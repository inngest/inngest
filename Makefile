.PHONY: help
help: ## Print help
	@grep -E '^[/a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: dev
dev: docs ## Build dev binary with goreleaser
	goreleaser build --single-target --snapshot --clean

.PHONY: run
run: ## Run dev server in test mode
	TEST_MODE=true LOG_LEVEL=info DEBUG=1 go run ./cmd dev --tick=50 --no-poll --verbose $(PARAMS)

.PHONY: debug
debug: ## Run dev server with Delve debugger
	TEST_MODE=true LOG_LEVEL=trace DEBUG=1 dlv debug ./cmd --headless --listen=127.0.0.1:40000 --continue --accept-multiclient --log -- dev --tick=50 --no-poll --no-discovery --verbose $(PARAMS)

.PHONY: xgo
xgo: ## Cross-compile for multiple platforms
	xgo -pkg cmd -ldflags="-s -w" -out build/inngest -targets "linux/arm64,linux/amd64,darwin/arm64,darwin/amd64" .

.PHONY: test
test: ## Run tests and linter
	go test $(shell go list ./... | grep -v tests) -race -count=1
	golangci-lint run

.PHONY: vendor
vendor: ## Update vendored dependencies
	go install github.com/goware/modvendor@latest
	go mod tidy && go mod vendor && modvendor -copy="**/*.a" -v

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run --verbose

.PHONY: e2e
e2e: ## Run end-to-end tests
	./tests.sh

.PHONY: e2e-golang
e2e-golang: ## Run Go SDK e2e tests
	SDK_URL=http://localhost:3000/api/inngest API_URL=http://localhost:8288 go test ./tests/golang -v -count=1

.PHONY: gen
gen: ## Run all code generators
	go generate ./...
	make gql queries constraintapi-snapshots tygo

.PHONY: protobuf
protobuf: ## Generate protobuf files
	buf generate
	buf generate --path proto/api/v2 --template proto/api/v2/buf.gen.yaml
	buf generate --path proto/connect/v1 --template proto/connect/v1/buf.gen.yaml
	buf generate --path proto/debug/v1 --template proto/debug/v1/buf.gen.yaml
	buf generate --path proto/state/v2 --template proto/state/v2/buf.gen.yaml
	buf generate --path proto/constraintapi/v1 --template proto/constraintapi/v1/buf.gen.yaml

# $GOBIN must be set and be in your path for this to work
.PHONY: queries
queries: ## Generate sqlc queries
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	sqlc generate

.PHONY: snapshot
snapshot: ## Build release snapshot
	goreleaser release --snapshot --skip publish --clean

.PHONY: build-ui
build-ui: ## Build dev server UI
	cd ui/apps/dev-server-ui && pnpm install --frozen-lockfile
	cd ui/apps/dev-server-ui && pnpm build
	cp -r ./ui/apps/dev-server-ui/dist/* ./pkg/devserver/static/

.PHONY: docs
docs: ## Generate OpenAPI documentation
	@echo "Validating examples JSON structure..."
	@cd tools/convert-openapi && go test -run TestExamplesJSONStructure -v
	@echo "Generating protobuf files..."
	@# Generate OpenAPI v2 directly using protoc due to buf configuration issues
	@mkdir -p docs/openapi/v2
	cd proto && protoc --proto_path=. --proto_path=third_party \
		--openapiv2_out=../docs/openapi/v2 \
		--openapiv2_opt=allow_delete_body=true \
		--openapiv2_opt=json_names_for_fields=false \
		api/v2/service.proto
	@echo "Converting OpenAPI v2 to v3..."
	go run ./tools/convert-openapi docs/openapi/v2 docs/openapi/v3

.PHONY: build
build: docs ## Build release binaries
	goreleaser build

.PHONY: gql
gql: ## Generate GraphQL code
	go run github.com/99designs/gqlgen --verbose --config ./pkg/coreapi/gqlgen.yml

.PHONY: tygo
tygo: ## Generate TypeScript types from Go structs
	go run github.com/gzuidhof/tygo@latest generate
	cd ui && pnpm prettier --write "packages/components/src/generated/**/*.ts"

.PHONY: constraintapi-snapshots
constraintapi-snapshots: ## Regenerate constraint API Lua snapshots
	@echo "Regenerating constraint API Lua script snapshots..."
	rm -rf pkg/constraintapi/testdata/snapshots
	cd pkg/constraintapi && go test -run TestLuaScriptSnapshots .

.PHONY: clean
clean: ## Remove build artifacts
	rm -f __debug_bin*
	rm -rf docs/openapi/v2/*
	rm -rf docs/openapi/v3/*
