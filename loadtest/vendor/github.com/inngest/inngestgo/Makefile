.PHONY: itest
itest:
	go test ./tests -v -count=1 -race

.PHONY: utest
utest:
	go test `go list ./... | grep -v "/tests"` -v -count=1 -race

.PHONY: lint
lint:
	golangci-lint run --verbose
