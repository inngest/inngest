.PHONY: dev
dev:
	goreleaser build --single-target --snapshot --rm-dist

xgo:
	xgo -pkg cmd -ldflags="-s -w" -out inngest -targets "linux/arm64,linux/amd64,darwin/arm64,darwin/amd64,windows/amd64" .

.PHONY: test
test:
	sh -c 'cd ./pkg/cuedefs && cue vet ./tests/... -c'
	sh -c 'cd ./pkg/cuedefs && cue eval ./tests/... -c'
	go test $(shell go list ./... | grep -v tests) -race -count=1
	golangci-lint run

.PHONY: vendor
vendor:
	go install github.com/goware/modvendor@latest
	go mod tidy && go mod vendor && modvendor -copy="**/*.a" -v

.PHONY: lint
lint:
	golangci-lint run --verbose

.PHONY: e2e
e2e:
	./tests.sh

queries:
	go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
	sqlc generate
	# sed -i 's#interface{}#uuid.UUID#' ./pkg/cqrs/ddb/queries/*.go

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
