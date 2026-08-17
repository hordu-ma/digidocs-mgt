#!/usr/bin/env bash
#
# 烟测隔离栈的生命周期管理。
#
# 背景：烟测会往目标库写入一次性数据，而它此前默认打的就是演示后端
# 127.0.0.1:18081，导致每跑一次客户演示环境就多一批测试垃圾。本脚本用
# docker-compose.smoke.yml 起一套完全独立的 PostgreSQL + 后端 + 前端
# （独立 project / 容器名 / 端口 / 数据库 / 网络，存储走 memory 不碰群晖），
# 让烟测有自己的地盘。
#
# 用法：
#   scripts/codex/smoke-stack.sh up [--with-worker]   # 拉起并灌入种子数据
#   scripts/codex/smoke-stack.sh down                 # 销毁（含数据卷）
#   scripts/codex/smoke-stack.sh status
#
# 拉起后烟测入口：
#   backend  http://127.0.0.1:18083
#   frontend http://127.0.0.1:18082

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

PROJECT="${SMOKE_COMPOSE_PROJECT:-digidocs-smoke}"
BACKEND_URL="${SMOKE_BACKEND_URL:-http://127.0.0.1:18083}"
FRONTEND_URL="${SMOKE_FRONTEND_URL:-http://127.0.0.1:18082}"
PG_CONTAINER="digidocs-smoke-postgres"
SMOKE_DB="digidocs_smoke"

compose() {
  docker compose -p "$PROJECT" \
    -f docker-compose.yml -f docker-compose.smoke.yml \
    "$@"
}

wait_for_backend() {
  echo "== 等待后端就绪 =="
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 2 "$BACKEND_URL/healthz" >/dev/null 2>&1; then
      echo "[OK] $BACKEND_URL/healthz"
      return 0
    fi
    sleep 2
  done
  echo "[MISS] $BACKEND_URL/healthz 在等待后仍不可达" >&2
  return 1
}

# 确认这套栈确实是隔离栈，避免脚本被改坏之后把种子数据灌进演示库。
assert_isolated() {
  local env_value
  env_value=$(curl -fsS --max-time 3 "$BACKEND_URL/api/v1/system/info" 2>/dev/null \
    | sed -n 's/.*"env"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
  if [[ "$env_value" != "smoke" ]]; then
    echo "[ABORT] $BACKEND_URL 的 APP_ENV=${env_value:-unknown}，不是隔离烟测栈；已中止。" >&2
    exit 1
  fi
}

cmd_up() {
  local with_worker=0
  [[ "${1:-}" == "--with-worker" ]] && with_worker=1

  echo "== 拉起烟测栈（project=$PROJECT）=="
  if [[ "$with_worker" == "1" ]]; then
    echo "[NOTE] 已包含 worker。它要访问宿主机 OpenClaw（172.30.0.1:18789），"
    echo "[NOTE] 而本机 UFW 默认 DROP，只放行了演示栈网段。未放行时表现为 60s"
    echo "[NOTE] 静默超时而非连接被拒——排查入口 grep 'UFW BLOCK' /var/log/ufw.log。"
    compose --profile app up -d --build postgres backend-go backend-py-worker frontend
  else
    compose --profile app up -d --build postgres backend-go frontend
  fi

  wait_for_backend
  assert_isolated

  echo "== 灌入种子数据 =="
  docker exec -i "$PG_CONTAINER" psql -v ON_ERROR_STOP=1 \
    -U "${POSTGRES_USER:-postgres}" -d "$SMOKE_DB" \
    < backend-go/sql/seed.sql >/dev/null
  echo "[OK] backend-go/sql/seed.sql 已载入 $SMOKE_DB"

  echo
  echo "[OK] 烟测栈就绪"
  echo "     backend  $BACKEND_URL"
  echo "     frontend $FRONTEND_URL"
  echo "     跑烟测： make smoke        （默认即打这套栈）"
  echo "     用完销毁：scripts/codex/smoke-stack.sh down"
}

cmd_down() {
  echo "== 销毁烟测栈（含数据卷）=="
  compose --profile app down -v --remove-orphans
  echo "[OK] 已销毁"
}

cmd_status() {
  compose --profile app ps
  echo
  if curl -fsS --max-time 2 "$BACKEND_URL/api/v1/system/info" 2>/dev/null; then
    echo
  else
    echo "[INFO] $BACKEND_URL 不可达（栈未启动？）"
  fi
}

case "${1:-}" in
  up) shift; cmd_up "${1:-}" ;;
  down) cmd_down ;;
  status) cmd_status ;;
  *) echo "用法: $0 {up [--with-worker]|down|status}" >&2; exit 2 ;;
esac
