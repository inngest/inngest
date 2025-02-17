.PHONY: itest
itest:
	go test ./tests -v -count=1

.PHONY: utest
utest:
	go test -test.v -short

.PHONY: lint
lint:
	golangci-lint run --verbose
