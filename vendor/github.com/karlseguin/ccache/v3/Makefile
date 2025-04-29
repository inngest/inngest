.DEFAULT_GOAL := help

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: bench
bench: ## Run benchmarks
	go test ./... -bench . -benchtime 5s -timeout 0 -run=XXX -benchmem

.PHONY: l
l: ## Lint Go source files
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && golangci-lint run

.PHONY: t
t: ## Run unit tests
	go test -race -count=1 ./...

.PHONY: f
f: ## Format code
	go fmt ./...

.PHONY: c
c: ## Measure code coverage
	go test -race -covermode=atomic ./... -coverprofile=cover.out