package org.cloudreve.android.data

import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.RequestBody.Companion.toRequestBody
import org.cloudreve.android.api.ApiClient
import org.cloudreve.android.api.CreateFileRequest
import org.cloudreve.android.api.CreateUploadSessionRequest
import org.cloudreve.android.api.DeleteFileRequest
import org.cloudreve.android.api.FileObject
import org.cloudreve.android.api.FileUrlRequest
import org.cloudreve.android.api.ListFileResponse
import org.cloudreve.android.api.RenameFileRequest
import org.cloudreve.android.api.SearchResponse
import org.cloudreve.android.api.ShareCreateRequest
import org.cloudreve.android.api.UploadSessionResponse
import org.cloudreve.android.util.CrUri
import retrofit2.Response
import java.io.File

class ApiException(val code: Int, message: String) : Exception(message)

class FileRepository(private val api: ApiClient) {

    private fun <T> Response<org.cloudreve.android.api.ApiResponse<T>>.unwrap(): T {
        if (!isSuccessful) throw ApiException(code(), "HTTP ${code()}")
        val body = body() ?: throw ApiException(-1, "Empty response")
        if (body.code != 0 || body.data == null) {
            throw ApiException(body.code, body.msg.ifEmpty { "API error ${body.code}" })
        }
        return body.data
    }

    private fun <T> Response<org.cloudreve.android.api.ApiResponse<T>>.unwrapEmpty() {
        if (!isSuccessful) throw ApiException(code(), "HTTP ${code()}")
        val body = body() ?: throw ApiException(-1, "Empty response")
        if (body.code != 0) {
            throw ApiException(body.code, body.msg.ifEmpty { "API error ${body.code}" })
        }
    }

    suspend fun list(uri: String, pageToken: String? = null): ListFileResponse =
        api.service().listFiles(uri = uri, nextPageToken = pageToken).unwrap()

    suspend fun downloadUrl(uri: String): String =
        api.service().fileUrls(FileUrlRequest(listOf(uri), download = true))
            .unwrap().urls.firstOrNull()?.url
            ?: throw ApiException(-1, "No download URL returned")

    suspend fun download(url: String) = api.service().download(url)

    /** Resolves a thumbnail URL for the file, or null when none exists. */
    suspend fun thumbUrl(uri: String): String? =
        runCatching { api.service().thumb(uri).unwrap().url }.getOrNull()

    /** Creates a public share link and returns its absolute URL. */
    suspend fun createShare(uri: String): String {
        val id = api.service().createShare(ShareCreateRequest(uri = uri)).unwrap()
        val base = api.serverBase()
        return "$base/s/$id"
    }

    suspend fun search(query: String, offset: Int = 0): SearchResponse =
        api.service().searchFiles(query = query, offset = offset).unwrap()

    suspend fun mkdir(uri: String) {
        api.service().createFile(CreateFileRequest(uri = uri, type = "folder")).unwrapEmpty()
    }

    suspend fun rename(uri: String, newName: String) {
        api.service().renameFile(RenameFileRequest(uri = uri, newName = newName)).unwrapEmpty()
    }

    suspend fun delete(uris: List<String>) {
        api.service().deleteFiles(DeleteFileRequest(uris = uris)).unwrapEmpty()
    }

    suspend fun createUploadSession(
        uri: String,
        size: Long,
        lastModified: Long,
        mimeType: String,
    ): UploadSessionResponse =
        api.service().createUploadSession(
            CreateUploadSessionRequest(
                uri = uri,
                size = size,
                lastModified = lastModified,
                mimeType = mimeType,
            )
        ).unwrap()

    /**
     * Streams a local file through the upload-session flow: create session,
     * PUT/POST each chunk to its resolved URL, then call completeURL when the
     * policy requires an explicit finalize step (e.g. S3 multipart).
     */
    suspend fun uploadFile(
        localFile: File,
        remoteParentUri: String,
        onProgress: (sent: Long, total: Long) -> Unit = { _, _ -> },
    ): UploadSessionResponse {
        val targetUri = CrUri.join(remoteParentUri, localFile.name)
        val mime = java.net.URLConnection.guessContentTypeFromName(localFile.name)
            ?: "application/octet-stream"
        val session = createUploadSession(
            uri = targetUri,
            size = localFile.length(),
            lastModified = localFile.lastModified(),
            mimeType = mime,
        )

        if (session.rapidUploaded) return session

        val urls = session.uploadUrls
        // Local/slave policies return no presigned URLs: chunks go to the
        // conventional POST /api/v4/file/upload/{session}/{index} endpoint and
        // the server completes the session once every chunk has landed.
        val conventional = urls.isNullOrEmpty()
        val chunkSize = if (session.chunkSize > 0) session.chunkSize else localFile.length().coerceAtLeast(1)
        val totalChunks = ((localFile.length() + chunkSize - 1) / chunkSize).toInt().coerceAtLeast(1)
        val svc = api.service()
        var sent = 0L
        localFile.inputStream().buffered().use { input ->
            val buffer = ByteArray(chunkSize.coerceAtMost(64L * 1024 * 1024).toInt())
            for (index in 0 until totalChunks) {
                val read = input.readNBytes(buffer, 0, buffer.size)
                if (index == 0 && read == 0 && localFile.length() > 0) break
                val target = if (conventional) {
                    session_serverBase() + "/api/v4/file/upload/${session.sessionId}/$index"
                } else {
                    if (index >= urls!!.size) break
                    resolveUrl(urls[index])
                }
                val body = buffer.copyOf(read)
                    .toRequestBody(mime.toMediaTypeOrNull())
                val resp = if (target.contains("/file/upload/")) {
                    svc.uploadChunk(target, body)
                } else {
                    // Presigned third-party URLs are conventionally PUT.
                    svc.uploadChunkPut(target, body)
                }
                if (!resp.isSuccessful) {
                    throw ApiException(resp.code(), "Chunk $index upload failed: HTTP ${resp.code()}")
                }
                resp.body()?.close()
                sent += read
                onProgress(sent, localFile.length())
                if (read <= 0) break
            }
        }

        if (session.completeUrl.isNotEmpty()) {
            val resp = svc.completeUpload(resolveUrl(session.completeUrl))
            if (!resp.isSuccessful) {
                throw ApiException(resp.code(), "Upload completion failed: HTTP ${resp.code()}")
            }
            resp.body()?.close()
        }
        return session
    }

    private suspend fun resolveUrl(url: String): String {
        if (url.startsWith("http")) return url
        return session_serverBase() + url
    }

    private suspend fun session_serverBase(): String {
        val base = api.serverBase()
        return if (base.endsWith('/')) base.dropLast(1) else base
    }
}
