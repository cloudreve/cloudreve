package org.cloudreve.android

import android.app.Application
import kotlinx.coroutines.flow.MutableStateFlow
import org.cloudreve.android.api.ApiClient
import org.cloudreve.android.data.CameraUploadSettings
import org.cloudreve.android.data.FavoritesStore
import org.cloudreve.android.data.FileRepository
import org.cloudreve.android.data.SessionManager
import org.cloudreve.android.data.TaskNotifySettings

class CloudreveApp : Application() {

    /** OAuth code+state arriving via the `cloudreve://mount` deep link. */
    data class OAuthCallback(val code: String, val state: String)
    val oauthCallback = MutableStateFlow<OAuthCallback?>(null)

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
        private set

    override fun onCreate() {
        super.onCreate()
        sessionManager = SessionManager(this)
        apiClient = ApiClient(sessionManager)
        fileRepository = FileRepository(apiClient)
        cameraUploadSettings = CameraUploadSettings(this)
        favoritesStore = FavoritesStore(this)
        taskNotifySettings = TaskNotifySettings(this)
    }
}
