package driver

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/stretchr/testify/require"
)

// stubExecer is a transport-free sqlExecer: it records the SQL conn sends
// and returns canned results, so these tests exercise conn alone (the
// transports have their own tests in internal/quack and internal/jsonlines).
type stubExecer struct {
	sent  []string
	cols  []string
	types []string
	rows  []result.Row
}

func (s *stubExecer) Exec(_ context.Context, sqlText string) ([]string, []result.Row, error) {
	s.sent = append(s.sent, sqlText)
	return s.cols, s.rows, nil
}

func (s *stubExecer) Query(_ context.Context, sqlText string) ([]string, []string, []result.Row, error) {
	s.sent = append(s.sent, sqlText)
	return s.cols, s.types, s.rows, nil
}

func TestConnExecContextSendsInterpolatedSQL(t *testing.T) {
	stub := &stubExecer{}
	c := &conn{sess: stub}

	_, err := c.ExecContext(t.Context(), "INSERT INTO t VALUES (?);", []driver.NamedValue{{Ordinal: 1, Value: int64(1)}})
	require.NoError(t, err)
	require.Equal(t, []string{"INSERT INTO t VALUES (1);"}, stub.sent)
}

func TestConnQueryContextReturnsRows(t *testing.T) {
	cols := []string{"id"}
	stub := &stubExecer{cols: cols, types: []string{"BIGINT"}, rows: testRows(cols, []any{int64(1)})}
	c := &conn{sess: stub}

	rows, err := c.QueryContext(t.Context(), "SELECT id FROM t;", nil)
	require.NoError(t, err)

	dest := make([]driver.Value, 1)
	require.NoError(t, rows.Next(dest))
	require.Equal(t, int64(1), dest[0])
}

// TestConnQueryContextColumnsMatchQueryOrder proves the driver.Rows a caller
// gets back reports Columns() in the order the transport reported them.
// "zebra" before "apple" sorts the wrong way alphabetically.
func TestConnQueryContextColumnsMatchQueryOrder(t *testing.T) {
	cols := []string{"zebra", "apple"}
	stub := &stubExecer{cols: cols, types: []string{"BIGINT", "BIGINT"}, rows: testRows(cols, []any{int64(1), int64(2)})}
	c := &conn{sess: stub}

	rows, err := c.QueryContext(t.Context(), "SELECT zebra, apple FROM t;", nil)
	require.NoError(t, err)
	require.Equal(t, []string{"zebra", "apple"}, rows.Columns())
}
