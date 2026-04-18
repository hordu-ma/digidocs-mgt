#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [[ -f .env ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    key=${line%%=*}
    if [[ -z "${!key+x}" ]]; then
      export "$line"
    fi
  done < ./.env
fi

strict="${STRICT_SMOKE:-0}"
backend_url="${BACKEND_BASE_URL:-http://127.0.0.1:18081}"
status=0
backend_ready=0
tmp_files=()
use_docker_curl=0
docker_network_name=""
assistant_poll_interval="${ASSISTANT_SMOKE_POLL_INTERVAL:-2}"
assistant_poll_attempts="${ASSISTANT_SMOKE_POLL_ATTEMPTS:-20}"
version_smoke_timeout="${VERSION_SMOKE_TIMEOUT:-20}"

is_truthy() {
  case "${1:-}" in
    1|true|TRUE|True|yes|YES|on|ON)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

cleanup() {
  for file in "${tmp_files[@]:-}"; do
    if [[ -n "$file" && -e "$file" ]]; then
      rm -f "$file"
    fi
  done
}

trap cleanup EXIT

require_or_skip() {
  local message="$1"
  if [[ "$strict" == "1" ]]; then
    echo "[MISS] $message"
    status=1
  else
    echo "[SKIP] $message"
  fi
}

extract_json_string() {
  local key="$1"
  grep -o '"'"$key"'":"[^"]*"' | head -1 | cut -d'"' -f4 || true
}

extract_synology_api_path() {
  local api_name="$1"
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; data=json.load(sys.stdin).get("data", {}); print(data.get(sys.argv[1], {}).get("path", ""), end="")' "$api_name" 2>/dev/null || true
    return
  fi
  echo ""
}

