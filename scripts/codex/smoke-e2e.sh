#!/usr/bin/env bash
#
# 端到端烟测：拉起隔离栈 → 灌种子数据 → 跑接口烟测 → 销毁。
#
# 此前本脚本直接 `docker compose --profile app up -d` 演示栈，并把 seed.sql 灌进
# 演示库，等于每跑一次就重置演示账号并写入一批测试数据。现在改为在
# docker-compose.smoke.yml 定义的隔离栈上执行，演示环境完全不受影响。
#
# 用法：
#   scripts/codex/smoke-e2e.sh              # 起栈 → 烟测 → 销毁
#   KEEP_SMOKE_STACK=1 scripts/codex/smoke-e2e.sh   # 跑完保留栈便于排查
#   RUN_ASSISTANT_SMOKE=1 scripts/codex/smoke-e2e.sh  # 连带 worker / OpenClaw 链路

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

backend_url="${SMOKE_BACKEND_URL:-http://127.0.0.1:18083}"
frontend_url="${SMOKE_FRONTEND_URL:-http://127.0.0.1:18082}"
assistant_smoke="${RUN_ASSISTANT_SMOKE:-0}"
keep_stack="${KEEP_SMOKE_STACK:-0}"

is_truthy() {
  case "${1:-}" in
    1|true|TRUE|True|yes|YES|on|ON) return 0 ;;
    *) return 1 ;;
  esac
}

cleanup() {
  local rc=$?
  if is_truthy "$keep_stack"; then
    echo
    echo "[KEEP] KEEP_SMOKE_STACK=1，保留烟测栈；用完执行 scripts/codex/smoke-stack.sh down"
  else
    echo
    ./scripts/codex/smoke-stack.sh down || true
  fi
  exit "$rc"
}
trap cleanup EXIT

if is_truthy "$assistant_smoke"; then
  ./scripts/codex/smoke-stack.sh up --with-worker
else
  ./scripts/codex/smoke-stack.sh up
fi

echo
echo '== 前端健康检查 =='
if curl -fsS --max-time 5 "$frontend_url/" >/dev/null; then
  echo "[OK] $frontend_url/"
else
  echo "[MISS] $frontend_url/ 不可达"
  exit 1
fi

echo
echo '== 接口烟测 =='
STRICT_SMOKE="${STRICT_SMOKE:-1}" \
RUN_ASSISTANT_SMOKE="$assistant_smoke" \
BACKEND_BASE_URL="$backend_url" \
  ./scripts/codex/smoke-local.sh
