package duckdb

import (
	"bufio"
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/logger"
)

// eofMarker is appended as a canary query after every real statement sent to
// the duckdb subprocess. -jsonlines mode has no explicit "end of statement"
// marker, so we read output lines until we see this value, treating
// everything read before it as the statement's result.
const eofMarker = "__inngest_duckdb_eof__"

// describeMarker is query()'s interior canary: it separates a batched
// DESCRIBE statement's own output from the real query's output within the
// single stdin write/read round trip query() uses — see query's doc comment.
const describeMarker = "__inngest_duckdb_describe_eof__"

// maxLineBytes bounds a single output line. bufio.Scanner's 64KiB default is
// too small for a wide SELECT row, and an over-long line would otherwise
// surface as a bare "token too long" scan error that looks like a dead
// subprocess.
const maxLineBytes = 4 << 20

// errStatementFailed marks an error DuckDB reported for the statement itself:
// a constraint violation, a type/conversion failure, or schema drift. The CLI
// writes those to stderr and *still* completes the eofMarker round trip
// (verified empirically), so before this existed the driver reported success
// for statements DuckDB had actually rejected — silent data loss for the most
// likely real failure mode of the dual-write path. The subprocess is healthy
// when this is returned, so process.exec must neither restart nor retry.
var errStatementFailed = errors.New("duckdb: statement failed")

// errSessionDesynced marks a session that abandoned a statement mid-read
// (the only way that happens is ctx cancellation inside exec). The
// subprocess's remaining output for that statement is still queued, so the
// session can no longer correlate output with statements and must be replaced
// by a fresh one — process.exec does that via restartLocked.
var errSessionDesynced = errors.New("duckdb: session abandoned an in-flight statement")

// session serializes SQL text over stdin and parses the subprocess's merged
// stdout+stderr stream (see process.spawnLocked for why the two are one
// pipe). It assumes a single caller at a time (the POC's SetMaxOpenConns(1)
// constraint, enforced by process's own mutex); mu here only guards against a
// second concurrent exec interleaving statements on the wire.
type session struct {
	mu    sync.Mutex
	stdin io.Writer

	// lines carries every output line from readLoop. Reading through a
	// channel rather than scanning inline is what makes exec's ctx
	// parameter meaningful: exec can select on ctx.Done() instead of
	// blocking indefinitely in a pipe read.
	lines chan []byte

	// scanErr is written by readLoop before it closes lines. The channel
	// close is the happens-before edge that makes it safe to read after a
	// receive observes the close, so it needs no additional synchronization.
	scanErr error

	// done retires the session: readLoop exits instead of blocking forever
	// trying to hand a line to an exec that will never come.
	done      chan struct{}
	closeOnce sync.Once

	// desynced is set by exec when it abandons an in-flight statement.
	// Guarded by mu.
	desynced bool
}

func newSession(stdin io.Writer, out io.Reader) *session {
	s := &session{
		stdin: stdin,
		lines: make(chan []byte, 64),
		done:  make(chan struct{}),
	}
	go s.readLoop(out)
	return s
}

// close retires the session's reader goroutine. Safe to call more than once,
// and safe to call concurrently with an in-flight exec.
func (s *session) close() {
	s.closeOnce.Do(func() { close(s.done) })
}

func (s *session) readLoop(out io.Reader) {
	defer close(s.lines)

	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	for sc.Scan() {
		// Scanner reuses its token buffer between Scan calls, so the line
		// has to be copied before it crosses the channel.
		line := make([]byte, len(sc.Bytes()))
		copy(line, sc.Bytes())

		select {
		case s.lines <- line:
		case <-s.done:
			return
		}
	}
	s.scanErr = sc.Err()
}

// exec sends sql followed by a canary query, and returns every JSON row read
// before the canary's marker line, alongside cols: the result's column names
// in the query's own left-to-right order, as found on the first data row's
// own key order. Because -jsonlines mode prints nothing for a zero-row
// result besides the marker, cols is nil whenever sql produced no rows —
// this transport has no other way to learn the column list.
//
// sql need not end in a semicolon — exec appends one if missing. Without it,
// the canary query written right after sql on the same stdin stream would
// merge into one statement with whatever sql it followed (DuckDB's CLI, in
// this non-interactive stdin mode, keeps buffering input until it sees a
// terminating ';' rather than erroring), hanging indefinitely rather than
// producing two separate results — confirmed against the real binary.
//
// Lines that are not parseable as JSON are the subprocess's stderr output,
// merged into the same stream (process.spawnLocked) precisely so they can be
// attributed to the statement in flight when they were written. They are
// logged, and if any of them is DuckDB error output the statement is reported
// as failed rather than silently succeeding.
//
// ctx cancels the read. Doing so abandons a statement whose output is still
// queued, which desyncs the protocol permanently, so the session marks itself
// unusable; process.exec responds by respawning the subprocess with a fresh
// session.
func (s *session) exec(ctx context.Context, sql string) (cols []string, rows []map[string]any, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.desynced {
		return nil, nil, errSessionDesynced
	}

	combined := fmt.Sprintf("%s\nSELECT '%s' AS __marker__;\n", normalizeStatement(sql), eofMarker)
	if _, err := io.WriteString(s.stdin, combined); err != nil {
		return nil, nil, fmt.Errorf("writing to duckdb subprocess: %w", err)
	}

	cols, rows, diags, err := s.readSegment(ctx, eofMarker)
	if err != nil {
		return nil, nil, err
	}
	if err := reportDiagnostics(ctx, diags); err != nil {
		return nil, nil, err
	}
	return cols, rows, nil
}

