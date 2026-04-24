FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/signalflow ./cmd/signalflow
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.21
RUN apk add --no-cache ca-certificates \
 && adduser -D -u 10001 -g '' appuser
COPY --from=builder /out/signalflow /usr/local/bin/signalflow
COPY --from=builder /out/migrate /usr/local/bin/migrate
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["signalflow"]
