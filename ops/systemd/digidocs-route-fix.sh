#!/usr/bin/env bash
set -euo pipefail

IP_BIN="${IP_BIN:-$(command -v ip)}"
ATTEMPTS="${DIGIDOCS_ROUTE_FIX_ATTEMPTS:-8}"
SLEEP_SECONDS="${DIGIDOCS_ROUTE_FIX_INTERVAL:-1}"

if [[ -z "$IP_BIN" ]]; then
  echo "ip command not found" >&2
  exit 1
fi

for ((attempt = 1; attempt <= ATTEMPTS; attempt++)); do
  for cidr in 172.17.0.0/16 172.18.0.0/16 192.168.1.0/24; do
    "$IP_BIN" route replace throw "$cidr" table 52
  done

  if (( attempt < ATTEMPTS )); then
    sleep "$SLEEP_SECONDS"
  fi
done

echo "digidocs route fix applied:"
"$IP_BIN" route show table 52 | grep -E '^(throw 172\.17\.0\.0/16|throw 172\.18\.0\.0/16|throw 192\.168\.1\.0/24)' || true
