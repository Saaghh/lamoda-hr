FROM golang:1.22 as builder

COPY ../.. /src
WORKDIR /src

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/apiserver cmd/apiserver/main.go

FROM debian:stable-slim

COPY --from=builder /src/bin/apiserver /app/bin/apiserver

WORKDIR /app

ENV LOG_LEVEL="info"
ENV PG_HOST="lamoda_postgres"

ENTRYPOINT ["./bin/apiserver"]