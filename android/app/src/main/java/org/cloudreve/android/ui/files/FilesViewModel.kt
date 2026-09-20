package org.cloudreve.android.ui.files

import android.content.Context
import android.net.Uri
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import org.cloudreve.android.api.FileObject
import org.cloudreve.android.api.SearchHit
import org.cloudreve.android.data.FileRepository
import org.cloudreve.android.util.CrUri
import java.io.File
import java.io.FileOutputStream

data class FilesUiState(
    val currentUri: String = CrUri.MY_PREFIX,
    val files: List<FileObject> = emptyList(),
    val loading: Boolean = false,
    val loadingMore: Boolean = false,
    val nextToken: String? = null,
    val error: String? = null,
    val snackbar: String? = null,
    val searchQuery: String? = null,
    val searchResults: List<SearchHit> = emptyList(),
    val searchTotal: Long = 0,
    val searchLoading: Boolean = false,
    val searchLoadingMore: Boolean = false,
    val searchError: String? = null,
)

class FilesViewModel(private val repo: FileRepository) : ViewModel() {

    private val _state = MutableStateFlow(FilesUiState())
    val state: StateFlow<FilesUiState> = _state

    fun refresh() {
        val uri = _state.value.currentUri
        _state.value = _state.value.copy(loading = true, error = null, files = emptyList(), nextToken = null)
        viewModelScope.launch {
            try {
                val res = repo.list(uri)
                _state.value = _state.value.copy(
                    loading = false,
                    files = sortFiles(res.files),
                    nextToken = res.pagination.nextToken,
                )
            } catch (e: Exception) {
                _state.value = _state.value.copy(loading = false, error = e.message)
            }
        }
    }

    fun loadMore() {
        val s = _state.value
        val token = s.nextToken ?: return
        if (s.loadingMore) return
        _state.value = s.copy(loadingMore = true)
        viewModelScope.launch {
            try {
                val res = repo.list(s.currentUri, pageToken = token)
                _state.value = _state.value.copy(
                    loadingMore = false,
                    files = sortFiles(_state.value.files + res.files),
                    nextToken = res.pagination.nextToken,
                )
            } catch (e: Exception) {
                _state.value = _state.value.copy(loadingMore = false, snackbar = e.message)
            }
        }
    }

    fun navigateTo(file: FileObject) {
        if (!file.isFolder) return
        _state.value = _state.value.copy(currentUri = file.path)
        refresh()
    }

    fun navigateUp(): Boolean {
        val cur = _state.value.currentUri
        if (CrUri.isRoot(cur)) return false
        _state.value = _state.value.copy(currentUri = CrUri.parent(cur))
        refresh()
        return true
    }

    fun search(query: String) {
        val q = query.trim()
        if (q.isEmpty()) return
        _state.value = _state.value.copy(
            searchQuery = q,
            searchResults = emptyList(),
            searchTotal = 0,
            searchLoading = true,
            searchError = null,
        )
        viewModelScope.launch {
            try {
                val res = repo.search(q, 0)
                _state.value = _state.value.copy(
                    searchLoading = false,
                    searchResults = res.hits,
                    searchTotal = res.total,
                )
            } catch (e: Exception) {
                _state.value = _state.value.copy(searchLoading = false, searchError = e.message)
            }
        }
    }

    fun searchMore() {
        val s = _state.value
        val q = s.searchQuery ?: return
        if (s.searchLoadingMore || s.searchResults.size >= s.searchTotal) return
        _state.value = s.copy(searchLoadingMore = true)
        viewModelScope.launch {
            try {
                val res = repo.search(q, s.searchResults.size)
                _state.value = _state.value.copy(
                    searchLoadingMore = false,
                    searchResults = _state.value.searchResults + res.hits,
                )
            } catch (e: Exception) {
                _state.value = _state.value.copy(searchLoadingMore = false, snackbar = e.message)
            }
        }
    }

    fun clearSearch() {
        _state.value = _state.value.copy(
            searchQuery = null,
            searchResults = emptyList(),
            searchTotal = 0,
            searchLoading = false,
            searchLoadingMore = false,
            searchError = null,
        )
    }

    fun mkdir(name: String) {
        if (name.isBlank()) return
        viewModelScope.launch {
            runCatching { repo.mkdir(CrUri.join(_state.value.currentUri, name)) }
                .onSuccess { refresh() }
                .onFailure { _state.value = _state.value.copy(snackbar = it.message) }
        }
    }

    fun rename(file: FileObject, newName: String) {
        if (newName.isBlank() || newName == file.name) return
        viewModelScope.launch {
            runCatching { repo.rename(file.path, newName) }
                .onSuccess { refresh() }
                .onFailure { _state.value = _state.value.copy(snackbar = it.message) }
        }
    }

    fun delete(file: FileObject) {
        viewModelScope.launch {
            runCatching { repo.delete(listOf(file.path)) }
                .onSuccess { refresh() }
                .onFailure { _state.value = _state.value.copy(snackbar = it.message) }
        }
    }

    /** Downloads into app cache and returns the file for the caller to open/share. */
    suspend fun downloadToCache(context: Context, file: FileObject): File {
        val url = repo.downloadUrl(file.path)
        val resp = repo.download(url)
        if (!resp.isSuccessful) throw Exception("Download failed: HTTP ${resp.code()}")
        val dir = File(context.cacheDir, "downloads").apply { mkdirs() }
        val out = File(dir, file.name)
        resp.body()!!.byteStream().use { input ->
            FileOutputStream(out).use { input.copyTo(it) }
        }
        return out
    }

    suspend fun shareLink(file: FileObject): String = repo.createShare(file.path)

    suspend fun thumbUrl(file: FileObject): String? =
        if (file.thumbnail == true) repo.thumbUrl(file.path) else null

    fun consumeSnackbar() {
        _state.value = _state.value.copy(snackbar = null)
    }

    private fun sortFiles(files: List<FileObject>): List<FileObject> =
        files.sortedWith(compareByDescending<FileObject> { it.isFolder }.thenBy { it.name.lowercase() })
}
