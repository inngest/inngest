package duckdb

import (
	"bytes"
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
