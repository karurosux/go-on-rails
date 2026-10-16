# Schema changes

## `gor g migration <name> [field:type...]`

Evolve the schema after a resource exists. With fields and a conventional
name, the column SQL is generated for you:

```bash
gor g migration add_slug_to_posts slug:string
gor g migration remove_views_from_posts views:int
gor g migration backfill_post_slugs          # no fields: empty skeleton
```

Added columns get `NOT NULL DEFAULT <zero>` so existing rows stay valid;
`uuid`/`references` columns (no safe default) are added nullable with a
backfill TODO.

**The rest is automatic:** gor records each resource's fields in
`.gor/manifest.json`, so an add/remove migration also regenerates the
queries, handler, TS client, Zod schema, and pages. Files you have edited
by hand are never overwritten — they're listed for you to update instead
(and the Go/TS compilers point at anything missed). Then just:

```bash
gor db migrate
```

## `gor g sync <resource>`

Re-runs that regeneration on demand (e.g. after deleting a file to get a
pristine copy), and can toggle the situational flags on an existing
resource — routes move on both sides automatically:

```bash
gor g sync post --live            # make it realtime
gor g sync post --protected       # move behind auth
gor g sync post --no-protected    # make it public again
gor g sync post --no-live         # stop publishing/subscribing
```
