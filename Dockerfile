FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o signalflow ./cmd/signalflow

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
RUN adduser -D -g '' appuser
COPY --from=builder /app/signalflow /usr/local/bin/signalflow
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["signalflow"]
