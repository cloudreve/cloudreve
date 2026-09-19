package org.cloudreve.android.ui.login

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import org.cloudreve.android.api.ApiClient
import org.cloudreve.android.api.PasswordLoginRequest
import org.cloudreve.android.data.SessionManager

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
}
