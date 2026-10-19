# Changelog

## Unreleased

- `gor dev` now refuses to start when the API port is already taken, naming
  the squatting process (command, pid, start time). Previously Air's server
  died on the bind while Vite kept proxying `/api` to the stale process, so
  every new route 404'd with no hint why.
- `gor dev` runs each process (air, npm, worker) in its own process group and
  signals the whole group on shutdown, so stopping the wrappers can no longer
  orphan the real server or Vite, leaving them squatting on the port.
- `gor dev` warns when nothing is listening on the API port 60s after start —
  Air stays alive even when the server it launched failed, which used to make
  a dead dev session look healthy.

## v0.3.0

- `gor plugins install --link`: dev installs — the plugin directory is
  symlinked instead of copied, so edits apply immediately (rerun with
  `--force`); `plugins list` shows the link target.
- `gor plugins new <name>`: scaffold a generator plugin with a working
  example and a README covering the manifest schema, markers, and dev
  loop. `gor skills install` now also installs a `gor-plugins` skill that
  teaches AI agents to author plugins.

- `gor g observability`: vendor-neutral monitoring — 5xx error reports to
  any Sentry-compatible service via `SENTRY_DSN` (Sentry, GlitchTip,
  Bugsink; unset = disabled, `obs.CaptureError` for jobs), and Prometheus
  request/runtime metrics at `/metrics`.

- Auth gains the guest-only guard: `auth.RequireGuest` middleware (login
  and register now return 403 for signed-in users) and a `<RequireGuest />`
  route guard wrapping /login and /register, redirecting signed-in
  visitors back to where they came from. Existing apps: re-run
  `gor g auth --force` (auth routes in routes.tsx are left as-is — wrap
  them in `<Route element={<RequireGuest />}>` yourself).

- Fix: the CSRF middleware rejected every mutation in development ("cross-
  origin request rejected") because the Vite proxy rewrites the Host header
  while the browser's Origin stays on the dev-server port. `auth.CSRF()` is
  now a no-op outside production, and in production it additionally allows
  the `APP_URL` origin for Host-rewriting proxies. Existing apps: replace
  the `CSRF` function in `internal/auth/middleware.go` (or re-run
  `gor g auth --force` if your auth files are unmodified).

## v0.2.0

- Generator plugins: `gor plugins install <git-url|path>` (user-wide or
  `--project`) installs declarative third-party generators that appear as
  `gor g <name>` — files, marker injections, env defaults, and allowlisted
  post-runs, no plugin code execution. Example: `examples/plugins/sysinfo`.

- `gor g model <name> <fields>`: data layer only (migration + sqlc
  queries) for internal tables.
- `gor g resource --headless`: frontend building blocks without routes —
  Zod schema, typed client + hooks, and a props-driven Form
  (`id?`, `onSuccess`, `onCancel`) to embed in modals or custom pages.
- Per-app generator defaults in `.gor/config.json`
  (`{ "resource": { "headless": true } }`), Rails `config.generators`
  style; explicit CLI flags always override.

- `gor g uploads` now generates two storage drivers selected at runtime by
  `STORAGE_DRIVER`: `disk` (default) and `s3` for any S3-compatible store
  (AWS S3, Cloudflare R2, MinIO, DigitalOcean Spaces) via `S3_*` env vars,
  with optional `S3_PUBLIC_URL` for CDN-fronted buckets.

## v0.1.0

First release.

### Scaffolding & core
- `gor new`: Go API (Echo, pgx, sqlc, goose, slog, graceful shutdown) +
  React/Vite/TypeScript SPA (React Router, TanStack Query, RHF + Zod,
  Tailwind v4, vendored shadcn/ui) served from one binary. Docker Postgres
  with per-app host ports, Go/Node version enforcement, `AGENTS.md`/`CLAUDE.md`
  for AI agents.
- `gor dev` (Air + Vite + worker, sqlc regen on save), `gor doctor`,
  `gor db setup|reset|migrate|rollback|status|new|seed|gen`.

### Generators
- `gor g resource`: full-stack CRUD with validation, pagination,
  search/sort, and flags `--service`, `--api-only`, `--protected`, `--live`.
  Field types incl. `references` (FK + index) and `file` (uploads).
- `gor g migration` / `gor g sync`: schema evolution with automatic
  regeneration of derived files via `.gor/manifest.json`; hand-edited files
  are never overwritten. Sync toggles `--live`/`--protected` on existing
  resources.
- `gor g auth`: cookie sessions (sliding, hashed tokens, revocable), roles,
  guards + frontend gates, CSRF origin checks, login timing equalization,
  per-IP credential rate limiting. `gor g auth-emails`: password reset
  (signs out everywhere) + email verification.
- `gor g jobs` / `g job`: River-backed background jobs with `--every`,
  `--cron`, and `--live` lifecycle streaming; worker process integrated in
  dev, build, and the Docker image.
- `gor g realtime`: SSE + Postgres LISTEN/NOTIFY with React hooks.
- `gor g mailer`: SMTP email with HTML + text templates and Mailpit dev
  catcher. `gor g uploads`: disk storage, upload endpoint + form control.
- `gor skills install`: Claude Code skill for AI-driven usage.

### Build & deploy
- `gor build`: single binary (+ worker). `--image`/`--tag`: distroless
  Docker image containing both binaries. All configuration via runtime env.
