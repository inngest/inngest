package duckdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScanRowByNameAndTypeCoercion(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE t (zzz BIGINT, aaa VARCHAR, ts TIMESTAMP);")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO t VALUES (42, 'hello', TIMESTAMP '2026-08-25 12:00:00');")
	require.NoError(t, err)

	rows, err := db.QueryContext(t.Context(), "SELECT zzz, aaa, ts FROM t;")
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())
	row, err := ScanRowByName(rows)
	require.NoError(t, err)
	require.Equal(t, "hello", row["aaa"])

	n, err := AsInt64(row["zzz"])
	require.NoError(t, err)
	require.Equal(t, int64(42), n)

	ts, err := AsTimestamp(row["ts"])
	require.NoError(t, err)
	require.Equal(t, 2026, ts.Year())
	require.Equal(t, time.August, ts.Month())
	require.Equal(t, 25, ts.Day())
}

func TestAsTimestampRejectsUnsupportedType(t *testing.T) {
	_, err := AsTimestamp(42)
	require.Error(t, err)
}
