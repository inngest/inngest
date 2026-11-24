package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/inngest/inngest/pkg/duckdb/worker"
)

// QuackColumnKind identifies one column's physical wire type for
// QuackAppender.AppendRow.
type QuackColumnKind = worker.StreamColumnKind

const (
	QuackColumnUUID        = worker.StreamColumnUUID
	QuackColumnVarchar     = worker.StreamColumnVarchar
	QuackColumnJSON        = worker.StreamColumnJSON
	QuackColumnTimestampTZ = worker.StreamColumnTimestampTZ
	QuackColumnBool        = worker.StreamColumnBool
	QuackColumnBigint      = worker.StreamColumnBigint
)

// QuackAppender bulk-loads rows into one table over the worker's binary
// columnar stream mechanism. AppendRow buffers rows in memory; Flush sends
// everything buffered as one batch.
type QuackAppender struct {
	session    *worker.Session
	schema     string
	table      string
	columns    []QuackColumnKind
	rows       [][]any
	pooledConn *sql.Conn
}

// NewQuackAppender returns a QuackAppender for catalog.schema.table, holding
// an acquired *sql.Conn open until Close is called.
func NewQuackAppender(ctx context.Context, db *sql.DB, catalog, schema, table string, columns []QuackColumnKind) (*QuackAppender, error) {
	sqlConn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("duckdb: quack appender: acquiring connection: %w", err)
	}

	var appender *QuackAppender
	err = sqlConn.Raw(func(driverConn any) error {
		var err error
		appender, err = newQuackAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, columns)
		return err
	})
	if err != nil {
		_ = sqlConn.Close()
		return nil, err
	}
	appender.pooledConn = sqlConn
	return appender, nil
}

// NewQuackAppenderFromConn is NewQuackAppender for a driver.Conn the caller
// already owns outright.
func NewQuackAppenderFromConn(ctx context.Context, driverConn driver.Conn, catalog, schema, table string, columns []QuackColumnKind) (*QuackAppender, error) {
	return newQuackAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, columns)
}

func newQuackAppenderFromDriverConn(ctx context.Context, driverConn any, catalog, schema, table string, columns []QuackColumnKind) (*QuackAppender, error) {
	sess, err := resolveWorkerSessionForAppender(ctx, driverConn, catalog, schema, "quack appender")
	if err != nil {
		return nil, err
	}
	return &QuackAppender{session: sess, schema: schema, table: table, columns: columns}, nil
}

func resolveWorkerSessionForAppender(ctx context.Context, driverConn any, catalog, schema, errPrefix string) (*worker.Session, error) {
	c, ok := driverConn.(*conn)
	if !ok {
		return nil, fmt.Errorf("duckdb: %s: unexpected driver connection type %T", errPrefix, driverConn)
	}

	var sess *worker.Session
	switch s := c.sess.(type) {
	case *workerSessionExecer:
		if s.sess.Abandoned() {
			return nil, fmt.Errorf("duckdb: %s: worker session was abandoned mid-statement", errPrefix)
		}
		sess = s.sess
	case *pooledQuackConn:
		var err error
		sess, err = s.currentSession(ctx)
		if err != nil {
			return nil, err
		}
	case *process:
		var err error
		sess, err = s.currentWorkerSession(ctx)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("duckdb: %s: unexpected transport %T", errPrefix, c.sess)
	}

	if catalog != "" {
		target := worker.Identifier(catalog)
		if schema != "" {
			target += "." + worker.Identifier(schema)
		}
		if err := sess.Exec(ctx, fmt.Sprintf("USE %s;", target)); err != nil {
			return nil, fmt.Errorf("duckdb: %s: switching to catalog %q: %w", errPrefix, catalog, translateWorkerErr(err))
		}
	}

	return sess, nil
}

// AppendRow buffers one row.
func (a *QuackAppender) AppendRow(vals ...any) error {
	if len(vals) != len(a.columns) {
		return fmt.Errorf("duckdb: quack appender: got %d values, expected %d columns", len(vals), len(a.columns))
	}
	a.rows = append(a.rows, vals)
	return nil
}

// Flush sends every buffered row as one batch and clears the buffer.
func (a *QuackAppender) Flush(ctx context.Context) error {
	if len(a.rows) == 0 {
		return nil
	}
	if err := a.session.Append(ctx, a.schema, a.table, a.columns, a.rows); err != nil {
		return translateWorkerErr(err)
	}
	a.rows = a.rows[:0]
	return nil
}

// Close flushes any remaining buffered rows and releases any acquired pooled connection.
func (a *QuackAppender) Close(ctx context.Context) error {
	flushErr := a.Flush(ctx)
	if a.pooledConn != nil {
		if closeErr := a.pooledConn.Close(); closeErr != nil && flushErr == nil {
			return closeErr
		}
	}
	return flushErr
}
