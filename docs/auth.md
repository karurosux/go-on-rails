# Authentication

## `gor g auth`

Cookie-session authentication, generated as plain code you own:

- `users` + `sessions` tables (bcrypt passwords, hashed session tokens,
  sliding 7-day expiry, `role` column defaulting to `user`)
- `internal/auth/`: middleware + guards — `auth.Require`,
  `auth.RequireRole("admin")`, `auth.CurrentUser(c)` — and endpoints
  `POST /api/auth/register|login|logout`, `GET /api/auth/me`
- `/login` and `/register` pages, session-aware nav, a `<RequireAuth />`
  route guard (plus `<RequireRole />`), and inline gates (`<SignedIn>`,
  `<SignedOut>`, `<RoleGate>`) on the frontend

## Protecting resources

```bash
gor g resource note title:string --protected     # new: behind auth.Require
gor g sync note --protected                      # existing: move behind auth
```

Or by hand: in `routes.go` change `note.Register(api, q)` to
`note.Register(protected, q)`. Role-gate route groups with
`admin := api.Group("/admin", auth.RequireRole("admin"))`.

Promote an admin: `UPDATE users SET role = 'admin' WHERE email = '...';`

Endpoints generated before auth (realtime, uploads) are moved behind
`auth.Require` automatically when you generate auth — order doesn't matter.

## `gor g auth-emails`

Password reset + email verification on top of auth (sets up the
[mailer](mailer.md) if needed): tokens table (hashed, single-use,
expiring), endpoints, both emails (HTML + text), and the
`/forgot-password`, `/reset-password`, `/verify-email` pages. A password
reset signs the user out of every session.

Two one-line integrations are printed for you to paste (gor never edits
your code): sending the verification email in `register()`, and the
"forgot password?" link on the login page.

## Security posture

Rails-style defaults: httpOnly + SameSite=Lax cookies (Secure in
production), CSRF protection via an Origin-check middleware
(`auth.CSRF()`), login timing equalization against email enumeration,
server-side revocable sessions storing only token hashes, per-IP rate
limiting on credential endpoints (`auth.LoginRateLimit()`), bcrypt
passwords, parameterized SQL (sqlc), React escaping, explicit input
structs (no mass assignment), security headers, and global rate limiting.
