# Cloudreve Android

Native Android client for Cloudreve. Kotlin + Jetpack Compose, Material 3.

Status: **Phase E — v0.1 implemented.** Sign in, browse, download, upload,
rename, mkdir, delete, thumbnails, share-link creation. See `../ROADMAP.md`
for the remaining scope.

## Build

```bash
cd android
ANDROID_HOME=$HOME/Android/Sdk ./gradlew :app:assembleDebug
# APK: app/build/outputs/apk/debug/app-debug.apk
```

CI runs `assembleDebug` on every PR.

## What's implemented

- `api/v4` auth: password login → access/refresh tokens, single-flight
  refresh on 401, server-side revocation on sign-out
- File browser: folder navigation, cursor pagination, pull-refresh,
  mkdir / rename / delete / download (cache + FileProvider open)
- Uploads: SAF document picker or share-sheet target, WorkManager
  background worker with progress notification, chunked through the
  upload-session flow (local + presigned remote policies)
- Thumbnails via `/file/thumb` (Coil, lazily resolved per row)
- Share-link creation (`PUT /share`), link copied to clipboard
- Full-text search (`GET /file/search`): query bar in the files screen,
  offset pagination, result rows show parent path + content snippet;
  folder hits navigate into place

## Planned next

- Camera auto-upload, offline-favorite files, DocumentsProvider
- No iOS. Ever.
