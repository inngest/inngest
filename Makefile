.PHONY: dev
dev:
	goreleaser build --single-target --snapshot --rm-dist

.PHONY: run
run:
	LOG_LEVEL=debug go run ./cmd/main.go dev -v $(PARAMS)

xgo:
	xgo -pkg cmd -ldflags="-s -w" -out build/inngest -targets "linux/arm64,linux/amd64,darwin/arm64,darwin/amd64" .

.PHONY: test
test:
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

.PHONY: gen
gen:
	go generate ./...
	make gql queries

.PHONY: protobuf
protobuf:
	buf generate

# $GOBIN must be set and be in your path for this to work
.PHONY: queries
queries:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	sqlc generate

.PHONY: snapshot
snapshot:
	goreleaser release --snapshot --skip-publish --rm-dist

.PHONY: build-ui
build-ui:
	cd ui/apps/dev-server-ui && pnpm install
	cd ui/apps/dev-server-ui && pnpm build
	cp -r ./ui/apps/dev-server-ui/dist/* ./pkg/devserver/static/
	cp -r ./ui/apps/dev-server-ui/.next/routes-manifest.json ./pkg/devserver/static/

.PHONY: build
build:
	goreleaser build

.PHONY: gql
gql:
	go run github.com/99designs/gqlgen --verbose --config ./pkg/coreapi/gqlgen.yml

.PHONY: nats-js
nats-js:
	nats-server -js -sd ./tmp/nats -m 8222

.PHONY: nats-stream
nats-stream:
	nats stream add \
		--subjects=inngest.run.spans \
		--description='Trace span delivery pipeline' \
		--retention=limits \
		--max-msg-size=10mb \
		--max-age=7d \
		--storage=file \
		--replicas=1 \
		--discard=old \
		--defaults \
		trace
