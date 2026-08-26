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

// exec sends sql (which must end in a semicolon) followed by a canary query,
// and returns every JSON row read before the canary's marker line, alongside
// cols: the result's column names in the query's own left-to-right order, as
// found on the first data row's own key order. Because -jsonlines mode prints
// nothing for a zero-row result besides the marker, cols is nil whenever sql
// produced no rows — this transport has no other way to learn the column
// list.
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

	combined := fmt.Sprintf("%s\nSELECT '%s' AS __marker__;\n", sql, eofMarker)
	if _, err := io.WriteString(s.stdin, combined); err != nil {
		return nil, nil, fmt.Errorf("writing to duckdb subprocess: %w", err)
	}

	var diags []string
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				if s.scanErr != nil {
					return nil, nil, fmt.Errorf("reading duckdb subprocess output: %w", s.scanErr)
				}
				return nil, nil, io.ErrUnexpectedEOF
			}
			if len(line) == 0 {
				continue
			}

			rowCols, row, derr := decodeOrderedRow(line)
			if derr != nil {
				diags = append(diags, string(line))
				continue
			}

			if marker, ok := row["__marker__"].(string); ok && marker == eofMarker {
				if err := reportDiagnostics(ctx, diags); err != nil {
					return nil, nil, err
				}
				return cols, rows, nil
			}

			if cols == nil {
				cols = rowCols
			}
			rows = append(rows, row)
		case <-ctx.Done():
			s.desynced = true
			return nil, nil, fmt.Errorf("%w: %w", errSessionDesynced, ctx.Err())
		}
	}
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

// mapRows adapts a []map[string]any into database/sql/driver.Rows.
type mapRows struct {
	cols []string
	rows []map[string]any
	pos  int
}

func newMapRows(cols []string, rows []map[string]any) *mapRows {
	return &mapRows{cols: cols, rows: rows}
}

func (r *mapRows) Columns() []string { return r.cols }
func (r *mapRows) Close() error      { return nil }

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
