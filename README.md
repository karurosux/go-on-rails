# gor — Go on Rails

A Rails-style CLI for full-stack **Go + React** apps.

- **Backend:** idiomatic Go API — [Echo](https://echo.labstack.com), Postgres via
  [pgx](https://github.com/jackc/pgx), SQL-first queries with [sqlc](https://sqlc.dev),
  migrations with [goose](https://github.com/pressly/goose), `log/slog`, graceful shutdown.
- **Frontend:** React + Vite + TypeScript (strict) SPA — React Router, TanStack Query,
  React Hook Form + Zod, Tailwind CSS v4 + [shadcn/ui](https://ui.shadcn.com).
  No server-rendered templates.
- **Production:** one Go binary that serves the API *and* the embedded React build.

Generated code is plain, readable Go and TypeScript that you own. No runtime
framework layer, no reflection magic — every template lives in [templates/](templates/)
and is easy to edit.

## Install

Straight from GitHub (needs Go 1.25+):

```bash
go install github.com/karurosux/go-on-rails/cmd/gor@latest
```

This installs the `gor` binary into `$(go env GOPATH)/bin` — make sure that's
on your PATH. Or from a clone:

```bash
git clone git@github.com:karurosux/go-on-rails.git
cd go-on-rails
go install ./cmd/gor
```

Check your toolchain with `gor doctor` — it tells you exactly what's
missing (`sqlc`, `goose`, `air`, `node`/`npm`, optionally Docker) and how
to install it.

## Quick start

```bash
gor new blog
cd blog
gor db setup        # starts Postgres via Docker + migrates
gor g resource post title:string body:text published:bool
gor db migrate
gor dev             # http://localhost:5173
```

You now have a CRUD API at `/api/posts` and working List/Detail/Create/Edit
pages at `/posts`, with validation on both sides.

## Documentation

| Guide | Commands | What you get |
|---|---|---|
| [Getting started](docs/getting-started.md) | `new`, `dev`, `doctor` | project scaffold, dev servers, theming |
| [Resources](docs/resources.md) | `g resource` | full-stack CRUD, field types, search/sort, `--protected`/`--live` |
| [Schema changes](docs/migrations.md) | `g migration`, `g sync` | evolving columns with auto-regeneration |
| [Database](docs/database.md) | `db setup/migrate/seed/...` | goose wrapper, Docker Postgres, seeds |
| [Authentication](docs/auth.md) | `g auth`, `g auth-emails` | sessions, roles, guards, password reset, verification |
| [Background jobs](docs/jobs.md) | `g jobs`, `g job` | River queue, schedules (`--every`, `--cron`), live jobs |
| [Realtime](docs/realtime.md) | `g realtime` | SSE + Postgres NOTIFY, live-updating pages |
| [Email](docs/mailer.md) | `g mailer` | SMTP mailers, templates, Mailpit dev catcher |
| [File uploads](docs/uploads.md) | `g uploads` | disk storage, upload control, `file` fields |
| [Deployment](docs/deployment.md) | `build`, `build --image` | single binary, Docker image, worker, runtime config |
| [AI agents](docs/ai.md) | `skills install` | AGENTS.md in every app, installable Claude Code skill |

## Conventions

- **One application, not two.** Every backend capability ships with its
  frontend counterpart, pre-connected: resources get typed clients and pages,
  auth gets guards and gates, realtime gets hooks — and flags like
  `--protected` and `--live` wire both sides together in one command.
- Resource names are **singular** (`post`); gor pluralizes for tables, routes, and URLs.
- Feature-based layout: one Go package and one React feature folder per resource.
- Generators insert at marker comments — keep `// gor:imports`, `// gor:routes`,
  `{/* gor:routes */}`, and `{/* gor:nav */}` where they are.
- Generators that introduce configuration append working dev defaults to
  `.env.example` (and `.env` if present) — values you've already set are
  never touched, and generated code runs without hand-editing env files.
- Everything gor writes is formatted (`gofmt`) and conflict-checked up front:
  a collision aborts the whole generation before any file is written.
- Generators never edit code you've written: hand-modified files are
  skipped with a warning, and the rare one-line integrations are printed
  for you to paste.

## Extending gor

Templates are embedded from [templates/](templates/) — edit them and
rebuild to change what gor generates. The marker-comment system and
feature-folder layout are the extension points for anything you add
(an admin panel is the next big one on the list).

## Development

```bash
go test ./...          # field parser, type mappings, injection, template rendering
./scripts/smoke.sh     # scaffold a throwaway app, run every generator, compile + typecheck
go install ./cmd/gor   # install your local build
```

CI runs both on every push and pull request. Releases are tagged
(`vX.Y.Z`) and recorded in [CHANGELOG.md](CHANGELOG.md).
