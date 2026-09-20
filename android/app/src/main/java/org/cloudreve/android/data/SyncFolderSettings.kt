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

private val Context.syncFolderStore by preferencesDataStore(name = "sync_folder")

/**
 * Local sync folder preferences: a user-picked SAF tree mirrored
 * (upload-only) into a remote Cloudreve folder by SyncFolderWorker.
 * `synced` holds "<document uri>|<lastModified>|<size>" signatures so
 * unchanged files are skipped; bounded like the camera-upload ID sets.
 */
class SyncFolderSettings(private val context: Context) {

    private object Keys {
        val enabled = booleanPreferencesKey("enabled")
        val treeUri = stringPreferencesKey("tree_uri")
        val remoteFolder = stringPreferencesKey("remote_folder")
        val wifiOnly = booleanPreferencesKey("wifi_only")
        val synced = stringSetPreferencesKey("synced")
        val lastSyncAt = longPreferencesKey("last_sync_at")
    }

    data class Snapshot(
        val enabled: Boolean = false,
        val treeUri: String? = null,
        val remoteFolder: String = DEFAULT_FOLDER,
        val wifiOnly: Boolean = true,
        val lastSyncAt: Long = 0L,
    )

    val snapshot: Flow<Snapshot> = context.syncFolderStore.data.map {
        Snapshot(
            enabled = it[Keys.enabled] ?: false,
            treeUri = it[Keys.treeUri],
            remoteFolder = it[Keys.remoteFolder] ?: DEFAULT_FOLDER,
            wifiOnly = it[Keys.wifiOnly] ?: true,
            lastSyncAt = it[Keys.lastSyncAt] ?: 0L,
        )
    }

    suspend fun snapshotNow(): Snapshot = snapshot.first()

    suspend fun setEnabled(enabled: Boolean) {
        context.syncFolderStore.edit { it[Keys.enabled] = enabled }
    }

    suspend fun setTreeUri(uri: String?) {
        context.syncFolderStore.edit { prefs ->
            if (uri == null) prefs.remove(Keys.treeUri) else prefs[Keys.treeUri] = uri
        }
    }

    suspend fun setRemoteFolder(folder: String) {
        context.syncFolderStore.edit { it[Keys.remoteFolder] = folder.trim() }
    }

    suspend fun setWifiOnly(wifiOnly: Boolean) {
        context.syncFolderStore.edit { it[Keys.wifiOnly] = wifiOnly }
    }

    suspend fun syncedNow(): Set<String> =
        context.syncFolderStore.data.first()[Keys.synced] ?: emptySet()

    suspend fun markSynced(signature: String) {
        context.syncFolderStore.edit { prefs ->
            val synced = (prefs[Keys.synced] ?: emptySet()).toMutableSet()
            synced.add(signature)
            prefs[Keys.synced] = synced.bounded()
            prefs[Keys.lastSyncAt] = System.currentTimeMillis()
        }
    }

    private fun Set<String>.bounded(): Set<String> =
        if (size <= MAX_TRACKED) this else this.drop(size - MAX_TRACKED).toSet()

    companion object {
        const val DEFAULT_FOLDER = "${CrUri.MY_PREFIX}/Sync"
        private const val MAX_TRACKED = 4000
    }
}
