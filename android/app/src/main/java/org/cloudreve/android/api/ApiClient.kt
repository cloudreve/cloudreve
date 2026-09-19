package org.cloudreve.android.api

import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.json.Json
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import org.cloudreve.android.BuildConfig
import org.cloudreve.android.data.SessionManager
import retrofit2.Retrofit
import java.util.concurrent.TimeUnit

/**
 * ApiClient owns the OkHttp/Retrofit wiring for one server. Auth requests
 * carry `Authorization: Bearer <access>`; a 401 triggers a single-flight
 * refresh and one retry, mirroring the desktop client's behavior.
 */
class ApiClient(private val session: SessionManager) {

    private val json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
    }

    private val refreshMutex = Mutex()

    @Volatile
    private var cachedBaseUrl: String? = null

    @Volatile
    private var cachedService: ApiService? = null

    suspend fun serverBase(): String = session.serverUrlNow()

    suspend fun service(): ApiService {
        val base = session.serverUrlNow().ifEmpty { "http://localhost" } + "/"
        cachedService?.let { if (cachedBaseUrl == base) return it }
        return buildService(base).also {
            cachedService = it
            cachedBaseUrl = base
        }
    }

    private fun buildService(baseUrl: String): ApiService {
        val authInterceptor = Interceptor { chain ->
            val token = runBlocking { session.accessTokenNow() }
            val request = if (token.isNotEmpty()) {
                chain.request().newBuilder()
                    .header("Authorization", "Bearer $token")
                    .build()
            } else {
                chain.request()
            }
            chain.proceed(request)
        }

        val refreshAuthenticator = okhttp3.Authenticator { _, response ->
            // Give up after a couple of retries or when there is no session.
            if (response.request.header("X-Retry-Count")?.toIntOrNull() ?: 0 >= 1) return@Authenticator null
            val refreshToken = runBlocking { session.refreshTokenNow() }
            if (refreshToken.isEmpty()) return@Authenticator null

            val newTokens = runBlocking {
                refreshMutex.withLock {
                    // Another request may already have refreshed while we queued.
                    val current = session.accessTokenNow()
                    if (current.isNotEmpty() && current != response.request.header("Authorization")
                        ?.removePrefix("Bearer ")
                    ) {
                        return@withLock current
                    }
                    val refreshed = kotlin.runCatching {
                        buildService(cachedBaseUrl ?: return@withLock null)
                            .refreshToken(RefreshTokenRequest(refreshToken))
                    }.getOrNull()
                    val body = refreshed?.body()?.data
                    if (refreshed?.isSuccessful == true && body != null) {
                        val expiresAt = parseExpiresAt(body.accessExpires)
                        session.saveTokens(body.accessToken, body.refreshToken, expiresAt)
                        body.accessToken
                    } else {
                        null
                    }
                }
            } ?: return@Authenticator null

            response.request.newBuilder()
                .header("Authorization", "Bearer $newTokens")
                .header("X-Retry-Count", "1")
                .build()
        }

        val logging = HttpLoggingInterceptor().apply {
            level = if (BuildConfig.DEBUG) HttpLoggingInterceptor.Level.BASIC
            else HttpLoggingInterceptor.Level.NONE
        }

        val client = OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS)
            .writeTimeout(60, TimeUnit.SECONDS)
            .addInterceptor(authInterceptor)
            .addInterceptor(logging)
            .authenticator(refreshAuthenticator)
            .build()

        return Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(ApiService::class.java)
    }

    companion object {
        // OffsetDateTime parses both "…Z" and "+02:00" offsets; unknown → refresh early.
        fun parseExpiresAt(iso: String): Long = runCatching {
            java.time.OffsetDateTime.parse(iso).toInstant().toEpochMilli()
        }.getOrDefault(0L)
    }
}
