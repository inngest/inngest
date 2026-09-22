package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestQuackAndJSONLinesReportTheSameColumnOrder runs the identical query
// through both sqlExecer transports (jsonlines' *session and quack's
// *quackSession, both reached via *process so restart/health-check plumbing
// is exercised too) and requires them to report the same column order — and
// for that order to match the query's own left-to-right column list, not
// whatever order the two transports' underlying map types happen to iterate
// in. "zebra" before "apple" sorts the wrong way alphabetically, so this
// fails if either transport (or newMapRows downstream) ever falls back to
// deriving column order from a map instead of carrying it through from the
// wire.
func TestQuackAndJSONLinesReportTheSameColumnOrder(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	const query = "SELECT 1 AS zebra, 2 AS apple, 3 AS mango;"
	want := []string{"zebra", "apple", "mango"}

	jsonlinesProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = jsonlinesProc.close(t.Context()) })
	_, isJSONLines := jsonlinesProc.sess.(*session)
	require.True(t, isJSONLines, "expected jsonlines transport when QuackAddr is unset")

	jsonlinesCols, _, err := jsonlinesProc.exec(t.Context(), query)
	require.NoError(t, err)
	require.Equal(t, want, jsonlinesCols)

	quackAddr := EphemeralQuackAddr
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })
	_, isQuack := quackProc.sess.(*quackSession)
	require.True(t, isQuack, "expected quack transport once bootstrapped")

	quackCols, _, err := quackProc.exec(t.Context(), query)
	require.NoError(t, err)
	require.Equal(t, want, quackCols)

	require.Equal(t, jsonlinesCols, quackCols, "quack and jsonlines must report identical column order for the same query")
}

// TestQuackAndJSONLinesReportTheSameColumnTypes is TestQuackAndJSONLinesReportTheSameColumnOrder's
// counterpart for query() (not exec()): it proves both transports report the
// same DuckDB column type names for the same query, and — the whole reason
// query() exists — that a query matching zero rows still reports its full
// column list and types rather than silently reporting none, on both
// transports identically.
func TestQuackAndJSONLinesReportTheSameColumnTypes(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	// CURRENT_TIMESTAMP is TIMESTAMP WITH TIME ZONE in DuckDB, not the plain
	// microsecond TIMESTAMP a literal `TIMESTAMP '...'` would produce —
	// confirmed against the real binary, not a guess.
	const query = "SELECT CAST(1 AS BIGINT) AS n, 'x' AS s, CURRENT_TIMESTAMP AS ts FROM (SELECT 1) WHERE false;"
	wantCols := []string{"n", "s", "ts"}
	wantTypes := []string{"BIGINT", "VARCHAR", "TIMESTAMP WITH TIME ZONE"}

	jsonlinesProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = jsonlinesProc.close(t.Context()) })

	jsonlinesCols, jsonlinesTypes, jsonlinesRows, err := jsonlinesProc.query(t.Context(), query)
	require.NoError(t, err)
	require.Equal(t, wantCols, jsonlinesCols)
	require.Equal(t, wantTypes, jsonlinesTypes)
	require.Empty(t, jsonlinesRows, "query matches zero rows")

	quackAddr := EphemeralQuackAddr
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	quackCols, quackTypes, quackRows, err := quackProc.query(t.Context(), query)
	require.NoError(t, err)
	require.Equal(t, wantCols, quackCols)
	require.Equal(t, wantTypes, quackTypes)
	require.Empty(t, quackRows, "query matches zero rows")
}

// TestRepeatedColumnNamesKeepEveryValue runs a query that repeats a column
// name through database/sql over both transports and requires every value
// back, in order — the shape of an Insights join like
// SELECT r.account_id, e.account_id ....
func TestRepeatedColumnNamesKeepEveryValue(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	const query = "SELECT 1 AS x, 'two' AS x, 3 AS y;"

	quackAddr := EphemeralQuackAddr
	for name, opts := range map[string]Options{
		"jsonlines": {BinaryPath: binPath, DBFile: ":memory:"},
		"quack":     {BinaryPath: binPath, DBFile: ":memory:", QuackAddr: &quackAddr},
	} {
		t.Run(name, func(t *testing.T) {
			db, err := Open(t.Context(), opts)
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
		})
	}
}
