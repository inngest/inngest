EXTERNAL_TOOLS=\
	golang.org/x/tools/cmd/cover \
	golang.org/x/tools/cmd/vet \
	github.com/mattn/goveralls \
	github.com/stretchr/testify/assert


all: all-tests
	@echo "*** Done!"

get:
	@echo "*** Resolve dependencies..."
	@go get -v .

all-tests:
	@echo "*** Run tests..."
	@go test .

benchmark:
	@echo "*** Run benchmarks..."
	@go test -v -benchmem -bench=. -run=^a

test-race:
	@echo "*** Run tests with race condition..."
	@go test --race -v .

test-cover-builder:
	@go test -covermode=count -coverprofile=/tmp/art.out .

	@rm -f /tmp/art_coverage.out
	@echo "mode: count" > /tmp/art_coverage.out
	@cat /tmp/art.out | tail -n +2 >> /tmp/art_coverage.out
	@rm /tmp/art.out

test-cover: test-cover-builder
	@go tool cover -html=/tmp/art_coverage.out

build:
	@echo "*** Build project..."
	@go build -v .

build-asm:
	@go build -a -work -v -gcflags="-S -B -C" .

build-race:
	@echo "*** Build project with race condition..."
	@go build --race -v .

bootstrap:
	@for tool in  $(EXTERNAL_TOOLS) ; do \
		echo "Installing $$tool" ; \
    	go get $$tool; \
	done
