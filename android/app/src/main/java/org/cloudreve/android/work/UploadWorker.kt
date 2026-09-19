package org.cloudreve.android.work

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.pm.ServiceInfo
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.work.CoroutineWorker
import androidx.work.Data
import androidx.work.ForegroundInfo
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import org.cloudreve.android.CloudreveApp
import org.cloudreve.android.R
import java.io.File
import java.io.FileOutputStream

/**
 * UploadWorker copies SAF/content URIs into cache, then streams them through
 * the Cloudreve upload-session flow. WorkManager survives process death for
 * large transfers; the foreground notification shows progress.
 */
class UploadWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val uris = inputData.getStringArray(KEY_URIS)?.toList() ?: return Result.failure()
        val parentUri = inputData.getString(KEY_PARENT_URI) ?: return Result.failure()
        val app = applicationContext as CloudreveApp

        createChannel()
        var failed = 0
        uris.forEachIndexed { i, uriString ->
            val uri = Uri.parse(uriString)
            val name = resolveDisplayName(uri)
            setForeground(progressInfo("Uploading $name", i, uris.size))
            try {
                val tmp = stageToCache(uri, name)
                try {
                    app.fileRepository.uploadFile(tmp, parentUri) { _, _ -> }
                } finally {
                    tmp.delete()
                }
            } catch (_: Exception) {
                failed++
            }
        }
        return if (failed == 0) Result.success() else Result.failure(
            Data.Builder().putInt(KEY_FAILED, failed).build()
        )
    }

    private fun resolveDisplayName(uri: Uri): String {
        val cursor = applicationContext.contentResolver.query(uri, null, null, null, null)
        cursor?.use {
            val idx = it.getColumnIndex(android.provider.OpenableColumns.DISPLAY_NAME)
            if (it.moveToFirst() && idx >= 0) return it.getString(idx)
        }
        return uri.lastPathSegment?.substringAfterLast('/') ?: "upload.bin"
    }

    private fun stageToCache(uri: Uri, name: String): File {
        val dir = File(applicationContext.cacheDir, "upload_staging").apply { mkdirs() }
        val tmp = File(dir, name)
        applicationContext.contentResolver.openInputStream(uri)!!.use { input ->
            FileOutputStream(tmp).use { input.copyTo(it) }
        }
        return tmp
    }

    private fun progressInfo(name: String, index: Int, total: Int): ForegroundInfo {
        val notification = NotificationCompat.Builder(applicationContext, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.stat_sys_upload)
            .setContentTitle("Cloudreve upload")
            .setContentText("$name (${index + 1}/$total)")
            .setOngoing(true)
            .build()
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            ForegroundInfo(NOTIF_ID, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC)
        } else {
            ForegroundInfo(NOTIF_ID, notification)
        }
    }

    private fun createChannel() {
        val manager = applicationContext.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_ID, "Uploads", NotificationManager.IMPORTANCE_LOW)
        )
    }

    companion object {
        const val KEY_URIS = "uris"
        const val KEY_PARENT_URI = "parent_uri"
        const val KEY_FAILED = "failed"
        private const val CHANNEL_ID = "uploads"
        private const val NOTIF_ID = 41

        fun enqueue(context: Context, uris: List<Uri>, parentUri: String) {
            val request = OneTimeWorkRequestBuilder<UploadWorker>()
                .setInputData(
                    Data.Builder()
                        .putStringArray(KEY_URIS, uris.map { it.toString() }.toTypedArray())
                        .putString(KEY_PARENT_URI, parentUri)
                        .build()
                )
                .build()
            WorkManager.getInstance(context).enqueue(request)
        }
    }
}
