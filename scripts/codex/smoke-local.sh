#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

strict="${STRICT_SMOKE:-0}"
backend_url="${BACKEND_BASE_URL:-http://127.0.0.1:18081}"
status=0

require_or_skip() {
  local message="$1"
  if [[ "$strict" == "1" ]]; then
    echo "[MISS] $message"
    status=1
  else
    echo "[SKIP] $message"
  fi
}

container_running() {
  local name="$1"
  docker inspect -f '{{.State.Running}}' "$name" 2>/dev/null | grep -qx 'true'
}

echo '== smoke prerequisites =='
if ! command -v docker >/dev/null 2>&1; then
  require_or_skip 'docker command is unavailable'
  exit "$status"
fi

echo '== compose services =='
for container in digidocs-postgres digidocs-redis digidocs-minio; do
  if container_running "$container"; then
    echo "[OK] $container running"
  else
    require_or_skip "$container is not running"
  fi
done

for container in digidocs-backend-go digidocs-backend-py-worker; do
  if container_running "$container"; then
    echo "[OK] $container running"
  else
    require_or_skip "$container is not running; start with docker compose --profile app up -d backend-go backend-py-worker"
  fi
done

echo '== backend healthz =='
if curl -fsS --max-time 3 "$backend_url/healthz" >/dev/null 2>&1; then
  echo "[OK] $backend_url/healthz"
else
  require_or_skip "$backend_url/healthz is unreachable"
fi

echo '== summary =='
if [[ "$status" == "0" ]]; then
  echo '[OK] local smoke checks passed or were skipped in non-strict mode'
fi

exit "$status"
