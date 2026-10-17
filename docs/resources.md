# Resources

## `gor g resource <name> <field:type>...`

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

## Field types

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
| `file`       | text (URL)   | string      | string     | upload control |

`author:references` creates an `author_id` column referencing `authors(id)`;
`attachment:file` stores an uploaded file's URL (needs
[`gor g uploads`](uploads.md)).

## Listing, search, and sort

List endpoints paginate (`page`, `perPage`), sort (`sort=newest|oldest`),
and — when the resource has string/text fields — search (`search=`, ILIKE
across those fields), with matching controls on the generated List page.

## Flags

- `--service` — adds `service.go` between handlers and the database, for domain logic
- `--api-only` — skip all frontend generation
- `--protected` — register behind `auth.Require` on both sides (needs
  [`gor g auth`](auth.md))
- `--live` — realtime resource (needs [`gor g realtime`](realtime.md)):
  handlers publish on create/update/delete and the List/Detail pages
  subscribe, so every open browser updates without a reload
- `--headless` — frontend building blocks only: Zod schema, typed client +
  hooks, and a props-driven `<XForm id? onSuccess? onCancel? />` with no
  router coupling — embed it in modals, wizards, or your own pages. No
  List/Detail pages, nothing injected into routes or nav
- `--force` — overwrite existing files (an existing create-table migration is
  kept, never duplicated)

Toggle `--live`/`--protected` on an existing resource later with
[`gor g sync`](migrations.md#gor-g-sync-resource).

## Per-app flag defaults

If your project always wants certain flags (e.g. you build custom pages and
only want headless building blocks), set defaults once in
`.gor/config.json` — the gor equivalent of Rails' `config.generators`:

```json
{ "resource": { "headless": true, "service": true } }
```

Explicit CLI flags always override the config (`--headless=false` brings
the pages back for one resource).

## `gor g model <name> <field:type>...`

The data layer only — migration + sqlc queries, no API and no frontend.
For internal tables (settings, join tables, logs) that handlers and jobs
query directly:

```bash
gor g model tag name:string
```

Promote it to a full resource later with
`gor g resource tag <same fields> --force`.
