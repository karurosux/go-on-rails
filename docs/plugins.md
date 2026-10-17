# Generator plugins

Things like Stripe billing don't belong in gor core — but they make great
**installable generators**. A plugin is a git repo (or directory) holding a
`plugin.json` and templates; once installed it appears as a first-class
subcommand:

```bash
gor plugins install github.com/someone/gor-stripe   # user-wide (~/.gor/plugins)
gor plugins install ../my-plugin --project          # ships with the repo (.gor/plugins)
gor plugins list
gor g stripe                                        # run it like any generator
gor plugins remove stripe
```

Project-level plugins shadow user-level ones of the same name; plugins
can't shadow built-in generators.

## Writing a plugin

Plugins are **declarative** — gor interprets the manifest, so running one
never executes plugin code. They get the same primitives built-in
generators use: render files, inject at `gor:` markers, append env
defaults, run allowlisted commands. See
[`examples/plugins/sysinfo`](../examples/plugins/sysinfo) for a complete
working example.

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

- **Templates** render with `{{.Module}}`, `{{.AppName}}`, and `{{.Args}}`
  (extra CLI arguments). Set `"delims": "square"` on files containing
  runtime `{{ }}` syntax — then use `[[ ]]` for generation-time values.
  `.go.tmpl` output is gofmt-checked.
- **Injections** insert a rendered snippet before any `gor:` marker,
  idempotently — rerunning the plugin never duplicates lines.
- **Env** blocks append to `.env.example`/`.env` unless the sentinel key is
  already present (user values are never touched).
- **postRun** is allowlisted: `tidy` (go mod tidy) or `sqlc`
  (sqlc generate + tidy).
- Standard semantics apply: existing files refuse to be overwritten
  without `--force`.

## Trust model

Declarative manifests mean installation runs no code — but running a
plugin writes its files into your app, and those files become part of your
build. Install plugins only from sources you'd accept a pull request from.
