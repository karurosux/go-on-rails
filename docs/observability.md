# Observability

Monitoring is the one place gor's "your database is enough" philosophy
doesn't apply: error aggregation, metric storage, and dashboards are
products. So gor generates **vendor-neutral emission** and leaves storage
to external services (hosted or self-hosted):

```bash
gor g observability
```

## Errors — Sentry protocol

Every 5xx response is reported with its request context to whatever
`SENTRY_DSN` points at — hosted [Sentry](https://sentry.io), or
self-hosted [GlitchTip](https://glitchtip.com)/[Bugsink](https://bugsink.com)
(both speak the Sentry protocol). Unset DSN = reporting disabled, zero
overhead. Client errors (4xx, validation) are never reported.

Outside the request cycle — jobs, goroutines:

```go
if err := doWork(); err != nil {
    obs.CaptureError(err)
    return err // still let River retry
}
```

## Metrics — Prometheus

`GET /metrics` serves request counts and latency histograms per route
(plus Go runtime metrics) in Prometheus format. Scrape it with
self-hosted Prometheus/Grafana, Grafana Cloud, or any vendor agent that
speaks the format. **Keep /metrics firewalled or behind your
infrastructure** — it's for your scraper, not the public.

## Logs

Nothing to generate: the app already writes structured JSON to stdout in
production (`log/slog`). Point your collector (Vector, Promtail, a PaaS
log drain) at it; `request_id` correlates lines per request.

## What's deliberately external

Dashboards, alerting rules, uptime checks, and tracing backends. If you
later want distributed tracing, OpenTelemetry is the vendor-neutral path —
an extension point, not generated today.
