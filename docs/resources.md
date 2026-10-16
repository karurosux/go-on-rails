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
- `--force` — overwrite existing files (an existing create-table migration is
  kept, never duplicated)

Toggle `--live`/`--protected` on an existing resource later with
[`gor g sync`](migrations.md#gor-g-sync-resource).
