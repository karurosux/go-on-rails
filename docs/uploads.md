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

## Storage

`STORAGE_DIR` overrides the location (default `./storage/uploads`;
gitignored). In production, mount a persistent volume there. To move to
S3-compatible storage later, swap the `Save` function in
`internal/storage/storage.go` — the handler and frontend stay the same.
