.PHONY: dev
dev:
	goreleaser build --single-target --snapshot --rm-dist

.PHONY: test
test:
	sh -c 'cd ./pkg/cuedefs && cue vet ./tests/... -c'
	sh -c 'cd ./pkg/cuedefs && cue eval ./tests/... -c'
	go test $(shell go list ./... | grep -v tests) -race -count=1
	golangci-lint run

.PHONY: lint
lint:
	golangci-lint run --verbose

.PHONY: e2e
e2e:
	./tests.sh

.PHONY: snapshot
snapshot:
	goreleaser release --snapshot --skip-publish --rm-dist

.PHONY: build-ui
build-ui:
	cd ui && pnpm install
	cd ui && pnpm build
	cp -r ./ui/dist/* ./pkg/devserver/static/

.PHONY: build
build:
	goreleaser build

.PHONY: gql
gql:
	go run github.com/99designs/gqlgen --verbose --config ./pkg/coreapi/gqlgen.yml
