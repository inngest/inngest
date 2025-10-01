.PHONY: dev
dev: docs
	goreleaser build --single-target --snapshot --clean

# specifically for tests
.PHONY: run
run:
	TEST_MODE=true LOG_LEVEL=info DEBUG=1 go run ./cmd dev --tick=50 --no-poll --verbose $(PARAMS)

# Start with debug mode in Delve
.PHONY: debug
debug:
	TEST_MODE=true LOG_LEVEL=trace DEBUG=1 dlv debug ./cmd --headless --listen=127.0.0.1:40000 --continue --accept-multiclient --log -- dev --tick=50 --no-poll --no-discovery --verbose $(PARAMS)

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


.PHONY: e2e-golang
e2e-golang:
	SDK_URL=http://localhost:3000/api/inngest API_URL=http://localhost:8288 go test ./tests/golang -v -count=1

.PHONY: gen
gen:
	go generate ./...
	make gql queries

.PHONY: protobuf
protobuf:
	buf generate
	buf generate --path proto/api/v2 --template proto/api/v2/buf.gen.yaml
	buf generate --path proto/connect/v1 --template proto/connect/v1/buf.gen.yaml
	buf generate --path proto/debug/v1 --template proto/debug/v1/buf.gen.yaml
	buf generate --path proto/state/v2 --template proto/state/v2/buf.gen.yaml

# $GOBIN must be set and be in your path for this to work
.PHONY: queries
queries:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	sqlc generate

.PHONY: snapshot
snapshot:
	goreleaser release --snapshot --skip publish --clean

.PHONY: build-ui
build-ui:
	cd ui/apps/dev-server-ui && pnpm install --frozen-lockfile
	cd ui/apps/dev-server-ui && pnpm build
	cp -r ./ui/apps/dev-server-ui/dist/* ./pkg/devserver/static/
	cp -r ./ui/apps/dev-server-ui/.next/routes-manifest.json ./pkg/devserver/static/

# Generate OpenAPI documentation from protobuf files
.PHONY: docs
docs:
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
build: docs
	goreleaser build

.PHONY: gql
gql:
	go run github.com/99designs/gqlgen --verbose --config ./pkg/coreapi/gqlgen.yml

.PHONY: clean
clean:
	rm -f __debug_bin*
	rm -rf docs/openapi/v2/*
	rm -rf docs/openapi/v3/*
