SHELL := /bin/bash

.PHONY: help up_build api worker app test test-verbose test-coverage test-unit test-integration clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

up_build: ## Build and start all services
	docker-compose down
	docker-compose up -d --build

api: ## Rebuild and restart API only
	docker-compose down api
	docker-compose up -d api --build

worker: ## Rebuild and restart Worker only
	docker-compose down worker
	docker-compose up -d worker --build

app: ## Rebuild and restart API + Worker
	docker-compose down worker api
	docker-compose up -d worker api --build

test: ## Run all tests
	go test ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

test-coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-unit: ## Run unit tests only
	go test -v -short ./...

test-integration: ## Run integration tests
	go test -v -run Integration ./...

test-rate-limiter: ## Run rate limiter tests
	go test -v ./internal/middleware -run RateLimiter

test-auth: ## Run auth tests
	go test -v ./internal/auth

test-controller: ## Run controller tests
	go test -v ./internal/task -run Controller

clean: ## Clean up test artifacts
	rm -f coverage.txt coverage.html
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