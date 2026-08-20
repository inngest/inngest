package duckdb

import (
	"bytes"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnExecContextSendsSQL(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(`{"__marker__":"` + eofMarker + `"}` + "\n")
	c := &conn{sess: newSession(&written, fakeStdout)}

	_, err := c.ExecContext(t.Context(), "INSERT INTO t VALUES (1);", nil)
	require.NoError(t, err)
	require.Contains(t, written.String(), "INSERT INTO t VALUES (1);")
}

func TestConnQueryContextReturnsRows(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(
		`{"id":1}` + "\n" + `{"__marker__":"` + eofMarker + `"}` + "\n",
	)
	c := &conn{sess: newSession(&written, fakeStdout)}

	rows, err := c.QueryContext(t.Context(), "SELECT id FROM t;", nil)
	require.NoError(t, err)

	dest := make([]driver.Value, 1)
	require.NoError(t, rows.Next(dest))
	require.Equal(t, float64(1), dest[0])
}

// TestConnQueryContextColumnsMatchQueryOrder proves the driver.Rows a caller
// gets back from database/sql reports Columns() in the query's own order,
// not whatever order newMapRows happened to derive from map iteration.
// "zebra" before "apple" sorts the wrong way alphabetically.
func TestConnQueryContextColumnsMatchQueryOrder(t *testing.T) {
	var written bytes.Buffer
	fakeStdout := strings.NewReader(
		`{"zebra":1,"apple":2}` + "\n" + `{"__marker__":"` + eofMarker + `"}` + "\n",
	)
	c := &conn{sess: newSession(&written, fakeStdout)}

	rows, err := c.QueryContext(t.Context(), "SELECT zebra, apple FROM t;", nil)
	require.NoError(t, err)
	require.Equal(t, []string{"zebra", "apple"}, rows.Columns())
}
