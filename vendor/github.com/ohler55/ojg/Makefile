
Q = $(if $(filter 1,$V),,@)

all: cover

lint:
	golangci-lint run

cover: lint
	go test -coverpkg github.com/ohler55/ojg -coverprofile=cov.out
	make -C oj
	make -C sen
	make -C pretty
	make -C alt
	make -C jp
	make -C gen
	make -C asm
	$Q grep github oj/cov.out >> cov.out
	$Q grep github sen/cov.out >> cov.out
	$Q grep github pretty/cov.out >> cov.out
	$Q grep github alt/cov.out >> cov.out
	$Q grep github jp/cov.out >> cov.out
	$Q grep github gen/cov.out >> cov.out
	$Q grep github asm/cov.out >> cov.out
	$Q go tool cover -func=cov.out | grep "total:"

test: cover

build:
	make -C cmd

.PHONY: all lint cover test build
