package org.cloudreve.android.work

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.pm.ServiceInfo
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.documentfile.provider.DocumentFile
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
import org.cloudreve.android.util.CrUri
import java.io.File
import java.io.FileOutputStream
import java.util.concurrent.TimeUnit

/**
 * Periodically mirrors a user-picked SAF directory tree into a remote
 * Cloudreve folder (upload-only — remote files are never deleted).
 * Files unchanged since their last upload are skipped via a
 * "<uri>|<mtime>|<size>" signature set kept in SyncFolderSettings.
 * Subdirectory structure is mirrored; remote folders are created lazily
 * with mkdir errors swallowed (already-exists is expected).
 */
class SyncFolderWorker(context: Context, params: WorkerParameters) :
    CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val app = applicationContext as CloudreveApp
        val settings = app.syncFolderSettings
        val cfg = settings.snapshotNow()
        val treeUri = cfg.treeUri?.let { Uri.parse(it) }
        if (!cfg.enabled || treeUri == null) return Result.success()

        val root = DocumentFile.fromTreeUri(applicationContext, treeUri)
            ?: return Result.failure()

        createChannel()

        val synced = settings.syncedNow()
        val ensuredDirs = mutableSetOf<String>()
        var uploaded = 0

        walk(root, "").forEach { entry ->
            if (uploaded >= MAX_PER_RUN) return@forEach
            if (entry.signature in synced) return@forEach

            setForeground(progressInfo(entry.name, uploaded))
            try {
                ensureRemoteDirs(app, cfg.remoteFolder, entry.relativeDir, ensuredDirs)
                val remoteParent = CrUri.join(cfg.remoteFolder, entry.relativeDir)
                val tmp = stageToCache(entry)
                try {
                    app.fileRepository.uploadFile(tmp, remoteParent) { _, _ -> }
                    settings.markSynced(entry.signature)
                    uploaded++
                } finally {
                    tmp.delete()
                }
            } catch (_: Exception) {
                // Skipped for this run — no failure set here, so a transient
                // failure simply retries on the next pass.
            }
        }
        return Result.success()
    }

    private data class Entry(
        val doc: DocumentFile,
        val name: String,
        val relativeDir: String,
        val signature: String,
    )

    /** Flattens the tree into uploadable file entries with relative dirs. */
    private fun walk(root: DocumentFile, prefix: String): List<Entry> {
        val out = mutableListOf<Entry>()
        val stack = ArrayDeque<Pair<DocumentFile, String>>()
        stack.add(root to prefix)
        while (stack.isNotEmpty()) {
            val (dir, rel) = stack.removeLast()
            dir.listFiles().forEach { child ->
                val name = child.name ?: return@forEach
                when {
                    child.isDirectory -> stack.add(child to if (rel.isEmpty()) name else "$rel/$name")
                    child.isFile -> out.add(
                        Entry(
                            doc = child,
                            name = name,
                            relativeDir = rel,
                            signature = "${child.uri}|${child.lastModified()}|${child.length()}",
                        )
                    )
                }
            }
        }
        return out
    }

    private suspend fun ensureRemoteDirs(
        app: CloudreveApp,
        base: String,
        relativeDir: String,
        ensured: MutableSet<String>,
    ) {
        if (relativeDir.isEmpty()) return
        var current = base
        relativeDir.split('/').forEach { segment ->
            current = CrUri.join(current, segment)
            if (ensured.add(current)) {
                runCatching { app.fileRepository.mkdir(current) }
            }
        }
    }

    private fun stageToCache(entry: Entry): File {
        val dir = File(applicationContext.cacheDir, "sync_staging").apply { mkdirs() }
        val tmp = File(dir, "sync-${entry.signature.hashCode()}-${entry.name}")
        applicationContext.contentResolver.openInputStream(entry.doc.uri)!!.use { input ->
            FileOutputStream(tmp).use { input.copyTo(it) }
        }
        return tmp
    }

    private fun progressInfo(text: String, done: Int): ForegroundInfo {
        val notification = NotificationCompat.Builder(applicationContext, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.stat_sys_upload)
            .setContentTitle("Cloudreve folder sync")
            .setContentText(if (done == 0) text else "$text ($done uploaded)")
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
            NotificationChannel(CHANNEL_ID, "Folder sync", NotificationManager.IMPORTANCE_LOW)
        )
    }

    companion object {
        private const val WORK_NAME = "sync_folder_periodic"
        private const val WORK_ONCE = "sync_folder_once"
        private const val CHANNEL_ID = "sync_folder"
        private const val NOTIF_ID = 43
        private const val MAX_PER_RUN = 100

        private fun constraints(wifiOnly: Boolean): Constraints =
            Constraints.Builder()
                .setRequiredNetworkType(
                    if (wifiOnly) NetworkType.UNMETERED else NetworkType.CONNECTED
                )
                .setRequiresBatteryNotLow(true)
                .build()

        fun apply(context: Context, wifiOnly: Boolean) {
            val wm = WorkManager.getInstance(context)
            val periodic = PeriodicWorkRequestBuilder<SyncFolderWorker>(15, TimeUnit.MINUTES)
                .setConstraints(constraints(wifiOnly))
                .build()
            wm.enqueueUniquePeriodicWork(WORK_NAME, ExistingPeriodicWorkPolicy.UPDATE, periodic)
            wm.enqueueUniqueWork(
                WORK_ONCE,
                ExistingWorkPolicy.REPLACE,
                OneTimeWorkRequestBuilder<SyncFolderWorker>()
                    .setConstraints(constraints(wifiOnly))
                    .build(),
            )
        }

        fun cancel(context: Context) {
            val wm = WorkManager.getInstance(context)
            wm.cancelUniqueWork(WORK_NAME)
            wm.cancelUniqueWork(WORK_ONCE)
        }

        fun syncNow(context: Context, wifiOnly: Boolean) {
            WorkManager.getInstance(context).enqueueUniqueWork(
                WORK_ONCE,
                ExistingWorkPolicy.REPLACE,
                OneTimeWorkRequestBuilder<SyncFolderWorker>()
                    .setConstraints(constraints(wifiOnly))
                    .build(),
            )
        }
    }
}
