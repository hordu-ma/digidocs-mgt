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

# `ip route show` prints a trailing space on every line, so a whole-line match
# (grep -Fqx "throw $cidr") can never succeed. Compare parsed fields instead.
has_throw_route() {
  "$IP_BIN" route show table 52 |
    awk -v want="$1" '$1 == "throw" && $2 == want { found = 1 } END { exit !found }'
}

declare -A repaired=()
for ((attempt = 1; attempt <= ATTEMPTS; attempt++)); do
  for cidr in "${ROUTE_FIX_CIDRS[@]}"; do
    if ! has_throw_route "$cidr"; then
      "$IP_BIN" route replace throw "$cidr" table 52
      repaired["$cidr"]=1
    fi
  done

  # The retry passes exist to win the race when tailscaled rewrites table 52 a
  # moment after we repair it. Once the table is converged they are read-only.
  if (( attempt < ATTEMPTS )); then
    sleep "$SLEEP_SECONDS"
  fi
done

# Log only real repairs, so a log line means something actually happened.
if (( ${#repaired[@]} > 0 )); then
  printf 'digidocs route fix repaired: %s\n' "${!repaired[*]}"
fi
