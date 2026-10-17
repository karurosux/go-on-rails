# Changelog

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
