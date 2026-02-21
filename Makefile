SHELL := /bin/bash

# Test database / redis for integration tests (override via env)
TEST_DB_URL ?= postgres://postgres:postgres@localhost:5432/task_db?sslmode=disable
TEST_REDIS_ADDR ?= localhost:6379

.PHONY: help up_build api worker app \
        test test-verbose test-coverage \
        test-unit test-integration test-integration-ci \
        test-auth test-controller \
        clean deps tidy lint fmt vet

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

up_build: ## Build and start all services
	docker-compose down
	docker-compose up -d --build

down:
	docker-compose down

api: ## Rebuild and restart API only
	docker-compose down api
	docker-compose up -d api --build

worker: ## Rebuild and restart Worker only
	docker-compose down worker
	docker-compose up -d worker --build

app: ## Rebuild and restart API + Worker
	docker-compose down worker api
	docker-compose up -d worker api --build

# ── Unit tests (no external dependencies) ───────────────────────────────────
test-unit: ## Run all unit tests (no DB/Redis required)
	go test -v -race -count=1 ./tests/unit/...

# ── Integration tests (require Postgres + Redis) ─────────────────────────────
test-integration: ## Run integration tests against local Docker services
	@echo "Starting Postgres + Redis …"
	docker compose up -d postgres redis
	@sleep 3
	TEST_DATABASE_URL=$(TEST_DB_URL) TEST_REDIS_ADDR=$(TEST_REDIS_ADDR) \
	  go test -v -race -count=1 -timeout 120s ./tests/integration/...

test-integration-ci: ## Run integration tests (CI — expects DB already up)
	TEST_DATABASE_URL=$(TEST_DB_URL) TEST_REDIS_ADDR=$(TEST_REDIS_ADDR) \
	  go test -v -race -count=1 -timeout 120s \
	  -coverprofile=coverage-integration.txt -covermode=atomic \
	  ./tests/integration/...

# ── All tests ────────────────────────────────────────────────────────────────
test: ## Run unit tests + integration tests
	$(MAKE) test-unit
	$(MAKE) test-integration

test-verbose: ## Run all unit tests with full verbose output
	go test -v -race -count=1 ./tests/unit/...

test-coverage: ## Unit tests with HTML coverage report
	go test -v -race -count=1 \
	  -coverprofile=coverage.txt -covermode=atomic \
	  ./tests/unit/...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

# ── Focused test targets ─────────────────────────────────────────────────────
test-auth: ## Run auth unit tests only
	go test -v -race -count=1 ./tests/unit/auth/...

test-controller: ## Run controller unit tests only
	go test -v -race -count=1 ./tests/unit/task/... ./tests/unit/user/...

test-middleware: ## Run middleware unit tests only
	go test -v -race -count=1 ./tests/unit/middleware/...

clean: ## Clean up test artifacts
	rm -f coverage.txt coverage.html coverage-integration.txt
	go clean -testcache

deps: ## Download dependencies
	go mod download
	go mod verify

tidy: ## Tidy and verify dependencies
	go mod tidy
	go mod verify

lint: ## Run linter (requires golangci-lint)
	golangci-lint run --timeout=5m

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...