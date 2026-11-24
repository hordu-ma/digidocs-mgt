#!/usr/bin/env bash
#
# 部署后检查：沿用户真实访问路径（前端 nginx → 后端）验证，而不是只看各容器
# 健康状态或直连后端端口。
#
# 背景（2026-11-10）：前端 nginx 曾在启动时把后端地址解析一次并缓存，后端
# 容器重建换 IP 后所有 API 经前端都返回 502；而容器健康检查和直连 18081 的
# 检查全部正常，问题因此没有被发现。
#
# 用法：
#   scripts/codex/post-deploy-check.sh
#   PUBLIC_URL=http://8.152.204.76 scripts/codex/post-deploy-check.sh   # 同时检查公网入口

set -uo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

FRONTEND_URL="${FRONTEND_URL:-http://127.0.0.1:18080}"
PUBLIC_URL="${PUBLIC_URL:-}"
status=0

check() {
  local label="$1" url="$2" expect="$3"
  local code
  code=$(curl -s -o /dev/null -m 10 -w '%{http_code}' "$url")
  if [[ "$code" == "$expect" ]]; then
    echo "[OK] $label -> $code"
  else
    echo "[FAIL] $label -> $code (expected $expect)"
    status=1
  fi
}

echo '== containers =='
for c in digidocs-postgres digidocs-backend-go digidocs-backend-py-worker digidocs-frontend; do
  h=$(docker inspect -f '{{.State.Health.Status}}' "$c" 2>/dev/null || echo missing)
  if [[ "$h" == "healthy" ]]; then echo "[OK] $c healthy"; else echo "[FAIL] $c $h"; status=1; fi
done

echo '== release manifest =='
for c in digidocs-postgres digidocs-backend-go digidocs-backend-py-worker digidocs-frontend; do
  docker inspect -f '{{.Name}} {{.Image}}' "$c" || status=1
done
expected_migrations=$(cd "$ROOT_DIR/backend-go/migrations" && printf '%s\n' *.sql | sed 's/\.sql$//' | sort)
built_migrations=$(docker exec digidocs-backend-go sh -c 'cd /app/migrations && printf "%s\n" *.sql' | sed 's/\.sql$//' | sort)
db_migrations=$(docker exec digidocs-postgres sh -c   'psql -X -At -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT name FROM schema_migrations ORDER BY version"')
if [[ -n "$expected_migrations" && "$expected_migrations" == "$built_migrations" && "$expected_migrations" == "$db_migrations" ]]; then
  echo '[OK] release, running image and database migrations match'
else
  echo '[FAIL] release, running image or database migrations differ'
  status=1
fi

echo '== through frontend nginx =='
check "frontend shell" "$FRONTEND_URL/" 200
check "database readiness via frontend" "$FRONTEND_URL/api/v1/system/readiness" 200
check "internal Worker routes blocked at frontend" "$FRONTEND_URL/api/v1/internal/poll-tasks" 404
check "API via frontend (system/info)" "$FRONTEND_URL/api/v1/system/info" 200
check "protected API via frontend rejects anonymous" "$FRONTEND_URL/api/v1/auth/me" 401

if [[ -n "$PUBLIC_URL" ]]; then
  echo '== public entry =='
  check "public shell" "$PUBLIC_URL/" 200
  check "public database readiness" "$PUBLIC_URL/api/v1/system/readiness" 200
  check "public API (system/info)" "$PUBLIC_URL/api/v1/system/info" 200
fi

exit "$status"
