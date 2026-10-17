# Deployment

## `gor build`

`npm run build` (with `tsc` typechecking), then `go build` — producing
`bin/<app>`: one binary serving the API and the embedded frontend.

```bash
gor build
APP_ENV=production DATABASE_URL=... ./bin/myapp
```

With jobs set up, `bin/<app>-worker` is built alongside the server.

## Docker image

A deployable image (~27 MB, distroless, runs as non-root):

```bash
gor build --image              # tags <app>:latest
gor build --tag myapp:v1.2     # custom tag
docker run -p 8080:8080 -e DATABASE_URL=postgres://... myapp:v1.2
```

The image contains the worker binary too — run it as a second container:

```bash
docker run -e DATABASE_URL=... --entrypoint /myapp-worker myapp:v1.2
```

The Dockerfile is multi-stage (node builds the frontend, Go embeds it) and
self-adjusts its Go toolchain to `go.mod` via `GOTOOLCHAIN=auto`.

## Configuration

Nothing is baked in at build time — everything is read at runtime from the
environment: `APP_ENV` (`production` enables Secure cookies and JSON
logs), `PORT`, `DATABASE_URL`, `SMTP_*`/`MAIL_FROM` (mailer), `APP_URL`
(links in auth emails), and uploads: `STORAGE_DRIVER=disk` with a
persistent volume at `STORAGE_DIR`, or `STORAGE_DRIVER=s3` with the
`S3_*` settings for any S3-compatible store. In development the same
values come from `.env`, which generators keep filled with working
defaults.
