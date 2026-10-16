# Database

## `gor db <subcommand>`

Thin goose wrapper using `DATABASE_URL` (from the environment or `.env`):

```bash
gor db setup          # docker compose up + wait for Postgres + migrate
gor db reset          # wipe the dev database (down -v) and set up fresh
gor db migrate        # goose up
gor db rollback       # goose down
gor db status         # goose status
gor db new add_slug   # create an empty timestamped migration
gor db seed           # run cmd/seed (created on demand) — keep seeds idempotent
gor db gen            # re-run sqlc generate + go mod tidy by hand
```

(`gor dev` also re-runs sqlc automatically whenever you save a file in
`queries/` or `migrations/` — `gor db gen` is for when Air isn't running.)

## Without Docker

Start Postgres however you like, set `DATABASE_URL`, and use
`gor db migrate`. Note: generated migrations default ids to `uuidv7()`
(Postgres 18+); on older Postgres switch them to `gen_random_uuid()`.

## Seeds

`gor db seed` runs `cmd/seed/main.go` (generated with the app, or created
on demand for older apps). It gets a connected pool and the sqlc queries —
fill in your data and keep it idempotent, so running it twice is safe.
