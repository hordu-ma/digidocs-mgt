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

echo '== through frontend nginx =='
check "frontend shell" "$FRONTEND_URL/" 200
check "API via frontend (system/info)" "$FRONTEND_URL/api/v1/system/info" 200
check "protected API via frontend rejects anonymous" "$FRONTEND_URL/api/v1/auth/me" 401

if [[ -n "$PUBLIC_URL" ]]; then
  echo '== public entry =='
  check "public shell" "$PUBLIC_URL/" 200
  check "public API (system/info)" "$PUBLIC_URL/api/v1/system/info" 200
fi

exit "$status"
