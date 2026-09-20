package org.cloudreve.android.ui.login

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import org.cloudreve.android.api.ApiClient
import org.cloudreve.android.api.PasswordLoginRequest
import org.cloudreve.android.data.SessionManager
import java.net.URLEncoder
import java.security.MessageDigest
import java.security.SecureRandom

data class LoginUiState(
    val serverUrl: String = "",
    val email: String = "",
    val password: String = "",
    val loading: Boolean = false,
    val error: String? = null,
    val loggedIn: Boolean = false,
)

class LoginViewModel(
    private val session: SessionManager,
    private val api: ApiClient,
) : ViewModel() {

    private val _state = MutableStateFlow(LoginUiState())
    val state: StateFlow<LoginUiState> = _state

    fun onServerUrlChange(v: String) = _state.value.let { _state.value = it.copy(serverUrl = v, error = null) }
    fun onEmailChange(v: String) = _state.value.let { _state.value = it.copy(email = v, error = null) }
    fun onPasswordChange(v: String) = _state.value.let { _state.value = it.copy(password = v, error = null) }

    fun login() {
        val s = _state.value
        if (s.serverUrl.isBlank() || s.email.isBlank() || s.password.isBlank()) {
            _state.value = s.copy(error = "Fill in server, email and password")
            return
        }
        _state.value = s.copy(loading = true, error = null)
        viewModelScope.launch {
            try {
                session.saveServerUrl(s.serverUrl)
                val resp = api.service().login(PasswordLoginRequest(s.email, s.password))
                val body = resp.body()
                if (!resp.isSuccessful || body == null || body.code != 0 || body.data == null) {
                    throw Exception(body?.msg ?: "Login failed (HTTP ${resp.code()})")
                }
                val login = body.data
                session.saveSession(
                    accessToken = login.token.accessToken,
                    refreshToken = login.token.refreshToken,
                    accessExpiresAt = ApiClient.parseExpiresAt(login.token.accessExpires),
                    email = login.user.email,
                    nick = login.user.nickname,
                )
                _state.value = _state.value.copy(loading = false, loggedIn = true)
            } catch (e: Exception) {
                _state.value = _state.value.copy(loading = false, error = e.message ?: "Login failed")
            }
        }
    }

    /**
     * Builds the server-side authorize URL for browser sign-in and stashes
     * the PKCE verifier + state. The consent page bounces the code to the
     * app through the `cloudreve://mount` deep link. Returns null when the
     * server URL is missing.
     */
    fun startOAuth(): String? {
        val s = _state.value
        if (s.serverUrl.isBlank()) {
            _state.value = s.copy(error = "Enter the server URL first")
            return null
        }
        val verifier = randomBase64Url(48)
        val challenge = base64Url(MessageDigest.getInstance("SHA-256").digest(verifier.toByteArray()))
        val state = randomBase64Url(24)
        viewModelScope.launch {
            session.saveServerUrl(s.serverUrl)
            session.savePendingOAuth(verifier, state)
        }
        val enc: (String) -> String = { URLEncoder.encode(it, "UTF-8") }
        return "${s.serverUrl.trimEnd('/')}/session/authorize" +
            "?response_type=code&client_id=${enc(OAUTH_CLIENT_ID)}" +
            "&redirect_uri=${enc(OAUTH_REDIRECT)}&state=${enc(state)}" +
            "&scope=${enc(OAUTH_SCOPES)}&code_challenge=${enc(challenge)}" +
            "&code_challenge_method=S256"
    }

    /** Exchanges the deep-link code for tokens. Called by the UI on `cloudreve://mount`. */
    fun completeOAuth(code: String, state: String) {
        _state.value = _state.value.copy(loading = true, error = null)
        viewModelScope.launch {
            try {
                val pending = session.takePendingOAuth()
                    ?: throw Exception("No pending sign-in — start again")
                if (pending.second != state) {
                    throw Exception("OAuth state mismatch")
                }
                val resp = api.service().oauthToken(
                    clientId = OAUTH_CLIENT_ID,
                    grantType = "authorization_code",
                    code = code,
                    redirectUri = OAUTH_REDIRECT,
                    codeVerifier = pending.first,
                )
                val token = resp.body()
                if (!resp.isSuccessful || token == null || token.accessToken.isEmpty()) {
                    throw Exception("Token exchange failed (HTTP ${resp.code()})")
                }
                // The token response carries no user; fetch profile from userinfo.
                val info = runCatching {
                    api.service().oauthUserInfo("Bearer ${token.accessToken}").body()
                }.getOrNull()
                session.saveSession(
                    accessToken = token.accessToken,
                    refreshToken = token.refreshToken,
                    accessExpiresAt = if (token.expiresIn > 0) {
                        System.currentTimeMillis() + token.expiresIn * 1000
                    } else 0L,
                    email = info?.email ?: "",
                    nick = info?.name?.ifEmpty { info.preferredUsername } ?: "",
                )
                _state.value = _state.value.copy(loading = false, loggedIn = true)
            } catch (e: Exception) {
                _state.value = _state.value.copy(loading = false, error = e.message ?: "Sign-in failed")
            }
        }
    }

    private fun randomBase64Url(bytes: Int): String =
        base64Url(ByteArray(bytes).also { SecureRandom().nextBytes(it) })

    private fun base64Url(data: ByteArray): String =
        android.util.Base64.encodeToString(data, android.util.Base64.URL_SAFE or android.util.Base64.NO_PADDING or android.util.Base64.NO_WRAP)
            .trim()

    companion object {
        // Built-in public OAuth client (seeded by the server migration).
        private const val OAUTH_CLIENT_ID = "393a1839-f52e-498e-9972-e77cc2241eee"
        private const val OAUTH_REDIRECT = "/callback/desktop"
        private const val OAUTH_SCOPES =
            "profile email openid offline_access UserInfo.Write UserSecurityInfo.Write Workflow.Write Files.Write Shares.Write"
    }
}
