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
   - [x] Paid shares — `share.price_points` + `share_purchase` (buyer debit → owner income at `share_score_rate`) + resume ticket; download/thumb gated in share navigator, listing stays visible; `share_sell` (bit 29) gates price-setting, `share_free` (bit 8) bypasses paywall; `PaidShareGate` UI + restore via `purchase_ticket`
   - [x] Hash dedup / instant upload — `entity.hash` + `(hash,size)` index; `FindEntityByHash` (completed version entities, owner scope default / `global` opt-in); `PrepareUpload` links existing entity (refcount++) → `rapid_uploaded` session, no transfer; media-meta/FTS queued per file; frontend streams SHA-256 via `hash-wasm`, skips encrypted policies; `upload_dedup_scope` admin setting (#191, fixes #3044)
   - [x] Multi-file share — one share covering N selected files (upstream #3032): share.files M:N join (anchor file edge preserved for validity/ACL/paid gate); synthetic union root in share navigator with per-file real parent chains; dead linked files drop out of listing; upload-only + write caps stripped at union root; uris[] create API (max 50, dedup); multi-select Share dialog; file_count + anchor source_uri in share info
2. **Storage policy advanced** — multiple policies per group, per-directory binding, load-balancer policy, file migration (fixes #3518, #2961, #2262). See §1.3a.
   - [x] PR #175 — resumable admin relocation task (entities or whole-policy scope), encryption-aware re-wrap, admin UI + per-policy migrate action (#9, #125, #136)
   - [x] Group→policies M:N (`allowed_policies` edge, empty = legacy single) + group-editor multi-select; per-directory `sys:preferred_policy` metadata marker with nearest-ancestor precedence (invalid marker cuts inheritance); user `preferred_policy` setting applied in own tree only; `load_balance` policy type with weighted children resolved before drivers (#182, fixes #2961)
   - [x] User-facing relocate — `POST /file/relocate` from FM More-actions dialog, entity expansion + same-policy skip + dedup, restricted to group-allowed policies (#182, fixes #2262)
3. **SSO** — generic OIDC, Logto, multi-account switching, sign-up email filtering (fixes #3464, #3056, #3505). See §1.3a.
   - [x] PR #141 — inbound OIDC consumer (auth-code + nonce, JWKS-verified RS256 id_tokens, userinfo fallback, auto-provisioning, one-time ticket handoff, SSRF-validated endpoints, redacted secret); covers Keycloak/Authentik/Logto/generic IdPs; sign-up email domain filtering (whitelist/blacklist + sub-address block) enforced at registration and SSO provisioning; multi-account lands via existing session `upsert` (fixes #3464, #3056)
   - [x] Multi-account switcher UI — N-token session store, avatar-menu switch + signed-out badges (#189)
   - [x] QQ Connect (non-OIDC protocol, separate integration), account linking UI for existing local accounts — `sso_binding` entity, link/unlink with lockout guard, admin accordion, sign-in button (#190); remainder: group/role claim mapping
4. **VAS/monetization-free** — credits + redemption codes as *free* features (gift codes for admin use), storage/membership SKU definitions; payment processors stay out of scope (fixes #3231). See §1.3a for the SKU/credits/gift-code spec.
   - [x] `sku` entity (storage-capacity + group-upgrade types, duration, cash+points price, label, bullets); points purchase → atomic debit+grant; admin SKU tables; `/shop` page (Memberships/Storage/Redeem tabs) + nav entry
   - [x] `user.credits` + `credit_txn` ledger (guarded atomic adjust); Finance settings tab (balance + grants + redeem + ledger); admin manual adjust (#183)
   - [x] `gift_code` entity (points/storage/group × amount × duration) + `user_grant` expiring grants + `grant_expire` cron; admin generate/list/revoke + user redeem (#183)
   - [x] Paid-share `price_points` wired to the ledger + purchase/resume-ticket flow
5. **System extensions** — activity/audit log, site announcements, node selection, report-abuse queue (fixes #3480, #3479 IP whitelist). See §1.3a.
   - [x] PR #144 — task `creator_ip` capture with CIDR-capable admin filter (#115 OSS half), group remote-download quotas per count + per volume (#16), yt-dlp downloader provider (#88), progressive image preview (#113), v3 migrator `DatabaseURL` passthrough (#42)
   - [x] `activity_event` entity (immutable, tx-aware, actor+IP+CID) + per-file Activity dialog + admin `/admin/event` feed + per-type enablement + retention cron (#184)
   - [x] Coverage wave 2: email/user-activated/token-refresh/share-viewed/version/metadata/view/thumb/live-photo/copy-from/webdav/profile+security/oauth/admin-ops/import (1bbaddf)
   - [x] link/unlink_account events wired via `sso_binding` flows (#190)
   - [x] Event coverage wave 3: `membership_unsubscribe` emitted by grant-expiry cron for reverted group grants; `user_exceed_quota_notified` at every quota rejection (upload pre-check, atomic reserve, copy)
   - [ ] Event coverage remainder (needs unbuilt features): payment_* (no payment processor), mount (policy-mount feature absent)
   - [x] site announcement: `announcement` setting (markdown) + post-login modal + per-user dismissal re-triggering on content change (#184)
   - [x] `abuse_report` entity + public `POST /abuse/report` (IP rate-limit + `abuse_captcha` gate) + admin `/admin/abuse` queue (resolve/dismiss + reversible share block) + share-menu Report entry (#185)
   - [x] group `allowed_nodes` pool + `allow_select_node` + task `target_node` dispatch (persisted in task state, weighted LB within pool); group admin multi-select + task-dialog node picker

## 5. Phase C — security + quality

- Own security review on top of upstream fixes: session/token entropy audit, SSRF guard re-test (NAT64 class), rate limiting on auth endpoints
- Fix upstream bug backlog by impact: ~~#3574 OOM~~ (done — paged tree walk + batched delete), ~~#3118/#3005 WebDAV large-file~~ (done — Content-Range assembly into one session; non-local policies get honest 501; single-PUT giant-file 500s are proxy/client timeouts, not fixable server-side), ~~#3375 SMTP auth discovery~~ (done — `smtp_auth` setting)
- #3454 (PG FK on upload) is **Pro-only** — `audit_logs` doesn't exist in this codebase. When B.5 adds our own audit log: insert the audit row in the same tx *after* the file row, never before.
- [x] #198 (upstream #3581) — "import files" task shows source storage policy "unknown": `ImportTaskState.PolicyName` baked at creation (resolved best-effort via `StoragePolicyClient`), summary emits `dst_policy_name` so admin views of other users' tasks work without a policy lookup; `policyOptionCache` retyped to `StoragePolicyBrief[]` and populated from `getAllowedPolicies()` at session init as fallback for legacy tasks
- [x] #199 (upstream #3584) — markdown editor lag: root cause was per-keystroke React re-renders (changedValue state fed back into the editor's initial-markdown prop) re-running every plugin's `update()` hook (RealmWithPlugins has a dep-less effect). `MarkdownEditor` is now `memo`'d with a `useMemo`'d plugins array + stable `translation`; `MarkdownViewer` keeps edits in a ref (read at save) and passes the immutable loaded content — zero re-renders per keystroke
- [x] #200 (upstream #3586) — SIGHUP "crash": the upstream log was a clean signal-driven shutdown (SIGHUP was registered in `signal.Notify`), triggered when the reporter's terminal/SSH session closed. `signal.Ignore(syscall.SIGHUP)` now — default disposition would terminate the process; verified live: server survives `kill -HUP` (HTTP stays 200) and still shuts down cleanly on SIGTERM
- [x] #89 (upstream #3277) — Android Motion Photo preview: `util/motionPhoto.ts` extracts the MP4 appended at EOF of MicroVideo/MotionPhoto JPEGs — XMP `GCamera:MicroVideoOffset` or `Container:Directory` `video/mp4` `Item:Length`, fetched via Range requests (128 KiB head + video tail only), `ftyp` sanity-checked; `Photo.tsx` feeds the blob URL into the same LivePhotosKit player used for iOS live photos — badge + press-to-play for free; degrades silently on CORS/fetch failure
- [x] `desloppify` pass — 73 review items dispositioned (46 fixed, 27 honestly skipped), strict score 77.1 (was 18.9); scorecard lives in README. `security-reviewer` pass done incrementally per batch (OAuth secrets, SSRF, process exec, path safety)
- [x] Tag management page (upstream #2962) — owner-scoped `tag:` metadata stats/rename/recolor/delete in `inventory.FileClient`, `GET/PATCH/DELETE /file/tag` routes, Settings → Tags tab with merge-on-rename semantics
- [x] Download URL shuffling (#173) — `download_cdn_shuffle` distributes generated download URLs randomly across the site URL + `download_cdn_routes` endpoints (`setting.DownloadURLBase`, honors `UseFirstSiteUrl`); covers entity downloads, archive sessions, and redirect-type direct links; manual route picker hidden client-side while active
- [x] Private space / vault (upstream #3447) — opt-in root folder flagged `sys:vault`; ancestry-based membership (zero flag maintenance; chain-less search results resolved lazily via `file_children`); `vaultNavigator` decorator gating `To`/`Children`/`Walk`/`ExecuteHook`; separate vault password (`salt:sha256`, sensitive) + 30-min cache-backed unlock session, unlock rate-limited 10/h; vault content never shareable and never direct-linkable; search filtered while locked; `vault_enabled`/`vault_unlocked` in user settings; unlock prompt in `ExplorerError`, Private space section in security settings, lock badge on vault folder (#195)
- [x] Saved share links (#147) — `POST /file/create` accepts `type: share` + `share_id`/`share_password`, materializing a symbolic shortcut (`sys:shared_redirect`) that lists under My Files and Shared with me; "Save to my files" in the share popover + "Save share link" dialog on /shares
- [x] Share `hide_readme` option (upstream #2729 item 6) — `ShareProps.HideReadMe` (only meaningful with `ShowReadMe`); share navigator filters `README.md`/`README.txt` (case-insensitive) from listings while direct-path resolution stays open for the readme viewer; `detectReadMe` URI fallback now probes unconditionally; owner-only `hide_readme` on share responses; Share dialog nested checkbox
- [x] 2FA recovery codes (upstream #2729 item 3) — `user.two_factor_backup_codes` sensitive JSON of `salt:sha256` digests; `PUT /user/setting/2fa/backup` regenerates 10 one-time codes behind a valid TOTP (rate-limited 5/h); `Verify2FA` falls back to single-use code consumption on TOTP failure; codes invalidated on secret rotation/disable; login phase gains a recovery-code input mode; security settings show remaining count + regenerate dialog
- [x] Public share directory (upstream #2729 items 4+5) — `share.listed_publicly` opt-in column gated by new `GroupPermissionSharePublicList` group bit; rejected on password-protected shares at the service layer and normalized off at creation; `GET /share/listed` anonymous endpoint (rate-limited 60/min/IP, cursor pagination) listing only non-expired passwordless listed shares with case-insensitive name search across anchor + covered files; `listed_publicly` owner-visible in share responses; `/discover` page (anonymous-visible nav item + sign-in link), admin group Share section switch, en+zh locales
- [x] Decompression-bomb guards (meta #2 item 10) — `DecompressSize` now bounds cumulative extracted output (its documented "total file size" intent), not just compressed input: `checkExtractGuards` aborts at the limit and at a 100k-entry cap (`maxExtractEntries`, bounds dir-creation bombs), each entry stream wrapped in `cappedFile` so understated size headers cannot overrun; slave path receives the limit via `SlaveExtractArchiveTaskState.ExtractLimit`; all failures carry `queue.CriticalErr` (no retry of the same bomb); resume-safe via cursor-skip size accounting
- [x] Storage policy total capacity (upstream #2178 item 1) — `PolicySetting.MaxTotalSize` caps cumulative entity bytes per policy; enforced in `PrepareUpload`, batch upload validation, and `copyFiles` (baseline usage + per-batch accumulation since tx writes are invisible to the usage query); canonical `ErrInsufficientCapacity` preserved so `errors.Is` quota handling still matches; admin policy editor gains a Max total capacity SizeInput, en+zh locales
- [x] Storage policy overflow chain (upstream #2178 item 4) — `PolicySetting.OverflowPolicyID` links a fallback policy; `overflowChain` walks hops with cycle guard + hop cap, skipping suspended members and resolving load-balance members to weighted children; `PrepareUpload` spills to the first member with headroom for the file size so name/size/extension rules apply to the landing policy; `PreValidateUpload` checks aggregate chain headroom since batches may split across members; admin policy editor gains an Overflow policy select, en+zh locales
- [x] Thumbnail generation controls (upstream #2178 items 7+8) — `PolicySetting.ThumbForceProxy` skips the backend's native thumbnail API even when supported, implying the local proxy pipeline; `PolicySetting.ThumbStoragePolicyID` redirects generated thumb entities to a designated policy (honored via `PreferredStoragePolicy` for thumbnail uploads in `PrepareUpload`); Thumbnails section gains both controls, en+zh locales
- [x] Per-user blob relocation (upstream #2729/misc) — `RelocateEntityService` gains a third scope `src_user_id` (mutually exclusive with `entity_ids`/`src_policy_id`); `NewRelocateUserTask` selects entities by `created_by` with the same cursor-resumable transfer path; admin user editor gains a "Relocate files" button opening the relocate dialog prefilled with the user scope; en+zh locales
- [x] Tencent Captcha (upstream #2178) — `captcha_type=tcaptcha` now performs real verification: `pkg/tcaptcha` calls Tencent Cloud `DescribeCaptchaResult` with full TC3-HMAC-SHA256 request signing (CaptchaAppId/AppSecretKey + SecretId/SecretKey, CaptchaType 9, client IP propagated); login/register/forgot-password flows emit `{ticket, randstr}` from the TCaptcha.js popup widget via a new `TCaptcha` verify-button component; admin Captcha section gains the provider option + four credential fields; en+zh locales
- [x] Localized admin strings (#25/#2691) — `setting.Provider.Localized` resolves any `<key>_i18n` JSON map by language tag (exact → bare primary subtag → wildcard `*` → base value); `SiteBasicLocalized` covers site name/title/description; consumed by site config, announcement endpoint, share-preview OG tags, index.html placeholders, WOPI breadcrumb, and email templates (recipient language); SKU gains `name_i18n`/`des_i18n` columns resolved per buyer language in the shop; admin gets a reusable `LocalizedFields` accordion (per-language inputs) wired into site name/description/announcement and SKU name/description; en+zh locales
- [x] Weighted policy selection (upstream #2178 item 2) — `GroupSetting.WeightedPolicies` spreads uploads across the group's allowed policies by free capacity: `pickByFreeCapacity` picks the member with the most remaining `MaxTotalSize` headroom that fits the file (uncapped/suspended members not weighed); explicit directory/user preferences still win; size-aware `getPreferredPolicyForSize` wired into both upload paths; admin group editor gains a switch, en+zh locales
- [x] Download source typing + torrent bomb guard (upstream #2178 离线下载) — `CreateDownloadTask` distinguishes plain URLs from BitTorrent sources: `src_file` must name a `.torrent`, `magnet:` links auto-pick a BT-capable provider (qBittorrent preferred, aria2 fallback) and fail fast when only non-BT nodes exist; explicit `provider=ytdlp` with a torrent source is rejected; `validateFiles` caps selected files per task at `maxDownloadFiles` (10k) with `queue.CriticalErr` so crafted torrents cannot flood the entity table
- [x] WeChat scan login (upstream #2729 item 2) — `GET /session/wechat/login` redirects to `open.weixin.qq.com/connect/qrconnect` (scope `snsapi_login`, `#wechat_redirect` fragment); callback exchanges the code at `sns/oauth2/access_token` and binds by unionid (openid fallback); shares the single-use SSO state/ticket machinery and `sso_binding` table; provisioned accounts use synthetic `@connect.wechat.local` addresses with nickname from `/sns/userinfo`; account linking via `?link=1` + unbind via the shared provider route; admin UserSession section gains a WeChat accordion (enabled/AppID/AppSecret/register-enabled, callback URL shown); login page + security settings gain WeChat buttons; en+zh locales
- [x] SMS verification-code sign-in + phone binding — generic HTTP SMS gateway (`sms_*` settings: endpoint/method/headers/body template with `{phone}`/`{code}` placeholders, SSRF-guarded outbound call); `users.phone` unique optional column; KV-stored 6-digit codes (5-min TTL, single-use, 60s resend throttle) across `login`/`bind`/`reset` scenes; `POST /session/sms/send` (IP rate-limit + login-CAPTCHA gate) / `POST /session/sms/login` (auto-provisions synthetic `sms_*@sms.local` accounts when enabled, 2FA continuation preserved) / `POST /user/reset_sms` / `PUT|DELETE /user/setting/phone`; masked phone in user settings response; login page gains an SMS phase + reset-via-SMS mode in forgot password, security settings gain a phone-binding section, admin UserSession gains an SMS gateway accordion; en+zh locales
- [x] Direct-link traffic packs (upstream #2178 item 11) — `users.dl_traffic` (bytes, `-1` = unlimited default preserving legacy behavior); `RedirectDirectLink` atomically charges the owner's balance by file size before issuing the signed entity URL (`CodeInsufficientTraffic` = 40094 on exhaustion, balance never goes negative); new `traffic` SKU + gift-code type top up the balance permanently (unlimited users stay unlimited); admin VAS gains a Traffic product section + traffic gift-code type, Shop gains a Traffic packs tab, Finance shows the remaining allowance; en+zh locales
- [x] CLI OAuth + consent denial (ported from upstream PR #3588) — built-in `Cloudreve CLI` public client (`http://127.0.0.1/callback`, desktop scope set, empty secret + mandatory PKCE per our public-client convention); `redirectURIMatches` implements RFC 8252 loopback matching (any port on `127.0.0.1`/`[::1]` literal only — no `localhost`, no userinfo/fragments/encoded paths, strict query equality); `POST /session/oauth/consent/deny` returns `access_denied` after full client+redirect validation (`Deny` is internal, `json:"-"`); token exchange rejects PKCE downgrade (verifier without a registered challenge); migration preserves admin edits; tests cover redirect matrix, denial, downgrade, and migration idempotence

## 6. Phase D — desktop, all platforms

Goal: Windows + macOS + Linux from the `desktop/` tree in this repo.

| Layer | Windows (exists) | macOS | Linux |
|---|---|---|---|
| Placeholders/hydration | cfapi (keep) | File Provider ext (Swift bridge) | FUSE (`fuser`) or plain sync folder |
| Shell integration | shellext (keep) | Finder sync extension | Nautilus/Dolphin plugin (later) |
| Notifications | win32_notif | `mac_notification_sys` | `notify_rust` |
| Sync core | shared: `cloudreve-api`, `inventory`, `tasks`, `uploader`, `drive/sync` | same | same |

- Status: (1) notifications already per-OS (`win32_notif` / `notify_rust` / `mac_notification_sys`) — no abstraction needed; (2) hydration abstracted via `drive/placeholder` cfg swap — `cfapi` on Windows, `placeholder_non_windows` full-sync adapter elsewhere (FUSE / File Provider still open); (3) CI matrix builds + tests all 3 OSes; (4) packaging: `desktop-release.yml` on `desktop-v*` tags ships .msi/.exe (Windows), .dmg (macOS), .deb/.AppImage (Linux) — MSIX deferred (needs store signing).
- Verified on Linux: `cargo test --workspace` green (49 tests), `cargo tauri build` produces working .deb + .AppImage.
- Feature fallback on Linux/macOS until providers land: full sync without placeholders (download-on-access still works via sync engine).
- [x] #167 (upstream desktop#49) — online-only thumbnails missing in Explorer: root cause was a client/server contract mismatch — the CE `/file/thumb` response carries only `url`/`expires` while the `cloudreve-api` model required `obfuscated`, failing deserialization on every thumbnail request (`E_FAIL` to Explorer; hydrated files were unaffected since Windows thumbs them locally). `obfuscated` is now `#[serde(default)]`; the decode path still runs when a server emits the flag

## 7. Phase E — Android app (native, no iOS)

Lives in `android/` in this repo. Kotlin + Jetpack Compose, Material 3.

- **API**: `api/v4` REST + OAuth token (entities exist: `oauthclient`, `oauthgrant`) — same surface the desktop `cloudreve-api` crate documents; port its models as the spec
- **Core features**: browse/download/upload files, share links, full-text search (done — query bar + offset pagination + parent-path/snippet rows), camera-upload (done — periodic WorkManager MediaStore sync, Wi-Fi-only constraint, ID dedup, settings dialog), offline-favorite files (done — "Keep offline" downloads to filesDir/offline, DataStore registry, star dialog with open/refresh/remove), local sync folder (done — SAF tree → remote mirror, upload-only, mtime/size dedup)
- **System integration** (the "native, complete" ask): share-sheet target (done — SEND/SEND_MULTIPLE → UploadWorker), DocumentsProvider (done — Files-app browse/open/thumb/rename/delete/search), quick-share tile (done — QS tile toggles camera backup), notifications on share/task events (done — periodic /workflow poll → terminal-state notifications)
- **Auth**: OAuth flow (done — browser consent + PKCE + `cloudreve://mount` deep link); later passkey if backend exposes
- **WebDAV bridge**: `/dav` works as fallback file access until SDK matures
- Non-goals: iOS, tablet-first layouts (works, not optimized)

## 8. Governance

- License/credit: keep `LICENSE` (GPL-3.0), add `AUTHORS`/credit line to original Cloudreve authors in README — attribution without endorsement
- Release cadence: tag `fork-4.19.x` line first (cherry-picks only), then `5.0.0-fork` once Phase B lands
- Every merge: build + test + lint green (pre-push gate, non-negotiable)

---

## Issue migration format

Each fork issue: title translated to English when needed, body = `Upstream: cloudreve/cloudreve#NNNN` + short restatement + group labels. Epics get `epic` label and link children. Upstream `wontfix` items we want get `revisit` label.
