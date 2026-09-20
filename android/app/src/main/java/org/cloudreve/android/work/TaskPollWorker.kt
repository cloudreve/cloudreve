package org.cloudreve.android.work

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import org.cloudreve.android.CloudreveApp
import java.util.concurrent.TimeUnit

/**
 * Periodically polls /workflow for the user's tasks and posts a
 * notification when a task reaches a terminal state (completed, error,
 * canceled). The first run after enabling only seeds the seen-map so
 * history doesn't flood the shade.
 */
class TaskPollWorker(context: Context, params: WorkerParameters) :
    CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val app = applicationContext as CloudreveApp
        val settings = app.taskNotifySettings
        if (!settings.enabledNow()) return Result.success()

        createChannel()

        val seen = settings.seenNow()
        val tasks = runCatching {
            app.apiClient.service().listTasks(pageSize = 20, category = "general")
                .body()?.data?.tasks.orEmpty() +
                app.apiClient.service().listTasks(pageSize = 20, category = "downloaded")
                    .body()?.data?.tasks.orEmpty()
        }.getOrDefault(emptyList())

        val fresh = seen.isNotEmpty()
        tasks.forEach { t ->
            val previous = seen[t.id]
            if (previous == t.status) return@forEach
            settings.markSeen(t.id, t.status)
            if (fresh && t.status in TERMINAL) {
                notify(t.id, t.type, t.status, t.error)
            }
        }
        return Result.success()
    }

    private fun notify(id: String, type: String, status: String, error: String?) {
        if (!NotificationManagerCompat.from(applicationContext).areNotificationsEnabled()) return
        val label = type.substringAfterLast('.').replace('_', ' ')
        val text = when (status) {
            "completed" -> "$label completed"
            "canceled" -> "$label canceled"
            else -> "$label failed" + (error?.let { ": $it" } ?: "")
        }
        val notification = NotificationCompat.Builder(applicationContext, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.stat_sys_download_done)
            .setContentTitle("Cloudreve task")
            .setContentText(text)
            .setAutoCancel(true)
            .build()
        NotificationManagerCompat.from(applicationContext)
            .notify(id.hashCode(), notification)
    }

    private fun createChannel() {
        val manager =
            applicationContext.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_ID, "Task updates", NotificationManager.IMPORTANCE_DEFAULT)
        )
    }

    companion object {
        private const val WORK_NAME = "task_poll_periodic"
        private const val CHANNEL_ID = "task_updates"
        private val TERMINAL = setOf("completed", "error", "canceled")

        private val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        fun apply(context: Context) {
            val wm = WorkManager.getInstance(context)
            val periodic = PeriodicWorkRequestBuilder<TaskPollWorker>(15, TimeUnit.MINUTES)
                .setConstraints(constraints)
                .build()
            wm.enqueueUniquePeriodicWork(WORK_NAME, ExistingPeriodicWorkPolicy.UPDATE, periodic)
            // Prime the seen-map immediately so enabling doesn't notify on history.
            wm.enqueueUniqueWork(
                "$WORK_NAME.once",
                ExistingWorkPolicy.REPLACE,
                OneTimeWorkRequestBuilder<TaskPollWorker>().setConstraints(constraints).build(),
            )
        }

        fun cancel(context: Context) {
            val wm = WorkManager.getInstance(context)
            wm.cancelUniqueWork(WORK_NAME)
            wm.cancelUniqueWork("$WORK_NAME.once")
        }
    }
}
