# Directory containing the Makefile.
PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

export GOBIN ?= $(PROJECT_ROOT)/bin
export PATH := $(GOBIN):$(PATH)

BENCH_FLAGS ?= -cpuprofile=cpu.pprof -memprofile=mem.pprof -benchmem

GO_FILES = $(shell \
	find . '(' -path '*/.*' -o -path './vendor' ')' -prune \
	-o -name '*.go' -print | cut -b3-)

.PHONY: all
all: build lint test

.PHONY: build
build:
	go build ./...

.PHONY: lint
lint: golangci-lint tidy-lint

.PHONY: test
test:
	go test -race ./...

.PHONY: cover
cover:
	go test -race -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

.PHONY: bench
BENCH ?= .
bench:
	go list ./... | xargs -n1 go test -bench=$(BENCH) -run="^$$" $(BENCH_FLAGS)

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: golangci-lint
golangci-lint:
	golangci-lint run

.PHONY: tidy-lint
tidy-lint:
	go mod tidy
	git diff --exit-code -- go.mod go.sum
