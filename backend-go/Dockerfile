FROM golang:1.26 AS build

WORKDIR /app

COPY . .
RUN go build -mod=vendor -o /out/digidocs-go-api ./cmd/api
# backup-sync is not part of the running server; it is invoked ad hoc via
# `docker exec` from scripts/codex/backup.sh to push a backup file to the
# Synology NAS this container is already configured to trust, reusing the
# same internal/storage/synology client instead of a bespoke curl-based
# reimplementation of DSM auth in the backup script.
RUN go build -mod=vendor -o /out/digidocs-backup-sync ./cmd/backup-sync \
    && go build -mod=vendor -o /out/digidocs-storage-audit ./cmd/storage-audit

FROM debian:bookworm-slim

WORKDIR /app

# curl is here for the container HEALTHCHECK (see docker-compose.yml); the base
# image ships neither curl nor wget, so without it the healthcheck can never run.
RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/digidocs-go-api /usr/local/bin/digidocs-go-api
COPY --from=build /out/digidocs-backup-sync /usr/local/bin/digidocs-backup-sync
COPY --from=build /out/digidocs-storage-audit /usr/local/bin/digidocs-storage-audit
COPY --from=build /app/migrations ./migrations

EXPOSE 8080

CMD ["digidocs-go-api"]
