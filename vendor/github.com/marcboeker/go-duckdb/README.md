# Go SQL driver for [DuckDB](https://github.com/duckdb/duckdb)

The DuckDB driver conforms to the built-in `database/sql` interface.

![Tests status](https://github.com/marcboeker/go-duckdb/actions/workflows/tests.yaml/badge.svg)

## Notice on v1.4.0

Version `1.4.0` changed the DuckDB decimal representation from `float64` to a new [`Decimal`](https://github.com/marcboeker/go-duckdb/blob/d722d9c9d2bc9364d2f22a3afec3cbd26ac07f41/types.go#L83) type, which is much more precise. If you are upgrading to `v1.4.0` and are using DuckDBs decimals, please make sure to update your code to make use of the new `Decimal` type.

## Installation

```
go get github.com/marcboeker/go-duckdb
```

`go-duckdb` uses `CGO` to make calls to DuckDB. You must build your binaries with `CGO_ENABLED=1`.

## Usage

`go-duckdb` hooks into the `database/sql` interface provided by the Go stdlib. To open a connection, simply specify the driver type as `duckdb`:

```go
db, err := sql.Open("duckdb", "")
```

This creates an in-memory instance of DuckDB. If you would like to store the data on the filesystem, you need to specify the path where to store the database:

```go
db, err := sql.Open("duckdb", "/path/to/foo.db")
```

If you want to set specific [config options for DuckDB](https://duckdb.org/docs/sql/configuration), you can add them as query style parameters in the form of `name=value` to the DSN, like:

```go
db, err := sql.Open("duckdb", "/path/to/foo.db?access_mode=read_only&threads=4")
```

Alternatively, you can also use `sql.OpenDB` when you want to perform some initialization before the connection is created and returned from the connection pool on call to `db.Conn`.
Here's an example that installs and loads the JSON extension for each connection:

```go
connector, err := duckdb.NewConnector("/path/to/foo.db?access_mode=read_only&threads=4", func(execer driver.Execer) error {
  bootQueries := []string{
    "INSTALL 'json'",
    "LOAD 'json'",
  }

  for _, qry := range bootQueries {
    _, err = execer.Exec(qry, nil)
    if err != nil {
      return err
    }
  }
  return nil
})
if err != nil {
  return nil, err
}

db := sql.OpenDB(connector)
db.SetMaxOpenConns(poolsize)
...
```

Please refer to the [database/sql](https://godoc.org/database/sql) GoDoc for further usage instructions.

## DuckDB Appender API

If you want to use the [DuckDB Appender API](https://duckdb.org/docs/data/appender.html), you can obtain a new Appender by supplying a DuckDB connection to `NewAppenderFromConn()`.

```go
connector, err := NewConnector("test.db", nil)
if err != {
  ...
}
conn, err := connector.Connect(context.Background())
if err != {
  ...
}
defer conn.Close()

// Retrieve appender from connection.
appender, err := NewAppenderFromConn(conn, "", "test")
if err != {
  ...
}
defer appender.Close()

err = appender.AppendRow(...)
if err != {
  ...
}

// Optional, if you want to access the appended rows immediately.
err = appender.Flush()
if err != {
  ...
}
```

## Linking DuckDB

By default, `go-duckdb` statically links DuckDB into your binary. Statically linking DuckDB adds around 30 MB to your binary size. On Linux (Intel) and macOS (Intel and ARM), `go-duckdb` bundles pre-compiled static libraries for fast builds. On other platforms, it falls back to compiling DuckDB from source, which takes around 10 minutes. You can force `go-duckdb` to build DuckDB from source by passing `-tags=duckdb_from_source` to `go build`.

Alternatively, you can dynamically link DuckDB by passing `-tags=duckdb_use_lib` to `go build`. You must have a copy of `libduckdb` available on your system (`.so` on Linux or `.dylib` on macOS), which you can download from the DuckDB [releases page](https://github.com/duckdb/duckdb/releases). For example:

```sh
# On Linux
CGO_ENABLED=1 CGO_LDFLAGS="-L/path/to/libs" go build -tags=duckdb_use_lib main.go
LD_LIBRARY_PATH=/path/to/libs ./main

# On macOS
CGO_ENABLED=1 CGO_LDFLAGS="-L/path/to/libs" go build -tags=duckdb_use_lib main.go
DYLD_LIBRARY_PATH=/path/to/libs ./main
```
