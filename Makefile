.PHONY: build run test test-short test-db-up test-db-down lint dev docker-build

TEST_DATABASE_URL ?= postgres://signalflow:secret@localhost:55432/signalflow_test?sslmode=disable
export TEST_DATABASE_URL

build:
	go build -o bin/signalflow ./cmd/signalflow

run:
	go run ./cmd/signalflow

test:
	go test ./... -race -p 1

# Unit-only run: integration tests skip themselves when TEST_DATABASE_URL is unset.
test-short:
	TEST_DATABASE_URL= go test ./... -race

# Spin up a disposable Postgres for integration tests on port 55432 (kept off
# 5432 so it doesn't collide with `make dev`).
test-db-up:
	docker run -d --rm --name signalflow-test-pg \
		-p 55432:5432 \
		-e POSTGRES_USER=signalflow -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=signalflow_test \
		postgres:17-alpine
	@until docker exec signalflow-test-pg pg_isready -U signalflow >/dev/null 2>&1; do sleep 1; done
	@echo "test postgres ready on localhost:55432"

test-db-down:
	-docker stop signalflow-test-pg

lint:
	golangci-lint run

dev:
	docker compose up --build

docker-build:
	docker build -t signalflow .
