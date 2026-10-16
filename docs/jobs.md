# Background jobs

Jobs run on [River](https://riverqueue.com) — Postgres-backed, so there's
no Redis or broker to run; enqueues are as durable as your database.

```bash
gor g jobs                             # once: jobs package + cmd/worker process
gor g job send_welcome_email           # per job: args + worker + test, registered
gor g job cleanup_sessions --every 1h  # also runs on a schedule (Go duration)
gor g job daily_report --cron "0 9 * * *"  # or at specific times (cron syntax)
gor g job nightly_sync --live              # streams job lifecycle to the browser
```

Enqueue from anywhere (handlers, services, other jobs):

```go
jobs.Enqueue(c.Request().Context(), jobs.SendWelcomeEmailArgs{UserID: id})
```

Workers get the app's sqlc queries injected (`w.q`), so a job can read the
database and act on it. Returning an error from `Work` retries the job
with exponential backoff.

## Schedules

- `--every 1h` runs on the interval, plus once at worker startup. Intervals
  are Go durations (`30m`, `1h`, `24h`).
- `--cron "0 9 * * *"` runs at specific times (standard 5-field syntax,
  worker's local time; prefix the expression with `CRON_TZ=<zone>` to pin a
  timezone). Cron jobs don't fire at startup.

Schedules live as one readable line each in `internal/jobs/jobs.go` — edit
or delete them there. A scheduled job can still be enqueued on demand.

## Live jobs

`--live` jobs publish `started`/`completed`/`failed` on the `"jobs"`
[realtime](realtime.md) channel — show progress in React with
`useRealtime("jobs", ...)`.

## Running the worker

`gor dev` runs it automatically (hot-reloaded, `[job]` prefix). `gor build`
produces `bin/<app>-worker` next to the server binary, and the Docker image
contains it too — see [deployment](deployment.md). River's queue tables
migrate themselves when the worker starts.
