package org.cloudreve.android.data

import android.content.Context
import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val Context.taskNotifyStore by preferencesDataStore(name = "task_notify")

/**
 * Task-notification preferences: the on/off toggle plus a map of
 * task id → last seen status, so TaskPollWorker only notifies on
 * transitions to a terminal state. The map is pruned to the newest
 * entries to stay small.
 */
class TaskNotifySettings(private val context: Context) {

    private val json = Json { ignoreUnknownKeys = true }

    @Serializable
    private data class Seen(val map: Map<String, String> = emptyMap())

    private object Keys {
        val enabled = booleanPreferencesKey("enabled")
        val seen = stringPreferencesKey("seen")
    }

    val enabled: Flow<Boolean> = context.taskNotifyStore.data.map {
        it[Keys.enabled] ?: false
    }

    suspend fun enabledNow(): Boolean = enabled.first()

    suspend fun setEnabled(enabled: Boolean) {
        context.taskNotifyStore.edit { it[Keys.enabled] = enabled }
    }

    suspend fun seenNow(): Map<String, String> =
        runCatching {
            json.decodeFromString<Seen>(
                context.taskNotifyStore.data.first()[Keys.seen] ?: "{}"
            ).map
        }.getOrDefault(emptyMap())

    suspend fun markSeen(id: String, status: String) {
        context.taskNotifyStore.edit { prefs ->
            val map = runCatching {
                json.decodeFromString<Seen>(prefs[Keys.seen] ?: "{}").map
            }.getOrDefault(emptyMap()).toMutableMap()
            map.remove(id)
            map[id] = status
            if (map.size > MAX_SEEN) {
                // Keep the newest tail; insertion order preserved.
                val trimmed = map.entries.toList().takeLast(MAX_SEEN).associate { it.toPair() }
                map.clear(); map.putAll(trimmed)
            }
            prefs[Keys.seen] = json.encodeToString(Seen(map))
        }
    }

    companion object {
        private const val MAX_SEEN = 200
    }
}
