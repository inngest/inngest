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

	_, rows, err := s.exec(t.Context(), "SELECT id, name FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, float64(1), rows[0]["id"])
	require.Equal(t, "a", rows[0]["name"])

	require.Contains(t, written.String(), "SELECT id, name FROM t;")
	require.Contains(t, written.String(), eofMarker)
}

// TestSessionExecPreservesColumnOrder pins jsonlines' column order to the
// query's own left-to-right order, not the incidental (random) order Go map
// iteration would otherwise produce. "zebra" before "apple" would sort the
// wrong way alphabetically, so this fails if cols is ever derived by ranging
// over the decoded row map instead of the JSON object's own key order.
func TestSessionExecPreservesColumnOrder(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(
		`{"zebra":1,"apple":2}` + "\n" +
			`{"__marker__":"` + eofMarker + `"}` + "\n",
	)

	s := newSession(&written, fakeStdout)

	cols, rows, err := s.exec(t.Context(), "SELECT zebra, apple FROM t;")
	require.NoError(t, err)
	require.Equal(t, []string{"zebra", "apple"}, cols)
	require.Len(t, rows, 1)
}

func TestSessionExecReturnsEmptyForNoRows(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(`{"__marker__":"` + eofMarker + `"}` + "\n")

	s := newSession(&written, fakeStdout)

	_, rows, err := s.exec(t.Context(), "CREATE TABLE t (id INTEGER);")
	require.NoError(t, err)
	require.Empty(t, rows)
}

// TestSessionQueryBatchesDescribeAndDataInOneRoundTrip proves query() writes
// a DESCRIBE statement and the real query together in a single stdin write
// (not two separate exec() round trips), and correctly splits the merged
// jsonlines output back into DESCRIBE's own rows (turned into cols/types)
// and the real query's own rows.
func TestSessionQueryBatchesDescribeAndDataInOneRoundTrip(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(
		`{"column_name":"id","column_type":"INTEGER","null":"YES","key":null,"default":null,"extra":null}` + "\n" +
			`{"column_name":"name","column_type":"VARCHAR","null":"YES","key":null,"default":null,"extra":null}` + "\n" +
			`{"__marker__":"` + describeMarker + `"}` + "\n" +
			`{"id":1,"name":"a"}` + "\n" +
			`{"__marker__":"` + eofMarker + `"}` + "\n",
	)

	s := newSession(&written, fakeStdout)

	cols, types, rows, err := s.query(t.Context(), "SELECT id, name FROM t;")
	require.NoError(t, err)
	require.Equal(t, []string{"id", "name"}, cols)
	require.Equal(t, []string{"INTEGER", "VARCHAR"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, float64(1), rows[0]["id"])

	// Both the DESCRIBE statement and the real query, plus both interior
	// canaries, went out together — not as two separate exec()-style round
	// trips (which would still produce this same text, just written and
	// read in two separate calls instead of one).
	require.Equal(t, 1, strings.Count(written.String(), "DESCRIBE SELECT id, name FROM t;"))
	require.Equal(t, 1, strings.Count(written.String(), describeMarker))
	require.Equal(t, 1, strings.Count(written.String(), eofMarker))
}

// TestSessionQueryReturnsColumnsEvenWhenDataMatchesZeroRows is the whole
// point of routing column metadata through DESCRIBE rather than the real
// query's own output: -jsonlines prints nothing at all for a zero-row
// result, so cols/types can only come from DESCRIBE's own rows, never from
// dataCols.
func TestSessionQueryReturnsColumnsEvenWhenDataMatchesZeroRows(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(
		`{"column_name":"id","column_type":"BIGINT","null":"YES","key":null,"default":null,"extra":null}` + "\n" +
			`{"__marker__":"` + describeMarker + `"}` + "\n" +
			`{"__marker__":"` + eofMarker + `"}` + "\n",
	)

	s := newSession(&written, fakeStdout)

	cols, types, rows, err := s.query(t.Context(), "SELECT id FROM t WHERE false;")
	require.NoError(t, err)
	require.Equal(t, []string{"id"}, cols)
	require.Equal(t, []string{"BIGINT"}, types)
	require.Empty(t, rows)
}

// TestSessionQueryReportsErrorDiagnosticFromEitherSegment mirrors
// TestSessionExecReportsErrorDiagnosticBeforeMarker for the two-segment
// query() path: an error diagnostic anywhere in the merged stream (whether
// attributed to the DESCRIBE statement or the real query) fails the whole
// call.
func TestSessionQueryReportsErrorDiagnosticFromEitherSegment(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(
		"Binder Error: Table with name t does not exist\n" +
			`{"__marker__":"` + describeMarker + `"}` + "\n" +
			`{"__marker__":"` + eofMarker + `"}` + "\n",
	)

	s := newSession(&written, fakeStdout)

	cols, types, rows, err := s.query(t.Context(), "SELECT id FROM t;")
	require.ErrorIs(t, err, errStatementFailed)
	require.Contains(t, err.Error(), "does not exist")
	require.Nil(t, cols)
	require.Nil(t, types)
	require.Nil(t, rows)
}

func TestMapRowsColumns(t *testing.T) {
	r := newMapRows([]string{"id"}, nil, []map[string]any{{"id": float64(1)}})
	require.Equal(t, []string{"id"}, r.Columns())
}

func TestMapRowsReportsColumnTypeDatabaseTypeName(t *testing.T) {
	r := newMapRows([]string{"id", "name"}, []string{"BIGINT", "VARCHAR"}, nil)
	require.Equal(t, "BIGINT", r.ColumnTypeDatabaseTypeName(0))
	require.Equal(t, "VARCHAR", r.ColumnTypeDatabaseTypeName(1))
}

func TestMapRowsNextIteratesAllRowsThenEOF(t *testing.T) {
	r := newMapRows([]string{"id"}, nil, []map[string]any{
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
	r := newMapRows(nil, nil, nil)
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

	_, rows, err := s.exec(t.Context(), "INSERT INTO t VALUES (NULL);")
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

	_, rows, err := s.exec(t.Context(), "SELECT id FROM t;")
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
	go func() { _, _, err := s.exec(ctx, "SELECT 1;"); done <- err }()

	select {
	case err := <-done:
		require.ErrorIs(t, err, errSessionDesynced)
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("exec ignored its context and blocked on the read")
	}

	_, _, err := s.exec(t.Context(), "SELECT 1;")
	require.ErrorIs(t, err, errSessionDesynced, "a desynced session must refuse further statements")
}
