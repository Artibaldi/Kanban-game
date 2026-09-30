#!/usr/bin/env bash
# All checks: Go unit + end-to-end API tests (throwaway database), web rules tests, type check and build.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
docker compose -f "$root/compose.local.yml" up -d --wait
export DATABASE_URL="${DATABASE_URL:-postgres://kanban:kanban-local-only@127.0.0.1:15433/kanban?sslmode=disable}"
( cd "$root/server" && go vet ./... && go test -count=1 ./... )
( cd "$root/web" && npm test && npm run build )
