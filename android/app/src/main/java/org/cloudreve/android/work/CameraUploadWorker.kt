package org.cloudreve.android.work

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.ContentUris
import android.content.Context
import android.content.pm.ServiceInfo
import android.os.Build
import android.provider.MediaStore
import androidx.core.app.NotificationCompat
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.ForegroundInfo
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import org.cloudreve.android.CloudreveApp
import java.io.File
import java.io.FileOutputStream
import java.util.concurrent.TimeUnit

/**
 * Periodically scans MediaStore for photos/videos captured since the last
 * sync and uploads them to the configured Cloudreve folder. A bounded set of
 * synced/failed MediaStore IDs prevents re-uploads; the date-added watermark
 * bounds each scan. Runs unmetered-only when the user chose Wi-Fi only.
 */
class CameraUploadWorker(context: Context, params: WorkerParameters) :
    CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val app = applicationContext as CloudreveApp
        val settings = app.cameraUploadSettings
        val cfg = settings.snapshotNow()
        if (!cfg.enabled) return Result.success()

        createChannel()

        val synced = settings.syncedIdsNow().toMutableSet()
        val failed = settings.failedIdsNow()
        val lastTs = settings.lastSyncTsNow()

        val items = queryNewMedia(lastTs, cfg.includeVideos)
            .filter { it.id !in synced && it.id !in failed }
            .take(MAX_PER_RUN)

        items.forEachIndexed { i, item ->
            setForeground(progressInfo("Backing up ${item.name}", i, items.size))
            try {
                val tmp = stageToCache(item)
                try {
                    app.fileRepository.uploadFile(tmp, cfg.remoteFolder) { _, _ -> }
                    settings.markSynced(item.id, item.dateAddedSec)
                } finally {
                    tmp.delete()
                }
            } catch (_: Exception) {
                // Permanently skipped: retried MediaStore rows that keep
                // failing would otherwise block every later photo.
                settings.markFailed(item.id)
            }
        }
        return Result.success()
    }

    private data class MediaItem(
        val id: String,
        val contentUri: android.net.Uri,
        val name: String,
        val dateAddedSec: Long,
    )

    private fun queryNewMedia(sinceSec: Long, includeVideos: Boolean): List<MediaItem> {
        val out = mutableListOf<MediaItem>()
        val collection = MediaStore.Files.getContentUri(MediaStore.VOLUME_EXTERNAL)
        val projection = arrayOf(
            MediaStore.Files.FileColumns._ID,
            MediaStore.Files.FileColumns.DISPLAY_NAME,
            MediaStore.Files.FileColumns.DATE_ADDED,
            MediaStore.Files.FileColumns.MEDIA_TYPE,
        )
        val mediaTypes = buildList {
            add(MediaStore.Files.FileColumns.MEDIA_TYPE_IMAGE)
            if (includeVideos) add(MediaStore.Files.FileColumns.MEDIA_TYPE_VIDEO)
        }
        val selection = "${MediaStore.Files.FileColumns.DATE_ADDED} > ? AND " +
            "${MediaStore.Files.FileColumns.MEDIA_TYPE} IN (${mediaTypes.joinToString(",") { "?" }})"
        val args = (listOf(sinceSec) + mediaTypes).map { it.toString() }.toTypedArray()
        val order = "${MediaStore.Files.FileColumns.DATE_ADDED} ASC"

        applicationContext.contentResolver.query(
            collection, projection, selection, args, order,
        )?.use { cursor ->
            val idIdx = cursor.getColumnIndexOrThrow(MediaStore.Files.FileColumns._ID)
            val nameIdx = cursor.getColumnIndexOrThrow(MediaStore.Files.FileColumns.DISPLAY_NAME)
            val dateIdx = cursor.getColumnIndexOrThrow(MediaStore.Files.FileColumns.DATE_ADDED)
            while (cursor.moveToNext()) {
                val id = cursor.getLong(idIdx)
                out.add(
                    MediaItem(
                        id = id.toString(),
                        contentUri = ContentUris.withAppendedId(collection, id),
                        name = cursor.getString(nameIdx) ?: "media_$id",
                        dateAddedSec = cursor.getLong(dateIdx),
                    )
                )
            }
        }
        return out
    }

    private fun stageToCache(item: MediaItem): File {
        val dir = File(applicationContext.cacheDir, "camera_staging").apply { mkdirs() }
        val tmp = File(dir, item.name)
        applicationContext.contentResolver.openInputStream(item.contentUri)!!.use { input ->
            FileOutputStream(tmp).use { input.copyTo(it) }
        }
        return tmp
    }

    private fun progressInfo(text: String, index: Int, total: Int): ForegroundInfo {
        val notification = NotificationCompat.Builder(applicationContext, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.stat_sys_upload)
            .setContentTitle("Cloudreve camera backup")
            .setContentText("$text (${index + 1}/$total)")
            .setOngoing(true)
            .build()
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            ForegroundInfo(NOTIF_ID, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC)
        } else {
            ForegroundInfo(NOTIF_ID, notification)
        }
    }

    private fun createChannel() {
        val manager =
            applicationContext.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_ID, "Camera backup", NotificationManager.IMPORTANCE_LOW)
        )
    }

    companion object {
        private const val WORK_NAME = "camera_upload_periodic"
        private const val WORK_ONCE = "camera_upload_once"
        private const val CHANNEL_ID = "camera_upload"
        private const val NOTIF_ID = 42
        private const val MAX_PER_RUN = 100

        private fun constraints(wifiOnly: Boolean): Constraints =
            Constraints.Builder()
                .setRequiredNetworkType(
                    if (wifiOnly) NetworkType.UNMETERED else NetworkType.CONNECTED
                )
                .setRequiresBatteryNotLow(true)
                .build()

        /** Schedules (or removes) the periodic sync and kicks an immediate run. */
        fun apply(context: Context, wifiOnly: Boolean) {
            val wm = WorkManager.getInstance(context)
            val periodic = PeriodicWorkRequestBuilder<CameraUploadWorker>(15, TimeUnit.MINUTES)
                .setConstraints(constraints(wifiOnly))
                .build()
            wm.enqueueUniquePeriodicWork(WORK_NAME, ExistingPeriodicWorkPolicy.UPDATE, periodic)
            wm.enqueueUniqueWork(
                WORK_ONCE,
                ExistingWorkPolicy.REPLACE,
                OneTimeWorkRequestBuilder<CameraUploadWorker>()
                    .setConstraints(constraints(wifiOnly))
                    .build(),
            )
        }

        fun cancel(context: Context) {
            val wm = WorkManager.getInstance(context)
            wm.cancelUniqueWork(WORK_NAME)
            wm.cancelUniqueWork(WORK_ONCE)
        }

        /** Manual "sync now" — the periodic schedule is untouched. */
        fun syncNow(context: Context, wifiOnly: Boolean) {
            WorkManager.getInstance(context).enqueueUniqueWork(
                WORK_ONCE,
                ExistingWorkPolicy.REPLACE,
                OneTimeWorkRequestBuilder<CameraUploadWorker>()
                    .setConstraints(constraints(wifiOnly))
                    .build(),
            )
        }
    }
}
