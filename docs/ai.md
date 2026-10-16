# AI agents

gor is built to be driven by AI coding agents as much as by humans — the
generators are the API, and two artifacts teach agents to use them instead
of hand-writing boilerplate.

## In every generated app: `AGENTS.md`

`gor new` generates an `AGENTS.md` (plus a `CLAUDE.md` that imports it, so
Claude Code picks it up automatically) containing the agent contract for
that app:

- the golden rule — prefer gor generators over hand-written boilerplate
- the command cheat sheet and field types
- hard rules: never remove `gor:` markers, never edit `internal/db/gen/`,
  schema changes go through `gor g migration`
- how to verify changes (`go test`, `tsc -b`, `go build`)

Any agent that reads repo instruction files (Claude Code, Cursor, Copilot,
etc.) gets it for free.

## The gor skill

For agents with skill support (Claude Code), install the gor skill so the
full CLI workflow is available even before an app exists — scaffolding new
projects included:

```bash
gor skills install             # user-wide: ~/.claude/skills/gor
gor skills install --project   # ships with the repo: .claude/skills/gor
```

The skill covers the complete command surface (generators, dependencies
between them, db/build commands, in-code APIs) and the editing rules.
Re-run the install after upgrading gor to refresh it.

## Why this matters

Generators keep both halves of the app consistent through
`.gor/manifest.json` — an agent that hand-writes a migration or CRUD
endpoint bypasses that machinery and loses auto-regeneration. Teaching the
agent the commands keeps its work on the same rails as yours.
