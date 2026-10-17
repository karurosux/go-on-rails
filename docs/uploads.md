# File uploads

File uploads on local disk (S3-ready seam), as the counterpart of the
`file` field type:

```bash
gor g uploads                                   # storage pkg + endpoint + upload control
gor g resource doc title:string attachment:file # forms get an upload button
```

`POST /api/uploads` (multipart `file` field, 10MB limit, behind auth when
it exists) stores the file under a random key and returns its
`/uploads/<key>` URL; the resource column stores that URL. Files are
served at `GET /uploads/<key>`.

The frontend gets `web/src/components/upload-input.tsx` — the control
generated forms use for `file` fields (upload button, view link, remove) —
and `uploadFile()` in `web/src/api/uploads.ts` for custom UIs.

## Storage drivers

`STORAGE_DRIVER` picks the backend at runtime — no code changes to deploy:

- **`disk`** (default): local directory, `STORAGE_DIR` overrides the
  location (default `./storage/uploads`; gitignored). Served at
  `/uploads/*`; mount a persistent volume in production.
- **`s3`**: any S3-compatible store — AWS S3, Cloudflare R2, MinIO,
  DigitalOcean Spaces. Configure `S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`,
  `S3_ACCESS_KEY`/`S3_SECRET_KEY`. Stored URLs point at
  `<endpoint>/<bucket>/<key>` by default, so the bucket must be publicly
  readable — or set `S3_PUBLIC_URL` to a CDN/public base in front of it.

The upload endpoint, form control, and stored URL columns are identical
under both drivers.
