package org.cloudreve.android.provider

import android.content.Context
import android.database.Cursor
import android.database.MatrixCursor
import android.graphics.Point
import android.os.CancellationSignal
import android.os.ParcelFileDescriptor
import android.provider.DocumentsContract
import android.provider.DocumentsContract.Document
import android.provider.DocumentsContract.Root
import android.provider.DocumentsProvider
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.runBlocking
import org.cloudreve.android.CloudreveApp
import org.cloudreve.android.R
import org.cloudreve.android.api.FileObject
import org.cloudreve.android.data.FileRepository
import org.cloudreve.android.util.CrUri
import java.io.File
import java.io.FileNotFoundException
import java.security.MessageDigest

/**
 * Exposes the Cloudreve filesystem to the system file picker / Files app.
 * Document IDs are the entries' `cloudreve://` URIs, matching the desktop
 * client's addressing scheme.
 *
 * Provider methods run on binder threads, so the suspend repository calls are
 * bridged with runBlocking. All network errors surface as
 * FileNotFoundException, which the Documents UI renders as "unavailable".
 */
class CloudreveDocumentsProvider : DocumentsProvider() {

    private val repo: FileRepository
        get() = (context!!.applicationContext as CloudreveApp).fileRepository

    private val ioScope = Dispatchers.IO

    override fun onCreate(): Boolean = true

    override fun queryRoots(projection: Array<out String>?): Cursor {
        val cursor = MatrixCursor(projection ?: DEFAULT_ROOT_COLUMNS)
        cursor.newRow().apply {
            add(Root.COLUMN_ROOT_ID, ROOT_ID)
            add(Root.COLUMN_DOCUMENT_ID, ROOT_ID)
            add(Root.COLUMN_TITLE, "Cloudreve")
            add(Root.COLUMN_SUMMARY, rootSummary())
            add(Root.COLUMN_FLAGS, Root.FLAG_SUPPORTS_SEARCH or Root.FLAG_SUPPORTS_IS_CHILD)
            add(Root.COLUMN_ICON, R.mipmap.ic_launcher)
            add(Root.COLUMN_MIME_TYPES, "*/*")
        }
        return cursor
    }

    private fun rootSummary(): String = runCatching {
        val base = runBlocking(ioScope) {
            (context!!.applicationContext as CloudreveApp).apiClient.serverBase()
        }
        java.net.URI(base).host
    }.getOrNull() ?: ""

    override fun isChildDocument(parentDocumentId: String, documentId: String): Boolean =
        documentId != parentDocumentId && documentId.startsWith("${parentDocumentId.trimEnd('/')}/")

    override fun queryDocument(documentId: String, projection: Array<out String>?): Cursor {
        val cursor = MatrixCursor(projection ?: DEFAULT_DOC_COLUMNS)
        if (documentId == ROOT_ID) {
            cursor.newRow().apply {
                add(Document.COLUMN_DOCUMENT_ID, ROOT_ID)
                add(Document.COLUMN_DISPLAY_NAME, "My files")
                add(Document.COLUMN_MIME_TYPE, Document.MIME_TYPE_DIR)
                add(Document.COLUMN_FLAGS, 0)
            }
            return cursor
        }
        val obj = findObject(documentId)
        cursor.addFileRow(obj)
        return cursor
    }

    override fun queryChildDocuments(
        parentDocumentId: String,
        projection: Array<out String>?,
        sortOrder: String?,
    ): Cursor {
        val cursor = MatrixCursor(projection ?: DEFAULT_DOC_COLUMNS)
        runBlocking(ioScope) {
            var token: String? = null
            do {
                val page = repo.list(parentDocumentId, token)
                page.files.forEach { cursor.addFileRow(it) }
                token = page.pagination.nextToken
            } while (token != null)
        }
        return cursor
    }

    override fun querySearchDocuments(
        rootId: String,
        query: String,
        projection: Array<out String>?,
    ): Cursor {
        val cursor = MatrixCursor(projection ?: DEFAULT_DOC_COLUMNS)
        runBlocking(ioScope) {
            repo.search(query).hits.forEach { cursor.addFileRow(it.file) }
        }
        return cursor
    }

    override fun openDocument(
        documentId: String,
        mode: String,
        signal: CancellationSignal?,
    ): ParcelFileDescriptor {
        if (mode.contains('w')) {
            throw FileNotFoundException("Cloudreve documents are read-only here")
        }
        val local = fetchToCache(documentId)
        return ParcelFileDescriptor.open(local, ParcelFileDescriptor.MODE_READ_ONLY)
    }

