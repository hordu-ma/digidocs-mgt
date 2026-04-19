FROM golang:1.26 AS build

WORKDIR /app

COPY . .
RUN go build -mod=vendor -o /out/digidocs-go-api ./cmd/api

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/digidocs-go-api /usr/local/bin/digidocs-go-api
COPY --from=build /app/migrations ./migrations

EXPOSE 8080

CMD ["digidocs-go-api"]
