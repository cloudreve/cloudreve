package org.cloudreve.android.data

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.longPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map

private val Context.sessionStore by preferencesDataStore(name = "session")

/**
 * SessionManager persists the server base URL and token pair. It is the
 * single source of truth for "which account, on which server".
 */
class SessionManager(private val context: Context) {

    private object Keys {
        val serverUrl = stringPreferencesKey("server_url")
        val accessToken = stringPreferencesKey("access_token")
        val refreshToken = stringPreferencesKey("refresh_token")
        // Epoch millis when access_token expires; 0 = unknown.
        val accessExpiresAt = longPreferencesKey("access_expires_at")
        val userEmail = stringPreferencesKey("user_email")
        val userNick = stringPreferencesKey("user_nick")
    }

    val serverUrl: Flow<String> = context.sessionStore.data.map { it[Keys.serverUrl] ?: "" }
    val accessToken: Flow<String> = context.sessionStore.data.map { it[Keys.accessToken] ?: "" }
    val userEmail: Flow<String> = context.sessionStore.data.map { it[Keys.userEmail] ?: "" }
    val userNick: Flow<String> = context.sessionStore.data.map { it[Keys.userNick] ?: "" }
    val isLoggedIn: Flow<Boolean> = context.sessionStore.data.map {
        !it[Keys.refreshToken].isNullOrEmpty() && !it[Keys.serverUrl].isNullOrEmpty()
    }

    suspend fun serverUrlNow(): String = serverUrl.first()
    suspend fun accessTokenNow(): String = accessToken.first()
    suspend fun refreshTokenNow(): String =
        context.sessionStore.data.first()[Keys.refreshToken] ?: ""
    suspend fun accessExpiresAtNow(): Long =
        context.sessionStore.data.first()[Keys.accessExpiresAt] ?: 0L

    suspend fun saveServerUrl(url: String) {
        context.sessionStore.edit { it[Keys.serverUrl] = url.trimEnd('/') }
    }

    suspend fun saveSession(
        accessToken: String,
        refreshToken: String,
        accessExpiresAt: Long,
        email: String,
        nick: String,
    ) {
        context.sessionStore.edit {
            it[Keys.accessToken] = accessToken
            it[Keys.refreshToken] = refreshToken
            it[Keys.accessExpiresAt] = accessExpiresAt
            it[Keys.userEmail] = email
            it[Keys.userNick] = nick
        }
    }

    suspend fun saveTokens(accessToken: String, refreshToken: String, accessExpiresAt: Long) {
        context.sessionStore.edit {
            it[Keys.accessToken] = accessToken
            it[Keys.refreshToken] = refreshToken
            it[Keys.accessExpiresAt] = accessExpiresAt
        }
    }

    suspend fun clear() {
        context.sessionStore.edit { it.clear() }
    }
}
