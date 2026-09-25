package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/quack"
)

// QuackColumnKind identifies one column's physical wire type for
// QuackAppender.AppendRow — see quack.ColumnKind for each kind's accepted
// Go values.
type QuackColumnKind = quack.ColumnKind

const (
	QuackColumnUUID        = quack.ColumnUUID
	QuackColumnVarchar     = quack.ColumnVarchar
	QuackColumnJSON        = quack.ColumnJSON
	QuackColumnTimestampMS = quack.ColumnTimestampMS
)

// QuackAppender bulk-loads rows into one table over quack's
// SEND_DATA_REQUEST mechanism (see internal/quack/senddata.go). AppendRow buffers
// rows in memory; Flush sends everything buffered as one batch. Not safe for
// concurrent use (one goroutine at a time per *QuackAppender — see
// NewQuackAppender's doc comment for why concurrent *appenders*, each on its
// own instance, are fine).
type QuackAppender struct {
	session *quack.Session
	schema  string
	table   string
	columns []QuackColumnKind
	rows    [][]any

	// pooledConn is non-nil only when this appender came from
	// NewQuackAppender (not NewQuackAppenderFromConn), which acquired it
	// from db's pool specifically to hold open — see NewQuackAppender's doc
	// comment for why releasing it any earlier than Close is unsafe. Close
	// releases it back to the pool.
	pooledConn *sql.Conn
}

// NewQuackAppender returns a QuackAppender for catalog.schema.table, reading
// db's underlying quack session directly (via *sql.Conn.Raw — the type
// assertion below rejects a jsonlines-only connection). db must have been
// opened with Options.QuackAddr set; otherwise this returns an error rather
// than silently falling back to a slower transport.
//
// The acquired *sql.Conn is held open for the returned QuackAppender's
// entire lifetime — released back to db's pool only once Close runs, not
// before. This one connection is safe for concurrent, independent
// NewQuackAppender calls (each gets its own): earlier, this acquired a
// connection just long enough to extract its underlying quack.Session, then
// released it back to the pool immediately, while the appender kept using
// that same session for the rest of its life. That let a second concurrent
// NewQuackAppender call be handed the exact same connection while the first
// appender's own stream was still open on it — the second appender's first
// PrepareRequest then supersedes the first's in-flight one server-side
// ("duckdb: statement failed: superseded by a new query"), corrupting or
// losing whichever appender lost the race. Confirmed via
// TestQuackAppenderConcurrentPooledUseDoesNotSuperseded and, before this
// fix, a real cmd/duckdbbench benchmark run.
//
// The wire protocol has no catalog field: it resolves schema.table against
// the target connection's own default catalog, which is per-connection, not
// shared with the bootstrapping CLI session. So when catalog is non-empty,
// NewQuackAppender issues "USE <catalog>;" once on the resolved session
// before returning. This is a real, global mutation of that session's
// default catalog for every later statement — safe here only because every
// other caller already fully qualifies table names with DuckLakeAlias (see
// cmd/duckdbseed/insert.go).
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
// already owns outright — typically via Connector.Connect directly (see
// OpenConnector), rather than one *sql.DB.Conn hands out from its pool. The
// caller must Close driverConn itself once done with the appender (Close on
// the returned QuackAppender does not release it, since it never acquired
// it).
func NewQuackAppenderFromConn(ctx context.Context, driverConn driver.Conn, catalog, schema, table string, columns []QuackColumnKind) (*QuackAppender, error) {
	return newQuackAppenderFromDriverConn(ctx, driverConn, catalog, schema, table, columns)
}

// newQuackAppenderFromDriverConn holds the logic NewQuackAppender and
// NewQuackAppenderFromConn share: resolve driverConn to its underlying
// quack.Session, optionally switch catalog, construct the QuackAppender.
func newQuackAppenderFromDriverConn(ctx context.Context, driverConn any, catalog, schema, table string, columns []QuackColumnKind) (*QuackAppender, error) {
	sess, err := resolveQuackSessionForAppender(ctx, driverConn, catalog, "quack appender")
	if err != nil {
		return nil, err
	}
	return &QuackAppender{session: sess, schema: schema, table: table, columns: columns}, nil
}

// resolveQuackSessionForAppender resolves driverConn to its underlying
// quack.Session and, if catalog is non-empty, switches that session's default
// catalog — the logic every *Appender constructor (QuackAppender,
// QuackMergeAppender) shares. errPrefix names the caller in error messages
// (e.g. "quack appender", "quack merge appender").
//
// The wire protocol has no catalog field: it resolves schema.table against
// the target connection's own default catalog, which is per-connection, not
// shared with the bootstrapping CLI session. So when catalog is non-empty,
// this issues "USE <catalog>;" once on the resolved session before
// returning. This is a real, global mutation of that session's default
// catalog for every later statement — safe here only because every other
// caller already fully qualifies table names with DuckLakeAlias (see
// cmd/duckdbseed/insert.go).
func resolveQuackSessionForAppender(ctx context.Context, driverConn any, catalog, errPrefix string) (*quack.Session, error) {
	c, ok := driverConn.(*conn)
	if !ok {
		return nil, fmt.Errorf("duckdb: %s: unexpected driver connection type %T", errPrefix, driverConn)
	}

	// The primary connection's sess is *process (crash-restart handling
	// wraps the real transport); an extra connection from Options.QuackConns
	// > 1 is a *pooledQuackConn. Either way the appender drives the current
	// *quack.Session directly, so a crash mid-append surfaces as a plain
	// error rather than a transparent retry.
	var sess *quack.Session
	switch s := c.sess.(type) {
	case *quack.Session:
		sess = s
	case *pooledQuackConn:
		var err error
		sess, err = s.currentSession(ctx)
		if err != nil {
			return nil, err
		}
	case *process:
		var err error
		sess, err = s.currentQuackSession()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("duckdb: %s: unexpected transport %T", errPrefix, c.sess)
	}

	if catalog != "" {
		if _, _, err := sess.Exec(ctx, fmt.Sprintf("USE %s;", quack.Identifier(catalog))); err != nil {
			return nil, fmt.Errorf("duckdb: %s: switching to catalog %q: %w", errPrefix, catalog, err)
		}
	}

	return sess, nil
}

// AppendRow buffers one row. len(vals) must equal len(columns) from
// NewQuackAppender, in the same order. No I/O happens here — call Flush or
// Close to actually send buffered rows.
func (a *QuackAppender) AppendRow(vals ...any) error {
	if len(vals) != len(a.columns) {
		return fmt.Errorf("duckdb: quack appender: got %d values, expected %d columns", len(vals), len(a.columns))
	}
	a.rows = append(a.rows, vals)
	return nil
}

// Flush sends every buffered row as one SEND_DATA_REQUEST batch and clears
// the buffer. A no-op if nothing is buffered.
func (a *QuackAppender) Flush(ctx context.Context) error {
	if len(a.rows) == 0 {
		return nil
	}
	if err := a.session.Append(ctx, a.schema, a.table, a.columns, a.rows); err != nil {
		return err
	}
	a.rows = a.rows[:0]
	return nil
}

// Close flushes any remaining buffered rows, then — for a QuackAppender
// from NewQuackAppender — releases the pooled connection it has held open
// since construction back to db's pool. The appender must not be used
// afterward.
func (a *QuackAppender) Close(ctx context.Context) error {
	flushErr := a.Flush(ctx)
	if a.pooledConn != nil {
		if closeErr := a.pooledConn.Close(); closeErr != nil && flushErr == nil {
			return closeErr
		}
	}
	return flushErr
}
