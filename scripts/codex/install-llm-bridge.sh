#!/usr/bin/env bash
#
# 安装 DigiDocs 本地模型桥接（systemd 用户级服务，无需 root）。
#
# 把 Docker 应用网关地址 172.29.0.1 上的端口转发到只监听回环地址的本地模型服务：
#   172.29.0.1:11501 -> 127.0.0.1:11500  llama.cpp Qwen（对话）
#   172.29.0.1:11502 -> 127.0.0.1:11434  Ollama（向量嵌入，预留）
#
# 本机 UFW 默认拒绝来自容器网段的入站，还需要 root 执行一次放行（脚本只打印命令）：
#   sudo ufw allow from 172.29.0.0/24 to 172.29.0.1 port 11501:11502 proto tcp
#   sudo ufw allow from 172.30.0.0/24 to 172.29.0.1 port 11501:11502 proto tcp   # 烟测隔离栈
#
# 用法：
#   scripts/codex/install-llm-bridge.sh
#   scripts/codex/install-llm-bridge.sh --uninstall

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
UNIT_DIR="$HOME/.config/systemd/user"
SERVICE="digidocs-llm-bridge.service"
BIN="$HOME/.local/bin/digidocs-llm-bridge"

if [[ "${1:-}" == "--uninstall" ]]; then
  systemctl --user disable --now "$SERVICE" 2>/dev/null || true
  rm -f "$UNIT_DIR/$SERVICE" "$BIN"
  systemctl --user daemon-reload
  echo "[OK] 已卸载 $SERVICE"
  exit 0
fi

mkdir -p "$UNIT_DIR" "$(dirname "$BIN")"
install -m 0755 "$ROOT_DIR/ops/systemd/digidocs-llm-bridge.py" "$BIN"

cat > "$UNIT_DIR/$SERVICE" <<UNIT
[Unit]
Description=DigiDocs bridge from Docker network to local model servers
Documentation=file://$ROOT_DIR/ops/systemd/digidocs-llm-bridge.py
After=network-online.target llama-mtp.service ollama.service

[Service]
ExecStart=/usr/bin/python3 $BIN
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
UNIT

systemctl --user daemon-reload
systemctl --user enable --now "$SERVICE"
sleep 1
systemctl --user --no-pager --lines=3 status "$SERVICE" | sed -n '1,3p'
echo
echo "[NEXT] 如尚未放行，请以 root 执行："
echo "  sudo ufw allow from 172.29.0.0/24 to 172.29.0.1 port 11501:11502 proto tcp"
echo "  sudo ufw allow from 172.30.0.0/24 to 172.29.0.1 port 11501:11502 proto tcp"
