<h1 align="center">
  <br>
  Cloudreve — Community Fork
  <br>
</h1>
<h4 align="center">Self-hosted file management and sharing platform — fully open source, actively maintained.</h4>

<p align="center">
  <a href="https://github.com/Dvorinka/cloudreve/actions"><img src="https://img.shields.io/github/actions/workflow/status/Dvorinka/cloudreve/ci.yml?branch=master" alt="CI"></a>
  <a href="https://github.com/Dvorinka/cloudreve/releases"><img src="https://img.shields.io/github/v/release/Dvorinka/cloudreve?include_prereleases" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-GPL--3.0-blue" alt="GPL-3.0"></a>
</p>

> **This is an actively maintained fork of [cloudreve/cloudreve](https://github.com/cloudreve/cloudreve).**
> All original work is by the Cloudreve authors (cloudreve.org). This fork exists because upstream
> development had slowed: it continues the project as a **complete, fully open-source distribution** —
> backend, web frontend, desktop clients for Windows/macOS/Linux, and a native Android app — with every
> "Pro"-class feature reimplemented as free software. See [NOTICE](NOTICE) for attribution.

## What this fork does differently

- **No Pro tier.** The upsell UI (`ProChip`/`ProDialog`) is removed. Pro-class capabilities are being
  reimplemented as open features: share collaboration (upload/edit/preview-only/drop-box shares),
  OIDC SSO, delegated admin roles — already shipped.
- **Upstream issue backlog triaged and fixed.** All 137 migrated upstream issues are tracked in this
  repo's issue tracker; ~70% are closed. WebDAV mounts/read-only/collision handling, stuck uploads,
  recycle-bin fail-safes, SQLite WAL, MySQL `parseTime`, unix-socket migrations, and dozens more.
- **Security hardening on top of upstream's fixes.** OAuth public clients no longer ship hardcoded
  secrets (PKCE only, per RFC 8252); SSRF validation on remote-download URLs; delegated-admin access
  is audit-logged; auth endpoints are rate-limited; the downloader layer was reviewed for process
  execution and path-safety.
- **One monorepo.** Backend + frontend + desktop + Android live here; no submodules.
- **Tests and CI are real.** GitHub Actions run backend tests, frontend typecheck/build, and the
  desktop matrix (Windows/macOS/Linux) on every PR.

## Repository layout

```
.                    Go backend — Gin + ent ORM (SQLite/MySQL/PostgreSQL)
frontend/            Web SPA — React + TypeScript + Vite + MUI (vendored, no submodule)
desktop/             Desktop client — Tauri/Rust sync engine (Windows cfapi today;
                     macOS/Linux hydration providers on the roadmap)
android/             Native Android client — Kotlin + Jetpack Compose (scaffolded, Phase E)
.github/workflows/   CI (backend, frontend, desktop matrix) + release pipeline
```

## Features

- Storage providers: local, remote node, S3-compatible, OneDrive, OSS, COS, Qiniu, Upyun, KS3, OBS.
- Direct upload/download between client and storage; chunked, resumable, parallel uploads.
- Remote download: aria2, qBittorrent, **and yt-dlp** providers, multi-node with per-node settings,
  group-level concurrent/size quotas.
- Share links with expiration — plus fork additions: upload-only drop boxes, edit-in-place,
  preview-only mode, anonymous upload, IP-restricted views.
- Archive compress/extract, media metadata extraction, metadata/tag search.
- WebDAV across all storage providers (read-only group enforcement fixed in this fork).
- SSO: generic OIDC inbound consumer (auth-code + nonce, JWKS-verified, auto-provisioning),
  OAuth public clients with PKCE, passkeys, TOTP 2FA.
- Multi-user, multi-group; admin task list with CIDR-capable creator-IP filtering; per-user trash
  retention; per-group remote-download quotas.
- Preview: image (progressive thumbnail→full-res), video, audio, ePub, Markdown, diagrams,
  Office documents (WOPI), 3D models.
- PWA, dark mode, i18n (en-US, zh-CN, and more), theme customization, custom HTML injection.

## Build from source

Prereqs: Go ≥ 1.24, Node ≥ 20 + Yarn, (desktop) Rust + platform Tauri deps.

```bash
# Frontend
cd frontend && yarn install
NODE_OPTIONS=--max-old-space-size=6144 yarn build   # emits build/ consumed by the Go embed

# Backend (repo root) — the binary serves frontend + API on :5212
go build -o cloudreve .
./cloudreve
```

Desktop client: see `desktop/CLAUDE.md` (`cargo tauri build`, Windows-first; other platforms WIP).

## Development

```bash
go build ./... && go vet ./... && go test ./...          # backend gate
cd frontend && yarn tsc --noEmit && yarn build           # frontend gate
```

`docker-compose.dev.yml` brings up postgres + redis + a source-built backend; `yarn dev` gives
frontend hot reload. PRs land via feature branches — never push to `master` — and must pass all CI
jobs before merge.

## Status scorecard

| Area | State |
|---|---|
| Backend / frontend | Stable — 4.19.1 line, all CI green |
| Upstream issues | ~70% of the 137 migrated issues closed; remainder are feature-scale, Pro-surface, or device-bound |
| Code health | desloppify strict score 77.1 (was 18.9); 73 review items dispositioned |
| Desktop client | Windows functional (cfapi sync + shell integration); macOS/Linux providers planned |
| Android client | Scaffolded — Kotlin/Compose skeleton, Phase E in [ROADMAP.md](ROADMAP.md) |
| Pro-free features | Share collaboration ✓, OIDC SSO ✓, delegated admins ✓; storage-policy migration, VAS/billing, audit surface in progress |

Known limitations and the full plan: [ROADMAP.md](ROADMAP.md) · issue tracker has honest per-issue status.

## Security

Report vulnerabilities privately via GitHub's "Report a vulnerability" on this repo — do not open a
public issue. All 16 published upstream GHSAs are patched at our baseline; our own additions are
reviewed for SSRF, path traversal, process execution, and session entropy before merge.

## Credits

Cloudreve was created by **Aaron Liu and the Cloudreve contributors** (cloudreve.org). This fork is
an independent continuation under the same GPL-3.0 license — attribution, not endorsement. See
[NOTICE](NOTICE) for the full attribution statement.

## License

[GPL-3.0](LICENSE) — same as upstream. Contributions are licensed identically.
