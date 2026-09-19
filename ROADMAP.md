# Cloudreve Fork — Analysis & Roadmap

Fork: `Dvorinka/cloudreve` · Upstream: `cloudreve/cloudreve` · Baseline: `4.19.1` (exact upstream HEAD, zero divergence)

Original work by the Cloudreve authors (cloudreve.org). This fork continues it as a fully open-source project — every "Pro" feature reimplemented and free, desktop client on all platforms, native Android app.

---

## 1. Analysis

### 1.1 Repo state

| Fact | Value |
|---|---|
| Fork vs upstream | `0 ahead / 0 behind` — clean mirror of `master` @ 4.19.1 |
| Backend | Go 1.26, Gin, ent ORM, `go build ./...` compiles (only `assets.zip` embed fails until frontend is built — expected) |
| Frontend | `assets/` submodule → `cloudreve/frontend` (React + Vite + TS, 69 deps), not yet initialized locally |
| Storage drivers present | local, S3, OSS, COS, Qiniu, Upyun, OneDrive, remote node (`service/explorer/slave.go`) |
| Auth present | password+2FA, passkey (`ent/schema/passkey.go`), generic OAuth client/grant, WebDAV accounts (`davaccount.go`) |

### 1.2 Security posture

16 published GHSAs on upstream — **all fixed at our baseline** (vuln ranges ≤ 4.17.0, we run 4.19.1). Spot-verified in tree, not just by version:

- GHSA-f8xp (account takeover, insecure PRNG): `pkg/util/common.go` uses `crypto/rand` primary, `math/rand` only on crypto failure — fix present
- GHSA-vgj4 (OAuth scope bypass, missing client_id): `service/oauth/oauth.go:159` validates `authCode.ClientID != s.ClientID` — fix present
- Remaining highs (WebDAV path traversal, quota TOCTOU, OneDrive cred update via Admin.Read) — all ≤ 4.16.x ranges, patched

Ongoing security work is in the roadmap (§5), not the backlog.

### 1.3 Pro feature map (cloudreve.org/pricing — all 5 slides captured)

The community repo contains **zero Pro code** — Pro ships as a separate licensed binary (`--license-key`, `proupgrade` DB migration). BUT the community **frontend already contains the full Pro UI skeleton** — every surface below renders a `ProChip` badge that opens `ProDialog.tsx`. Implementation = backend endpoints + remove the chips.

| Pro slide | Features | Frontend hooks found |
|---|---|---|
| Sharing & collaboration | write/upload/delete via share link, paid share links, granular file permissions (users/groups/anonymous), anonymous upload via share, default shares for new users | `ShareSection.tsx` (group editor) |
| Storage policy mgmt | multiple policies per group, per-directory policies, load-balancer policy, file migration between policies | `SelectProvider.tsx`, `storage_policy_id` exists (single) on group |
| User & auth | multi-account switching, Logto SSO, OIDC SSO, QQ Connect, sign-up email filtering | `SSOSettings.tsx`, `SSO/` |
| Monetization (VAS) | storage plans, membership plans, redemption codes, credits | `VAS/` dir: `GroupProducts`, `StorageProducts`, `PaymentProviders`, `GiftCodes` — **full UI exists** |
| System extensions | activity/audit logs, site announcements, node selection, report abuse | `Events.tsx` (admin), `Home.tsx` |

Backend gaps are concrete: `ShareProps` = `{share_view, show_read_me}` only; `group.storage_policy_id` is single; no order/product/credit entities at all. `NavigatorCapability_CommunityPlaceholder1–9` in `pkg/filemanager/fs/dbfs/navigator.go` are the reserved capability slots Pro fills.

#### 1.3a Pro UI reference — extracted from `cloudreve.org/imgs/features/en_*.png`

All 22 reference shots vendored to `output/playwright/pro-features/`. What each actually shows (implementation-level detail, not marketing text):

