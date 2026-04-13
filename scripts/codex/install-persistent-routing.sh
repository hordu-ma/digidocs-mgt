#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

BIN_SOURCE="$ROOT_DIR/ops/systemd/digidocs-route-fix.sh"
DROPIN_SOURCE="$ROOT_DIR/ops/systemd/tailscaled-digidocs-routing.conf"

BIN_TARGET="/usr/local/bin/digidocs-route-fix.sh"
DROPIN_DIR="/etc/systemd/system/tailscaled.service.d"
DROPIN_TARGET="$DROPIN_DIR/digidocs-routing.conf"

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

if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemctl is required" >&2
  exit 1
fi

echo "installing digidocs persistent routing assets..."
run_as_root install -d -m 0755 /usr/local/bin
run_as_root install -m 0755 "$BIN_SOURCE" "$BIN_TARGET"
run_as_root install -d -m 0755 "$DROPIN_DIR"
run_as_root install -m 0644 "$DROPIN_SOURCE" "$DROPIN_TARGET"
run_as_root systemctl daemon-reload

echo "applying route fix immediately..."
run_as_root "$BIN_TARGET"

echo "verifying tailscaled drop-in..."
systemctl show tailscaled --property=DropInPaths --no-pager
ip route show table 52 | grep -E '^(throw 172\.17\.0\.0/16|throw 172\.18\.0\.0/16|throw 192\.168\.1\.0/24)'

echo "installed:"
echo "  script:  $BIN_TARGET"
echo "  drop-in: $DROPIN_TARGET"
