package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkerReportsExactColumnOrder(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	const query = "SELECT 1 AS zebra, 2 AS apple, 3 AS mango;"
	want := []string{"zebra", "apple", "mango"}

	proc, err := startProcessWithLake(t.Context(), binPath, ":memory:", nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = proc.close(t.Context()) })
	_, isWorker := proc.sess.(*workerSessionExecer)
	require.True(t, isWorker, "expected worker transport")

	cols, _, err := proc.Exec(t.Context(), query)
	require.NoError(t, err)
	require.Equal(t, want, cols)
}

func TestWorkerReportsExactColumnTypesOnEmptyResult(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	const query = "SELECT CAST(1 AS BIGINT) AS n, 'x' AS s, CURRENT_TIMESTAMP AS ts FROM (SELECT 1) WHERE false;"
	wantCols := []string{"n", "s", "ts"}
	wantTypes := []string{"BIGINT", "VARCHAR", "TIMESTAMP WITH TIME ZONE"}

	proc, err := startProcessWithLake(t.Context(), binPath, ":memory:", nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = proc.close(t.Context()) })

	cols, types, rows, err := proc.Query(t.Context(), query)
	require.NoError(t, err)
	require.Equal(t, wantCols, cols)
	require.Equal(t, wantTypes, types)
	require.Empty(t, rows, "query matches zero rows")
}

func TestRepeatedColumnNamesKeepEveryValue(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	const query = "SELECT 1 AS x, 'two' AS x, 3 AS y;"

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	rows, err := db.QueryContext(t.Context(), query)
	require.NoError(t, err)
	defer rows.Close()

	cols, err := rows.Columns()
	require.NoError(t, err)
	require.Equal(t, []string{"x", "x", "y"}, cols)

	require.True(t, rows.Next())
	var a, c int64
	var b string
	require.NoError(t, rows.Scan(&a, &b, &c))
	require.Equal(t, int64(1), a)
	require.Equal(t, "two", b)
	require.Equal(t, int64(3), c)
}
