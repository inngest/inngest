package duckdb

import (
	"context"
	"database/sql/driver"
	"errors"
)

// conn implements database/sql/driver.Conn, ExecerContext, and QueryerContext
// over a single duckdb subprocess session. Parameter binding is handled by
// literal.go (encodeArgs) since this transport has no wire-level bind
// protocol — see docs/plans/006-duckdb-poc-subprocess-dual-write.md.
type conn struct {
	sess *session
}

func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("duckdb: prepared statements not supported; use ExecContext/QueryContext")
}

func (c *conn) Close() error { return nil }

func (c *conn) Begin() (driver.Tx, error) {
	return nil, errors.New("duckdb: transactions not supported in this POC")
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	sql, err := interpolate(query, args)
	if err != nil {
		return nil, err
	}
	if _, err := c.sess.exec(ctx, sql); err != nil {
		return nil, err
	}
	return driver.RowsAffected(0), nil
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	sql, err := interpolate(query, args)
	if err != nil {
		return nil, err
	}
	rows, err := c.sess.exec(ctx, sql)
	if err != nil {
		return nil, err
	}
	return newMapRows(rows), nil
}

// interpolate is a temporary no-op stub. This transport has no wire-level
// bind protocol, so parameters must be encoded as SQL literals directly into
// the query text; Task 6 replaces this with real literal encoding
// (encodeArgs) and reintroduces test coverage for parameter substitution.
func interpolate(query string, args []driver.NamedValue) (string, error) {
	return query, nil
}
