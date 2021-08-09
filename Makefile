snapshot:
	goreleaser release --snapshot --skip-publish --rm-dist

build:
	goreleaser build
