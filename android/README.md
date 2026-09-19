# Cloudreve Android

Native Android client for Cloudreve. Kotlin + Jetpack Compose, Material 3.

Status: **Phase E — v0 implemented.** Sign in, browse, download, upload,
rename, mkdir, delete. See `../ROADMAP.md` for the remaining scope.

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

## Planned next

- Camera auto-upload, offline-favorite files, DocumentsProvider
- Thumbnails via `/file/thumb`, share-link creation, search
- No iOS. Ever.
