# Copyright 2024 The Sqlite Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

.PHONY:	all build_all_targets clean edit editor test vendor work

# Pinned deduplicator. undup folds byte-identical declarations shared across the
# per-target generated files in lib/ and vec/ into build-tagged shared files,
# keeping each tag's module download under Go's 500MB cap. Bump deliberately.
UNDUP = modernc.org/undup@v0.0.5

all: editor
	golint 2>&1
	staticcheck 2>&1

build_all_targets:
	GOOS=darwin GOARCH=amd64 go test -c -o /dev/null
	GOOS=darwin GOARCH=amd64 go build -v ./...
	GOOS=darwin GOARCH=arm64 go test -c -o /dev/null
	GOOS=darwin GOARCH=arm64 go build -v ./...
	GOOS=freebsd GOARCH=amd64 go test -c -o /dev/null
	GOOS=freebsd GOARCH=amd64 go build -v ./...
	GOOS=freebsd GOARCH=386 go test -c -o /dev/null
	GOOS=freebsd GOARCH=386 go build -v ./...
	GOOS=freebsd GOARCH=arm go test -c -o /dev/null
	GOOS=freebsd GOARCH=arm go build -v ./...
	GOOS=freebsd GOARCH=arm64 go test -c -o /dev/null
	GOOS=freebsd GOARCH=arm64 go build -v ./...
	GOOS=linux GOARCH=386 go test -c -o /dev/null
	GOOS=linux GOARCH=386 go build -v ./...
	GOOS=linux GOARCH=amd64 go test -c -o /dev/null
	GOOS=linux GOARCH=amd64 go build -v ./...
	GOOS=linux GOARCH=arm go test -c -o /dev/null
	GOOS=linux GOARCH=arm go build -v ./...
	GOOS=linux GOARCH=arm64 go test -c -o /dev/null
	GOOS=linux GOARCH=arm64 go build -v ./...
	GOOS=linux GOARCH=loong64 go test -c -o /dev/null
	GOOS=linux GOARCH=loong64 go build -v ./...
	GOOS=linux GOARCH=ppc64le go test -c -o /dev/null
	GOOS=linux GOARCH=ppc64le go build -v ./...
	GOOS=linux GOARCH=riscv64 go test -c -o /dev/null
	GOOS=linux GOARCH=riscv64 go build -v ./...
	GOOS=linux GOARCH=s390x go test -c -o /dev/null
	GOOS=linux GOARCH=s390x go build -v ./...
	GOOS=netbsd GOARCH=amd64 go test -c -o /dev/null
	GOOS=netbsd GOARCH=amd64 go build -v ./...
	GOOS=openbsd GOARCH=amd64 go test -c -o /dev/null
	GOOS=openbsd GOARCH=amd64 go build -v ./...
	GOOS=openbsd GOARCH=arm64 go test -c -o /dev/null
	GOOS=openbsd GOARCH=arm64 go build -v ./...
	GOOS=windows GOARCH=386 go test -c -o /dev/null
	GOOS=windows GOARCH=386 go build -v ./...
	GOOS=windows GOARCH=amd64 go test -c -o /dev/null
	GOOS=windows GOARCH=amd64 go build -v ./...
	GOOS=windows GOARCH=arm64 go test -c -o /dev/null
	GOOS=windows GOARCH=arm64 go build -v ./...
	echo done

clean:
	rm -f log-* cpu.test mem.test *.out go.work*
	go clean

edit:
	@if [ -f "Session.vim" ]; then gvim -S & else gvim -p Makefile go.mod builder.json all_test.go & fi

editor:
	go test -c -o /dev/null
	go build -v  -o /dev/null ./...
	cd vendor_libs && go build -o /dev/null main.go

test:
	go test -v -timeout 24h
	
vendor:
	# Reconstruct full per-target files (a no-op the first time), so the freshly
	# vendored transpiles overwrite a clean tree with no stale shared files.
	go run $(UNDUP) -expand -dir lib
	go run $(UNDUP) -expand -dir vec
	cd vendor_libs && go build -o ../vendor main.go
	# ../libsqlite3 and ../libsqlite_vec are read one full per-target file at a
	# time. They ship expanded today, but either may adopt the deduplicated
	# layout (modernc.org/builder's NW autogen); the tool detects that and
	# expands a temporary copy, leaving those checkouts untouched. Hence the pin:
	# one version of record for this repo, wherever undup is invoked.
	./vendor -undup=$(UNDUP)
	rm -f vendor
	# Fold byte-identical declarations back into build-tagged shared files. undup
	# only touches files carrying the generated-code marker, never hand-written
	# platform files (libsqlite3_*.go, hooks_*.go, ...).
	go run $(UNDUP) -dir lib
	go run $(UNDUP) -dir vec
	gofmt -s -w lib/sqlite*.go vec/vec*.go
	make build_all_targets

work:
	rm -f go.work*
	go work init
	go work use .
	go work use ../cc/v4
	go work use ../ccgo/v3
	go work use ../ccgo/v4
	go work use ../libc
	go work use ../libtcl8.6
	go work use ../libsqlite3
	go work use ../libz
