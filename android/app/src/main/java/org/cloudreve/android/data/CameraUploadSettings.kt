package org.cloudreve.android.data

import android.content.Context
import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.longPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.core.stringSetPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import org.cloudreve.android.util.CrUri

private val Context.cameraUploadStore by preferencesDataStore(name = "camera_upload")

/**
 * Camera auto-upload preferences plus the incremental-sync bookkeeping
 * (which MediaStore items were already uploaded or permanently failed).
 * ID sets are bounded so storage stays small; evicted entries may rarely
 * cause a duplicate upload of an old item.
 */
class CameraUploadSettings(private val context: Context) {

    private object Keys {
        val enabled = booleanPreferencesKey("enabled")
        val remoteFolder = stringPreferencesKey("remote_folder")
        val wifiOnly = booleanPreferencesKey("wifi_only")
        val includeVideos = booleanPreferencesKey("include_videos")
        val syncedIds = stringSetPreferencesKey("synced_ids")
        val failedIds = stringSetPreferencesKey("failed_ids")
        val lastSyncTs = longPreferencesKey("last_sync_ts")
        val lastSyncAt = longPreferencesKey("last_sync_at")
    }

    data class Snapshot(
        val enabled: Boolean = false,
        val remoteFolder: String = DEFAULT_FOLDER,
        val wifiOnly: Boolean = true,
        val includeVideos: Boolean = true,
        val lastSyncAt: Long = 0L,
    )

    val snapshot: Flow<Snapshot> = context.cameraUploadStore.data.map {
        Snapshot(
            enabled = it[Keys.enabled] ?: false,
            remoteFolder = it[Keys.remoteFolder] ?: DEFAULT_FOLDER,
            wifiOnly = it[Keys.wifiOnly] ?: true,
            includeVideos = it[Keys.includeVideos] ?: true,
            lastSyncAt = it[Keys.lastSyncAt] ?: 0L,
        )
    }

    suspend fun snapshotNow(): Snapshot = snapshot.first()

    suspend fun setEnabled(enabled: Boolean) {
        context.cameraUploadStore.edit { it[Keys.enabled] = enabled }
    }

    suspend fun setRemoteFolder(folder: String) {
        context.cameraUploadStore.edit { it[Keys.remoteFolder] = folder.trim() }
    }

    suspend fun setWifiOnly(wifiOnly: Boolean) {
        context.cameraUploadStore.edit { it[Keys.wifiOnly] = wifiOnly }
    }

    suspend fun setIncludeVideos(include: Boolean) {
        context.cameraUploadStore.edit { it[Keys.includeVideos] = include }
    }

    suspend fun syncedIdsNow(): Set<String> =
        context.cameraUploadStore.data.first()[Keys.syncedIds] ?: emptySet()

    suspend fun failedIdsNow(): Set<String> =
        context.cameraUploadStore.data.first()[Keys.failedIds] ?: emptySet()

    suspend fun lastSyncTsNow(): Long =
        context.cameraUploadStore.data.first()[Keys.lastSyncTs] ?: 0L

    suspend fun markSynced(id: String, dateAddedSec: Long) {
        context.cameraUploadStore.edit { prefs ->
            val synced = (prefs[Keys.syncedIds] ?: emptySet()).toMutableSet()
            synced.add(id)
            prefs[Keys.syncedIds] = synced.bounded()
            if (dateAddedSec > (prefs[Keys.lastSyncTs] ?: 0L)) {
                prefs[Keys.lastSyncTs] = dateAddedSec
            }
            prefs[Keys.lastSyncAt] = System.currentTimeMillis()
        }
    }

    suspend fun markFailed(id: String) {
        context.cameraUploadStore.edit { prefs ->
            val failed = (prefs[Keys.failedIds] ?: emptySet()).toMutableSet()
            failed.add(id)
            prefs[Keys.failedIds] = failed.bounded()
        }
    }

    private fun Set<String>.bounded(): Set<String> =
        if (size <= MAX_TRACKED_IDS) this else this.drop(size - MAX_TRACKED_IDS).toSet()

    companion object {
        const val DEFAULT_FOLDER = "${CrUri.MY_PREFIX}/Photos"
        private const val MAX_TRACKED_IDS = 2000
    }
}
