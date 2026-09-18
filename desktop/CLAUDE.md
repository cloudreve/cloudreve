# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cloudreve Desktop is a Tauri-based Windows desktop application that synchronizes files with a Cloudreve cloud drive server using the Windows Cloud Files API (cfapi). It provides:
- Real-time bidirectional file synchronization
- On-demand file hydration (files appear locally but are downloaded only when accessed)
- Windows Explorer shell integration (context menus, thumbnails, custom states)
- Multiple storage provider support for uploads (S3, OneDrive, Qiniu, Upyun, local)
- System tray application with React-based UI

## Build Commands

```bash
# Backend (Rust) - run from project root
cargo build                    # Build all workspace crates
cargo build --release          # Release build
cargo check                    # Check for compilation errors
cargo test                     # Run all tests
cargo test --package cloudreve-sync --lib inventory::db::tests  # Run specific module tests
cargo test --package cloudreve-api  # Run API crate tests

# Frontend (React/TypeScript) - run from ui/ directory
cd ui
yarn install                   # Install dependencies
yarn dev                       # Start Vite dev server (localhost:5173)
yarn build                     # Build for production
yarn lint                      # Run ESLint

# Full Tauri Application - run from project root
cargo tauri dev                # Development mode with hot reload
cargo tauri build              # Production build
```

## Architecture

### Workspace Structure

```
├── src-tauri/           # Tauri application shell
├── crates/
│   ├── cloudreve-sync/  # Core sync service (main logic)
│   ├── cloudreve-api/   # Async REST client for Cloudreve server
│   └── win32_notif/     # Windows notification utilities
└── ui/                  # React frontend (Vite + MUI)
```

### Tauri Layer (`src-tauri/`)

- `lib.rs`: Application entry, initializes sync service, sets up system tray, spawns event bridge
- `commands.rs`: Tauri IPC commands exposed to frontend (`list_drives`, `add_drive`, `remove_drive`, etc.)
- `event_handler.rs`: Bridges `EventBroadcaster` events to Tauri frontend events

**Initialization Flow**: App starts → system tray setup → async `init_sync_service()` spawned → DriveManager created → shell services initialized → event bridge connects EventBroadcaster to Tauri

### Core Sync Module (`crates/cloudreve-sync/`)

**Drive Management:**
- `drive/manager.rs`: Central `DriveManager` coordinating all mounted drives via command channel
- `drive/mounts.rs`: Individual mount point (`Mount`) handling
- `drive/callback.rs`: Windows Cloud Filter callbacks (`SyncFilter` trait)
- `drive/sync.rs`: Sync logic and placeholder/metadata conversion
- `drive/remote_events.rs`: SSE handling for server-pushed changes

**Windows Cloud Files API (`cfapi/`):**
- `filter/`: Callback traits (`SyncFilter`, `Filter`) and request handling
- `placeholder.rs`, `placeholder_file.rs`: Cloud file placeholder management
- `root/`: Sync root registration and connection

**Persistence (`inventory/`):**
- SQLite via Diesel ORM at `~/.cloudreve/meta.db`
- Stores file metadata, task queue, upload sessions, drive properties
- Migrations in `migrations/inventory/`

**Uploads (`uploader/`):**
- Chunked upload with provider backends: S3, OneDrive, Qiniu, Upyun, local
- Encryption and resumable upload support

**Shell Extensions (`shellext/`):**
- Context menus, thumbnails, custom file states, status UI
- `shell_service.rs`: COM-based Windows Explorer integration

### Frontend (`ui/`)

React 19 + TypeScript + MUI + Vite application:
- `src/pages/popup/`: Main tray popup (drive list, task progress)
- `src/pages/AddDrive.tsx`: Add drive wizard
- `src/pages/settings/`: Settings pages
- i18n via react-i18next, translations in `ui/public/locales/`

### Key Patterns

**Command Channels**: `DriveManager` and `Mount` use `mpsc::UnboundedSender` for async command dispatch from shell extensions and Tauri commands.

**Callback Threading**: Windows Cloud Filter callbacks run on OS threads, using `blocking_recv()` on oneshot channels to await async operations.

**Event Broadcasting**: `EventBroadcaster` (tokio broadcast channel) pushes events to both the Tauri frontend (via event bridge) and any SSE subscribers.

**Global State**: `APP_STATE` (tokio `OnceCell`) holds initialized `DriveManager`, `EventBroadcaster`, and service handles for the application lifetime.

## Windows-Specific Notes

- Requires Windows Cloud Files API (`windows` crate with extensive feature flags in Cargo.toml)
- COM shell extensions for Explorer integration
- Deep link protocol: `cloudreve://`
- Single instance enforcement via `tauri-plugin-single-instance`

## Database Migrations

Migrations are embedded and run automatically on startup. To add a new migration:
1. Create folder in `migrations/inventory/` (e.g., `0005_new_table/`)
2. Add `up.sql` and `down.sql` files
3. Use idempotent SQL (`IF NOT EXISTS`) for clean upgrades

## Localization

- Backend: `rust-i18n` macro with translations in `locales/`
- Frontend: `react-i18next` with translations in `ui/public/locales/{locale}/common.json`
