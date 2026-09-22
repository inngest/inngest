package driver

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

// sqlExecer abstracts over what runs a SQL statement and collects its JSON
// rows. *session (rows.go) implements it directly — conn_test.go exercises
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
//
// query is exec's counterpart for conn.QueryContext: it additionally
// returns types, the result's DuckDB column type names in the same order as
// cols — see rows.go's session.query and quack_session.go's quackSession.query
// for how each transport gets there (a batched DESCRIBE for jsonlines;
// already-decoded PrepareResponse metadata, no extra statement at all, for
// quack).
type sqlExecer interface {
	exec(ctx context.Context, sqlText string) (cols []string, rows []map[string]any, err error)
	query(ctx context.Context, sqlText string) (cols []string, types []string, rows []map[string]any, err error)
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
	start := time.Now()
	_, _, err = c.sess.exec(ctx, sql)
	logStatement(ctx, "exec", query, len(args), start, -1, err)
	if err != nil {
		return nil, err
	}
	return driver.RowsAffected(0), nil
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	sql, err := interpolate(query, args)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	cols, types, rows, err := c.sess.query(ctx, sql)
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
		// literal instead, not what a VARIANT column needs — see
		// encodeLiteral's json.RawMessage case). Returning nil instead
		// tells database/sql to accept nv.Value exactly as given.
		return nil
	}

	return driver.ErrSkip
}
