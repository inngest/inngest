test:
	go test ./... -race -count=1
	golangci-lint run

snapshot:
	goreleaser release --snapshot --skip-publish --rm-dist

build:
	goreleaser build
