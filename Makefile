default: help

SHELL:= /usr/bin/bash -e
GOLANGCI_LINT_VERSION:=1.52.0
# Download from https://github.com/golangci/golangci-lint/releases/tag/v1.52.2

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: linter
linter: ## Install golangci-lint executable via curl.
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v${GOLANGCI_LINT_VERSION}

.PHONY: lint
lint: ## Updates modules and execute linters.
	## use `make install-linter` to install linters if missing
	## or download the executable file from https://github.com/golangci/golangci-lint/releases/
	go mod tidy
	golangci-lint -v --timeout=5m run

.PHONY: clean
clean: ## Remove temporary files and cached tests results.
	go clean -testcache

.PHONY: test
test: clean ## Remove cache and Run unit tests only.
	go test -v ./... -count=1

.PHONY: coverc
coverc: clean ## Testing coverage and view stats in console.
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

.PHONY: coverh
coverh: clean ## Testing coverage and view stats in browser.
	go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

.PHONY: cover
cover: coverc coverh ## Coverage stats on console and browser.

.PHONY: format
format: ## Format all codebase.
	gofumpt -l -w .