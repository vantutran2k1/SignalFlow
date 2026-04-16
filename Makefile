.PHONY: build run test lint dev docker-build

build:
	go build -o bin/signalflow ./cmd/signalflow

run:
	go run ./cmd/signalflow

test:
	go test ./... -race

lint:
	golangci-lint run

dev:
	docker compose up --build

docker-build:
	docker build -t signalflow .
