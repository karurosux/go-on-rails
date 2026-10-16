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

Check your toolchain:

```bash
gor doctor
```

`gor` shells out to `sqlc`, `goose`, `air`, `node`/`npm`, and (optionally) Docker.
`gor doctor` tells you exactly what's missing and how to install it.

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

## Commands

### `gor new <name>`

Scaffolds a full project, runs `go mod tidy` and `npm install`, and initializes git.

```bash
gor new myapp
gor new myapp --module github.com/me/myapp   # custom Go module path
gor new myapp --skip-install                 # skip tidy/npm/git (offline, CI)
```

Generated layout:

```
myapp/
  cmd/server/main.go     wiring, middleware, graceful shutdown
  internal/
    config/              env config, .env support in dev
    db/                  pgx pool; db/gen = sqlc output
    httpx/               JSON helpers, validation, error envelope
    middleware/          slog request logging, SPA serving
    routes/routes.go     route registration (generators append here)
  migrations/            goose SQL migrations
  queries/               sqlc query files
  web/                   React + Vite + TS app (src/api, src/features)
    src/components/ui/   vendored shadcn/ui components (button, input, table, ...)
    src/index.css        Tailwind v4 theme — CSS variables retheme everything
    components.json      shadcn CLI config: npx shadcn@latest add <component>
  embed.go               //go:embed web/dist with SPA fallback
  docker-compose.yml     dev Postgres (18), host port 5433
  sqlc.yaml  .air.toml  Makefile  Procfile.dev  .env.example
```

### `gor g resource <name> <field:type>...`

```bash
gor g resource post title:string body:text published:bool author:references
```

Generates, idempotently (refuses to overwrite without `--force`):

- goose migration (`id` UUIDv7 PK, your fields, `created_at`/`updated_at`;
  `references` fields get a foreign key + index)
- sqlc queries: paginated List, Count, Get, Create, partial Update, Delete —
  then runs `sqlc generate` and `go mod tidy`
- `internal/<name>/handler.go` — REST handlers calling sqlc directly, with
  validation and a consistent JSON error envelope:
  `GET|POST /api/<plural>`, `GET|PATCH|DELETE /api/<plural>/:id`
- `internal/<name>/handler_test.go` — `httptest`-based handler tests
- `web/src/api/<plural>.ts` — typed fetch client + TanStack Query hooks
- `web/src/features/<plural>/` — `List`, `Detail`, `Form` (create + edit) pages
  with a Zod schema wired into React Hook Form
- Route + nav registration on both sides, inserted at `// gor:` marker comments

Field types and their mappings:

| type         | Postgres     | Go          | TypeScript | form input     |
|--------------|--------------|-------------|------------|----------------|
| `string`     | text         | string      | string     | text           |
| `text`       | text         | string      | string     | textarea       |
| `int`        | integer      | int32       | number     | number         |
| `bigint`     | bigint       | int64       | number     | number         |
| `bool`       | boolean      | bool        | boolean    | checkbox       |
| `decimal`    | numeric      | string      | string     | text           |
| `timestamp`  | timestamptz  | time.Time   | string     | datetime-local |
| `uuid`       | uuid         | uuid.UUID   | string     | text           |
| `references` | uuid + FK    | uuid.UUID   | string     | text           |

`author:references` creates an `author_id` column referencing `authors(id)`.

Flags:

- `--service` — adds `service.go` between handlers and the database, for domain logic
- `--api-only` — skip all frontend generation
- `--force` — overwrite existing files (an existing create-table migration is
  kept, never duplicated)

### `gor g auth`

Cookie-session authentication, generated as plain code you own:

- `users` + `sessions` tables (bcrypt passwords, hashed session tokens,
  7-day expiry, `role` column defaulting to `user`)
- `internal/auth/`: middleware + guards — `auth.Require`,
  `auth.RequireRole("admin")`, `auth.CurrentUser(c)` — and endpoints
  `POST /api/auth/register|login|logout`, `GET /api/auth/me`
- `/login` and `/register` pages, session-aware nav, and a `<RequireAuth />`
  route guard (plus `<RequireRole />`) on the frontend

Protect resources:

```bash
gor g resource note title:string --protected     # new: behind auth.Require
# existing: in routes.go change  note.Register(api, q) -> note.Register(protected, q)
# role-gate: admin := api.Group("/admin", auth.RequireRole("admin"))
```

Promote an admin: `UPDATE users SET role = 'admin' WHERE email = '...';`

Security posture (Rails-style defaults): httpOnly + SameSite=Lax cookies
(Secure in production), CSRF protection via an Origin-check middleware
(`auth.CSRF()`), login timing equalization against email enumeration,
server-side revocable sessions storing only token hashes, bcrypt passwords,
parameterized SQL (sqlc), React escaping, explicit input structs (no mass
assignment), security headers, and rate limiting. Email verification,
password reset, and per-login rate limits are left as extension points.

