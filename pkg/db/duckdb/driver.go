package duckdb

import "database/sql/driver"

// Driver implements database/sql/driver.Driver for the duckdb-CLI-subprocess
// transport. Use Connector (process.go) via sql.OpenDB to construct a *sql.DB
// backed by a real subprocess; Driver.Open is not exercised by this POC and
// exists only to satisfy the driver.Driver interface.
type Driver struct{}

func (d *Driver) Open(name string) (driver.Conn, error) {
	return nil, errUnsupportedOpen
}

type unsupportedOpenError struct{}

func (*unsupportedOpenError) Error() string {
	return "duckdb: use duckdb.Open(ctx, opts) / duckdb.Connector, not sql.Open"
}

var errUnsupportedOpen = &unsupportedOpenError{}
