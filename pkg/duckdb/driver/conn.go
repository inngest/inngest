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

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/inngest/inngest/pkg/duckdb/worker"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

// sqlExecer abstracts over what runs a SQL statement and collects its rows.
// *process implements it on top of the primary worker session (adding crash
// detection, restart recovery, and locking), and *pooledQuackConn implements
// it for additional concurrent worker sessions. conn is transport-agnostic;
// conn_test.go exercises it against a stub.
type sqlExecer interface {
	Exec(ctx context.Context, sqlText string) (cols []string, rows []result.Row, err error)
	Query(ctx context.Context, sqlText string) (cols []string, types []string, rows []result.Row, err error)
}

// conn implements database/sql/driver.Conn, ExecerContext, and QueryerContext
// over a single DuckDB worker session. Parameter binding is handled by
// literal.go (encodeArgs).
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
	return nil, errors.New("duckdb: transactions not supported via Begin; use explicit BEGIN/COMMIT")
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
		return nil
	case meta.EventSessions:
		return nil
	case json.RawMessage:
		return nil
	}

	return driver.ErrSkip
}

// AtomicMultiTableCommit reports whether a transaction on c that writes
// several lake tables commits them all or none: SupportsTransactions, and a
// lake whose commits span tables (plain duckdb tables, not Iceberg).
func AtomicMultiTableCommit(c *sql.Conn) bool {
	if !SupportsTransactions(c) {
		return false
	}
	ok := true
	_ = c.Raw(func(driverConn any) error {
		if dc, isConn := driverConn.(*conn); isConn {
			if p := processOf(dc); p != nil {
				ok = p.atomicMultiTableCommit()
			}
		}
		return nil
	})
	return ok
}

func processOf(dc *conn) *process {
	switch s := dc.sess.(type) {
	case *process:
		return s
	case *pooledQuackConn:
		return s.p
	}
	return nil
}

func (p *process) atomicMultiTableCommit() bool { return p.iceberg == nil }

// SupportsTransactions reports whether c can run an explicit
// `BEGIN ... COMMIT` script and recover from a failure partway through it
// (by running `ROLLBACK` on the same connection afterwards). Every worker-backed
// connection supports transactions natively.
func SupportsTransactions(c *sql.Conn) bool {
	var ok bool
	_ = c.Raw(func(driverConn any) error {
		dc, isConn := driverConn.(*conn)
		if !isConn {
			return nil
		}
		switch dc.sess.(type) {
		case *workerSessionExecer, *pooledQuackConn, *process:
			ok = true
		}
		return nil
	})
	return ok
}

// Keep worker import referenced cleanly.
var _ = (*worker.Session)(nil)
