package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/inngest/inngest/pkg/duckdb/worker"
)

// QuackMergeConfig describes the MERGE INTO statement a QuackMergeAppender drives.
type QuackMergeConfig = worker.MergeConfig

// QuackMergeColumn names one column of a QuackMergeAppender's source rows.
type QuackMergeColumn = worker.MergeColumn

// QuackMergeAppender bulk-merges rows into one table over the worker's binary
// columnar stream mechanism.
type QuackMergeAppender struct {
	session    *worker.Session
	schema     string
	table      string
	cfg        QuackMergeConfig
	columns    []QuackMergeColumn
	rows       [][]any
	pooledConn *sql.Conn
}

// NewQuackMergeAppender returns a QuackMergeAppender for catalog.schema.table.
func NewQuackMergeAppender(ctx context.Context, db *sql.DB, catalog, schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn) (*QuackMergeAppender, error) {
	sqlConn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("duckdb: quack merge appender: acquiring connection: %w", err)
	}

	var appender *QuackMergeAppender
	err = sqlConn.Raw(func(driverConn any) error {
		var err error
		appender, err = newQuackMergeAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, cfg, columns)
		return err
	})
	if err != nil {
		_ = sqlConn.Close()
		return nil, err
	}
	appender.pooledConn = sqlConn
	return appender, nil
}

// NewQuackMergeAppenderFromConn is NewQuackMergeAppender for a driver.Conn
// the caller already owns outright.
func NewQuackMergeAppenderFromConn(ctx context.Context, driverConn driver.Conn, catalog, schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn) (*QuackMergeAppender, error) {
	return newQuackMergeAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, cfg, columns)
}

func newQuackMergeAppenderFromDriverConn(ctx context.Context, driverConn any, catalog, schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn) (*QuackMergeAppender, error) {
	sess, err := resolveWorkerSessionForAppender(ctx, driverConn, catalog, schema, "quack merge appender")
	if err != nil {
		return nil, err
	}
	return &QuackMergeAppender{session: sess, schema: schema, table: table, cfg: cfg, columns: columns}, nil
}

// AppendRow buffers one row.
func (a *QuackMergeAppender) AppendRow(vals ...any) error {
	if len(vals) != len(a.columns) {
		return fmt.Errorf("duckdb: quack merge appender: got %d values, expected %d columns", len(vals), len(a.columns))
	}
	a.rows = append(a.rows, vals)
	return nil
}

// Flush runs the MERGE with every buffered row as one batch and clears the buffer.
func (a *QuackMergeAppender) Flush(ctx context.Context) error {
	if len(a.rows) == 0 {
		return nil
	}
	if err := a.session.Merge(ctx, a.schema, a.table, a.cfg, a.columns, a.rows); err != nil {
		return translateWorkerErr(err)
	}
	a.rows = a.rows[:0]
	return nil
}

// Close flushes any remaining buffered rows and releases any acquired pooled connection.
func (a *QuackMergeAppender) Close(ctx context.Context) error {
	flushErr := a.Flush(ctx)
	if a.pooledConn != nil {
		if closeErr := a.pooledConn.Close(); closeErr != nil && flushErr == nil {
			return closeErr
		}
	}
	return flushErr
}
