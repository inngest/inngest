package duckdb

import (
	"context"
	"database/sql/driver"
	"errors"
)

// sqlExecer abstracts over what runs a SQL statement and collects its JSON
// rows. *session (rows.go) implements it directly — Task 4's tests exercise
// conn against a fake io.ReadWriter transport via a bare *session. *process
// (process.go) implements it too, additionally providing crash detection,
// one-restart-then-permanently-disable recovery, and locking around the real
// subprocess. conn is deliberately transport-agnostic so both keep working
// unchanged.
type sqlExecer interface {
	exec(ctx context.Context, sqlText string) ([]map[string]any, error)
}

// conn implements database/sql/driver.Conn, ExecerContext, and QueryerContext
// over a single duckdb subprocess session. Parameter binding is handled by
// literal.go (encodeArgs) since this transport has no wire-level bind
// protocol — see docs/plans/006-duckdb-poc-subprocess-dual-write.md.
type conn struct {
	sess sqlExecer
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
