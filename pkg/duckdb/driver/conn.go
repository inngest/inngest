package driver

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/quack"
	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

// sqlExecer abstracts over what runs a SQL statement and collects its rows.
// The transports (*jsonlines.Session, *quack.Session) implement it directly;
// *process implements it on top of whichever one is active, adding crash
// detection, one-restart-then-permanently-disable recovery, and locking
// around the real subprocess, and *pooledQuackConn does the same for extra
// pooled connections. conn is deliberately transport-agnostic; conn_test.go
// exercises it against a stub.
//
// cols is the result's column names in the query's own left-to-right order.
// It may be nil for a statement that returns no columns (DDL) or, for the
// jsonlines transport specifically, no rows (see jsonlines.Session.Exec).
//
// Query is Exec's counterpart for conn.QueryContext: it additionally
// returns types, the result's DuckDB column type names in the same order as
// cols — see jsonlines.Session.Query and quack.Session.Query for how each
// transport gets there (a batched DESCRIBE for jsonlines;
// already-decoded PrepareResponse metadata, no extra statement at all, for
// quack).
type sqlExecer interface {
	Exec(ctx context.Context, sqlText string) (cols []string, rows []result.Row, err error)
	Query(ctx context.Context, sqlText string) (cols []string, types []string, rows []result.Row, err error)
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

// Close releases c's own session if it has one (a pooledQuackConn). The
// primary connection's *process is shared by every conn Connector.Connect
// hands out and outlives them, so it's only closed by Connector.Close.
func (c *conn) Close() error {
	if closer, ok := c.sess.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

func (c *conn) Begin() (driver.Tx, error) {
	return nil, errors.New("duckdb: transactions not supported in this POC")
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	sql, err := interpolate(query, args)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	_, _, err = c.sess.Exec(ctx, sql)
	logStatement(ctx, "exec", query, len(args), start, -1, err)
	if err != nil {
		return nil, err
	}
	return unknownResult{}, nil
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	sql, err := interpolate(query, args)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	var cols, types []string
	var rows []result.Row
	if streams := quackStreamsFrom(ctx); len(streams) > 0 {
		// The streams are fed on the same quack session the query runs on,
		// under the transport's own locking and restart policy.
		sq, ok := c.sess.(streamQuerier)
		if !ok {
			return nil, errQuackStreamsTransport
		}
		cols, types, rows, err = sq.QueryWithStreams(ctx, sql, toSendStreams(streams))
	} else {
		cols, types, rows, err = c.sess.Query(ctx, sql)
	}
	logStatement(ctx, "query", query, len(args), start, len(rows), err)
	if err != nil {
		return nil, err
	}
	return newMapRows(cols, types, rows), nil
}

// logStatement debug-logs one statement without its SQL text. The
// interpolated statement carries bound values — dual-write's batch INSERTs
// include event payloads and step output — so it is identified by a
// fingerprint of the un-interpolated query instead, which is stable across
// executions of the same statement shape. rows < 0 means "not applicable".
func logStatement(ctx context.Context, op, query string, args int, start time.Time, rows int, err error) {
	attrs := []any{
		"op", op,
		"fingerprint", queryFingerprint(query),
		"args", args,
		"duration", time.Since(start),
	}
	if rows >= 0 {
		attrs = append(attrs, "rows", rows)
	}
	if err != nil {
		attrs = append(attrs, "error", err)
	}
	logger.StdlibLogger(ctx).Debug("duckdb: statement", attrs...)
}

func queryFingerprint(query string) string {
	sum := sha256.Sum256([]byte(query))
	return hex.EncodeToString(sum[:6])
}

// errRowsAffectedUnsupported is returned by unknownResult: neither transport
// reports DuckDB's changed-row count, and fabricating 0 would claim a
// successful statement touched nothing.
var errRowsAffectedUnsupported = errors.New("duckdb: RowsAffected is not supported by this driver")

type unknownResult struct{}

func (unknownResult) LastInsertId() (int64, error) {
	return 0, errors.New("duckdb: LastInsertId is not supported by this driver")
}

func (unknownResult) RowsAffected() (int64, error) { return 0, errRowsAffectedUnsupported }

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
	case json.RawMessage:
		// Load-bearing, not just a style choice: json.RawMessage's
		// underlying kind is []byte, so returning driver.ErrSkip here (the
		// default for an unhandled case) would fall through to
		// database/sql's own DefaultParameterConverter, which converts any
		// named []byte-kind type down to a plain []byte via reflection —
		// silently discarding the json.RawMessage type, and with it
		// encodeLiteral's ::JSON cast, before this value ever reaches
		// interpolate (encodeLiteral's own []byte case encodes as a BLOB
		// literal instead, not what a JSON column needs — see
		// encodeLiteral's json.RawMessage case). Returning nil instead
		// tells database/sql to accept nv.Value exactly as given.
		return nil
	}

	return driver.ErrSkip
}

// SupportsTransactions reports whether c can run an explicit
// `BEGIN ... COMMIT` script and recover from a failure partway through it
// (by running `ROLLBACK` on the same connection afterwards), IE whether
// c's transport is quack. The jsonlines transport can't: a failed statement
// leaves the session's transaction aborted, so the trailing marker SELECT
// jsonlines uses to find the end of a statement's output fails too and the
// read never returns.
//
// c must be pinned (*sql.DB.Conn) for the answer to mean anything about the
// statements the caller runs next.
func SupportsTransactions(c *sql.Conn) bool {
	var ok bool
	_ = c.Raw(func(driverConn any) error {
		dc, isConn := driverConn.(*conn)
		if !isConn {
			return nil
		}
		switch s := dc.sess.(type) {
		case *quack.Session, *pooledQuackConn:
			ok = true
		case *process:
			s.mu.Lock()
			_, ok = s.sess.(*quack.Session)
			s.mu.Unlock()
		}
		return nil
	})
	return ok
}