// query is exec's counterpart for callers that also need the result's
// column types (conn.go's QueryContext) — see docs/plans/010's Execute
// column-metadata section and pkg/duckdb/insights/execute.go, which this
// exists to let stop running its own separate "DESCRIBE <sql>" query.
//
// -jsonlines gives no way to learn a column's real DuckDB type from its own
// output: encoding/json only tells you the value's JSON shape (string,
// number, bool, ...), which can't distinguish INTEGER from BIGINT, DATE
// from a TIMESTAMP that happens to serialize as a string, DECIMAL from
// DOUBLE, and so on — and prints nothing at all for a zero-row result, so
// there's no data row to sniff a column list from either way (see exec's
// own doc comment). A "DESCRIBE <sql>" statement is the only way this
// transport can learn a query's schema, so query still has to run one — but
// as a single stdin write/read round trip together with the real query
// (two statements, two interior markers: describeMarker then eofMarker)
// rather than two separate exec() calls, each its own mutex-guarded write
// and read. cols/types are always derived from DESCRIBE's own rows, never
// from the real query's, since DESCRIBE's rows are the only ones guaranteed
// present regardless of how many rows the real query actually matches.
func (s *session) query(ctx context.Context, sql string) (cols []string, types []string, rows []map[string]any, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.desynced {
		return nil, nil, nil, errSessionDesynced
	}

	sql = normalizeStatement(sql)
	combined := fmt.Sprintf(
		"DESCRIBE %s\nSELECT '%s' AS __marker__;\n%s\nSELECT '%s' AS __marker__;\n",
		sql, describeMarker, sql, eofMarker,
	)
	if _, err := io.WriteString(s.stdin, combined); err != nil {
		return nil, nil, nil, fmt.Errorf("writing to duckdb subprocess: %w", err)
	}

	_, describeRows, describeDiags, err := s.readSegment(ctx, describeMarker)
	if err != nil {
		return nil, nil, nil, err
	}
	_, dataRows, dataDiags, err := s.readSegment(ctx, eofMarker)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := reportDiagnostics(ctx, append(describeDiags, dataDiags...)); err != nil {
		return nil, nil, nil, err
	}

	cols, types, err = parseDescribeRows(describeRows)
	if err != nil {
		return nil, nil, nil, err
	}
	return cols, types, dataRows, nil
}

// readSegment reads output lines until it observes a "__marker__" row equal
// to marker, returning that segment's own column order/rows (nil cols if
// the segment produced no rows before its marker — see exec's doc comment)
// plus every line that failed to parse as JSON along the way, unclassified
// into pass/fail (see reportDiagnostics). ctx cancellation desyncs the
// session exactly as exec's read loop always has: the subprocess's
// remaining output for the abandoned statement(s) is still queued, so the
// session can no longer correlate output with statements.
func (s *session) readSegment(ctx context.Context, marker string) (cols []string, rows []map[string]any, diags []string, err error) {
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				if s.scanErr != nil {
					return nil, nil, diags, fmt.Errorf("reading duckdb subprocess output: %w", s.scanErr)
				}
				return nil, nil, diags, io.ErrUnexpectedEOF
			}
			if len(line) == 0 {
				continue
			}

			rowCols, row, derr := decodeOrderedRow(line)
			if derr != nil {
				diags = append(diags, string(line))
				continue
			}

			if m, ok := row["__marker__"].(string); ok && m == marker {
				return cols, rows, diags, nil
			}

			if cols == nil {
				cols = rowCols
			}
			rows = append(rows, row)
		case <-ctx.Done():
			s.desynced = true
			return nil, nil, diags, fmt.Errorf("%w: %w", errSessionDesynced, ctx.Err())
		}
	}
}

// normalizeStatement trims trailing whitespace and appends a terminating
// ';' if missing. Every statement written to the subprocess's stdin needs
// this — without it, a canary query written right after sql on the same
// stdin stream would merge into one statement with whatever sql it
// followed (DuckDB's CLI, in this non-interactive stdin mode, keeps
// buffering input until it sees a terminating ';' rather than erroring),
// hanging indefinitely rather than producing a separate result — confirmed
// against the real binary.
func normalizeStatement(sql string) string {
	sql = strings.TrimRight(sql, " \t\n\r")
	if !strings.HasSuffix(sql, ";") {
		sql += ";"
	}
	return sql
}

