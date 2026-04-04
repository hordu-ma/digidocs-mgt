FROM golang:1.26 AS build

WORKDIR /app

COPY . .
RUN go build -mod=vendor -o /out/digidocs-go-api ./cmd/api

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=build /out/digidocs-go-api /usr/local/bin/digidocs-go-api

EXPOSE 8080

CMD ["digidocs-go-api"]
