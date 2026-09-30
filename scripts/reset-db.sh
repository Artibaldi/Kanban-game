#!/usr/bin/env bash
# Drops all data and recreates the demo database, then keeps serving on :8090 (local prototype only).
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
docker compose -f "$root/compose.local.yml" up -d --wait
cd "$root/server" && go run ./cmd/server -reset
