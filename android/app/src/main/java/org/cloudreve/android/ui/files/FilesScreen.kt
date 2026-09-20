package org.cloudreve.android.ui.files

import android.content.ActivityNotFoundException
import android.content.Intent
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.automirrored.filled.InsertDriveFile
import androidx.compose.material.icons.filled.CreateNewFolder
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.filled.DriveFileRenameOutline
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Share
import androidx.compose.material.icons.filled.UploadFile
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.produceState
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.core.content.FileProvider
import kotlinx.coroutines.launch
import org.cloudreve.android.CloudreveApp
import org.cloudreve.android.api.FileObject
import org.cloudreve.android.data.CameraUploadSettings
import org.cloudreve.android.util.CrUri
import org.cloudreve.android.work.CameraUploadWorker
import org.cloudreve.android.work.UploadWorker
import java.text.DecimalFormat

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FilesScreen(
    viewModel: FilesViewModel,
    onSignOut: () -> Unit,
) {
    val state by viewModel.state.collectAsState()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val snackbar = remember { SnackbarHostState() }
    val listState = rememberLazyListState()

    var mkdirOpen by remember { mutableStateOf(false) }
    var renameTarget by remember { mutableStateOf<FileObject?>(null) }
    var deleteTarget by remember { mutableStateOf<FileObject?>(null) }
    var searchOpen by remember { mutableStateOf(false) }
    var searchText by remember { mutableStateOf("") }
    var cameraSettingsOpen by remember { mutableStateOf(false) }
    val searchFocus = remember { FocusRequester() }

    val inSearch = state.searchQuery != null

    val uploadLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.OpenMultipleDocuments()
    ) { uris ->
        if (uris.isNotEmpty()) {
            UploadWorker.enqueue(context, uris, state.currentUri)
            scope.launch { snackbar.showSnackbar("Upload queued (${uris.size} file${if (uris.size > 1) "s" else ""})") }
        }
    }

    LaunchedEffect(Unit) { viewModel.refresh() }

    LaunchedEffect(state.snackbar) {
        state.snackbar?.let {
            snackbar.showSnackbar(it)
            viewModel.consumeSnackbar()
        }
    }

    // Infinite scroll — load next page when the end is near.
    LaunchedEffect(listState) {
        snapshotFlow { listState.layoutInfo.visibleItemsInfo.lastOrNull()?.index }
            .collect { last ->
                if (last == null) return@collect
                if (inSearch) {
                    if (last >= state.searchResults.size - 5) viewModel.searchMore()
                } else if (last >= state.files.size - 5) {
                    viewModel.loadMore()
                }
            }
    }

    LaunchedEffect(searchOpen) {
        if (searchOpen) searchFocus.requestFocus()
    }

    fun closeSearch() {
        viewModel.clearSearch()
        searchOpen = false
        searchText = ""
    }

    BackHandler(enabled = inSearch || !CrUri.isRoot(state.currentUri)) {
        if (inSearch) closeSearch() else viewModel.navigateUp()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    if (inSearch || searchOpen) {
                        OutlinedTextField(
                            value = searchText,
                            onValueChange = { searchText = it },
                            modifier = Modifier
                                .fillMaxWidth()
                                .focusRequester(searchFocus),
                            placeholder = { Text("Search files") },
                            singleLine = true,
                            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                            keyboardActions = KeyboardActions(
                                onSearch = { viewModel.search(searchText) },
                            ),
                        )
                    } else {
                        Column {
                            Text("Files")
                            Text(
                                pathLabel(state.currentUri),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                    }
                },
                navigationIcon = {
                    if (inSearch || searchOpen) {
                        IconButton(onClick = { closeSearch() }) {
                            Icon(Icons.Default.Close, contentDescription = "Close search")
                        }
                    } else if (!CrUri.isRoot(state.currentUri)) {
                        IconButton(onClick = { viewModel.navigateUp() }) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Up")
                        }
                    }
                },
                actions = {
                    if (!inSearch && !searchOpen) {
                        IconButton(onClick = { searchOpen = true }) {
                            Icon(Icons.Default.Search, contentDescription = "Search")
                        }
                        IconButton(onClick = { cameraSettingsOpen = true }) {
                            Icon(Icons.Default.Settings, contentDescription = "Camera backup")
                        }
                        IconButton(onClick = { mkdirOpen = true }) {
                            Icon(Icons.Default.CreateNewFolder, contentDescription = "New folder")
                        }
                        IconButton(onClick = { viewModel.refresh() }) {
                            Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                        }
                        IconButton(onClick = onSignOut) {
                            Icon(Icons.AutoMirrored.Filled.ExitToApp, contentDescription = "Sign out")
                        }
                    } else if (!inSearch) {
                        IconButton(onClick = { viewModel.search(searchText) }) {
                            Icon(Icons.Default.Search, contentDescription = "Search")
                        }
                    }
                },
            )
        },
        floatingActionButton = {
            FloatingActionButton(onClick = { uploadLauncher.launch(arrayOf("*/*")) }) {
                Icon(Icons.Default.UploadFile, contentDescription = "Upload")
            }
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = state.loading,
            onRefresh = { viewModel.refresh() },
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            when {
                inSearch -> {
                    SearchResults(
                        state = state,
                        viewModel = viewModel,
                        listState = listState,
                        onOpenFolder = { file ->
                            closeSearch()
                            viewModel.navigateTo(file)
                        },
                        onOpenFile = { file ->
                            scope.launch {
                                runCatching { viewModel.downloadToCache(context, file) }
                                    .onSuccess { local -> openFile(context, local) }
                                    .onFailure { snackbar.showSnackbar(it.message ?: "Download failed") }
                            }
                        },
                    )
                }
                state.loading && state.files.isEmpty() -> {
                    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator()
                    }
                }
                state.error != null -> {
                    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            Text(state.error ?: "Error", color = MaterialTheme.colorScheme.error)
                            TextButton(onClick = { viewModel.refresh() }) { Text("Retry") }
                        }
                    }
                }
                state.files.isEmpty() -> {
                    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        Text("Empty folder", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                else -> {
                    LazyColumn(state = listState, modifier = Modifier.fillMaxSize()) {
                        items(state.files, key = { it.id + it.path }) { file ->
                            FileRow(
                                file = file,
                                thumbUrl = thumbUrlFor(viewModel, file),
                                onClick = {
                                    if (file.isFolder) {
                                        viewModel.navigateTo(file)
                                    } else {
                                        scope.launch {
                                            runCatching { viewModel.downloadToCache(context, file) }
                                                .onSuccess { local -> openFile(context, local) }
                                                .onFailure { snackbar.showSnackbar(it.message ?: "Download failed") }
                                        }
                                    }
                                },
                                onRename = { renameTarget = file },
                                onDelete = { deleteTarget = file },
                                onShare = {
                                    scope.launch {
                                        runCatching { viewModel.shareLink(file) }
                                            .onSuccess { link ->
                                                val cm = context.getSystemService(android.content.Context.CLIPBOARD_SERVICE)
                                                    as android.content.ClipboardManager
                                                cm.setPrimaryClip(
                                                    android.content.ClipData.newPlainText("share", link)
                                                )
                                                snackbar.showSnackbar("Share link copied")
                                            }
                                            .onFailure { snackbar.showSnackbar(it.message ?: "Share failed") }
                                    }
                                },
                                onDownload = {
                                    scope.launch {
                                        runCatching { viewModel.downloadToCache(context, file) }
                                            .onSuccess { snackbar.showSnackbar("Saved to cache: ${it.name}") }
                                            .onFailure { snackbar.showSnackbar(it.message ?: "Download failed") }
                                    }
                                },
                            )
                            HorizontalDivider()
                        }
                        if (state.loadingMore) {
                            item {
                                Row(
                                    Modifier.fillMaxWidth().padding(12.dp),
                                    horizontalArrangement = Arrangement.Center,
                                ) { CircularProgressIndicator() }
                            }
                        }
                    }
                }
            }
        }
    }

    if (mkdirOpen) {
        TextFieldDialog(
            title = "New folder",
            label = "Folder name",
            confirm = "Create",
            onConfirm = {
                viewModel.mkdir(it)
                mkdirOpen = false
            },
            onDismiss = { mkdirOpen = false },
        )
    }

    renameTarget?.let { file ->
        TextFieldDialog(
            title = "Rename",
            label = "New name",
            initial = file.name,
            confirm = "Rename",
            onConfirm = {
                viewModel.rename(file, it)
                renameTarget = null
            },
            onDismiss = { renameTarget = null },
        )
    }

    deleteTarget?.let { file ->
        AlertDialog(
            onDismissRequest = { deleteTarget = null },
            title = { Text("Delete ${file.name}?") },
            text = { Text("The file will be moved to trash.") },
            confirmButton = {
                TextButton(onClick = {
                    viewModel.delete(file)
                    deleteTarget = null
                }) { Text("Delete", color = MaterialTheme.colorScheme.error) }
            },
            dismissButton = {
                TextButton(onClick = { deleteTarget = null }) { Text("Cancel") }
            },
        )
    }

    if (cameraSettingsOpen) {
        CameraUploadDialog(onDismiss = { cameraSettingsOpen = false })
    }
}

