package duckdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestQuackQueryDateColumn is the end-to-end proof for quackColumn.values()'s
// Date case: a real duckdb subprocess over the quack transport, not just
// hand-built wire bytes. Before that case existed, any DATE-typed column
// (LogicalTypeId 15) crashed the subprocess with "duckdb: quack column
// decode: unsupported LogicalTypeId 15" — reproduced with a query shaped
// like the one that originally surfaced it (`SELECT '...'::DATE, * FROM
// ...`), minus the table dependency.
func TestQuackQueryDateColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(), "SELECT '2026-01-01'::DATE AS d, 1 AS n;")
	require.NoError(t, err)
	require.Equal(t, []string{"d", "n"}, cols)
	require.Equal(t, []string{"DATE", "INTEGER"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), rows[0]["d"])
}

// TestQuackQueryTimestampTZColumn covers LogicalTypeId 32 (TIMESTAMP WITH
// TIME ZONE), the type that crashed the same way as Date once QuackConns'
// caller hit it: "unsupported LogicalTypeId 32". DuckDB stores TIMESTAMPTZ
// as the same UTC microseconds-since-epoch instant as plain TIMESTAMP, so
// decoding it must produce the same absolute instant regardless of session
// TimeZone.
func TestQuackQueryTimestampTZColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	// Fix the session's display TimeZone so a wrong (display-affected rather
	// than instant-preserving) decode would be caught: if quack decoded the
	// text-rendered, zone-shifted value instead of the raw UTC instant, this
	// offset would corrupt the result.
	_, _, err = quackProc.exec(t.Context(), "SET TimeZone = 'America/New_York';")
	require.NoError(t, err)

	cols, types, rows, err := quackProc.query(t.Context(), "SELECT '2026-01-01 12:00:00+05:30'::TIMESTAMPTZ AS ts;")
	require.NoError(t, err)
	require.Equal(t, []string{"ts"}, cols)
	require.Equal(t, []string{"TIMESTAMP WITH TIME ZONE"}, types)
	require.Len(t, rows, 1)

	got, ok := rows[0]["ts"].(time.Time)
	require.True(t, ok, "expected a time.Time, got %T", rows[0]["ts"])
	want := time.Date(2026, 1, 1, 12, 0, 0, 0, time.FixedZone("", 5*3600+30*60))
	require.True(t, got.Equal(want), "got %v, want %v (same instant)", got, want)
}

// TestQuackQueryTimeColumn covers LogicalTypeId 16 (TIME): microseconds
// since midnight, decoded onto the zero date to match what Go's time.Parse
// produces for jsonlines' TIME layout.
func TestQuackQueryTimeColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(), "SELECT '12:34:56.5'::TIME AS t;")
	require.NoError(t, err)
	require.Equal(t, []string{"t"}, cols)
	require.Equal(t, []string{"TIME"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, time.Date(0, 1, 1, 12, 34, 56, 500000000, time.UTC), rows[0]["t"])
}

// TestQuackQueryTimeTZColumn covers LogicalTypeId 34 (TIME WITH TIME ZONE),
// whose physical encoding packs microseconds-since-midnight and a biased
// UTC offset into one uint64 (see quackTimeTZToTime) rather than the
// straightforward fixed-width layout every other case here decodes.
func TestQuackQueryTimeTZColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(), "SELECT '12:34:56.5+05:30'::TIMETZ AS t;")
	require.NoError(t, err)
	require.Equal(t, []string{"t"}, cols)
	require.Equal(t, []string{"TIME WITH TIME ZONE"}, types)
	require.Len(t, rows, 1)

	got, ok := rows[0]["t"].(time.Time)
	require.True(t, ok, "expected a time.Time, got %T", rows[0]["t"])
	require.Equal(t, 12, got.Hour())
	require.Equal(t, 34, got.Minute())
	require.Equal(t, 56, got.Second())
	require.Equal(t, 500000000, got.Nanosecond())
	_, offset := got.Zone()
	require.Equal(t, 5*3600+30*60, offset)
}
