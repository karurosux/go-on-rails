# Email

Transactional email over plain SMTP (works with Resend, Postmark, SES,
anything — no provider SDK), with templates embedded in the binary:

```bash
gor g mailer welcome
```

Each mailer gets a typed `Send` function, an HTML template rendered in a
shared layout, a plain-text alternative (better spam scores), and a render
test:

```go
mailer.SendWelcome(ctx, user.Email, mailer.WelcomeData{Name: user.Name})
```

First use also sets up `internal/mailer/` (SMTP client + shared layout)
and adds **Mailpit** to docker-compose — the dev mail catcher: every email
sent in development lands in a web UI instead of a real inbox (the
generator prints its URL; ports are derived per app).

## Configuration

Via env: `SMTP_ADDR` (host:port), `SMTP_USERNAME`/`SMTP_PASSWORD`,
`MAIL_FROM`. Dev defaults point at Mailpit and are appended to
`.env.example`/`.env` automatically. With no `SMTP_ADDR` at all, emails
are logged rather than erroring — development never fails on missing mail
config.

## Reliability

For emails that must not get lost, call the mailer from a
[background job](jobs.md) — River retries failed deliveries with backoff.
Inline sending from a handler is fine for non-critical mail.
