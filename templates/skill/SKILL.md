---
name: gor
description: Build and evolve full-stack Go + React apps with the gor CLI (Go on Rails). Use when working inside a gor-generated app (go.mod + sqlc.yaml + .gor/ or AGENTS.md mentioning gor), or when the user asks to scaffold a Go + React app, add resources/CRUD, auth, background jobs, realtime, email, file uploads, or migrations in one.
---

# gor — Go on Rails

gor scaffolds full-stack apps (Go API: Echo + pgx + sqlc + goose; React +
Vite + TS SPA: TanStack Query, RHF + Zod, Tailwind v4 + shadcn/ui) and
generates features as plain code the user owns. **Always prefer running a
gor generator over hand-writing the equivalent code** — generators produce
backend + frontend pre-wired and record fields in `.gor/manifest.json` so
future schema changes regenerate every derived file.

Detect a gor app: `go.mod` + `sqlc.yaml` at the root (plus `.gor/manifest.json`
once resources exist). If `gor` is not on PATH:
`go install github.com/karurosux/go-on-rails/cmd/gor@latest`.

## Core workflow

```bash
gor new myapp                 # scaffold (--module, --skip-install)
cd myapp && gor db setup      # Docker Postgres + migrate
gor g resource post title:string body:text published:bool
gor db migrate
gor dev                       # API :8080, web :5173, worker if present
```

Field types: `string text int bigint bool decimal timestamp uuid references file`.
Resource names are singular; gor pluralizes. List endpoints support
`page`, `perPage`, `sort=newest|oldest`, and `search=` (when string/text
fields exist).

## Generators

```bash
gor g resource <name> <field:type>... [--service] [--api-only] [--protected] [--live] [--force]
gor g migration add_<x>_to_<table> <field:type>...     # or remove_<x>_from_<table>; auto-regens derived files
gor g sync <resource> [--live|--no-live] [--protected|--no-protected]   # regen + toggle flags
gor g auth                    # cookie sessions, roles, guards, login/register UI
gor g auth-emails             # password reset + email verification (needs auth)
gor g jobs                    # River (Postgres) job runner + cmd/worker
gor g job <name> [--every 1h] [--cron "0 9 * * *"] [--live] [--force]
gor g realtime                # SSE + Postgres NOTIFY; React hooks
gor g mailer <name>           # SMTP email + Mailpit dev catcher
gor g uploads                 # disk storage + upload control for `file` fields
```

Dependencies between generators: `--protected` needs `g auth` first;
`--live` needs `g realtime`; `file` fields need `g uploads`;
`g auth-emails` needs auth (and sets up the mailer itself). Generation
order is otherwise free — `g auth` retrofits earlier realtime/uploads
endpoints behind auth automatically.

## Database

```bash
gor db setup|reset|migrate|rollback|status|seed|gen
gor db new <name>             # empty migration skeleton
```

`gor db gen` re-runs sqlc + go mod tidy (gor dev does it on save).
`gor db reset` wipes the dev Docker volume — use for "relation already
exists" errors from stale volumes.

## Build & deploy

```bash
gor build                     # bin/<app> (+ bin/<app>-worker with jobs)
gor build --image             # distroless Docker image; worker binary included
docker run --entrypoint /<app>-worker <image>   # run worker container
```

All config is runtime env: APP_ENV, PORT, DATABASE_URL, SMTP_*, APP_URL,
STORAGE_DIR. Generators append working dev defaults to .env.example/.env.

## Rules when editing a gor app

- Never remove `gor:` marker comments (`// gor:imports`, `// gor:routes`,
  `{/* gor:routes */}`, `{/* gor:protected-routes */}`, `{/* gor:nav */}`,
  `// gor:workers`, `// gor:periodic`, `# gor:services`).
- Never edit `internal/db/gen/` (sqlc output) — edit `queries/*.sql` or
  migrations, then `gor db gen`.
- Don't create migration files by hand; use `gor g migration` / `gor db new`.
- gor skips files the user hand-edited (hash check) and prints what to
  update manually — read generator output and finish those items.
- After registration flows exist, check generator output for printed
  one-line integrations (e.g. SendVerificationEmail in register()).

## In-code APIs (generated packages)

- `auth.Require`, `auth.RequireRole("admin")`, `auth.CurrentUser(c)`;
  frontend `<RequireAuth/>`, `<RequireRole/>`, `<SignedIn/>`, `<RoleGate/>`
- `jobs.Enqueue(ctx, jobs.XArgs{...})`
- `realtime.Publish(ctx, "channel", payload)`; frontend
  `useRealtime(ch, fn)`, `useRealtimeInvalidate(ch, queryKey)`
- `mailer.SendX(ctx, to, mailer.XData{...})`
- `storage.Save(...)` / POST /api/uploads → URL stored in `file` columns

## Verify changes

```bash
go test ./... && go build ./... && (cd web && npx tsc -b)
```
