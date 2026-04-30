#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

BIN_SOURCE="$ROOT_DIR/ops/systemd/digidocs-route-fix.sh"
DROPIN_SOURCE="$ROOT_DIR/ops/systemd/tailscaled-digidocs-routing.conf"
SERVICE_SOURCE="$ROOT_DIR/ops/systemd/digidocs-route-fix.service"
TIMER_SOURCE="$ROOT_DIR/ops/systemd/digidocs-route-fix.timer"

BIN_TARGET="/usr/local/bin/digidocs-route-fix.sh"
DROPIN_DIR="/etc/systemd/system/tailscaled.service.d"
DROPIN_TARGET="$DROPIN_DIR/digidocs-routing.conf"
SYSTEMD_DIR="/etc/systemd/system"
SERVICE_TARGET="$SYSTEMD_DIR/digidocs-route-fix.service"
TIMER_TARGET="$SYSTEMD_DIR/digidocs-route-fix.timer"

run_as_root() {
  if [[ "${EUID}" -eq 0 ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

if [[ ! -f "$BIN_SOURCE" ]]; then
  echo "missing route fix script: $BIN_SOURCE" >&2
  exit 1
fi

if [[ ! -f "$DROPIN_SOURCE" ]]; then
  echo "missing tailscaled drop-in template: $DROPIN_SOURCE" >&2
  exit 1
fi

if [[ ! -f "$SERVICE_SOURCE" ]]; then
  echo "missing route fix service unit: $SERVICE_SOURCE" >&2
  exit 1
fi

if [[ ! -f "$TIMER_SOURCE" ]]; then
  echo "missing route fix timer unit: $TIMER_SOURCE" >&2
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemctl is required" >&2
  exit 1
fi

echo "installing digidocs persistent routing assets..."
run_as_root install -d -m 0755 /usr/local/bin
run_as_root install -m 0755 "$BIN_SOURCE" "$BIN_TARGET"
run_as_root install -d -m 0755 "$DROPIN_DIR"
run_as_root install -m 0644 "$DROPIN_SOURCE" "$DROPIN_TARGET"
run_as_root install -m 0644 "$SERVICE_SOURCE" "$SERVICE_TARGET"
run_as_root install -m 0644 "$TIMER_SOURCE" "$TIMER_TARGET"
run_as_root systemctl daemon-reload
run_as_root systemctl enable --now digidocs-route-fix.timer

echo "applying route fix immediately..."
run_as_root env DIGIDOCS_ROUTE_FIX_EXTRA_CIDRS=8.152.204.76 "$BIN_TARGET"

echo "verifying tailscaled drop-in..."
systemctl show tailscaled --property=DropInPaths --no-pager
systemctl list-timers digidocs-route-fix.timer --no-pager
for cidr in 8.152.204.76 172.17.0.0/16 172.18.0.0/16 172.29.0.0/24 192.168.1.0/24; do
  ip route show table 52 | grep -F "throw $cidr"
done

echo "installed:"
echo "  script:  $BIN_TARGET"
echo "  drop-in: $DROPIN_TARGET"
echo "  service: $SERVICE_TARGET"
echo "  timer:   $TIMER_TARGET"
