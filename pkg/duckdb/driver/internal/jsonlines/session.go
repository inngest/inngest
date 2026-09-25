// Package jsonlines speaks the duckdb CLI's -jsonlines framing over its
// stdin and merged stdout+stderr: each statement is followed by a canary
// query whose marker row ends that statement's output, and non-JSON lines are
// attributed to the in-flight statement as diagnostics.
//
// The parent driver package uses a Session as the control channel for every
// freshly spawned subprocess (and as the data-plane transport when quack is
// off), and owns the subprocess itself.
package jsonlines

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
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

// Session serializes SQL text over stdin and parses the subprocess's merged
// stdout+stderr stream (see process.spawnLocked for why the two are one
// pipe). It assumes a single caller at a time (the POC's SetMaxOpenConns(1)
// constraint, enforced by process's own mutex); mu here only guards against a
// second concurrent exec interleaving statements on the wire.
type Session struct {
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

	// Redact, when set, scrubs secrets out of every diagnostic line before
	// it is logged or returned (the driver passes its secret redactor:
	// bootstrap statements carry tokens inline, and DuckDB echoes a failing
	// statement in some diagnostics).
	Redact func(string) string
}

func NewSession(stdin io.Writer, out io.Reader) *Session {
	s := &Session{
		stdin: stdin,
		lines: make(chan []byte, 64),
		done:  make(chan struct{}),
	}
	go s.readLoop(out)
	return s
}

// Close retires the session's reader goroutine. Safe to call more than once,
// and safe to call concurrently with an in-flight exec.
func (s *Session) Close() {
	s.closeOnce.Do(func() { close(s.done) })
}

