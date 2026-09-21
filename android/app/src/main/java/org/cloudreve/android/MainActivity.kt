package org.cloudreve.android

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import com.google.android.play.core.appupdate.AppUpdateManagerFactory
import com.google.android.play.core.appupdate.AppUpdateOptions
import com.google.android.play.core.install.model.AppUpdateType
import com.google.android.play.core.install.model.UpdateAvailability
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import kotlinx.coroutines.launch
import org.cloudreve.android.api.RefreshTokenRequest
import org.cloudreve.android.ui.files.FilesScreen
import org.cloudreve.android.ui.files.FilesViewModel
import org.cloudreve.android.ui.login.LoginScreen
import org.cloudreve.android.ui.login.LoginViewModel
import org.cloudreve.android.ui.theme.CloudreveTheme
import org.cloudreve.android.util.CrUri
import org.cloudreve.android.work.UploadWorker

class MainActivity : ComponentActivity() {

    private val updateLauncher =
        registerForActivityResult(ActivityResultContracts.StartIntentSenderForResult()) { }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        handleShareIntent(intent)
        setContent { App() }
    }

    override fun onResume() {
        super.onResume()
        checkForPlayUpdate()
    }

    // Ask Play for an update; if one is available and this install came from
    // Play, surface the in-app prompt once per version, then hand off to the
    // official IMMEDIATE update sheet.
    private fun checkForPlayUpdate() {
        val manager = AppUpdateManagerFactory.create(this)
        manager.appUpdateInfo.addOnSuccessListener { info ->
            if (info.updateAvailability() != UpdateAvailability.UPDATE_AVAILABLE ||
                !info.isUpdateTypeAllowed(AppUpdateType.IMMEDIATE)
            ) return@addOnSuccessListener
            val version = info.availableVersionCode()
            val prefs = getSharedPreferences("updates", MODE_PRIVATE)
            if (prefs.getInt("prompted_version", -1) == version) return@addOnSuccessListener
            prefs.edit().putInt("prompted_version", version).apply()
            (application as CloudreveApp).playUpdateAvailable.value = info
        }
    }

    fun launchPlayUpdate() {
        val info = (application as CloudreveApp).playUpdateAvailable.value ?: return
        runCatching {
            AppUpdateManagerFactory.create(this).startUpdateFlowForResult(
                info,
                updateLauncher,
                AppUpdateOptions.newBuilder(AppUpdateType.IMMEDIATE).build(),
            )
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handleShareIntent(intent)
    }

    // Share-sheet target + OAuth deep link.
    private fun handleShareIntent(intent: Intent?) {
        if (intent?.action == Intent.ACTION_VIEW) {
            val data = intent.data
            if (data?.scheme == "cloudreve" && data.host == "mount") {
                val code = data.getQueryParameter("code")
                val state = data.getQueryParameter("state") ?: ""
                if (!code.isNullOrEmpty()) {
                    (application as CloudreveApp).oauthCallback.value =
                        CloudreveApp.OAuthCallback(code, state)
                }
            }
            return
        }
        val uris = when (intent?.action) {
            Intent.ACTION_SEND -> listOfNotNull(
                @Suppress("DEPRECATION")
                intent.getParcelableExtra<Uri>(Intent.EXTRA_STREAM)
            )
            Intent.ACTION_SEND_MULTIPLE ->
                @Suppress("DEPRECATION")
                intent.getParcelableArrayListExtra<Uri>(Intent.EXTRA_STREAM)
            else -> null
        }
        if (!uris.isNullOrEmpty()) {
            UploadWorker.enqueue(this, uris, CrUri.MY_PREFIX)
        }
    }
}

private class AppViewModelFactory(private val app: CloudreveApp) : ViewModelProvider.Factory {
    @Suppress("UNCHECKED_CAST")
    override fun <T : ViewModel> create(modelClass: Class<T>): T = when {
        modelClass.isAssignableFrom(LoginViewModel::class.java) ->
            LoginViewModel(app.sessionManager, app.apiClient) as T
        modelClass.isAssignableFrom(FilesViewModel::class.java) ->
            FilesViewModel(app.fileRepository, app.favoritesStore) as T
        else -> throw IllegalArgumentException("Unknown ViewModel ${modelClass.name}")
    }
}

@Composable
fun App() {
    val context = LocalContext.current
    val app = context.applicationContext as CloudreveApp
    val factory = remember { AppViewModelFactory(app) }
    val loggedIn by app.sessionManager.isLoggedIn.collectAsState(initial = null)
    val nav = rememberNavController()
    val scope = rememberCoroutineScope()

    val signOut: () -> Unit = {
        scope.launch {
            runCatching {
                app.apiClient.service()
                    .signOut(RefreshTokenRequest(app.sessionManager.refreshTokenNow()))
            }
            app.sessionManager.clear()
        }
    }

    CloudreveTheme {
        val updateInfo by app.playUpdateAvailable.collectAsState()
        if (updateInfo != null) {
            AlertDialog(
                onDismissRequest = { app.playUpdateAvailable.value = null },
                title = { Text("Update available") },
                text = { Text("A new version of Cloudreve Mobile is available on Google Play.") },
                confirmButton = {
                    TextButton(onClick = {
                        app.playUpdateAvailable.value = null
                        (context as? MainActivity)?.launchPlayUpdate()
                    }) { Text("Update") }
                },
                dismissButton = {
                    TextButton(onClick = { app.playUpdateAvailable.value = null }) {
                        Text("Later")
                    }
                },
            )
        }
        when (loggedIn) {
            null -> Unit // session store still loading; keep frame empty
            false -> NavHost(navController = nav, startDestination = "login") {
                composable("login") {
                    LoginScreen(
                        viewModel = viewModel(factory = factory),
                        onLoggedIn = { nav.navigate("files") { popUpTo(0) } },
                    )
                }
            }
            true -> NavHost(navController = nav, startDestination = "files") {
                composable("files") {
                    FilesScreen(
                        viewModel = viewModel(factory = factory),
                        onSignOut = signOut,
                    )
                }
            }
        }
    }
}
