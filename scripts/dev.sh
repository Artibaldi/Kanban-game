#!/usr/bin/env bash
# Local development: Postgres in Docker, Go API on :8090, Vite dev server on :5173 (proxies /api to the API).
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
docker compose -f "$root/compose.local.yml" up -d --wait
( cd "$root/web" && [ -d node_modules ] || npm install )
( cd "$root/server" && go run ./cmd/server ) &
api=$!
trap 'kill $api 2>/dev/null' EXIT
cd "$root/web" && npm run dev
