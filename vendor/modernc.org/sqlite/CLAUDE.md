# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this package is

`modernc.org/sqlite` is a pure-Go `database/sql/driver` for SQLite — **CGo-free**. The SQLite C amalgamation is transpiled to Go via `modernc.org/ccgo`; the generated code lives in this repo as per-`GOOS`/`GOARCH` files under `lib/` (SQLite itself), `vec/` (the `sqlite-vec` extension), and `vfs/` (the C side of the Go-fs VFS bridge). Runtime support — `malloc`, `pthread`, syscalls, etc. — is provided by `modernc.org/libc`.

The hand-written Go on top of that transpiled core implements the `database/sql/driver` shim and additional Go-facing APIs (virtual tables, VFS, hooks, UDFs).

## Repository layout (the parts that aren't self-evident)

- `sqlite.go`, `conn.go`, `driver.go`, `stmt.go`, `rows.go`, `tx.go`, `backup.go`, `error.go`, `result.go`, `convert.go` — hand-written `database/sql/driver` implementation calling into `lib/`.
- `vtab.go`, `pre_update_hook.go`, `fcntl.go`, `mutex.go` — Go-facing extensions wired to SQLite hooks/trampolines.
- `lib/` — transpiled SQLite 3.53.3. One `sqlite_<goos>_<goarch>.go` per supported triple; `defs.go`, `hooks.go`, `hooks_linux_arm64.go`, `mutex.go`, plus `libsqlite3_freebsd.go`/`libsqlite3_windows.go` hold hand-written patches that augment the generated code. Import as `sqlite3 "modernc.org/sqlite/lib"`.
- `vec/` — transpiled `sqlite-vec` v0.1.9, auto-registers via `sqlite3_auto_extension` in `patches.go` on package init. Activate by blank-importing: `_ "modernc.org/sqlite/vec"`. Covers the same 19 targets `lib/` does; `vec_test.go`'s `//go:build` constrains by GOOS only.
- `vfs/` — exposes a Go `fs.FS` as a read-only SQLite VFS. `vfs.New(fsys)` returns a registered VFS name; open with `?vfs=<name>`. C side is transpiled per platform from `vfs/c/vfs.c` via the `vfs/Makefile`.
- `vtab/` — Go-facing virtual-table API (no dependency on the transpiled C). `vtab.RegisterModule(db, name, module)` registers modules on **new connections only**; a nil `db` targets the driver registered as `sqlite`, a non-nil `db` the driver backing it (via `vtab.ModuleRegisterer`). The bridge to C lives in the top-level `vtab.go`. See `vtab/doc.go` for the contract (Updater/Renamer/Transactional optional interfaces, re-entrancy rules, ArgIndex/Omit semantics).
- `vendor_libs/main.go` (build tag `none`) — regeneration tool. Reads transpiled `ccgo_<goos>_<goarch>.go` from sibling repos `../libsqlite3` and `../libsqlite_vec`, rewrites package names and imports, and writes `lib/sqlite_*.go` / `vec/vec_*.go`. Invoked by `make vendor`.
- `examples/` — runnable samples: `example1`, `connector`, `vtab_basic`, `vtab_csv`, `vtab_match`, `vtab_regexp`.
- `addport.go`, `issue198/`, `issue120.diff` — porting/regression scaffolding kept around for reference; not built.

## Commands

```bash
make editor              # quick local check: go test -c + go build ./... + vendor_libs build
make test                # go test -v -timeout 24h (the full suite is long)
make build_all_targets   # cross-build every supported GOOS/GOARCH
make vendor              # regenerate lib/ and vec/ from sibling ../libsqlite3 + ../libsqlite_vec
make all                 # editor + golint + staticcheck
make work                # set up go.work pointing at sibling cc/ccgo/libc/libtcl8.6/libsqlite3/libz repos
make clean               # removes log-*, *.test, *.out, go.work*
```

