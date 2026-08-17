#!/usr/bin/env bash
#
# 安装 DigiDocs 定时备份（systemd 用户级 timer）。
#
# 为什么是用户级：备份只需要访问 docker 与用户家目录，不需要 root。本机
# sdu-procurement-monitor 用的也是同一套模式。用户级 timer 要在未登录时也能跑，
# 依赖 linger——脚本会检查并在缺失时给出开启命令。
#
# 用法：
#   scripts/codex/install-backup-timer.sh              # 安装并启用（默认每天 03:30）
#   BACKUP_ON_CALENDAR='*-*-* 02:00:00' scripts/codex/install-backup-timer.sh
#   scripts/codex/install-backup-timer.sh --uninstall

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
UNIT_DIR="$HOME/.config/systemd/user"
SERVICE="digidocs-backup.service"
TIMER="digidocs-backup.timer"
ON_CALENDAR="${BACKUP_ON_CALENDAR:-*-*-* 03:30:00}"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-14}"
BACKUP_DIR="${DIGIDOCS_BACKUP_DIR:-$HOME/ops/digidocs-backups}"

if [[ "${1:-}" == "--uninstall" ]]; then
  systemctl --user disable --now "$TIMER" 2>/dev/null || true
  rm -f "$UNIT_DIR/$SERVICE" "$UNIT_DIR/$TIMER"
  systemctl --user daemon-reload
  echo "[OK] 已卸载 $TIMER / $SERVICE（备份文件保留在 $BACKUP_DIR）"
  exit 0
fi

mkdir -p "$UNIT_DIR" "$BACKUP_DIR"

cat > "$UNIT_DIR/$SERVICE" <<EOF
[Unit]
Description=DigiDocs database and code-repo backup
Documentation=file://$ROOT_DIR/scripts/codex/backup.sh

[Service]
Type=oneshot
WorkingDirectory=$ROOT_DIR
Environment=BACKUP_KEEP_DAYS=$KEEP_DAYS
Environment=DIGIDOCS_BACKUP_DIR=$BACKUP_DIR
ExecStart=$ROOT_DIR/scripts/codex/backup.sh
EOF

cat > "$UNIT_DIR/$TIMER" <<EOF
[Unit]
Description=Run DigiDocs backup on a schedule

[Timer]
OnCalendar=$ON_CALENDAR
# 关机/休眠错过的那次开机后补跑，否则停机一晚就等于少一天备份。
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now "$TIMER"

echo "[OK] 已安装并启用 $TIMER"
echo "     计划：$ON_CALENDAR   保留：${KEEP_DAYS} 天   目录：$BACKUP_DIR"

if [[ "$(loginctl show-user "$USER" --property=Linger --value 2>/dev/null)" != "yes" ]]; then
  echo
  echo "[WARN] 当前用户未开启 linger，未登录时 timer 不会触发。请执行："
  echo "       sudo loginctl enable-linger $USER"
fi

echo
echo "查看状态： systemctl --user list-timers digidocs-backup.timer"
echo "立即试跑： systemctl --user start $SERVICE && journalctl --user -u $SERVICE -n 30 --no-pager"
