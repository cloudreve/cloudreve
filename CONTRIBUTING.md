# Contributing

This is an actively maintained community fork of Cloudreve. Contributions are welcome — the goal is
a complete, fully open-source distribution, so Pro-class features land here as free software rather
than behind a license key.

## Before you start

- **Check the issue tracker first.** Migrated upstream issues are labeled by group
  (`group:*`, `pro-free`, `security`, `revisit`, `epic`); each carries an honest status note.
  [ROADMAP.md](ROADMAP.md) describes the phase plan (B.2 storage policies, B.4 VAS, B.5 system
  extensions, Phase D desktop, Phase E Android).
- **One change per PR.** Split large features into reviewable increments.
- **No CLA.** Unlike upstream there is no contributor agreement — contributions are GPL-3.0 like
  the project itself. Do not submit code copied from Cloudreve Pro sources.

## Workflow

1. Branch from `master` — never push to `master` directly.
2. Keep changes idiomatic: Gin + ent on the backend, React + MUI + Redux conventions in `frontend/`,
   existing provider/interface seams over new abstractions.
3. Run the pre-push gate, all of it:
   - `go build ./... && go vet ./... && go test ./...`
   - `cd frontend && yarn tsc --noEmit && yarn build` (use `NODE_OPTIONS=--max-old-space-size=6144`
     if Vite hits the default heap limit)
   - Boot the binary and smoke the endpoints you touched.
4. Open the PR against `Dvorinka/cloudreve` `master` and wait for all CI jobs (backend, frontend,
   desktop matrix) to go green.

## Style notes

- Security-relevant changes (auth, SSRF, process execution, file paths, secrets) get extra scrutiny —
  say so in the PR description.
- Comments only where they carry information; Go doc conventions for exported symbols.
- AI-assisted contributions are fine — you are responsible for what you submit.