    override fun openDocumentThumbnail(
        documentId: String,
        sizeHint: Point?,
        signal: CancellationSignal?,
    ): android.content.res.AssetFileDescriptor {
        val local = runBlocking(ioScope) {
            val url = repo.thumbUrl(documentId)
                ?: throw FileNotFoundException("No thumbnail")
            downloadTo(url, cacheFile("thumb", documentId))
        }
        val pfd = ParcelFileDescriptor.open(local, ParcelFileDescriptor.MODE_READ_ONLY)
        return android.content.res.AssetFileDescriptor(pfd, 0, local.length())
    }

    override fun deleteDocument(documentId: String) {
        runBlocking(ioScope) { repo.delete(listOf(documentId)) }
        notifyChange(documentId)
    }

    override fun renameDocument(documentId: String, displayName: String): String {
        runBlocking(ioScope) { repo.rename(documentId, displayName) }
        val newId = CrUri.join(CrUri.parent(documentId), displayName)
        notifyChange(documentId)
        return newId
    }

    private fun notifyChange(documentId: String) {
        context!!.contentResolver.notifyChange(
            DocumentsContract.buildDocumentUri(authority(), documentId),
            null,
        )
    }

    private fun authority(): String = "${context!!.packageName}.documents"

    /** Looks up a file by listing its parent directory. */
    private fun findObject(documentId: String): FileObject = runBlocking(ioScope) {
        val parent = CrUri.parent(documentId)
        var token: String? = null
        do {
            val page = repo.list(parent, token)
            page.files.firstOrNull { it.path == documentId }?.let { return@runBlocking it }
            token = page.pagination.nextToken
        } while (token != null)
        throw FileNotFoundException("Not found: $documentId")
    }

    private fun fetchToCache(documentId: String): File = runBlocking(ioScope) {
        val url = repo.downloadUrl(documentId)
        downloadTo(url, cacheFile("doc", documentId))
    }

    private suspend fun downloadTo(url: String, dest: File): File {
        val resp = repo.download(url)
        if (!resp.isSuccessful) {
            resp.body()?.close()
            throw FileNotFoundException("Download failed: HTTP ${resp.code()}")
        }
        dest.parentFile?.mkdirs()
        resp.body()!!.byteStream().use { input ->
            dest.outputStream().use { output -> input.copyTo(output) }
        }
        return dest
    }

    private fun cacheFile(kind: String, documentId: String): File {
        val hash = MessageDigest.getInstance("SHA-256")
            .digest(documentId.toByteArray())
            .joinToString("") { "%02x".format(it) }
            .take(16)
        val ext = CrUri.fileName(documentId).substringAfterLast('.', "")
        val suffix = if (ext.isEmpty()) "" else ".$ext"
        return File(context!!.cacheDir, "docprovider/$kind-$hash$suffix")
    }

    private fun MatrixCursor.addFileRow(file: FileObject) {
        val isDir = file.isFolder
        val flags = Document.FLAG_SUPPORTS_DELETE or
            Document.FLAG_SUPPORTS_RENAME or
            (if (!isDir && file.thumbnail == true) Document.FLAG_SUPPORTS_THUMBNAIL else 0)
        newRow().apply {
            add(Document.COLUMN_DOCUMENT_ID, file.path)
            add(Document.COLUMN_DISPLAY_NAME, file.name)
            add(
                Document.COLUMN_MIME_TYPE,
                if (isDir) Document.MIME_TYPE_DIR
                else java.net.URLConnection.guessContentTypeFromName(file.name)
                    ?: "application/octet-stream",
            )
            add(Document.COLUMN_SIZE, if (isDir) null else file.size)
            add(Document.COLUMN_LAST_MODIFIED, parseInstant(file.updatedAt))
            add(Document.COLUMN_FLAGS, flags)
        }
    }

    private fun parseInstant(value: String): Long? {
        if (value.isBlank()) return null
        return runCatching { java.time.Instant.parse(value).toEpochMilli() }
            .recoverCatching { java.time.OffsetDateTime.parse(value).toInstant().toEpochMilli() }
            .getOrNull()
    }

    companion object {
        private const val ROOT_ID = CrUri.MY_PREFIX

        private val DEFAULT_ROOT_COLUMNS = arrayOf(
            Root.COLUMN_ROOT_ID,
            Root.COLUMN_DOCUMENT_ID,
            Root.COLUMN_TITLE,
            Root.COLUMN_SUMMARY,
            Root.COLUMN_FLAGS,
            Root.COLUMN_ICON,
            Root.COLUMN_MIME_TYPES,
        )

        private val DEFAULT_DOC_COLUMNS = arrayOf(
            Document.COLUMN_DOCUMENT_ID,
            Document.COLUMN_DISPLAY_NAME,
            Document.COLUMN_MIME_TYPE,
            Document.COLUMN_SIZE,
            Document.COLUMN_LAST_MODIFIED,
            Document.COLUMN_FLAGS,
        )
    }
}