private fun pathLabel(uri: String): String {
    val path = uri.removePrefix(CrUri.MY_PREFIX).trim('/')
    return if (path.isEmpty()) "My files" else "My files / ${java.net.URLDecoder.decode(path, "UTF-8")}"
}

@Composable
private fun SearchResults(
    state: FilesUiState,
    viewModel: FilesViewModel,
    listState: androidx.compose.foundation.lazy.LazyListState,
    onOpenFolder: (FileObject) -> Unit,
    onOpenFile: (FileObject) -> Unit,
) {
    when {
        state.searchLoading -> {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
        }
        state.searchError != null -> {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(state.searchError ?: "Search failed", color = MaterialTheme.colorScheme.error)
                    TextButton(onClick = { viewModel.search(state.searchQuery ?: "") }) {
                        Text("Retry")
                    }
                }
            }
        }
        state.searchResults.isEmpty() -> {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text(
                    "No results for \"${state.searchQuery}\"",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        else -> {
            LazyColumn(state = listState, modifier = Modifier.fillMaxSize()) {
                items(state.searchResults, key = { it.file.id + it.file.path }) { hit ->
                    val file = hit.file
                    ListItem(
                        modifier = Modifier.clickable {
                            if (file.isFolder) onOpenFolder(file) else onOpenFile(file)
                        },
                        headlineContent = {
                            Text(file.name, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        },
                        supportingContent = {
                            Column {
                                Text(
                                    pathLabel(CrUri.parent(file.path)),
                                    style = MaterialTheme.typography.labelMedium,
                                    maxLines = 1,
                                    overflow = TextOverflow.Ellipsis,
                                )
                                if (hit.content.isNotBlank()) {
                                    Text(
                                        hit.content.trim(),
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        maxLines = 2,
                                        overflow = TextOverflow.Ellipsis,
                                    )
                                }
                            }
                        },
                        leadingContent = {
                            Icon(
                                if (file.isFolder) Icons.Default.Folder
                                else Icons.AutoMirrored.Filled.InsertDriveFile,
                                contentDescription = null,
                                tint = if (file.isFolder) MaterialTheme.colorScheme.primary
                                else MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        },
                    )
                    HorizontalDivider()
                }
                if (state.searchLoadingMore) {
                    item {
                        Row(
                            Modifier.fillMaxWidth().padding(12.dp),
                            horizontalArrangement = Arrangement.Center,
                        ) { CircularProgressIndicator() }
                    }
                }
            }
        }
    }
}

@Composable
private fun CameraUploadDialog(onDismiss: () -> Unit) {
    val context = LocalContext.current
    val app = context.applicationContext as CloudreveApp
    val settings = app.cameraUploadSettings
    val scope = rememberCoroutineScope()
    val snap by settings.snapshot.collectAsState(initial = null)
    var folderText by remember(snap?.remoteFolder) {
        mutableStateOf(snap?.remoteFolder ?: CameraUploadSettings.DEFAULT_FOLDER)
    }

    val permLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { grants ->
        if (grants.values.all { it }) {
            scope.launch {
                settings.setRemoteFolder(folderText)
                settings.setEnabled(true)
                CameraUploadWorker.apply(context, snap?.wifiOnly ?: true)
            }
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Camera backup") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Row(
                    Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("Back up new photos & videos", Modifier.weight(1f))
                    Switch(
                        checked = snap?.enabled == true,
                        onCheckedChange = { want ->
                            if (want) {
                                permLauncher.launch(mediaPermissions())
                            } else {
                                scope.launch {
                                    settings.setEnabled(false)
                                    CameraUploadWorker.cancel(context)
                                }
                            }
                        },
                    )
                }
                OutlinedTextField(
                    value = folderText,
                    onValueChange = { folderText = it },
                    label = { Text("Remote folder") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                Row(
                    Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("Wi-Fi only", Modifier.weight(1f))
                    Switch(
                        checked = snap?.wifiOnly ?: true,
                        onCheckedChange = { wifiOnly ->
                            scope.launch {
                                settings.setWifiOnly(wifiOnly)
                                if (snap?.enabled == true) {
                                    CameraUploadWorker.apply(context, wifiOnly)
                                }
                            }
                        },
                    )
                }
                Row(
                    Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("Include videos", Modifier.weight(1f))
                    Switch(
                        checked = snap?.includeVideos ?: true,
                        onCheckedChange = { scope.launch { settings.setIncludeVideos(it) } },
                    )
                }
                snap?.lastSyncAt?.takeIf { it > 0 }?.let {
                    Text(
                        "Last backup: " + java.text.DateFormat.getDateTimeInstance(
                            java.text.DateFormat.SHORT, java.text.DateFormat.SHORT,
                        ).format(java.util.Date(it)),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                scope.launch {
                    settings.setRemoteFolder(folderText)
                    CameraUploadWorker.syncNow(context, snap?.wifiOnly ?: true)
                }
            }) { Text("Sync now") }
        },
        dismissButton = {
            TextButton(onClick = {
                scope.launch { settings.setRemoteFolder(folderText) }
                onDismiss()
            }) { Text("Done") }
        },
    )
}

private fun mediaPermissions(): Array<String> =
    if (android.os.Build.VERSION.SDK_INT >= 33) {
        arrayOf(
            android.Manifest.permission.READ_MEDIA_IMAGES,
            android.Manifest.permission.READ_MEDIA_VIDEO,
        )
    } else {
        arrayOf(android.Manifest.permission.READ_EXTERNAL_STORAGE)
    }

private fun openFile(context: android.content.Context, local: java.io.File) {
    val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", local)
    val intent = Intent(Intent.ACTION_VIEW).apply {
        setDataAndType(uri, context.contentResolver.getType(uri) ?: "*/*")
        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
    }
    try {
        context.startActivity(intent)
    } catch (_: ActivityNotFoundException) {
        // No handler — the file is still in cache for manual access.
    }
}

@Composable
private fun thumbUrlFor(viewModel: FilesViewModel, file: FileObject): String? {
    val url by produceState<String?>(initialValue = null, file.id) {
        value = viewModel.thumbUrl(file)
    }
    return url
}

@Composable
private fun FileRow(
    file: FileObject,
    thumbUrl: String?,
    onClick: () -> Unit,
    onRename: () -> Unit,
    onDelete: () -> Unit,
    onShare: () -> Unit,
    onDownload: () -> Unit,
) {
    var menuOpen by remember { mutableStateOf(false) }
    ListItem(
        modifier = Modifier.clickable(onClick = onClick),
        headlineContent = { Text(file.name, maxLines = 1, overflow = TextOverflow.Ellipsis) },
        supportingContent = {
            Text(
                if (file.isFolder) "Folder" else formatSize(file.size),
                style = MaterialTheme.typography.labelMedium,
            )
        },
        leadingContent = {
            if (thumbUrl != null) {
                coil.compose.AsyncImage(
                    model = thumbUrl,
                    contentDescription = null,
                    modifier = Modifier.size(40.dp),
                    contentScale = androidx.compose.ui.layout.ContentScale.Crop,
                )
            } else {
                Icon(
                    if (file.isFolder) Icons.Default.Folder else Icons.AutoMirrored.Filled.InsertDriveFile,
                    contentDescription = null,
                    tint = if (file.isFolder) MaterialTheme.colorScheme.primary
                    else MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        trailingContent = {
            Box {
                IconButton(onClick = { menuOpen = true }) {
                    Icon(Icons.Default.MoreVert, contentDescription = "More")
                }
                DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                    if (!file.isFolder) {
                        DropdownMenuItem(
                            text = { Text("Download") },
                            leadingIcon = { Icon(Icons.Default.Download, null) },
                            onClick = { menuOpen = false; onDownload() },
                        )
                    }
                    DropdownMenuItem(
                        text = { Text("Share") },
                        leadingIcon = { Icon(Icons.Default.Share, null) },
                        onClick = { menuOpen = false; onShare() },
                    )
                    DropdownMenuItem(
                        text = { Text("Rename") },
                        leadingIcon = { Icon(Icons.Default.DriveFileRenameOutline, null) },
                        onClick = { menuOpen = false; onRename() },
                    )
                    DropdownMenuItem(
                        text = { Text("Delete") },
                        leadingIcon = { Icon(Icons.Default.Delete, null) },
                        onClick = { menuOpen = false; onDelete() },
                    )
                }
            }
        },
    )
}

@Composable
private fun TextFieldDialog(
    title: String,
    label: String,
    confirm: String,
    initial: String = "",
    onConfirm: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    var value by remember { mutableStateOf(initial) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            OutlinedTextField(
                value = value,
                onValueChange = { value = it },
                label = { Text(label) },
                singleLine = true,
            )
        },
        confirmButton = { TextButton(onClick = { onConfirm(value) }) { Text(confirm) } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

private fun formatSize(size: Long): String {
    if (size <= 0) return "0 B"
    val units = arrayOf("B", "KB", "MB", "GB", "TB")
    val group = (Math.log10(size.toDouble()) / Math.log10(1024.0)).toInt().coerceAtMost(units.size - 1)
    return DecimalFormat("#,##0.#").format(size / Math.pow(1024.0, group.toDouble())) + " " + units[group]
}
