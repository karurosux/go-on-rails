---
name: gor-plugins
description: Author generator plugins for the gor CLI (Go on Rails). Use when the user asks to create, write, or modify a gor plugin — an installable generator (plugin.json + templates) that runs as `gor g <name>` inside gor apps — or when working in a directory containing a gor plugin.json.
---

# Writing gor generator plugins

A gor plugin is a **declarative** generator: a directory with `plugin.json`
and templates. gor interprets the manifest — plugin code never executes —
using the same primitives as built-in generators: render files, inject at
markers, append env defaults, run allowlisted commands.

## Start from the scaffold

```bash
gor plugins new <name>        # creates ./gor-<name>/ with a working example
```

Names are lowercase `[a-z][a-z0-9-]*` and may not shadow built-in
generators (resource, model, migration, sync, auth, auth-emails, jobs,
job, realtime, mailer, uploads).

## plugin.json schema

```json
{
  "name": "stripe",
  "description": "Stripe billing scaffolding",
  "files": [
    {"template": "templates/billing.go.tmpl", "path": "internal/billing/billing.go"},
    {"template": "templates/Page.tsx.tmpl", "path": "web/src/features/billing/Page.tsx", "delims": "square"}
  ],
  "injections": [
    {"file": "internal/routes/routes.go", "marker": "// gor:imports", "snippet": "templates/import.snippet"},
    {"file": "internal/routes/routes.go", "marker": "// gor:routes", "snippet": "templates/routes.snippet"}
  ],
  "env": {"sentinel": "STRIPE_SECRET_KEY", "file": "templates/env.tmpl"},
  "postRun": ["tidy"],
  "message": "Next: set STRIPE_SECRET_KEY, then gor db migrate"
}
```

Rules that matter:
- Template data is {{.Module}} (app module path), {{.AppName}}, {{.Args}}
  (extra CLI args after `gor g <name>`). Default delimiters are {{ }};
  set `"delims": "square"` on files containing runtime Go-template syntax
  (email/HTML templates), then use [[ ]] for plugin-time values there.
- `.go.tmpl` output must be valid Go — gor gofmt-checks it and fails the
  run otherwise.
- Injections are idempotent (keyed on the snippet's first line) and indent
  to match the marker. Markers in generated apps: `// gor:imports` and
  `// gor:routes` in internal/routes/routes.go (an `api` group, plus
  `protected` behind auth when the app has it); `{/* gor:routes */}`,
  `{/* gor:protected-routes */}` in web/src/routes.tsx;
  `{/* gor:nav */}` in web/src/App.tsx; `// gor:workers`, `// gor:periodic`
  in internal/jobs/jobs.go; `# gor:services` in docker-compose.yml.
- `postRun` allowlist: "tidy" (go mod tidy) and "sqlc" (sqlc generate +
  tidy — use when the plugin adds migrations/ or queries/ files).
- Existing files are never overwritten without the user passing --force;
  design templates to be rerunnable.
- Plugins can add migrations: use a `{{timestamp}}` token in the path
  (`"path": "migrations/{{timestamp}}_create_billing.sql"`) — it resolves
  to a fresh goose version at run time (and re-runs reuse the existing
  file instead of duplicating it). Include goose `-- +goose Up/Down`
  sections, then postRun sqlc.

## Dev loop

```bash
gor new testapp && cd testapp
gor plugins install ../gor-<name> --project --link   # dev install: symlinked, edits are live
gor g <name>                                         # run; --force to rerun after edits
go build ./... && go test ./... && (cd web && npx tsc -b)
```

Without --link, install copies the plugin (what end users get); linked
installs are for development only — don't commit the symlink.

## Conventions for good plugins

Follow gor's own: feature packages under internal/<name>/, frontend under
web/src/features/<name>/ importing "@/components/ui/*" (shadcn) and
"../../api/client", env-configured with dev-safe defaults, and generate the
frontend counterpart when a natural one exists. Keep the README honest
about what the plugin writes.

## Publish

Push the plugin directory as a git repo; users run
`gor plugins install <git-url>`. Installation copies files only — but the
generated code becomes part of their build, so write it like a PR you'd
send them.
