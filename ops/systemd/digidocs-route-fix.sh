#!/usr/bin/env bash
set -euo pipefail

IP_BIN="${IP_BIN:-$(command -v ip)}"
ATTEMPTS="${DIGIDOCS_ROUTE_FIX_ATTEMPTS:-8}"
SLEEP_SECONDS="${DIGIDOCS_ROUTE_FIX_INTERVAL:-1}"
ROUTE_FIX_CIDRS=(172.17.0.0/16 172.18.0.0/16 172.29.0.0/24 192.168.1.0/24)

if [[ -n "${DIGIDOCS_ROUTE_FIX_EXTRA_CIDRS:-}" ]]; then
  # Allow ops to append additional destinations that must bypass tailscale policy routing.
  read -r -a EXTRA_CIDRS <<<"$DIGIDOCS_ROUTE_FIX_EXTRA_CIDRS"
  ROUTE_FIX_CIDRS+=("${EXTRA_CIDRS[@]}")
fi

if [[ -z "$IP_BIN" ]]; then
  echo "ip command not found" >&2
  exit 1
fi

for ((attempt = 1; attempt <= ATTEMPTS; attempt++)); do
  for cidr in "${ROUTE_FIX_CIDRS[@]}"; do
    "$IP_BIN" route replace throw "$cidr" table 52
  done

  if (( attempt < ATTEMPTS )); then
    sleep "$SLEEP_SECONDS"
  fi
done

echo "digidocs route fix applied:"
for cidr in "${ROUTE_FIX_CIDRS[@]}"; do
  "$IP_BIN" route show table 52 | grep -F "throw $cidr" || true
done
