package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/quack"
)

// QuackMergeConfig describes the MERGE INTO statement a QuackMergeAppender
// drives — see quack.MergeConfig.
type QuackMergeConfig = quack.MergeConfig

// QuackMergeColumn names one column of a QuackMergeAppender's source rows —
// see quack.MergeColumn.
type QuackMergeColumn = quack.MergeColumn

// QuackMergeAppender bulk-merges rows into one table over quack's
// SEND_DATA_REQUEST mechanism (see internal/quack/senddata.go). AppendRow buffers
// rows in memory; Flush runs the MERGE with everything buffered as one
// batch. Not safe for concurrent use (one goroutine at a time per
// *QuackMergeAppender — see NewQuackMergeAppender/NewQuackAppender's doc
// comments for why concurrent *appenders*, each on its own instance, are
// fine).
type QuackMergeAppender struct {
	session *quack.Session
	schema  string
	table   string
	cfg     QuackMergeConfig
	columns []QuackMergeColumn
	rows    [][]any

	// pooledConn is non-nil only when this appender came from
	// NewQuackMergeAppender (not NewQuackMergeAppenderFromConn) — see
	// QuackAppender.pooledConn's doc comment, which this shares in full.
	pooledConn *sql.Conn
}

// NewQuackMergeAppender returns a QuackMergeAppender for catalog.schema.table
// — see NewQuackAppender's doc comment for db/catalog requirements and the
// pooled-connection lifetime hazard it fixes, both of which this shares in
// full (including the USE <catalog> caveat).
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
// the caller already owns outright — see NewQuackAppenderFromConn's doc
// comment, which this shares in full.
func NewQuackMergeAppenderFromConn(ctx context.Context, driverConn driver.Conn, catalog, schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn) (*QuackMergeAppender, error) {
	return newQuackMergeAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, cfg, columns)
}

func newQuackMergeAppenderFromDriverConn(ctx context.Context, driverConn any, catalog, schema, table string, cfg QuackMergeConfig, columns []QuackMergeColumn) (*QuackMergeAppender, error) {
	sess, err := resolveQuackSessionForAppender(ctx, driverConn, catalog, "quack merge appender")
	if err != nil {
		return nil, err
	}
	return &QuackMergeAppender{session: sess, schema: schema, table: table, cfg: cfg, columns: columns}, nil
}

// AppendRow buffers one row. len(vals) must equal len(columns) from
// NewQuackMergeAppender, in the same order. No I/O happens here — call
// Flush or Close to actually run the MERGE.
func (a *QuackMergeAppender) AppendRow(vals ...any) error {
	if len(vals) != len(a.columns) {
		return fmt.Errorf("duckdb: quack merge appender: got %d values, expected %d columns", len(vals), len(a.columns))
	}
	a.rows = append(a.rows, vals)
	return nil
}

// Flush runs the MERGE with every buffered row as one SEND_DATA_REQUEST
// batch and clears the buffer. A no-op if nothing is buffered.
func (a *QuackMergeAppender) Flush(ctx context.Context) error {
	if len(a.rows) == 0 {
		return nil
	}
	if err := a.session.Merge(ctx, a.schema, a.table, a.cfg, a.columns, a.rows); err != nil {
		return err
	}
	a.rows = a.rows[:0]
	return nil
}

// Close flushes any remaining buffered rows, then — for a
// QuackMergeAppender from NewQuackMergeAppender — releases the pooled
// connection it has held open since construction back to db's pool. The
// appender must not be used afterward.
func (a *QuackMergeAppender) Close(ctx context.Context) error {
	flushErr := a.Flush(ctx)
	if a.pooledConn != nil {
		if closeErr := a.pooledConn.Close(); closeErr != nil && flushErr == nil {
			return closeErr
		}
	}
	return flushErr
}
