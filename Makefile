.PHONY: build run test tidy lint snapshot migrate

build:
	go build -ldflags="-X main.version=$(shell git rev-parse --short HEAD)" -o bin/server ./cmd/server

run:
	go run ./cmd/server

# Snapshot: create a timestamped Neon branch (safe point to roll back to)
snapshot:
	@./scripts/snapshot.sh pre-migrate

# Migrate: ALWAYS snapshot first, then apply migrations
migrate: snapshot
	@echo "🚀 Running migrations..."
	go run ./cmd/migrate/main.go
	@echo "✅ Done. Your pre-migrate snapshot is saved in Neon branches."

tidy:
	go mod tidy

test:
	go test ./... -v -race -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

docker-build:
	docker build -t publishing-backend:local .

env-copy:
	cp .env.example .env