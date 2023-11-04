default: help

SHELL:=/usr/bin/env bash
GOLANGCI_LINT_VERSION:=1.52.0
# Download from https://github.com/golangci/golangci-lint/releases/tag/v1.52.2

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: linter
linter: ## Install golangci-lint executable via curl.
	## manual download from https://github.com/golangci/golangci-lint/releases/
	which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v${GOLANGCI_LINT_VERSION}

.PHONY: lint
lint: linter ## Updates modules and execute linters.	
	go mod tidy
	golangci-lint -v --timeout=5m run	

.PHONY: clean
clean: ## Remove temporary files and cached tests results.
	go clean -testcache

.PHONY: test
test: clean ## Remove cache and Run unit tests only.
	go test -v -timeout 2m -count=1 ./...

.PHONY: test-cover
test-cover: clean ## Remove tests cache and run tests with coverage.
	go test -race -v -timeout 2m -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

.PHONY: coverc
coverc: clean ## Testing coverage and view stats in console.
	go test -timeout 2m -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

.PHONY: coverh
coverh: clean ## Testing coverage and view stats in browser.
	go test -timeout 2m -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

.PHONY: cover
cover: coverc coverh ## Coverage stats on console and browser.

.PHONY: format
format: ## Format all codebase.
	gofumpt -l -w .