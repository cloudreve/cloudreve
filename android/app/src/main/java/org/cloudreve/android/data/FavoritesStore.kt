package org.cloudreve.android.data

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val Context.favoritesStore by preferencesDataStore(name = "favorites")

@Serializable
data class FavoriteEntry(
    /** cloudreve:// URI of the remote file. */
    val path: String,
    val name: String,
    val size: Long,
    /** Absolute local path under filesDir/offline. */
    val localFile: String,
    val savedAt: Long,
    /** Remote updated_at at download time — staleness signal for refresh. */
    val remoteUpdatedAt: String = "",
)

/**
 * Offline-favorite registry: remote path → downloaded local copy.
 * JSON list in DataStore — jarvis: ceiling a few hundred entries,
 * upgrade to Room if sync status or per-file history is ever needed.
 */
class FavoritesStore(private val context: Context) {

    private object Keys {
        val entriesJson = stringPreferencesKey("entries_json")
    }

    private val json = Json { ignoreUnknownKeys = true }

    val entries: Flow<List<FavoriteEntry>> = context.favoritesStore.data.map {
        decode(it[Keys.entriesJson])
    }

    suspend fun listNow(): List<FavoriteEntry> = entries.first()

    suspend fun put(entry: FavoriteEntry) {
        context.favoritesStore.edit { prefs ->
            val list = decode(prefs[Keys.entriesJson])
                .filterNot { it.path == entry.path } + entry
            prefs[Keys.entriesJson] = json.encodeToString(list)
        }
    }

    suspend fun remove(path: String) {
        context.favoritesStore.edit { prefs ->
            val list = decode(prefs[Keys.entriesJson]).filterNot { it.path == path }
            prefs[Keys.entriesJson] = json.encodeToString(list)
        }
    }

    private fun decode(raw: String?): List<FavoriteEntry> =
        raw?.let {
            runCatching { json.decodeFromString<List<FavoriteEntry>>(it) }.getOrNull()
        } ?: emptyList()
}
