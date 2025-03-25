.PHONY: itest
itest:
	go test . -v -count=1 -race

.PHONY: utest
utest:
	go test -test.v -short -race

.PHONY: lint
lint:
	golangci-lint run --verbose
