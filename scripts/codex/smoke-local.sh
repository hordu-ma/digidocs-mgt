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
for container in digidocs-postgres; do
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

echo '== business endpoint smoke =='
if curl -fsS --max-time 3 "$backend_url/healthz" >/dev/null 2>&1; then
  # Login and get a token
  login_resp=$(curl -sS --max-time 5 -X POST "$backend_url/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"admin123"}' 2>/dev/null)
  token=$(echo "$login_resp" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4)

  if [[ -n "$token" ]]; then
    echo "[OK] auth/login returned token"
    auth_header="Authorization: Bearer $token"

    # GET /documents
    doc_status=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/documents?page=1&page_size=5" 2>/dev/null)
    if [[ "$doc_status" == "200" ]]; then
      echo "[OK] GET /documents -> $doc_status"
    else
      require_or_skip "GET /documents -> $doc_status (expected 200)"
    fi

    # GET /dashboard/overview
    dash_status=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/dashboard/overview" 2>/dev/null)
    if [[ "$dash_status" == "200" ]]; then
      echo "[OK] GET /dashboard/overview -> $dash_status"
    else
      require_or_skip "GET /dashboard/overview -> $dash_status (expected 200)"
    fi

    # GET /handovers
    ho_status=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/handovers" 2>/dev/null)
    if [[ "$ho_status" == "200" ]]; then
      echo "[OK] GET /handovers -> $ho_status"
    else
      require_or_skip "GET /handovers -> $ho_status (expected 200)"
    fi

    # GET /audit-events
    ae_status=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/audit-events?page=1" 2>/dev/null)
    if [[ "$ae_status" == "200" ]]; then
      echo "[OK] GET /audit-events -> $ae_status"
    else
      require_or_skip "GET /audit-events -> $ae_status (expected 200)"
    fi

    # GET /audit-events/summary
    as_status=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/audit-events/summary" 2>/dev/null)
    if [[ "$as_status" == "200" ]]; then
      echo "[OK] GET /audit-events/summary -> $as_status"
    else
      require_or_skip "GET /audit-events/summary -> $as_status (expected 200)"
    fi

    # POST /assistant/ask + poll /assistant/requests/{id}
    ask_resp=$(curl -sS --max-time 10 -X POST "$backend_url/api/v1/assistant/ask" \
      -H "$auth_header" \
      -H 'Content-Type: application/json' \
      -d '{"question":"请用一句话确认 smoke 已打通 AI 链路","scope":{"project_id":null,"document_id":null}}' 2>/dev/null)
    request_id=$(echo "$ask_resp" | grep -o '"request_id":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [[ -n "$request_id" ]]; then
      echo "[OK] POST /assistant/ask queued request_id=$request_id"
      final_status=""
      final_body=""
      for _ in 1 2 3 4 5 6 7 8; do
        final_body=$(curl -sS --max-time 10 -H "$auth_header" "$backend_url/api/v1/assistant/requests/$request_id" 2>/dev/null)
        final_status=$(echo "$final_body" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
        if [[ "$final_status" == "completed" || "$final_status" == "failed" ]]; then
          break
        fi
        sleep 2
      done

      if [[ "$final_status" == "completed" ]]; then
        echo "[OK] GET /assistant/requests/$request_id -> completed"
      else
        require_or_skip "GET /assistant/requests/$request_id -> ${final_status:-unknown}; body=$final_body"
      fi
    else
      require_or_skip "POST /assistant/ask did not return request_id"
    fi
  else
    require_or_skip "auth/login did not return a token (seed data may not be loaded)"
  fi
else
  echo "[SKIP] backend not reachable, skipping business endpoint smoke"
fi

echo '== summary =='
if [[ "$status" == "0" ]]; then
  echo '[OK] local smoke checks passed or were skipped in non-strict mode'
fi

exit "$status"
