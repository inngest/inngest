# Contributing

## Upgrading DuckDB

`go-duckdb` includes a copy of the DuckDB amalgamation (`duckdb.cpp`, `duckdb.h` and `duckdb.hpp`) and pre-compiled static libraries for faster builds on common platforms (see the `deps` directory). It uses Github Actions to pre-compile the static libraries based on the amalgamation code. This approach is inspired by a combination of [go-sqlite3](https://github.com/mattn/go-sqlite3) and [v8go](https://github.com/rogchap/v8go).

To upgrade to a new version of DuckDB:

1. Change `DUCKDB_VERSION` in `Makefile`
2. Run `make deps.source`
3. Push the updated amalgamation files in a new PR
4. Wait for Github Actions to pre-compile the static libraries in `deps` and push them to the PR
5. Once the static libraries have been pushed, you can merge the PR to master
