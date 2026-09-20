package org.cloudreve.android.tile

import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import org.cloudreve.android.CloudreveApp
import org.cloudreve.android.work.CameraUploadWorker

/**
 * Quick Settings tile: shows whether camera backup is armed and toggles
 * it. Tap → backup on; tap again → off.
 */
class SyncTileService : TileService() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onStartListening() {
        super.onStartListening()
        scope.launch { refresh() }
    }

    override fun onClick() {
        super.onClick()
        val app = applicationContext as CloudreveApp
        scope.launch {
            val cam = app.cameraUploadSettings.snapshotNow()
            if (cam.enabled) {
                app.cameraUploadSettings.setEnabled(false)
                CameraUploadWorker.cancel(applicationContext)
            } else {
                app.cameraUploadSettings.setEnabled(true)
                CameraUploadWorker.apply(applicationContext, cam.wifiOnly)
            }
            refresh()
        }
    }

    private suspend fun refresh() {
        val app = applicationContext as CloudreveApp
        val enabled = app.cameraUploadSettings.snapshotNow().enabled
        qsTile?.apply {
            state = if (enabled) Tile.STATE_ACTIVE else Tile.STATE_INACTIVE
            subtitle = if (enabled) "Camera backup on" else "Camera backup off"
            updateTile()
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        scope.cancel()
    }
}