func (s *Session) readLoop(out io.Reader) {
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

// Exec sends sql followed by a canary query, and returns every JSON row read
// before the canary's marker line, alongside cols: the result's column names
// in the query's own left-to-right order, as found on the first data row's
// own key order. Because -jsonlines mode prints nothing for a zero-row
// result besides the marker, cols is nil whenever sql produced no rows —
// this transport has no other way to learn the column list.
//
// sql need not end in a semicolon — Exec appends one if missing. Without it,
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
// unusable; process.Exec responds by respawning the subprocess with a fresh
// session.
func (s *Session) Exec(ctx context.Context, sql string) (cols []string, rows []result.Row, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.desynced {
		return nil, nil, result.ErrSessionDesynced
	}

	combined := fmt.Sprintf("%s\nSELECT '%s' AS __marker__;\n", normalizeStatement(sql), eofMarker)
	if _, err := io.WriteString(s.stdin, combined); err != nil {
		return nil, nil, fmt.Errorf("writing to duckdb subprocess: %w", err)
	}

	cols, rows, diags, err := s.readSegment(ctx, eofMarker)
	if err != nil {
		return nil, nil, err
	}
	if err := reportDiagnostics(ctx, s.Redact, diags); err != nil {
		return nil, nil, err
	}
	return cols, rows, nil
}

// Query is exec's counterpart for callers that also need the result's
// column types (conn.go's QueryContext) — see docs/plans/010's Execute
// column-metadata section and pkg/duckdb/insights/execute.go, which this
// exists to let stop running its own separate "DESCRIBE <sql>" Query.
//
// -jsonlines gives no way to learn a column's real DuckDB type from its own
// output: encoding/json only tells you the value's JSON shape (string,
// number, bool, ...), which can't distinguish INTEGER from BIGINT, DATE
// from a TIMESTAMP that happens to serialize as a string, DECIMAL from
// DOUBLE, and so on — and prints nothing at all for a zero-row result, so
// there's no data row to sniff a column list from either way (see exec's
// own doc comment). A "DESCRIBE <sql>" statement is the only way this
// transport can learn a Query's schema, so Query still has to run one — but
// as a single stdin write/read round trip together with the real Query
// (two statements, two interior markers: describeMarker then eofMarker)
// rather than two separate exec() calls, each its own mutex-guarded write
// and read. cols/types are always derived from DESCRIBE's own rows, never
// from the real Query's, since DESCRIBE's rows are the only ones guaranteed
// present regardless of how many rows the real Query actually matches.
func (s *Session) Query(ctx context.Context, sql string) (cols []string, types []string, rows []result.Row, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.desynced {
		return nil, nil, nil, result.ErrSessionDesynced
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
	if err := reportDiagnostics(ctx, s.Redact, append(describeDiags, dataDiags...)); err != nil {
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
func (s *Session) readSegment(ctx context.Context, marker string) (cols []string, rows []result.Row, diags []string, err error) {
	var shared *result.Columns
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				if s.scanErr != nil {
					return nil, nil, diags, fmt.Errorf("reading duckdb subprocess output: %w", s.scanErr)
				}
				return nil, nil, diags, io.ErrUnexpectedEOF
			}
			line = stripANSISGR(line)
			if len(line) == 0 {
				continue
			}

			lineCols, vals, derr := decodeOrderedRow(line)
			if derr != nil {
				diags = append(diags, string(line))
				continue
			}

			if len(lineCols) == 1 && lineCols[0] == "__marker__" && vals[0] == marker {
				return cols, rows, diags, nil
			}

			// Every row of one statement has the same columns, so the
			// first row's are shared by the rest.
			if shared == nil {
				cols = lineCols
				shared = result.NewColumns(cols)
			}
			rows = append(rows, result.Row{Cols: shared, Vals: vals})
		case <-ctx.Done():
			s.desynced = true
			return nil, nil, diags, fmt.Errorf("%w: %w", result.ErrSessionDesynced, ctx.Err())
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
func parseDescribeRows(rows []result.Row) (names []string, types []string, err error) {
	names = make([]string, len(rows))
	types = make([]string, len(rows))
	for i, r := range rows {
		name, ok := r.Get("column_name").(string)
		if !ok {
			return nil, nil, fmt.Errorf("duckdb: DESCRIBE row missing column_name: %v", r.Vals)
		}
		dbType, ok := r.Get("column_type").(string)
		if !ok {
			return nil, nil, fmt.Errorf("duckdb: DESCRIBE row missing column_type: %v", r.Vals)
		}
		names[i] = name
		types[i] = dbType
	}
	return names, types, nil
}

// stripANSISGR removes ANSI "\x1b[<params>m" color/reset sequences from
// line. DuckDB's CLI colors warning/error text it writes into the merged
// stdout+stderr stream (see spawnLocked), and when a colored diagnostic —
// e.g. the "extension_directory is deprecated" warning — is the last thing
// printed before a canary marker row, its trailing reset code lands on the
// same output line as the marker's JSON with no separating newline. Left in
// place, that prefix makes decodeOrderedRow fail to parse an otherwise valid
// marker row, so readSegment silently files it under diags and blocks
// forever waiting for a marker that already went by (verified against the
// real binary: this is what was hanging `SET extension_directory=...`).
func stripANSISGR(line []byte) []byte {
	if !bytes.ContainsRune(line, 0x1b) {
		return line
	}
	out := make([]byte, 0, len(line))
	for i := 0; i < len(line); i++ {
		if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
			j := i + 2
			for j < len(line) && line[j] != 'm' {
				j++
			}
			if j < len(line) {
				i = j
				continue
			}
		}
		out = append(out, line[i])
	}
	return out
}

// decodeOrderedRow parses one -jsonlines output line into cols, its
// top-level keys in on-the-wire order, and vals, the matching values.
// encoding/json's map-decoding path loses key order (and collapses repeated
// keys, which the CLI emits for a query that repeats a column name), so this
// walks the token stream instead — the JSON text itself still has the
// query's column order, since that's how the DuckDB CLI writes each row.
func decodeOrderedRow(line []byte) (cols []string, vals []any, err error) {
	dec := json.NewDecoder(bytes.NewReader(line))
	tok, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, nil, fmt.Errorf("duckdb: expected a JSON object, got %v", tok)
	}

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
		vals = append(vals, val)
	}
	if _, err := dec.Token(); err != nil { // closing '}'
		return nil, nil, err
	}
	return cols, vals, nil
}

// reportDiagnostics logs every non-JSON line the subprocess emitted while the
// statement was in flight — preserving the "stderr routed to the main
// process's logger" behaviour the previous dedicated stderr goroutine
// provided — and returns result.ErrStatementFailed if any of them is DuckDB error
// output. Every line is passed through redact first, when set.
func reportDiagnostics(ctx context.Context, redact func(string) string, diags []string) error {
	if len(diags) == 0 {
		return nil
	}

	l := logger.StdlibLogger(ctx)
	var errLines []string
	for _, line := range diags {
		if redact != nil {
			line = redact(line)
		}
		l.Warn("duckdb subprocess stderr", "line", line)
		if isErrorDiagnostic(line) {
			errLines = append(errLines, line)
		}
	}
	if len(errLines) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", result.ErrStatementFailed, strings.Join(errLines, "; "))
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
