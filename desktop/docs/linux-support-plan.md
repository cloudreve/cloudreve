# Linux Support Plan

## Phase 1: Baseline Linux Support

Status: implemented.

- Build `cloudreve-sync` and the Tauri app on Linux.
- Keep Windows CFAPI and shell extension code behind `cfg(windows)`.
- Provide non-Windows compatibility layers for shell services, notifications, app resource roots, and CFAPI-shaped file metadata.
- On Linux and other non-Windows platforms, use full sync only:
  - Create local directories for remote folders.
  - Update inventory for remote entries.
  - Queue downloads for remote files instead of creating placeholders.
  - Keep local filesystem watching and upload/download task queues active.
- Do not expose placeholder semantics on Linux yet.

## Phase 2: Linux Full Sync Only

Status: implemented.

- Linux does not implement Windows-style on-demand placeholder sync.
- KDE does not receive a separate placeholder backend.
- All Linux desktop environments use the Phase 1 full sync path.
- Linux sync roots are ordinary local directories watched by the filesystem watcher.
- Remote files are downloaded fully instead of represented as dehydrated placeholders.
- Placeholder semantics remain Windows-only through CFAPI.

## Phase 3: UI And Packaging

Status: implemented.

- Add Linux autostart support through freedesktop `.desktop` files. Implemented.
- Add Linux packaging metadata. Implemented with a freedesktop `.desktop` entry template.
- Add tests for Linux full sync behavior. Implemented:
  - Non-Windows remote file metadata commits update inventory without creating placeholder files.
  - Non-Windows remote folder metadata commits create local directories.
  - Linux autostart entries quote and escape executable paths correctly.