// parseDescribeRows extracts the column name/type list from a DESCRIBE
// statement's own decoded rows, in DESCRIBE's own row order — the same
// order the described query itself projects columns in. column_name/
// column_type are DESCRIBE's first two output columns (followed by null/
// key/default/extra, none of which query() has a use for).
func parseDescribeRows(rows []map[string]any) (names []string, types []string, err error) {
	names = make([]string, len(rows))
	types = make([]string, len(rows))
	for i, row := range rows {
		name, ok := row["column_name"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("duckdb: DESCRIBE row missing column_name: %v", row)
		}
		dbType, ok := row["column_type"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("duckdb: DESCRIBE row missing column_type: %v", row)
		}
		names[i] = name
		types[i] = dbType
	}
	return names, types, nil
}

// decodeOrderedRow parses one -jsonlines output line into both a row map (for
// value lookup) and cols: its top-level keys in on-the-wire order.
// encoding/json's map-decoding path loses key order, so this walks the token
// stream instead — the JSON text itself still has the query's column order,
// since that's how the DuckDB CLI writes each row.
func decodeOrderedRow(line []byte) (cols []string, row map[string]any, err error) {
	dec := json.NewDecoder(bytes.NewReader(line))
	tok, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, nil, fmt.Errorf("duckdb: expected a JSON object, got %v", tok)
	}

	row = make(map[string]any)
	for dec.More() {
		keyTok, kerr := dec.Token()
		if kerr != nil {
			return nil, nil, kerr
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, nil, fmt.Errorf("duckdb: expected a string object key, got %v", keyTok)
		}

		var val any
		if verr := dec.Decode(&val); verr != nil {
			return nil, nil, verr
		}

		cols = append(cols, key)
		row[key] = val
	}
	if _, err := dec.Token(); err != nil { // closing '}'
		return nil, nil, err
	}
	return cols, row, nil
}

// reportDiagnostics logs every non-JSON line the subprocess emitted while the
// statement was in flight — preserving the "stderr routed to the main
// process's logger" behaviour the previous dedicated stderr goroutine
// provided — and returns errStatementFailed if any of them is DuckDB error
// output.
func reportDiagnostics(ctx context.Context, diags []string) error {
	if len(diags) == 0 {
		return nil
	}

	l := logger.StdlibLogger(ctx)
	var errLines []string
	for _, line := range diags {
		l.Warn("duckdb subprocess stderr", "line", line)
		if isErrorDiagnostic(line) {
			errLines = append(errLines, line)
		}
	}
	if len(errLines) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", errStatementFailed, strings.Join(errLines, "; "))
}

// isErrorDiagnostic reports whether a diagnostic line is DuckDB error output.
// Every error the CLI prints leads with a "<Kind> Error: " token — observed
// kinds include Constraint, Conversion, Catalog, Binder, Parser, and Invalid
// Input — followed by optional continuation lines ("LINE 1: ...", "^", "Did
// you mean ..."). Matching the shared "Error: " token covers all of them
// without enumerating DuckDB's error taxonomy, and only ever sees lines that
// already failed to parse as a JSON result row.
func isErrorDiagnostic(line string) bool {
	return strings.Contains(line, "Error: ")
}

// mapRows adapts a []map[string]any into database/sql/driver.Rows. types is
// nil for a result conn.go's ExecContext path produced (no caller ever asks
// ExecContext's driver.Result for column types); QueryContext always
// supplies it (see conn.go), which is what backs *sql.Rows.ColumnTypes()'s
// DatabaseTypeName() — see ColumnTypeDatabaseTypeName below.
type mapRows struct {
	cols  []string
	types []string
	rows  []map[string]any
	pos   int
}

func newMapRows(cols []string, types []string, rows []map[string]any) *mapRows {
	return &mapRows{cols: cols, types: types, rows: rows}
}

func (r *mapRows) Columns() []string { return r.cols }
func (r *mapRows) Close() error      { return nil }

// ColumnTypeDatabaseTypeName implements
// database/sql/driver.RowsColumnTypeDatabaseTypeName, which is what
// *sql.Rows.ColumnTypes()[i].DatabaseTypeName() calls through to. This is
// the mechanism that lets pkg/duckdb/insights.Execute learn a query's real
// DuckDB column types from the query it already ran, instead of a second
// "DESCRIBE <sql>" query of its own.
func (r *mapRows) ColumnTypeDatabaseTypeName(index int) string {
	if index < 0 || index >= len(r.types) {
		return ""
	}
	return r.types[index]
}

func (r *mapRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.rows) {
		return io.EOF
	}
	row := r.rows[r.pos]
	for i, col := range r.cols {
		dest[i] = row[col]
	}
	r.pos++
	return nil
}
