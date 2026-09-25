package driver

import (
	"database/sql/driver"
	"io"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/stretchr/testify/require"
)

func TestMapRowsColumns(t *testing.T) {
	r := newMapRows([]string{"id"}, nil, testRows([]string{"id"}, []any{float64(1)}))
	require.Equal(t, []string{"id"}, r.Columns())
}

// testRows builds positional rows sharing one *result.Columns.
func testRows(names []string, vals ...[]any) []result.Row {
	cols := result.NewColumns(names)
	out := make([]result.Row, len(vals))
	for i, v := range vals {
		out[i] = result.Row{Cols: cols, Vals: v}
	}
	return out
}

// TestMapRowsKeepsEveryValueOfARepeatedColumnName pins that a query
// repeating a column name (SELECT a.x, b.x) returns both values, where
// map-keyed rows used to collapse them into the second.
func TestMapRowsKeepsEveryValueOfARepeatedColumnName(t *testing.T) {
	r := newMapRows([]string{"x", "x", "y"}, nil, testRows([]string{"x", "x", "y"}, []any{"a", "b", "c"}))
	require.Equal(t, []string{"x", "x", "y"}, r.Columns())
	dest := make([]driver.Value, 3)
	require.NoError(t, r.Next(dest))
	require.Equal(t, []driver.Value{"a", "b", "c"}, dest)
}

func TestMapRowsReportsColumnTypeDatabaseTypeName(t *testing.T) {
	r := newMapRows([]string{"id", "name"}, []string{"BIGINT", "VARCHAR"}, nil)
	require.Equal(t, "BIGINT", r.ColumnTypeDatabaseTypeName(0))
	require.Equal(t, "VARCHAR", r.ColumnTypeDatabaseTypeName(1))
}

func TestMapRowsNextIteratesAllRowsThenEOF(t *testing.T) {
	r := newMapRows([]string{"id"}, nil, testRows([]string{"id"},
		[]any{float64(1)},
		[]any{float64(2)},
		[]any{float64(3)},
	))
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
