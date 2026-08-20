package duckdb

import (
	"bufio"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// eofMarker is appended as a canary query after every real statement sent to
// the duckdb subprocess. -jsonlines mode has no explicit "end of statement"
// marker, so we read stdout lines until we see this value, treating
// everything read before it as the statement's result.
const eofMarker = "__inngest_duckdb_eof__"

// session serializes SQL text over stdin and parses newline-delimited JSON
// results from stdout. It assumes a single caller at a time (the POC's
// SetMaxOpenConns(1) constraint) — no internal locking beyond what's needed
// to make Close safe to call concurrently with an in-flight exec.
type session struct {
	mu     sync.Mutex
	stdin  io.Writer
	stdout *bufio.Scanner
}

func newSession(stdin io.Writer, stdout io.Reader) *session {
	return &session{
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
	}
}

// exec sends sql (which must end in a semicolon) followed by a canary query,
// and returns every JSON row read before the canary's marker line.
func (s *session) exec(ctx context.Context, sql string) ([]map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	combined := fmt.Sprintf("%s\nSELECT '%s' AS __marker__;\n", sql, eofMarker)
	if _, err := io.WriteString(s.stdin, combined); err != nil {
		return nil, fmt.Errorf("writing to duckdb subprocess: %w", err)
	}

	var rows []map[string]any
	for s.stdout.Scan() {
		line := s.stdout.Bytes()
		if len(line) == 0 {
			continue
		}

		var row map[string]any
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, fmt.Errorf("parsing duckdb jsonlines output %q: %w", line, err)
		}

		if marker, ok := row["__marker__"].(string); ok && marker == eofMarker {
			return rows, nil
		}

		rows = append(rows, row)
	}
	if err := s.stdout.Err(); err != nil {
		return nil, fmt.Errorf("reading duckdb subprocess output: %w", err)
	}
	return nil, io.ErrUnexpectedEOF
}

// mapRows adapts a []map[string]any into database/sql/driver.Rows.
type mapRows struct {
	cols []string
	rows []map[string]any
	pos  int
}

func newMapRows(rows []map[string]any) *mapRows {
	var cols []string
	if len(rows) > 0 {
		for k := range rows[0] {
			cols = append(cols, k)
		}
	}
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
