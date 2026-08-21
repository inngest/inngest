package duckdb

import (
	"bytes"
	"context"
	"database/sql/driver"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSessionExecReadsUntilEOFMarker(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(
		`{"id":1,"name":"a"}` + "\n" +
			`{"id":2,"name":"b"}` + "\n" +
			`{"__marker__":"` + eofMarker + `"}` + "\n",
	)

	s := newSession(&written, fakeStdout)

	rows, err := s.exec(t.Context(), "SELECT id, name FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, float64(1), rows[0]["id"])
	require.Equal(t, "a", rows[0]["name"])

	require.Contains(t, written.String(), "SELECT id, name FROM t;")
	require.Contains(t, written.String(), eofMarker)
}

func TestSessionExecReturnsEmptyForNoRows(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(`{"__marker__":"` + eofMarker + `"}` + "\n")

	s := newSession(&written, fakeStdout)

	rows, err := s.exec(t.Context(), "CREATE TABLE t (id INTEGER);")
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestMapRowsColumns(t *testing.T) {
	r := newMapRows([]map[string]any{{"id": float64(1)}})
	require.Equal(t, []string{"id"}, r.Columns())
}

func TestMapRowsNextIteratesAllRowsThenEOF(t *testing.T) {
	r := newMapRows([]map[string]any{
		{"id": float64(1)},
		{"id": float64(2)},
		{"id": float64(3)},
	})
	require.Equal(t, []string{"id"}, r.Columns())

	dest := make([]driver.Value, 1)

	require.NoError(t, r.Next(dest))
	require.Equal(t, float64(1), dest[0])

	require.NoError(t, r.Next(dest))
	require.Equal(t, float64(2), dest[0])

	require.NoError(t, r.Next(dest))
	require.Equal(t, float64(3), dest[0])

	require.ErrorIs(t, r.Next(dest), io.EOF)
	// A further call must keep returning EOF, not panic or wrap around.
	require.ErrorIs(t, r.Next(dest), io.EOF)
}

func TestMapRowsNextOnEmptyResultReturnsEOFImmediately(t *testing.T) {
	r := newMapRows(nil)
	require.Empty(t, r.Columns())

	dest := make([]driver.Value, 0)
	require.ErrorIs(t, r.Next(dest), io.EOF)
}

// TestSessionExecReportsErrorDiagnosticBeforeMarker is the fake-transport
// counterpart to process_test.go's real-subprocess constraint tests: DuckDB
// error output arrives on the merged stream ahead of the marker line, and
// exec must fail the statement instead of returning the (partial) rows it
// managed to read.
func TestSessionExecReportsErrorDiagnosticBeforeMarker(t *testing.T) {
	var written bytes.Buffer
	fakeOut := strings.NewReader(
		"Constraint Error: NOT NULL constraint failed: t.id\n" +
			`{"__marker__":"` + eofMarker + `"}` + "\n",
	)

	s := newSession(&written, fakeOut)

	rows, err := s.exec(t.Context(), "INSERT INTO t VALUES (NULL);")
	require.ErrorIs(t, err, errStatementFailed)
	require.Contains(t, err.Error(), "NOT NULL constraint failed")
	require.Nil(t, rows)
}

// TestSessionExecIgnoresNonErrorDiagnostics keeps ordinary subprocess chatter
// (anything on the merged stream that isn't a JSON row and isn't DuckDB error
// output) from failing a statement that actually succeeded.
func TestSessionExecIgnoresNonErrorDiagnostics(t *testing.T) {
	var written bytes.Buffer
	fakeOut := strings.NewReader(
		"some incidental subprocess chatter\n" +
			`{"id":1}` + "\n" +
			`{"__marker__":"` + eofMarker + `"}` + "\n",
	)

	s := newSession(&written, fakeOut)

	rows, err := s.exec(t.Context(), "SELECT id FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

// TestSessionExecReturnsWhenContextCancelled proves exec honours its ctx
// instead of blocking forever on a subprocess that never answers, and that it
// refuses to reuse a session whose framing it abandoned.
func TestSessionExecReturnsWhenContextCancelled(t *testing.T) {
	var written bytes.Buffer
	// A reader that never yields a line and never EOFs, i.e. a hung
	// subprocess.
	never, _ := io.Pipe()

	s := newSession(&written, never)
	t.Cleanup(s.close)

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	done := make(chan error, 1)
	go func() { _, err := s.exec(ctx, "SELECT 1;"); done <- err }()

	select {
	case err := <-done:
		require.ErrorIs(t, err, errSessionDesynced)
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("exec ignored its context and blocked on the read")
	}

	_, err := s.exec(t.Context(), "SELECT 1;")
	require.ErrorIs(t, err, errSessionDesynced, "a desynced session must refuse further statements")
}
