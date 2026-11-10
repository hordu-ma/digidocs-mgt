#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

container="digidocs-postgres-it-$$"

cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo '== postgres integration container =='
docker run --rm -d \
  --name "$container" \
  -e POSTGRES_DB=digidocs_mgt \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 127.0.0.1::5432 \
  pgvector/pgvector:pg17-trixie@sha256:01215e09bd4bd582d20230e025b6c971deeeffee49ceef1aae167e839cb62a7e >/dev/null

for _ in $(seq 1 60); do
  if docker exec "$container" pg_isready -U postgres -d digidocs_mgt >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! docker exec "$container" pg_isready -U postgres -d digidocs_mgt >/dev/null 2>&1; then
  echo '[MISS] PostgreSQL integration container is not ready'
  exit 1
fi

host_port=$(docker inspect -f '{{(index (index .NetworkSettings.Ports "5432/tcp") 0).HostPort}}' "$container")
dsn="postgres://postgres:postgres@127.0.0.1:${host_port}/digidocs_mgt?sslmode=disable"

echo '== go postgres integration tests =='
cd backend-go
DIGIDOCS_POSTGRES_TEST_DSN="$dsn" go test ./internal/repository/postgres -run TestPostgresIntegration -count=1 -v
