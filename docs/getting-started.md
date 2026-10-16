# Getting started

## `gor new <name>`

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
  cmd/seed/main.go       dev seed data (gor db seed)
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
  docker-compose.yml     dev Postgres (18); host port derived from the app name
  sqlc.yaml  .air.toml  Makefile  Procfile.dev  .env.example
```

Version enforcement is baked in: `go.mod` pins your Go version, and the
frontend gets `"engines"` + `engine-strict` + `.nvmrc` pinned to your Node
major, so the wrong toolchain fails loudly instead of mysteriously.

## `gor dev`

Runs Air (Go hot reload) and Vite together with prefixed, colored output;
with jobs set up, the worker runs too (`[job]` prefix). Ctrl+C stops
everything cleanly. Vite proxies `/api` to the Go server (following `PORT`
from `.env`), so there's no CORS in development. Editing `queries/*.sql` or
`migrations/*.sql` re-runs `sqlc generate` automatically before each rebuild.

## `gor doctor`

Checks every external tool (`sqlc`, `goose`, `air`, `node`/`npm`, Docker)
with install instructions for anything missing.

## Styling & theming

The frontend uses Tailwind CSS v4 with vendored shadcn/ui components
(`web/src/components/ui/` — yours to edit). Retheme the whole app by
editing the CSS variables in `web/src/index.css`; a `.dark` palette is
included. Add more components with `npx shadcn@latest add <component>`.