Single test: `go test -v -run TestScalar` (pattern is a regexp; tests live in `all_test.go`, `module_test.go`, `func_test.go`, `pre_update_hook_test.go`, `vec_test.go`, `leak_test.go`, `fcntl_test.go`, `backup_test.go`, `null_test.go`). VFS tests: `go test ./vfs/...`.

Build/debug tags:
- `-tags=sqlite.dmesg` — enables this package's `dmesg(...)` (writes to `/tmp/libc.log`); see `dmesg.go` / `nodmesg.go`.
- `-tags=libc.dmesg` — enables debug logs from `modernc.org/libc` (must be combined with patching `libc` itself — see the worked example in `doc.go`).

There is no `go generate` in this repo — `generator.go` lives in `../libsqlite3`, which owns the transpilation and its SQLite compile-time options. To produce a debug-instrumented transpilation, change the options there, `make generate` in that repo, then `make vendor` here.

## Fragile `modernc.org/libc` coupling

Downstream `go.mod` files **must pin the exact `modernc.org/libc` version that this repo's `go.mod` pins** — the transpiled code in `lib/` is closely tied to that specific `libc`. This is documented in `doc.go` and tracked in [issue #177](https://gitlab.com/cznic/sqlite/-/issues/177). Bumping `libc` here without re-transpiling (or vice-versa) breaks consumers; that's why `v1.33.0`, `v1.34.3`, and `v1.42.0` are retracted in `go.mod`.

When debugging into `libc`, use `make work` (or a manual `go work init && go work use . <path-to-libc>`) — `doc.go` has a worked example showing how to enable `Xwrite` dmesg logging in a local `libc` checkout.

## Repository / release workflow

- The canonical repo is GitLab `cznic/sqlite`. The GitHub `modernc-org/sqlite` mirror **does accept** issues and PRs, but PRs land via a manual cross-merge into GitLab — there can be a delay. The PRs listed in `CHANGELOG.md` (e.g. "merge request #113") are GitLab MR numbers, not GitHub PRs.
- Per `HACKING.md`: this repo is **not** auto-tagged — `builder.json` has `"autotag": "<none>"` because too many projects depend on `modernc.org/sqlite` to risk bot tagging. Releases are tagged **manually by the maintainer**; don't tag unless asked, and only once [the builder dashboard](https://modern-c.appspot.com/-/builder/?importpath=modernc.org%2fsqlite) is green for all platforms listed in `builder.json`.
- `go.mod` `retract` directives encode known-broken versions; treat them as load-bearing — don't remove entries when bumping the module.

## Driver registration model

`init()` in `sqlite.go` calls `sql.Register("sqlite", defaultDriver())` with a single package-level `*Driver` (`var d` in `driver.go`). The package-level registration functions (`RegisterFunction`, `RegisterScalarFunction`, `RegisterDeterministicScalarFunction`, `RegisterCollationUtf8`, `RegisterConnectionHook`) attach to that driver alone; its vtab modules — `vtab.RegisterModule` with a nil `db` — are the one category held process-globally and reach every driver's connections. A caller-constructed `Driver` carries its own registrations through the mirroring `*Driver` methods (`RegisterFunction`, `RegisterScalarFunction`, `RegisterDeterministicScalarFunction`, `RegisterCollationUtf8`, `RegisterConnectionHook`, `RegisterModule`, plus `Must*` variants), and `vtab.RegisterModule` with a non-nil `db` lands on the driver backing that `db`. Same-name module on both: the package-level implementation wins on that driver's connections. All registration applies to connections opened **afterwards**; registrations made after a connection is open do not affect it — open a new one. See the `Driver` type doc in `driver.go` and `vtab/doc.go`.

DSN query params are parsed in `conn.go`/`driver.go`: `_pragma`, `_time_format`, `_time_integer_format`, `_inttotime`, `_texttotime`, `_timezone`, `_txlock`, plus `vfs=<name>` to select a VFS registered via `vfs.New`.
