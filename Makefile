.PHONY: build run test tidy lint

build:
	go build -ldflags="-X main.version=$(shell git rev-parse --short HEAD)" -o bin/server ./cmd/server

run:
	go run ./cmd/server

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