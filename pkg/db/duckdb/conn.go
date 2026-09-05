package duckdb

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

// sqlExecer abstracts over what runs a SQL statement and collects its JSON
// rows. *session (rows.go) implements it directly — Task 4's tests exercise
// conn against a fake io.ReadWriter transport via a bare *session. *process
// (process.go) implements it too, additionally providing crash detection,
// one-restart-then-permanently-disable recovery, and locking around the real
// subprocess. conn is deliberately transport-agnostic so both keep working
// unchanged.
//
// cols is the result's column names in the query's own left-to-right order —
// carried separately from rows because a map can't preserve it. It may be
// nil for a statement that returns no columns (DDL) or, for *session
// specifically, no rows (see rows.go's session.exec).
type sqlExecer interface {
	exec(ctx context.Context, sqlText string) (cols []string, rows []map[string]any, err error)
}

// conn implements database/sql/driver.Conn, ExecerContext, and QueryerContext
// over a single duckdb subprocess session. Parameter binding is handled by
// literal.go (encodeArgs) since this transport has no wire-level bind
// protocol.
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
	logger.StdlibLogger(ctx).Debug("duckdb: exec", "sql", sql)
	if _, _, err := c.sess.exec(ctx, sql); err != nil {
		return nil, err
	}
	return driver.RowsAffected(0), nil
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	sql, err := interpolate(query, args)
	if err != nil {
		return nil, err
	}
	logger.StdlibLogger(ctx).Debug("duckdb: query", "sql", sql)
	cols, rows, err := c.sess.exec(ctx, sql)
	if err != nil {
		return nil, err
	}
	return newMapRows(cols, rows), nil
}

func (c *conn) CheckNamedValue(nv *driver.NamedValue) error {
	switch v := nv.Value.(type) {
	case []json.RawMessage:
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		nv.Value = string(data)
		return nil
	case enums.RunStatus:
		nv.Value = v.String()
		return nil
	case enums.StepStatus:
		nv.Value = v.String()
		return nil
	case ulid.ULID:
		nv.Value = v.String()
	case []string:
		// Not one of driver.Value's standard kinds, but encodeLiteral (see
		// literal.go) handles it directly as a DuckDB array literal — this
		// transport binds by SQL-text interpolation, not a wire-level bind
		// protocol, so there's no driver.Value validity constraint to honor
		// here beyond what encodeLiteral itself understands.
		return nil
	case meta.EventSessions:
		// Not one of driver.Value's standard kinds either — encodeLiteral
		// handles it directly as a DuckDB list-of-struct literal, same
		// reasoning as []string above.
		return nil
	}

	return driver.ErrSkip
}
