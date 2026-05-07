#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

backend_url="${BACKEND_BASE_URL:-http://127.0.0.1:18081}"
frontend_url="${FRONTEND_BASE_URL:-http://127.0.0.1:18080}"
assistant_smoke="${RUN_ASSISTANT_SMOKE:-0}"

echo '== e2e compose up =='
if [[ "$assistant_smoke" == "1" || "$assistant_smoke" == "true" || "$assistant_smoke" == "TRUE" ]]; then
  docker compose --profile app up -d postgres backend-go backend-py-worker frontend
else
  docker compose --profile app up -d postgres backend-go frontend
fi

echo '== wait backend =='
backend_ready=0
for _ in $(seq 1 60); do
  if curl -fsS --max-time 2 "$backend_url/healthz" >/dev/null 2>&1; then
    backend_ready=1
    break
  fi
  sleep 2
done
if [[ "$backend_ready" != "1" ]]; then
  echo "[MISS] $backend_url/healthz is unreachable after waiting"
  exit 1
fi
echo "[OK] $backend_url/healthz"

echo '== load seed data =='
docker exec -i digidocs-postgres psql \
  -v ON_ERROR_STOP=1 \
  -U "${POSTGRES_USER:-postgres}" \
  -d "${POSTGRES_DB:-digidocs_mgt}" \
  < backend-go/sql/seed.sql >/dev/null
echo '[OK] backend-go/sql/seed.sql loaded'

echo '== frontend health =='
if curl -fsS --max-time 5 "$frontend_url/" >/dev/null; then
  echo "[OK] $frontend_url/"
else
  echo "[MISS] $frontend_url/ is unreachable"
  exit 1
fi

echo '== api smoke =='
STRICT_SMOKE="${STRICT_SMOKE:-1}" RUN_ASSISTANT_SMOKE="$assistant_smoke" BACKEND_BASE_URL="$backend_url" ./scripts/codex/smoke-local.sh
