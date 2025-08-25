VERSION = $(shell godzil show-version)
CURRENT_REVISION = $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS = "-s -w -X github.com/Songmu/gotesplit.revision=$(CURRENT_REVISION)"
u := $(if $(update),-u)

.PHONY: deps
deps:
	go get ${u} -d
	go mod tidy

.PHONY: devel-deps
devel-deps:
	go install github.com/Songmu/godzil/cmd/godzil@latest
	go install github.com/tcnksm/ghr@latest

.PHONY: test
test:
	go test

.PHONY: build
build:
	go build -ldflags=$(BUILD_LDFLAGS) ./cmd/gotesplit

.PHONY: install
install:
	go install -ldflags=$(BUILD_LDFLAGS) ./cmd/gotesplit

.PHONY: release
release: devel-deps
	godzil release

CREDITS: deps devel-deps
	godzil credits -w

DIST_DIR = dist
.PHONY: go.sum crossbuild
crossbuild: CREDITS
	rm -rf $(DIST_DIR)
	env CGO_ENABLED=0 godzil crossbuild -pv=v$(VERSION) -build-ldflags=$(BUILD_LDFLAGS) \
      -os=linux,darwin,windows -d=$(DIST_DIR) ./cmd/*
	cd $(DIST_DIR) && shasum -a 256 $$(find * -type f -maxdepth 0) > SHA256SUMS

.PHONY: upload
upload:
	ghr v$(VERSION) $(DIST_DIR)
