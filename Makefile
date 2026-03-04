# Makefile
.PHONY: build lint test test-race migrate-up docker-up check

build:
	go build ./cmd/api/...

lint:
	golangci-lint run ./...

test:
	go test ./...

test-race:
	go test -race ./...

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

docker-up:
	docker compose up --build -d

check: lint test-race build
