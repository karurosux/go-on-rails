# Realtime

Server-push for live UIs: SSE to the browser, fanned out through Postgres
`LISTEN/NOTIFY` — no WebSocket infrastructure or broker, works across
multiple app instances, and the jobs worker can publish too.

```bash
gor g realtime
```

Publish from any handler, service, or job:

```go
realtime.Publish(ctx, "posts", map[string]any{"event": "created", "id": post.ID})
```

Subscribe in React (`EventSource` reconnects automatically):

```tsx
useRealtimeInvalidate("posts", ["posts"]);          // live-refetch a query
useRealtime("chat", (data) => setMessages(...));    // or handle raw events
```

The endpoint is `GET /api/realtime/:channel`; if auth exists, it's
registered behind `auth.Require` (retrofitted automatically if auth comes
later). Payloads cap at ~8KB (NOTIFY's limit) — send events and IDs, not
whole records.

## Live resources

`gor g resource post ... --live` wires the whole loop for you: handlers
publish on create/update/delete, and the generated List/Detail pages
subscribe — every open browser updates without a reload.
