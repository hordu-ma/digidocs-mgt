#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ASSET_DIR="${WUHAO_ASSET_DIR:-$HOME/workspace/asset-base/internal/五好爱学}"
API_BASE="${API_BASE:-http://localhost:18081/api/v1}"
BACKEND_HEALTH_URL="${BACKEND_HEALTH_URL:-http://localhost:18081/healthz}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-digidocs-postgres}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-digidocs_mgt}"
HTTP_CLIENT_CONTAINER="${HTTP_CLIENT_CONTAINER:-}"
CONTAINER_API_BASE="${CONTAINER_API_BASE:-http://backend-go:8080/api/v1}"
CONTAINER_ASSET_DIR="${CONTAINER_ASSET_DIR:-/tmp/digidocs-wuhao-assets}"

ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"
WUHAO_LOAD_MODE="${WUHAO_LOAD_MODE:-metadata}"

TEAM_SPACE_ID="10000000-0000-0000-0000-000000000002"
PROJECT_ID="20000000-0000-0000-0000-000000000004"
PROJECT_OWNER_ID="00000000-0000-0000-0000-000000000010"
DEFAULT_OWNER_ID="00000000-0000-0000-0000-000000000011"

require_cmd() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    echo "[ERROR] missing command: $name" >&2
    exit 1
  fi
}

sql_escape() {
  printf "%s" "$1" | sed "s/'/''/g"
}

psql_exec() {
  docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"
}

curl_api() {
  if [[ -n "$HTTP_CLIENT_CONTAINER" ]]; then
    docker exec "$HTTP_CLIENT_CONTAINER" curl "$@"
    return
  fi
  curl "$@"
}

prepare_http_client() {
  if [[ -n "$HTTP_CLIENT_CONTAINER" ]]; then
    API_BASE="$CONTAINER_API_BASE"
    docker exec "$HTTP_CLIENT_CONTAINER" mkdir -p "$CONTAINER_ASSET_DIR"
    docker cp "$ASSET_DIR/." "$HTTP_CLIENT_CONTAINER:$CONTAINER_ASSET_DIR/"
    return
  fi

  if curl --max-time 3 -fsS "$BACKEND_HEALTH_URL" >/dev/null; then
    return
  fi

  HTTP_CLIENT_CONTAINER="${FALLBACK_HTTP_CLIENT_CONTAINER:-digidocs-frontend}"
  API_BASE="$CONTAINER_API_BASE"
  docker exec "$HTTP_CLIENT_CONTAINER" curl --max-time 3 -fsS "http://backend-go:8080/healthz" >/dev/null
  docker exec "$HTTP_CLIENT_CONTAINER" mkdir -p "$CONTAINER_ASSET_DIR"
  docker cp "$ASSET_DIR/." "$HTTP_CLIENT_CONTAINER:$CONTAINER_ASSET_DIR/"
  echo "[INFO] host backend port is unavailable; using $HTTP_CLIENT_CONTAINER as HTTP client"
}

folder_id_for() {
  case "$1" in
    "产品规划") printf "30000000-0000-0000-0000-000000000006" ;;
    "用户手册") printf "30000000-0000-0000-0000-000000000007" ;;
    "生涯规划") printf "30000000-0000-0000-0000-000000000008" ;;
    "资质证明") printf "30000000-0000-0000-0000-000000000009" ;;
    "素材图片") printf "30000000-0000-0000-0000-000000000010" ;;
    *) echo "[ERROR] unknown folder alias: $1" >&2; exit 1 ;;
  esac
}

ensure_project_and_folders() {
  psql_exec <<SQL
BEGIN;

INSERT INTO projects (id, team_space_id, name, code, description, owner_id, status)
VALUES (
  '$PROJECT_ID',
  '$TEAM_SPACE_ID',
  '五好爱学 AI 教育产品资料库',
  'wuhao-ai',
  '五好爱学产品、规划、用户手册、生涯规划、资质与素材测试项目',
  '$PROJECT_OWNER_ID',
  'active'
)
ON CONFLICT ON CONSTRAINT uq_projects_team_space_code DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    status = EXCLUDED.status,
    updated_at = NOW();

INSERT INTO folders (id, project_id, parent_id, name, path, sort_order)
VALUES
  ('30000000-0000-0000-0000-000000000006', '$PROJECT_ID', NULL, '产品规划', '/产品规划', 10),
  ('30000000-0000-0000-0000-000000000007', '$PROJECT_ID', NULL, '用户手册', '/用户手册', 20),
  ('30000000-0000-0000-0000-000000000008', '$PROJECT_ID', NULL, '生涯规划', '/生涯规划', 30),
  ('30000000-0000-0000-0000-000000000009', '$PROJECT_ID', NULL, '资质证明', '/资质证明', 40),
  ('30000000-0000-0000-0000-000000000010', '$PROJECT_ID', NULL, '素材图片', '/素材图片', 50)
ON CONFLICT ON CONSTRAINT uq_folders_project_path DO UPDATE
SET name = EXCLUDED.name,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();

COMMIT;
SQL
}

login_token() {
  curl_api --max-time 30 -fsS \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$ADMIN_USERNAME\",\"password\":\"$ADMIN_PASSWORD\"}" \
    "$API_BASE/auth/login" \
    | jq -r ".data.access_token"
}