### `gor g jobs` / `gor g job <name>`

Background jobs on [River](https://riverqueue.com) — Postgres-backed, so
there's no Redis or broker to run; enqueues are as durable as your database.

```bash
gor g jobs                             # once: jobs package + cmd/worker process
gor g job send_welcome_email           # per job: args + worker + test, registered
gor g job cleanup_sessions --every 1h  # also runs on a schedule (Go duration)
gor g job daily_report --cron "0 9 * * *"  # or at specific times (cron syntax)
```

Enqueue from anywhere (handlers, services, other jobs):

```go
jobs.Enqueue(c.Request().Context(), jobs.SendWelcomeEmailArgs{UserID: id})
```

Returning an error from a job's `Work` retries it with exponential backoff.
`--every` jobs run on their interval (plus once at worker startup) with no
cron infrastructure — the schedule lives in `internal/jobs/jobs.go`, where
you can tune or remove it. A scheduled job can still be enqueued on demand.
`gor dev` runs the worker automatically (hot-reloaded, `[job]` prefix);
`gor build` produces `bin/<app>-worker` next to the server binary. River's
queue tables migrate themselves when the worker starts.

### `gor g migration <name> [field:type...]`

Evolve the schema after a resource exists. With fields and a conventional
name, the column SQL is generated for you:

```bash
gor g migration add_slug_to_posts slug:string
gor g migration remove_views_from_posts views:int
gor g migration backfill_post_slugs          # no fields: empty skeleton
```

Added columns get `NOT NULL DEFAULT <zero>` so existing rows stay valid;
`uuid`/`references` columns (no safe default) are added nullable with a
backfill TODO.

**The rest is automatic:** gor records each resource's fields in
`.gor/manifest.json`, so an add/remove migration also regenerates the
queries, handler, TS client, Zod schema, and pages. Files you have edited
by hand are never overwritten — they're listed for you to update instead
(and the Go/TS compilers point at anything missed). Then just:

```bash
gor db migrate
```

`gor g sync <resource>` re-runs that regeneration on demand (e.g. after
deleting a file to get a pristine copy).

### `gor dev`

Runs Air (Go hot reload) and Vite together with prefixed, colored output.
Ctrl+C stops both cleanly. Vite proxies `/api` to the Go server, so there's
no CORS in development. Editing `queries/*.sql` re-runs `sqlc generate`
automatically before each rebuild.

### `gor db <subcommand>`

Thin goose wrapper using `DATABASE_URL` (from the environment or `.env`):

```bash
gor db setup          # docker compose up + wait for Postgres + migrate
gor db reset          # wipe the dev database (down -v) and set up fresh
gor db migrate        # goose up
gor db rollback       # goose down
gor db status         # goose status
gor db new add_slug   # create an empty timestamped migration
gor db gen            # re-run sqlc generate + go mod tidy by hand
```

(`gor dev` also re-runs sqlc automatically whenever you save a file in
`queries/` or `migrations/` — `gor db gen` is for when Air isn't running.)

No Docker? Start Postgres however you like, set `DATABASE_URL`, and use
`gor db migrate`. Note: generated migrations default ids to `uuidv7()`
(Postgres 18+); on older Postgres switch them to `gen_random_uuid()`.

### `gor build`

`npm run build` (with `tsc` typechecking), then `go build` — producing
`bin/<app>`: one binary serving the API and the embedded frontend.

```bash
gor build
APP_ENV=production DATABASE_URL=... ./bin/myapp
```

Or build a deployable Docker image (~27 MB, distroless, runs as non-root):

```bash
gor build --image              # tags <app>:latest
gor build --tag myapp:v1.2     # custom tag
docker run -p 8080:8080 -e DATABASE_URL=postgres://... myapp:v1.2
```

### `gor doctor`

Checks every external tool with install instructions for anything missing.

## Conventions

- Resource names are **singular** (`post`); gor pluralizes for tables, routes, and URLs.
- Feature-based layout: one Go package and one React feature folder per resource.
- Generators insert at marker comments — keep `// gor:imports`, `// gor:routes`,
  `{/* gor:routes */}`, and `{/* gor:nav */}` where they are.
- Everything gor writes is formatted (`gofmt`) and conflict-checked up front:
  a collision aborts the whole generation before any file is written.

## Extending gor

Templates are embedded from [templates/](templates/) — edit them and rebuild to
change what gor generates. Auth, background jobs, mailers, file storage, and an
admin panel are intentionally out of scope for v0.1; the marker-comment system
and feature-folder layout are the extension points they'll use.

## Development

```bash
go test ./...          # field parser, type mappings, injection, template rendering
go install ./cmd/gor   # install your local build
```
