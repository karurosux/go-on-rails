#!/usr/bin/env bash
# Smoke test: builds gor, scaffolds a throwaway app, runs every generator,
# and verifies the result compiles, passes its tests, and typechecks.
# Database-free: nothing here needs Postgres (sqlc works from the schema).
#
# Requires: go, node/npm, sqlc on PATH.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "==> building gor"
GOR="$WORK/gor"
(cd "$ROOT" && go build -o "$GOR" ./cmd/gor)

command -v sqlc >/dev/null || {
  echo "sqlc is required: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"
  exit 1
}

cd "$WORK"
echo "==> gor new"
"$GOR" new smoke --skip-install
cd smoke
go mod tidy

echo "==> generators"
"$GOR" g realtime
"$GOR" g auth
"$GOR" g auth-emails
"$GOR" g uploads
"$GOR" g jobs
"$GOR" g job cleanup --every 1h --live
"$GOR" g job report --cron "0 9 * * *"
"$GOR" g mailer welcome
"$GOR" g resource post title:string body:text published:bool views:int price:decimal published_at:timestamp attachment:file --live --protected
"$GOR" g resource author name:string bio:text --service --api-only
"$GOR" g resource widget title:string body:text in_stock:bool --headless
"$GOR" g model tag name:string
"$GOR" g migration add_slug_to_posts slug:string
"$GOR" g sync post --no-live
"$GOR" skills install --project

echo "==> backend: vet + build + tests"
go vet ./...
go build ./...
go test ./...

echo "==> frontend: install + strict typecheck"
cd web
npm install --no-fund --no-audit
npx tsc -b

echo
echo "SMOKE OK"
