FROM golang:1.26 AS build

WORKDIR /app

COPY . .
RUN go build -mod=vendor -o /out/digidocs-go-api ./cmd/api

FROM debian:bookworm-slim

WORKDIR /app

# curl is here for the container HEALTHCHECK (see docker-compose.yml); the base
# image ships neither curl nor wget, so without it the healthcheck can never run.
RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/digidocs-go-api /usr/local/bin/digidocs-go-api
COPY --from=build /app/migrations ./migrations

EXPOSE 8080

CMD ["digidocs-go-api"]