normalize_synology_api_path() {
  local api_path="${1:-}"
  if [[ -z "$api_path" ]]; then
    echo ""
  elif [[ "$api_path" == /* ]]; then
    echo "$api_path"
  elif [[ "$api_path" == webapi/* ]]; then
    echo "/$api_path"
  else
    echo "/webapi/$api_path"
  fi
}

extract_response_data_id() {
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; data=json.load(sys.stdin).get("data"); print(data.get("id", "") if isinstance(data, dict) else "", end="")' 2>/dev/null || true
    return
  fi
  extract_json_string id
}

extract_version_smoke_document_id() {
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; data=json.load(sys.stdin).get("data") or []; print(next((item.get("id", "") for item in data if isinstance(item, dict) and item.get("current_version_no") is not None), ""), end="")' 2>/dev/null || true
    return
  fi
  extract_json_string id
}

container_running() {
  local name="$1"
  docker inspect -f '{{.State.Running}}' "$name" 2>/dev/null | grep -qx 'true'
}

smoke_curl() {
  if [[ "$use_docker_curl" == "1" ]]; then
    docker run --rm --network "$docker_network_name" -v /tmp:/tmp curlimages/curl:8.7.1 "$@"
  else
    curl "$@"
  fi
}

synology_curl() {
  curl "$@"
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
if smoke_curl -fsS --max-time 3 "$backend_url/healthz" >/dev/null 2>&1; then
  echo "[OK] $backend_url/healthz"
  backend_ready=1
elif container_running "digidocs-backend-go"; then
  docker_network_name=$(docker inspect -f '{{range $k, $v := .NetworkSettings.Networks}}{{$k}}{{end}}' digidocs-backend-go 2>/dev/null || true)
  if [[ -n "$docker_network_name" ]]; then
    backend_url="http://backend-go:8080"
    use_docker_curl=1
  fi
fi

if [[ "$use_docker_curl" == "1" ]] && smoke_curl -fsS --max-time 3 "$backend_url/healthz" >/dev/null 2>&1; then
  echo "[OK] $backend_url/healthz via docker network ($docker_network_name)"
  backend_ready=1
elif [[ "$backend_ready" != "1" ]]; then
  require_or_skip "$backend_url/healthz is unreachable"
fi

echo '== business endpoint smoke =='
if [[ "$backend_ready" == "1" ]]; then
  # Login and get a token
  login_resp=$(smoke_curl -sS --max-time 5 -X POST "$backend_url/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"admin123"}' 2>/dev/null || true)
  token=$(echo "$login_resp" | extract_json_string access_token)

  if [[ -n "$token" ]]; then
    echo "[OK] auth/login returned token"
    auth_header="Authorization: Bearer $token"

    # GET /documents
    doc_status=$(smoke_curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/documents?page=1&page_size=5" 2>/dev/null)
    if [[ "$doc_status" == "200" ]]; then
      echo "[OK] GET /documents -> $doc_status"
    else
      require_or_skip "GET /documents -> $doc_status (expected 200)"
    fi

    # GET /dashboard/overview
    dash_status=$(smoke_curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/dashboard/overview" 2>/dev/null)
    if [[ "$dash_status" == "200" ]]; then
      echo "[OK] GET /dashboard/overview -> $dash_status"
    else
      require_or_skip "GET /dashboard/overview -> $dash_status (expected 200)"
    fi

    # GET /handovers
    ho_status=$(smoke_curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/handovers" 2>/dev/null)
    if [[ "$ho_status" == "200" ]]; then
      echo "[OK] GET /handovers -> $ho_status"
    else
      require_or_skip "GET /handovers -> $ho_status (expected 200)"
    fi

    # GET /audit-events
    ae_status=$(smoke_curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/audit-events?page=1" 2>/dev/null)
    if [[ "$ae_status" == "200" ]]; then
      echo "[OK] GET /audit-events -> $ae_status"
    else
      require_or_skip "GET /audit-events -> $ae_status (expected 200)"
    fi

    # GET /audit-events/summary
    as_status=$(smoke_curl -sS -o /dev/null -w '%{http_code}' --max-time 5 \
      -H "$auth_header" "$backend_url/api/v1/audit-events/summary" 2>/dev/null)
    if [[ "$as_status" == "200" ]]; then
      echo "[OK] GET /audit-events/summary -> $as_status"
    else
      require_or_skip "GET /audit-events/summary -> $as_status (expected 200)"
    fi

    # POST /documents/{id}/versions + GET /versions/{id}/download|preview
    documents_resp=$(smoke_curl -sS --max-time 5 -H "$auth_header" "$backend_url/api/v1/documents?page=1&page_size=5" 2>/dev/null || true)
    document_id=$(echo "$documents_resp" | extract_version_smoke_document_id)

    if [[ -n "$document_id" ]]; then
      upload_tmp=$(mktemp)
      tmp_files+=("$upload_tmp")
      printf 'smoke second version\n' >"$upload_tmp"
      chmod 644 "$upload_tmp"

      version_resp=$(smoke_curl -sS --max-time "$version_smoke_timeout" -X POST "$backend_url/api/v1/documents/$document_id/versions" \
        -H "$auth_header" \
        -F 'commit_message=smoke second upload' \
        -F "file=@$upload_tmp;filename=smoke-v2.txt;type=text/plain" 2>/dev/null || true)
      version_id=$(echo "$version_resp" | extract_response_data_id)

      if [[ -n "$version_id" ]]; then
        download_tmp=$(mktemp)
        tmp_files+=("$download_tmp")
        download_status=$(smoke_curl -sS -o "$download_tmp" -w '%{http_code}' --max-time "$version_smoke_timeout" \
          -H "$auth_header" "$backend_url/api/v1/versions/$version_id/download" 2>/dev/null)
        if [[ "$download_status" == "200" && "$(cat "$download_tmp")" == 'smoke second version' ]]; then
          echo "[OK] version upload/download -> $download_status"
        else
          require_or_skip "version upload/download smoke failed (status=$download_status version_id=$version_id)"
        fi

        preview_tmp=$(mktemp)
        tmp_files+=("$preview_tmp")
        preview_status=$(smoke_curl -sS -o "$preview_tmp" -w '%{http_code}' --max-time "$version_smoke_timeout" \
          -H "$auth_header" "$backend_url/api/v1/versions/$version_id/preview" 2>/dev/null)
        if [[ "$preview_status" == "200" && "$(cat "$preview_tmp")" == 'smoke second version' ]]; then
          echo "[OK] version preview -> $preview_status"
        else
          require_or_skip "version preview smoke failed (status=$preview_status version_id=$version_id)"
        fi
      else
        require_or_skip "version smoke did not return expected version id; body=$version_resp"
      fi
    else
      require_or_skip "document lookup failed; cannot run version smoke"
    fi

    # POST /assistant/ask + poll /assistant/requests/{id}
    ask_resp=$(smoke_curl -sS --max-time 10 -X POST "$backend_url/api/v1/assistant/ask" \
      -H "$auth_header" \
      -H 'Content-Type: application/json' \
      -d "{\"question\":\"请用一句话确认 smoke 已打通 AI 链路\",\"scope\":{\"document_id\":\"$document_id\"}}" 2>/dev/null || true)
    request_id=$(echo "$ask_resp" | extract_json_string request_id)

    if [[ -n "$request_id" ]]; then
      echo "[OK] POST /assistant/ask queued request_id=$request_id"
      final_status=""
      final_body=""
      for ((poll_index=1; poll_index<=assistant_poll_attempts; poll_index++)); do
        final_body=$(smoke_curl -sS --max-time 10 -H "$auth_header" "$backend_url/api/v1/assistant/requests/$request_id" 2>/dev/null || true)
        final_status=$(echo "$final_body" | extract_json_string status)
        if [[ "$final_status" == "completed" || "$final_status" == "failed" ]]; then
          break
        fi
        sleep "$assistant_poll_interval"
      done

      if [[ "$final_status" == "completed" ]]; then
        echo "[OK] GET /assistant/requests/$request_id -> completed"
      else
        require_or_skip "GET /assistant/requests/$request_id -> ${final_status:-unknown} after ${assistant_poll_attempts} polls x ${assistant_poll_interval}s; body=$final_body"
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

if is_truthy "${RUN_SYNOLOGY_PREFLIGHT:-0}" || [[ "${STORAGE_BACKEND:-}" == "synology" ]]; then
  echo '== synology preflight =='
  synology_scheme="http"
  synology_port="${SYNOLOGY_PORT:-5000}"
  if is_truthy "${SYNOLOGY_HTTPS:-false}"; then
    synology_scheme="https"
    if [[ -z "${SYNOLOGY_PORT:-}" ]]; then
      synology_port="5001"
    fi
  fi

  if [[ -z "${SYNOLOGY_HOST:-}" || -z "${SYNOLOGY_ACCOUNT:-}" || -z "${SYNOLOGY_PASSWORD:-}" || -z "${SYNOLOGY_SHARE_PATH:-}" ]]; then
    require_or_skip 'synology preflight requires SYNOLOGY_HOST/SYNOLOGY_ACCOUNT/SYNOLOGY_PASSWORD/SYNOLOGY_SHARE_PATH'
  else
    synology_base_url="$synology_scheme://$SYNOLOGY_HOST:$synology_port"
    synology_curl_opts=(--silent --show-error --max-time 15)
    if is_truthy "${SYNOLOGY_INSECURE_SKIP_VERIFY:-0}"; then
      synology_curl_opts+=(-k)
    fi

    info_resp=$(synology_curl "${synology_curl_opts[@]}" --get "$synology_base_url/webapi/query.cgi" \
      --data-urlencode 'api=SYNO.API.Info' \
      --data-urlencode 'version=1' \
      --data-urlencode 'method=query' \
      --data-urlencode 'query=SYNO.API.Auth,SYNO.FileStation.Upload,SYNO.FileStation.List,SYNO.FileStation.Download,SYNO.FileStation.CreateFolder,SYNO.FileStation.Sharing' 2>/dev/null || true)
    if [[ "$info_resp" == *'"success":true'* ]]; then
      echo '[OK] SYNO.API.Info reachable'
    else
      require_or_skip 'SYNO.API.Info query failed'
    fi

    auth_path=$(echo "$info_resp" | extract_synology_api_path 'SYNO.API.Auth')
    if [[ -z "$auth_path" ]]; then
      auth_path='/webapi/auth.cgi'
    else
      auth_path=$(normalize_synology_api_path "$auth_path")
    fi

    login_resp=$(synology_curl "${synology_curl_opts[@]}" --get "$synology_base_url$auth_path" \
      --data-urlencode 'api=SYNO.API.Auth' \
      --data-urlencode 'version=3' \
      --data-urlencode 'method=login' \
      --data-urlencode "account=$SYNOLOGY_ACCOUNT" \
      --data-urlencode "passwd=$SYNOLOGY_PASSWORD" \
      --data-urlencode 'session=FileStation' \
      --data-urlencode 'format=sid' 2>/dev/null || true)
    synology_sid=$(echo "$login_resp" | extract_json_string sid)

    if [[ -n "$synology_sid" ]]; then
      echo '[OK] DSM/File Station login succeeded'
      smoke_folder="${SYNOLOGY_SHARE_PATH%/}/_digidocs_smoke_$$"
      smoke_parent=$(dirname "$smoke_folder")
      smoke_name=$(basename "$smoke_folder")
      smoke_file="${smoke_folder}/smoke.txt"

      create_resp=$(synology_curl "${synology_curl_opts[@]}" --get "$synology_base_url/webapi/entry.cgi" \
        --data-urlencode 'api=SYNO.FileStation.CreateFolder' \
        --data-urlencode 'version=2' \
        --data-urlencode 'method=create' \
        --data-urlencode "folder_path=$smoke_parent" \
        --data-urlencode "name=$smoke_name" \
        --data-urlencode 'force_parent=true' \
        --data-urlencode "_sid=$synology_sid" 2>/dev/null || true)
      if [[ "$create_resp" == *'"success":true'* ]]; then
        echo "[OK] created synology smoke folder $smoke_folder"
      else
        require_or_skip "failed to create synology smoke folder $smoke_folder"
      fi

      synology_tmp=$(mktemp)
      tmp_files+=("$synology_tmp")
      printf 'digidocs synology smoke\n' >"$synology_tmp"
      chmod 644 "$synology_tmp"
      upload_resp=$(synology_curl "${synology_curl_opts[@]}" -X POST "$synology_base_url/webapi/entry.cgi?_sid=$synology_sid" \
        -F 'api=SYNO.FileStation.Upload' \
        -F 'version=2' \
        -F 'method=upload' \
        -F "path=$smoke_folder" \
        -F 'create_parents=true' \
        -F 'overwrite=true' \
        -F "file=@$synology_tmp;filename=smoke.txt;type=text/plain" 2>/dev/null || true)
      if [[ "$upload_resp" == *'"success":true'* ]]; then
        echo '[OK] synology upload succeeded'
      else
        require_or_skip 'synology upload failed'
      fi

      list_resp=$(synology_curl "${synology_curl_opts[@]}" --get "$synology_base_url/webapi/entry.cgi" \
        --data-urlencode 'api=SYNO.FileStation.List' \
        --data-urlencode 'version=2' \
        --data-urlencode 'method=list' \
        --data-urlencode "folder_path=$smoke_folder" \
        --data-urlencode 'additional=["size","time"]' \
        --data-urlencode "_sid=$synology_sid" 2>/dev/null || true)
      if [[ "$list_resp" == *'smoke.txt'* ]]; then
        echo '[OK] synology list/getinfo succeeded'
      else
        require_or_skip 'synology list/getinfo failed'
      fi

      share_resp=$(synology_curl "${synology_curl_opts[@]}" --get "$synology_base_url/webapi/entry.cgi" \
        --data-urlencode 'api=SYNO.FileStation.Sharing' \
        --data-urlencode 'version=3' \
        --data-urlencode 'method=create' \
        --data-urlencode "path=$smoke_file" \
        --data-urlencode "_sid=$synology_sid" 2>/dev/null || true)
      if [[ "$share_resp" == *'"url":'* ]]; then
        echo '[OK] synology share link creation succeeded'
      else
        require_or_skip 'synology share link creation failed'
      fi

      synology_download=$(mktemp)
      tmp_files+=("$synology_download")
      download_status=$(synology_curl "${synology_curl_opts[@]}" -o "$synology_download" -w '%{http_code}' --get "$synology_base_url/webapi/entry.cgi" \
        --data-urlencode 'api=SYNO.FileStation.Download' \
        --data-urlencode 'version=2' \
        --data-urlencode 'method=download' \
        --data-urlencode "path=$smoke_file" \
        --data-urlencode 'mode=download' \
        --data-urlencode "_sid=$synology_sid" 2>/dev/null || true)
      if [[ "$download_status" == "200" && "$(cat "$synology_download")" == 'digidocs synology smoke' ]]; then
        echo '[OK] synology download succeeded'
      else
        require_or_skip 'synology download failed'
      fi

      delete_resp=$(synology_curl "${synology_curl_opts[@]}" --get "$synology_base_url/webapi/entry.cgi" \
        --data-urlencode 'api=SYNO.FileStation.Delete' \
        --data-urlencode 'version=2' \
        --data-urlencode 'method=delete' \
        --data-urlencode "path=$smoke_folder" \
        --data-urlencode 'recursive=true' \
        --data-urlencode "_sid=$synology_sid" 2>/dev/null || true)
      if [[ "$delete_resp" == *'"success":true'* ]]; then
        echo '[OK] synology cleanup succeeded'
      else
        require_or_skip 'synology cleanup failed'
      fi
    else
      require_or_skip 'DSM/File Station login failed'
    fi
  fi
fi

echo '== summary =='
if [[ "$status" == "0" ]]; then
  echo '[OK] local smoke checks passed or were skipped in non-strict mode'
fi

exit "$status"