find_document_id() {
  local title="$1"
  local escaped_title
  escaped_title="$(sql_escape "$title")"
  psql_exec -At -c "SELECT d.id FROM documents d WHERE d.project_id = '$PROJECT_ID'::uuid AND d.title = '$escaped_title' AND d.is_deleted = false ORDER BY d.created_at LIMIT 1;"
}

upload_fixture() {
  local token="$1"
  local folder_alias="$2"
  local title="$3"
  local rel_path="$4"
  local description="$5"
  local folder_id
  local file_path
  local upload_path
  local doc_id
  local response

  folder_id="$(folder_id_for "$folder_alias")"
  file_path="$ASSET_DIR/$rel_path"
  if [[ ! -f "$file_path" ]]; then
    echo "[ERROR] fixture file not found: $file_path" >&2
    exit 1
  fi
  upload_path="$file_path"
  if [[ -n "$HTTP_CLIENT_CONTAINER" ]]; then
    upload_path="$CONTAINER_ASSET_DIR/$rel_path"
  fi

  doc_id="$(find_document_id "$title")"
  if [[ -n "$doc_id" ]]; then
    response="$(curl_api --max-time 120 -fsS \
      -H "Authorization: Bearer $token" \
      -F "commit_message=刷新 wuhao-ai 测试资源文件" \
      -F "file=@$upload_path" \
      "$API_BASE/documents/$doc_id/versions")"
    echo "[OK] uploaded new version: $title -> $doc_id ($(jq -r '.data.version_no // "n/a"' <<<"$response"))"
    return
  fi

  response="$(curl_api --max-time 120 -fsS \
    -H "Authorization: Bearer $token" \
    -F "team_space_id=$TEAM_SPACE_ID" \
    -F "project_id=$PROJECT_ID" \
    -F "folder_id=$folder_id" \
    -F "title=$title" \
    -F "description=$description" \
    -F "current_owner_id=$DEFAULT_OWNER_ID" \
    -F "commit_message=导入 wuhao-ai 测试资源" \
    -F "file=@$upload_path" \
    "$API_BASE/documents")"
  echo "[OK] created document: $title -> $(jq -r '.data.id' <<<"$response")"
}

main() {
  require_cmd docker
  require_cmd curl
  require_cmd jq
  require_cmd sed

  if [[ ! -d "$ASSET_DIR" ]]; then
    echo "[ERROR] asset directory not found: $ASSET_DIR" >&2
    exit 1
  fi

  cd "$ROOT_DIR"
  if [[ "$WUHAO_LOAD_MODE" == "metadata" ]]; then
    psql_exec -f - < backend-go/sql/wuhao_ai_seed.sql
    psql_exec -c "SELECT p.code, count(d.id) AS document_count FROM projects p LEFT JOIN documents d ON d.project_id = p.id AND d.is_deleted = false WHERE p.id = '$PROJECT_ID'::uuid GROUP BY p.code;"
    echo "[OK] loaded wuhao-ai metadata fixtures. Set WUHAO_LOAD_MODE=upload to push files through the storage backend."
    return
  fi

  ensure_project_and_folders
  prepare_http_client

  local token
  token="$(login_token)"
  if [[ -z "$token" || "$token" == "null" ]]; then
    echo "[ERROR] login failed for $ADMIN_USERNAME" >&2
    exit 1
  fi

  upload_fixture "$token" "产品规划" "五好教育发展及规划" "“五好教育”发展及规划.docx" "五好教育发展方向、产品规划与资料沉淀。"
  upload_fixture "$token" "产品规划" "学情雷达系统建设路径" "学情雷达系统建设路径.pdf" "学情雷达系统建设方案与路径说明。"
  upload_fixture "$token" "用户手册" "Wuhao Tutor 用户手册" "Wuhao-tutor_User_Manual.pdf" "Wuhao Tutor 产品用户手册。"
  upload_fixture "$token" "用户手册" "寒假特训营日志" "寒假特训营日志.pdf" "寒假特训营运营与执行日志。"
  upload_fixture "$token" "生涯规划" "五好生涯规划介绍" "生涯规划/五好教育.pdf" "生涯规划产品介绍材料。"
  upload_fixture "$token" "生涯规划" "MBTI 职业性格测评解析" "生涯规划/MBTI职业性格测评解析.docx" "MBTI 职业性格测评解析资料。"
  upload_fixture "$token" "生涯规划" "职业生涯规划分享课件" "生涯规划/314 职业生涯规划分享1110.pptx" "职业生涯规划分享课件。"
  upload_fixture "$token" "生涯规划" "五好生涯报价体系" "生涯规划/五好生涯报价体系.xlsx" "五好生涯服务报价体系。"
  upload_fixture "$token" "资质证明" "网信算法备案公示内容" "老马识学/公示内容_网信算备330110507206401240101号.pdf" "网信算法备案公示材料。"
  upload_fixture "$token" "资质证明" "通义千问大模型 API 接口合作证明" "老马识学/通义千问大模型API接口合作证明.pdf" "通义千问大模型 API 接口合作证明。"
  upload_fixture "$token" "素材图片" "寒假特训营海报" "五好爱学（寒假特训）1.jpeg" "寒假特训营宣传素材。"

  psql_exec -c "SELECT p.code, count(d.id) AS document_count FROM projects p LEFT JOIN documents d ON d.project_id = p.id AND d.is_deleted = false WHERE p.id = '$PROJECT_ID'::uuid GROUP BY p.code;"
}

main "$@"
