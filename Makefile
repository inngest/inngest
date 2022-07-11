test:
	sh -c 'cd ./pkg/cuedefs && cue vet ./tests/... -c'
	sh -c 'cd ./pkg/cuedefs && cue eval ./tests/... -c'
	go test ./... -race -count=1
	golangci-lint run

snapshot:
	goreleaser release --snapshot --skip-publish --rm-dist

build:
	goreleaser build
