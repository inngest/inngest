package duckdb

import (
	"bytes"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

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
