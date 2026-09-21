package org.cloudreve.android

import android.app.Application
import com.google.android.play.core.appupdate.AppUpdateInfo
import kotlinx.coroutines.flow.MutableStateFlow
import org.cloudreve.android.api.ApiClient
import org.cloudreve.android.data.CameraUploadSettings
import org.cloudreve.android.data.FavoritesStore
import org.cloudreve.android.data.FileRepository
import org.cloudreve.android.data.SessionManager
import org.cloudreve.android.data.TaskNotifySettings
import org.cloudreve.android.data.SyncFolderSettings

class CloudreveApp : Application() {

    /** OAuth code+state arriving via the `cloudreve://mount` deep link. */
    data class OAuthCallback(val code: String, val state: String)
    val oauthCallback = MutableStateFlow<OAuthCallback?>(null)

    /** Pending Play in-app update, set by MainActivity's update check. */
    val playUpdateAvailable = MutableStateFlow<AppUpdateInfo?>(null)

    lateinit var sessionManager: SessionManager
        private set
    lateinit var apiClient: ApiClient
        private set
    lateinit var fileRepository: FileRepository
        private set
    lateinit var cameraUploadSettings: CameraUploadSettings
    lateinit var favoritesStore: FavoritesStore
        private set
    lateinit var taskNotifySettings: TaskNotifySettings
    lateinit var syncFolderSettings: SyncFolderSettings
        private set

    override fun onCreate() {
        super.onCreate()
        sessionManager = SessionManager(this)
        apiClient = ApiClient(sessionManager)
        fileRepository = FileRepository(apiClient)
        cameraUploadSettings = CameraUploadSettings(this)
        favoritesStore = FavoritesStore(this)
        taskNotifySettings = TaskNotifySettings(this)
        syncFolderSettings = SyncFolderSettings(this)
    }
}
