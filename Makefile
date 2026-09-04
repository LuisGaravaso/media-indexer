.PHONY: run build test vet lint coverage clean db-up db-down db-reset proxy proxy-sandbox proxy-prod help

# Default target
.DEFAULT_GOAL := help

## run: Run the application
run:
	go run ./cmd/server/main.go

## build: Build binary
build:
	go build -o bin/server ./cmd/server/main.go

## test: Run unit tests with race detection
test:
	go test -v -race ./...

## test-integration: Run integration tests with live services
test-integration:
	go test -v -race -tags=integration ./tests/integration/...

## coverage: Run tests with coverage and display function breakdown
coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

## vet: Run go vet
vet:
	go vet ./...

GOPATH ?= $(shell go env GOPATH)
GOLANGCI_LINT ?= $(GOPATH)/bin/golangci-lint

## lint: Run golangci-lint (installs if missing)
lint:
	@which golangci-lint > /dev/null 2>&1 || test -f $(GOLANGCI_LINT) || { \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2; \
	}
	@if which golangci-lint > /dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		$(GOLANGCI_LINT) run ./...; \
	fi

## docker-build: Build docker image
docker-build:
	docker build -t media-indexer:latest .

## docker-run: Run container locally
docker-run:
	docker run --rm -p 8080:8080 -e PORT=8080 media-indexer:latest

## clean: Remove build artifacts
clean:
	rm -rf bin/ coverage.out

## db-up: Start local PostgreSQL (with pgvector) database container
db-up:
	docker compose up -d db

## db-down: Stop local PostgreSQL container
db-down:
	docker compose down

## db-reset: Reset local database and re-apply all migrations from scratch
db-reset:
	docker compose down -v
	docker compose up -d db

## db-logs: View logs from local database container
db-logs:
	docker compose logs -f db

## hooks-install: Configure git to use project pre-commit hooks
hooks-install:
	git config core.hooksPath .githooks
	chmod +x .githooks/*
	@echo "Git hooks configured successfully with .githooks/"

## proxy-sandbox: Run local authenticated proxy to Cloud Run sandbox (default port 8080)
proxy-sandbox:
	@echo "Starting local proxy to reeler-sandbox media-indexer on http://localhost:8080..."
	gcloud run services proxy media-indexer --project=reeler-sandbox --region=us-central1 --port=8080

## proxy-prod: Run local authenticated proxy to Cloud Run prod (default port 8080)
proxy-prod:
	@echo "Starting local proxy to reeler-prod media-indexer on http://localhost:8080..."
	gcloud run services proxy media-indexer --project=reeler-prod --region=us-central1 --port=8080

## proxy: Alias for proxy-sandbox
proxy: proxy-sandbox

## help: Display available targets
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'