**Sharing & collaboration**
| Feature | What the UI reveals | Implementation shape |
|---|---|---|
| Modify/delete via share | "Create share link" dialog embeds the ACL picker itself | Share entity carries a capability set; already landed (#140) |
| File permissions | Context menu → More actions → **Permissions**: "Explicit access" list (users by email, groups like "Same group with me") each with a dot-joined capability dropdown + ×; "General access" rows: **Anonymous visitors**, **Everyone else**; search box "Search for emails or groups…" | ACL entity: `(file_id|dir, subject_type, subject_id)` → bitmask of **Read·Create·Update·Delete**. Create = folder-only (upload/move/copy into); Update = rename/metadata/log-view; atoms documented per-atom in the dialog |
| Paid share links | "Pay to download" section in share dialog: **points price** field + "Gain N points to you per purchasing" hint (take-rate visible: 50 → 40, i.e. ~80% to sharer). Buyer flow: gate page "You need to pay N Points" → dialog: *Pay with points* (login req.) / *Pay with cash* (¥0.50 equiv ⇒ **1 pt = ¥0.01**) / **Restore purchase** via "Resume ticket" from order email | `share.price_points`; purchase record + resume-ticket token; credits ledger (B.4) funds both sides; take-rate = admin setting |
| Anonymous upload | Same ACL dialog — anonymous grant of Create/Update/Delete on a folder share | Covered by ACL anonymous tier + existing `allow_upload` |
| Default shares | Admin "Default share shortcuts": chip-input of share IDs with search; new users get share-shortcut objects (folder icon + share badge) in home root | `setting.default_shares` = share ID list; on user create → insert share-shortcut fs entries |

**Storage policy management**
| Feature | What the UI reveals | Implementation shape |
|---|---|---|
| Multiple policies per group | Group editor "Available storage policies" — multi-select chips of all policies; "Modifying will not affect uploaded files" | `group_storage_policies` join table replaces `storage_policy_id` single |
| Per-directory policies | FM toolbar policy button (battery icon) → dropdown of group-allowed policies, check = current dir's policy | `file.metadata.policy_id` or per-dir row; upload session resolves dir→policy |
| Load balancer | "Load Balance" appears in the **provider type grid** beside Local/Remote/S3…; editor = name + child-policy list with **Weight** number per child + Add/Remove | New `StoragePolicyTypeLB`; driver picks weighted child per upload session |
| File migration | **User-facing**: context menu → More actions → "Relocate storage policy" → policy dropdown → background task "Queue → Index files → Transfer → Commit changes" | We have admin variant (#175); extend to user scope limited to group-allowed policies |

**User & auth**
| Feature | What the UI reveals | Implementation shape |
|---|---|---|
| Multi-account | "Select an account" page: remembered sessions, avatar+email+trash, "Signed out" badges, "Use another account"; avatar menu = inline switcher | Frontend session store holds N token sets; swap active credential; sign-out revokes one |
| OIDC SSO | Admin "Third-party sign-in": ClientID/secret/Scope, **"OIDC Wellknown Config — Import from URL"**, "Sign-in method name" (i18n-capable), **"Login without registration"** (auto-provision, locks user to that IdP) | Mostly landed (#141) — verify wellknown import + display-name parity |
| Logto SSO | Same panel, Logto-flavored preset | Falls out of generic OIDC — needs no extra backend; document Logto setup |
| QQ Connect | Separate OAuth (non-OIDC) provider button | Own small provider; low priority outside CN audience |
| Email filtering | "Filter email provider" (Whitelist/Blacklist) + domain list textarea + **"Disable sub-address email"** (blocks `+` aliases); registration-only, SSO exempt; toast "This email provider is forbidden" | Landed (#141) |

**Monetization (VAS)**
| Feature | What the UI reveals | Implementation shape |
|---|---|---|
| Storage SKUs | "Edit SKU": name, size+unit, **duration seconds** (packs expire!), price in min currency unit (700=¥7), optional Label badge, "Allow paying with points"+points price | `sku` entity; purchase → capacity grant with expiry |
| Membership SKUs | "Edit group product": name, **Group ID dropdown** (upgrade target), duration, price, label, description (one bullet per line), points toggle | Same SKU entity, `type=group`, grants group for duration then reverts |
| Credits | Settings gets **Finance tab**: balance + Recharge; ledger table (±Change, Time, Reason: Shop purchase/Manual adjustment/Share link purchased) | `credit_account` + `credit_txn` ledger; admin manual adjust endpoint |
| Redemption | Shop→Redeem: code field (UUID fmt). Admin Gift Codes table: product type (Points/VIP/Storage), qty, code, Used/Available, × revoke. "Generate": count + product type + SKU + units-per-code | `gift_code` entity → redeem applies SKU/points atomically |

**System extensions**
| Feature | What the UI reveals | Implementation shape |
|---|---|---|
| Activity log | Per-file "Activity" dialog (context menu): actor avatar+time, typed actions ("Updated file content → Look for this version", "Moved from Trash to My Files", "Triggered thumbnail generation", "Was copied from X to His files") — **cross-actor** (collaborators' ops on shared files). Admin "Event details": ID/Type/IP/Time/**Correlation ID**/linked user+file+blob+share/UA/raw JSON | `activity_event` entity: type enum, actor, correlation_id, subject edges (user/file/blob/share), ip+ua, json payload; per-file feed + admin feed |
| Site announcements | Post-login modal: title+body, "Don't show anymore" + OK | `setting.announcement` (markdown); per-user dismissal bit |
| Node selection | Group "Allowed nodes" chips (empty = all); task dialogs get "Target node" dropdown — **Auto dispatch** default + allowed nodes; covers remote-download + compress/decompress; other tasks → master | `group.allowed_nodes`; task-create accepts `target_node`, scheduler validates against group list |
| Report abuse | Dialog on shares/users: target chip, Reason dropdown ("Copyright infringement"…), description, **CAPTCHA** | `abuse_report` entity + admin review queue (resolve/dismiss/block share) |

### 1.4 Org repo decisions — **monorepo**

Everything ships from `Dvorinka/cloudreve`. No submodules, no sibling repos.

```
cloudreve/           Go backend (existing code, repo root)
├── frontend/        web SPA — vendored from cloudreve/frontend (was `assets` submodule)
├── desktop/         Tauri app — vendored from cloudreve/desktop
├── android/         native Android app — Kotlin + Compose (Phase E)
└── .github/workflows/  GitHub Actions CI/CD (replaces azure-pipelines)
```

| Repo | Verdict | Reason |
|---|---|---|
| `cloudreve` (this fork) | **Keep — base + monorepo root** | The core |
| `frontend` | **Vendored → `frontend/`** | Submodule replaced; source lives in-repo, CI builds it |
| `desktop` | **Vendored → `desktop/`** | Tauri+React; portable core (`cloudreve-api`, `inventory`, `tasks`, `uploader`, `drive/sync`) vs Windows-only glue (`cfapi`, `shellext`, `win32_notif`) |
| `docs` | **Skip for now** | Fork when we ship public docs |
| `docker-compose` | **Vendored file** | Root `docker-compose.yml` already exists; extend for dev stack |
| `taskqueue` | **Skip** | Dead since 2024; superseded by in-app queue |
| `remote-server` | **Skip** | Dead PHP-era remote; v4 has native remote nodes |
| `ios-feedback` | **Skip** | Tracker for closed-source iOS app; we do Android instead |
| `theme-editor`, `frontend_v2`, `v2` | **Skip** | Archived/ancient |

### 1.5 Upstream issues — 137 open, grouped

| Group | Count | Examples |
|---|---|---|
| Bug — upload/download/sync | ~25 | #3574 trash_bin_collect OOM, #3454 PG FK on upload, #3118 WebDAV 500 on large files, #3005 WebDAV fragments, #2938 upload stuck |
| Bug — WebDAV | ~8 | #2878 mount path 404 after move, #3118, #3409 read-only groups |
| Bug — DB/migration | ~8 | #3452 MySQL HeatWave, #2880 unix socket, #2934 v3→v4 sqlite, #2981 PG18 |
| Enhancement — sharing/permissions | ~15 | #3555 preview-only shares, #3390 default share visibility, #3340 upload-only folders, #3033/#3032 multi-share ops |
| Enhancement — storage policy | ~8 | #3518 encrypt on relocation, #2961 enable/disable policy, #2262 site-wide migration |
| Enhancement — auth/SSO | ~7 | #3464 OIDC, #3056 auto-OIDC, #2179 TOTP manual key, #3479 IP whitelist |
| Enhancement — download/tasks | ~10 | #3491 advanced remote download, #3259 yt-dlp, #2427 download quota, #2270 cancel tasks |
| UX/polish | ~20 | #3288 breadcrumb restore, #3223 deselect on empty click, #3508 loop video, #3507 gallery names |
| Pro-related (becomes free here) | ~10 | #3572 ADFS OIDC, #3515 recurring billing, #3180 group-expiry downgrade, #3171 subaddress ban |
| Mobile/iOS requests | ~5 | #3003 captcha incompat, #2826 login fail, #2863 capacity 0KB — Android solves |
| Chinese-titled (mixed) | ~40 | folded into groups above after translation |
| wontfix by upstream (revisit) | ~10 | #2494 multi-group, #2883 slide captcha, #2842 file audit — **candidates for us** |

### 1.6 Upstream PRs — verdicts

| PR | Change | Verdict |
|---|---|---|
| #3524 | revalidate share/direct links after permission change (+185/-1, CLEAN) — fixes #3544, same class as GHSA-vx2m | **Merge** — security |
| #3549 | gorilla/websocket 1.5.0→1.5.3 | **Merge** — dep CVE hygiene |
| #2964 | nwaples/rardecode 2.1.0→2.2.0 | **Merge** — dep hygiene |
| #2851 | ulikunitz/xz 0.5.12→0.5.14 | **Merge** — dep hygiene |
| #3490 | IP range/CIDR filter in event logs (+668, 2 files) — fixes #3480 | **Merge** — small, self-contained |
| #3472 | OIDC provider support (+292/-6, 10 files) — fixes #3464, overlaps Pro SSO | **Merge after review** — strategic (free OIDC) |
| #2481 | multi-stage Dockerfile (+18/-2) | **Review** — conflicts w/ our own docker work likely; take if clean |
| #2507 | p2p QUIC (draft) | **Skip** — draft, scope creep |
| #2499 | multi-group users (draft, WIP) | **Watch** — big feature, matches #2494 wontfix; revisit when upstream matures it |
| #1802 | checksum on WOPI/text-edit update (draft, 2024) | **Skip** — stale draft |

---

## 2. Immediate actions (this change set)

- [x] Analysis + this roadmap
- [x] Enable Issues on fork; label taxonomy (`group:*`, `pro-free`, `security`, `revisit`, `epic`)
- [x] Migrate all 137 upstream issues → fork #1–137 (translated, grouped, linked)
- [x] Merge upstream PRs: #3524, #3472, #3490, #3549, #2964, #2851, #2481
- [x] `go build ./...` + `go test ./...` green (incl. upstream's own stale-test fixes)
- [x] Monorepo restructure: `frontend/` + `desktop/` vendored, `assets` submodule removed, `android/` scaffolded
- [x] CI/CD rewrite: GitHub Actions `ci.yml` (backend/frontend/desktop matrix) + `release.yml` (goreleaser→ghcr.io); azure-pipelines removed

## 3. Phase A — foundation hardening (first weeks)

- [x] Sync-upstream automation: weekly `upstream → fork` merge workflow (`.github/workflows/upstream-sync.yml`)
- [x] Dependabot on the fork (Go, npm, cargo — `.github/dependabot.yml`)
- [x] `docker-compose.dev.yml` dev stack (postgres + redis + backend built from source; frontend via `yarn dev` hot reload)
- [x] Remove `ProDialog`/`ProChip` gates in `frontend/` — all upsell interception stripped, `ProDialog.tsx` deleted (backend feature work remains, Phase B)
- [x] `NOTICE` attribution file — upstream authorship + independent-Pro-implementation statement
- [x] `go vet ./...` clean (upstream lint debt repaired: lock-by-value receivers, unkeyed literals, GobDecode signature)

## 4. Phase B — Pro features, free (the big one)

Order = user-visible value first; each ships with backend + UI + tests.

1. **Share collaboration** — write/upload/delete via share link, anonymous upload, share ACL (users/groups), preview-only mode (fixes #3555, #3390, #3340, #3517, #3578; uses `NavigatorCapability` placeholder slots + `ShareProps` extension + `share` entity fields). See §1.3a for the extracted UI spec.
   - [x] PR #140 — `allow_upload`/`allow_edit`/`preview_only`/`upload_only` props, props-derived capability sets enforced server-side (`writePermitted`), same-share move/copy, anonymous upload, drop-box listing suppression, download denial via `IsDownloadCtxKey` hooks (fixes #3555 preview-only, #3340 drop-box)
   - [x] File/dir ACL entity — `(subject_type ∈ user|group|anonymous|everyone, subject_id)` → R/C/U/D bitmask; Permissions dialog under More actions; enforced in share-navigator capability checks; group bit 15 gate (#181, fixes #3517)
   - [x] Default shares — `setting.default_symbolics` + group `default_pinned` chip-input of share IDs; materialize as share-shortcut entries on fs init (#180)
   - [ ] Paid shares — `share.price_points` + gate page + purchase/resume-ticket flow; needs B.4 credits first
2. **Storage policy advanced** — multiple policies per group, per-directory binding, load-balancer policy, file migration (fixes #3518, #2961, #2262). See §1.3a.
   - [x] PR #175 — resumable admin relocation task (entities or whole-policy scope), encryption-aware re-wrap, admin UI + per-policy migrate action (#9, #125, #136)
   - [x] Group→policies M:N (`allowed_policies` edge, empty = legacy single) + group-editor multi-select; per-directory `sys:preferred_policy` metadata marker with nearest-ancestor precedence (invalid marker cuts inheritance); user `preferred_policy` setting applied in own tree only; `load_balance` policy type with weighted children resolved before drivers (#182, fixes #2961)
   - [x] User-facing relocate — `POST /file/relocate` from FM More-actions dialog, entity expansion + same-policy skip + dedup, restricted to group-allowed policies (#182, fixes #2262)
3. **SSO** — generic OIDC, Logto, multi-account switching, sign-up email filtering (fixes #3464, #3056, #3505). See §1.3a.
   - [x] PR #141 — inbound OIDC consumer (auth-code + nonce, JWKS-verified RS256 id_tokens, userinfo fallback, auto-provisioning, one-time ticket handoff, SSRF-validated endpoints, redacted secret); covers Keycloak/Authentik/Logto/generic IdPs; sign-up email domain filtering (whitelist/blacklist + sub-address block) enforced at registration and SSO provisioning; multi-account lands via existing session `upsert` (fixes #3464, #3056)
   - [ ] Multi-account switcher UI — N-token session store, avatar-menu switch + signed-out badges (frontend-only, backend already supports)
   - [ ] QQ Connect (non-OIDC protocol, separate integration), account linking UI for existing local accounts, group/role claim mapping
4. **VAS/monetization-free** — credits + redemption codes as *free* features (gift codes for admin use), storage/membership SKU definitions; payment processors stay out of scope (fixes #3231). See §1.3a for the SKU/credits/gift-code spec.
   - [ ] `sku` entity (storage-capacity + group-upgrade types, duration, cash+points price, label, bullets); Shop page (Memberships/Storage/Redeem tabs)
   - [x] `user.credits` + `credit_txn` ledger (guarded atomic adjust); Finance settings tab (balance + grants + redeem + ledger); admin manual adjust (#183)
   - [x] `gift_code` entity (points/storage/group × amount × duration) + `user_grant` expiring grants + `grant_expire` cron; admin generate/list/revoke + user redeem (#183)
   - [ ] Paid-share `price_points` wired to the ledger + purchase/resume-ticket flow
5. **System extensions** — activity/audit log, site announcements, node selection, report-abuse queue (fixes #3480, #3479 IP whitelist). See §1.3a.
   - [x] PR #144 — task `creator_ip` capture with CIDR-capable admin filter (#115 OSS half), group remote-download quotas per count + per volume (#16), yt-dlp downloader provider (#88), progressive image preview (#113), v3 migrator `DatabaseURL` passthrough (#42)
   - [x] `activity_event` entity (immutable, tx-aware, actor+IP+CID) + per-file Activity dialog + admin `/admin/event` feed + per-type enablement + retention cron (#184)
   - [x] Coverage wave 2: email/user-activated/token-refresh/share-viewed/version/metadata/view/thumb/live-photo/copy-from/webdav/profile+security/oauth/admin-ops/import (1bbaddf)
   - [ ] Event coverage remainder (needs unbuilt features): payment_*, link/unlink_account, membership_unsubscribe, mount, quota-notify
   - [x] site announcement: `announcement` setting (markdown) + post-login modal + per-user dismissal re-triggering on content change (#184)
   - [x] `abuse_report` entity + public `POST /abuse/report` (IP rate-limit + `abuse_captcha` gate) + admin `/admin/abuse` queue (resolve/dismiss + reversible share block) + share-menu Report entry (#185)
   - [x] group `allowed_nodes` pool + `allow_select_node` + task `target_node` dispatch (persisted in task state, weighted LB within pool); group admin multi-select + task-dialog node picker

## 5. Phase C — security + quality

- Own security review on top of upstream fixes: session/token entropy audit, SSRF guard re-test (NAT64 class), rate limiting on auth endpoints
- Fix upstream bug backlog by impact: ~~#3574 OOM~~ (done — paged tree walk + batched delete), ~~#3118/#3005 WebDAV large-file~~ (done — Content-Range assembly into one session; non-local policies get honest 501; single-PUT giant-file 500s are proxy/client timeouts, not fixable server-side), ~~#3375 SMTP auth discovery~~ (done — `smtp_auth` setting)
- #3454 (PG FK on upload) is **Pro-only** — `audit_logs` doesn't exist in this codebase. When B.5 adds our own audit log: insert the audit row in the same tx *after* the file row, never before.
- [x] `desloppify` pass — 73 review items dispositioned (46 fixed, 27 honestly skipped), strict score 77.1 (was 18.9); scorecard lives in README. `security-reviewer` pass done incrementally per batch (OAuth secrets, SSRF, process exec, path safety)

## 6. Phase D — desktop, all platforms

Goal: Windows + macOS + Linux from the `desktop/` tree in this repo.

| Layer | Windows (exists) | macOS | Linux |
|---|---|---|---|
| Placeholders/hydration | cfapi (keep) | File Provider ext (Swift bridge) | FUSE (`fuser`) or plain sync folder |
| Shell integration | shellext (keep) | Finder sync extension | Nautilus/Dolphin plugin (later) |
| Notifications | win32_notif → replace | `tauri-plugin-notification` (all platforms) | same |
| Sync core | shared: `cloudreve-api`, `inventory`, `tasks`, `uploader`, `drive/sync` | same | same |

- Port order: (1) strip `win32_notif`→tauri notifications (all platforms benefit), (2) abstract `drive/` behind a `HydrationProvider` trait (cfapi impl on Windows, stub→FUSE on Linux, FileProvider on macOS), (3) CI matrix build all 3, (4) MSIX→also ship .dmg/.AppImage/.deb.
- Feature fallback on Linux/macOS until providers land: full sync without placeholders (download-on-access still works via sync engine).

## 7. Phase E — Android app (native, no iOS)

Lives in `android/` in this repo. Kotlin + Jetpack Compose, Material 3.

- **API**: `api/v4` REST + OAuth token (entities exist: `oauthclient`, `oauthgrant`) — same surface the desktop `cloudreve-api` crate documents; port its models as the spec
- **Core features**: browse/download/upload files, share links, camera-upload (auto photo backup), offline-favorite files, local sync folder via SAF/WorkManager
- **System integration** (the "native, complete" ask): share-sheet target (upload to Cloudreve from any app), DocumentsProvider (Cloudreve in Files app), quick-share tile, notifications on share/task events
- **Auth**: webview OAuth flow → token; later passkey if backend exposes
- **WebDAV bridge**: `/dav` works as fallback file access until SDK matures
- Non-goals: iOS, tablet-first layouts (works, not optimized)

## 8. Governance

- License/credit: keep `LICENSE` (GPL-3.0), add `AUTHORS`/credit line to original Cloudreve authors in README — attribution without endorsement
- Release cadence: tag `fork-4.19.x` line first (cherry-picks only), then `5.0.0-fork` once Phase B lands
- Every merge: build + test + lint green (pre-push gate, non-negotiable)

---

## Issue migration format

Each fork issue: title translated to English when needed, body = `Upstream: cloudreve/cloudreve#NNNN` + short restatement + group labels. Epics get `epic` label and link children. Upstream `wontfix` items we want get `revisit` label.
