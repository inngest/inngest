package duckdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestQuackQueryIntegerFamilyColumns covers the plain fixed-width integer
// LogicalTypeIds that were missing from quackColumn.values(): TinyInt(11),
// UTinyInt(28), USmallInt(29), UInteger(30) all decode straightforwardly
// into int64 (they fit safely, unlike UBigInt/Hugeint below). Float(22) is
// included here too: a real 4-byte IEEE754 single, decoded via
// math.Float32frombits and widened to float64 — the precision loss
// (12345.6 -> 12345.599609375) is expected and confirmed against a real
// duckdb binary.
func TestQuackQueryIntegerFamilyColumns(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(),
		"SELECT (-5)::TINYINT AS ti, 200::UTINYINT AS uti, 300::USMALLINT AS us, "+
			"4000000000::UINTEGER AS ui, 12345.6::FLOAT AS f;")
	require.NoError(t, err)
	require.Equal(t, []string{"ti", "uti", "us", "ui", "f"}, cols)
	require.Equal(t, []string{"TINYINT", "UTINYINT", "USMALLINT", "UINTEGER", "FLOAT"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, int64(-5), rows[0]["ti"])
	require.Equal(t, int64(200), rows[0]["uti"])
	require.Equal(t, int64(300), rows[0]["us"])
	require.Equal(t, int64(4000000000), rows[0]["ui"])
	require.InDelta(t, 12345.599609375, rows[0]["f"], 1e-9)
}

// TestQuackQueryUBigIntColumn covers UBigInt(31): its range overflows
// int64, so it's decoded as a decimal-digit string — the same choice
// DuckDB's own JSON output makes for the same reason (confirmed against a
// real binary).
func TestQuackQueryUBigIntColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(), "SELECT 18000000000000000000::UBIGINT AS ub;")
	require.NoError(t, err)
	require.Equal(t, []string{"ub"}, cols)
	require.Equal(t, []string{"UBIGINT"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, "18000000000000000000", rows[0]["ub"])
}

// TestQuackQueryHugeintColumns covers Hugeint(50) and UHugeint(49): both
// physically a 16-byte little-endian lower/upper pair (hugeint.hpp),
// decoded via math/big into a decimal string — verified here against
// values right at HUGEINT's actual min/max and UHUGEINT's max, each
// confirmed independently against a real duckdb binary.
func TestQuackQueryHugeintColumns(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(),
		"SELECT 170141183460469231731687303715884105727::HUGEINT AS hg_max, "+
			"(-170141183460469231731687303715884105728)::HUGEINT AS hg_min, "+
			"340282366920938463463374607431768211455::UHUGEINT AS uhg_max;")
	require.NoError(t, err)
	require.Equal(t, []string{"hg_max", "hg_min", "uhg_max"}, cols)
	require.Equal(t, []string{"HUGEINT", "HUGEINT", "UHUGEINT"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, "170141183460469231731687303715884105727", rows[0]["hg_max"])
	require.Equal(t, "-170141183460469231731687303715884105728", rows[0]["hg_min"])
	require.Equal(t, "340282366920938463463374607431768211455", rows[0]["uhg_max"])
}

// TestQuackQueryDecimalColumns covers Decimal(21) across all four physical
// storage widths DuckDB picks by declared width (types.cpp's
// LogicalType::GetInternalType / decimal.hpp's Decimal::MAX_WIDTH_INT*):
// <=4 -> int16, <=9 -> int32, <=18 -> int64, <=38 -> hugeint. Values and
// expected text are pinned against a real duckdb binary, including the
// always-exactly-`scale`-digits (no trimming) and no-decimal-point-at-
// scale-0 conventions.
func TestQuackQueryDecimalColumns(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(),
		"SELECT 12.3::DECIMAL(4,1) AS d16, 123.400::DECIMAL(9,3) AS d32, "+
			"(-123.456)::DECIMAL(18,3) AS d64, "+
			"12345678901234567890.123456789::DECIMAL(38,9) AS d128, "+
			"5::DECIMAL(5,0) AS d_scale0;")
	require.NoError(t, err)
	require.Equal(t, []string{"d16", "d32", "d64", "d128", "d_scale0"}, cols)
	require.Equal(t, []string{"DECIMAL(4,1)", "DECIMAL(9,3)", "DECIMAL(18,3)", "DECIMAL(38,9)", "DECIMAL(5,0)"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, "12.3", rows[0]["d16"])
	require.Equal(t, "123.400", rows[0]["d32"])
	require.Equal(t, "-123.456", rows[0]["d64"])
	require.Equal(t, "12345678901234567890.123456789", rows[0]["d128"])
	require.Equal(t, "5", rows[0]["d_scale0"])
}

// TestQuackQueryBlobColumn covers Blob(26): BLOB shares VARCHAR's
// variable-length wire shape (both are PhysicalType::VARCHAR per DuckDB's
// LogicalType::GetInternalType), decoded as raw []byte rather than an
// escaped display string.
func TestQuackQueryBlobColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(), `SELECT '\xAA\xBB\xCC'::BLOB AS b;`)
	require.NoError(t, err)
	require.Equal(t, []string{"b"}, cols)
	require.Equal(t, []string{"BLOB"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, []byte{0xAA, 0xBB, 0xCC}, rows[0]["b"])
}

// TestQuackQueryBitColumn covers Bit(36): BIT is a bitstring_t, an alias
// for string_t (bit.hpp), so it shares BLOB/VARCHAR's variable-length wire
// shape too — decoded into the '0'/'1' text DuckDB's own Bit::ToString
// produces (bit.cpp), confirmed against a real binary.
func TestQuackQueryBitColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(), "SELECT BIT '101010' AS b, BIT '1' AS b2, BIT '11111111' AS b3;")
	require.NoError(t, err)
	require.Equal(t, []string{"b", "b2", "b3"}, cols)
	require.Equal(t, []string{"BIT", "BIT", "BIT"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, "101010", rows[0]["b"])
	require.Equal(t, "1", rows[0]["b2"])
	require.Equal(t, "11111111", rows[0]["b3"])
}

// TestQuackQueryIntervalColumn covers Interval(27): a 16-byte
// interval_t{months int32; days int32; micros int64} (interval.hpp),
// formatted to match DuckDB's own Interval::ToString exactly, including
// independently-signed components and unbounded (not mod-24) hours —
// every expected string here was confirmed against a real duckdb binary.
func TestQuackQueryIntervalColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(),
		"SELECT INTERVAL '1 year 2 months 3 days 04:05:06.789' AS a, "+
			"INTERVAL '25 hours' AS b, "+
			"(-INTERVAL '1 year 2 days 3 hours') AS c, "+
			"(INTERVAL '0 seconds') AS d;")
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "c", "d"}, cols)
	require.Equal(t, []string{"INTERVAL", "INTERVAL", "INTERVAL", "INTERVAL"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, "1 year 2 months 3 days 04:05:06.789", rows[0]["a"])
	require.Equal(t, "25:00:00", rows[0]["b"])
	require.Equal(t, "-1 year -2 days -03:00:00", rows[0]["c"])
	require.Equal(t, "00:00:00", rows[0]["d"])
}

// TestQuackQueryEnumColumn covers Enum(104): a small (<=255 value)
// dictionary uses a 1-byte index (EnumTypeInfo::DictType,
// extra_type_info.cpp) into the values already parsed onto
// quackLogicalType.enumValues by decodeQuackLogicalType.
func TestQuackQueryEnumColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	_, _, err = quackProc.exec(t.Context(), "CREATE TYPE mood AS ENUM ('sad', 'ok', 'happy');")
	require.NoError(t, err)

	cols, types, rows, err := quackProc.query(t.Context(), "SELECT 'happy'::mood AS m, 'sad'::mood AS m2;")
	require.NoError(t, err)
	require.Equal(t, []string{"m", "m2"}, cols)
	// typeName() renders ENUM's full value-list expansion (matching
	// DuckDB's own DESCRIBE), not the CREATE TYPE alias "mood".
	require.Equal(t, []string{"ENUM('sad', 'ok', 'happy')", "ENUM('sad', 'ok', 'happy')"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, "happy", rows[0]["m"])
	require.Equal(t, "sad", rows[0]["m2"])
}

// TestQuackQueryMapColumn covers Map(102): physically (and, per this
// package's comments, on the wire) a LIST of STRUCT{key,value}
// (LogicalType::GetInternalType, types.cpp) — decodeQuackVector routes it
// through the exact same decodeQuackListVector path as LIST, so no
// Map-specific values() case exists; this is the end-to-end proof that
// reuse actually works against a real subprocess.
func TestQuackQueryMapColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(), "SELECT MAP {'a': 1, 'b': 2} AS m;")
	require.NoError(t, err)
	require.Equal(t, []string{"m"}, cols)
	require.Equal(t, []string{"MAP(VARCHAR, INTEGER)"}, types)
	require.Len(t, rows, 1)

	got, ok := rows[0]["m"].([]any)
	require.True(t, ok, "expected a []any, got %T", rows[0]["m"])
	want := []any{
		map[string]any{"key": "a", "value": int64(1)},
		map[string]any{"key": "b", "value": int64(2)},
	}
	require.Equal(t, want, got)
}

// TestQuackQueryArrayColumn covers Array(108): a fixed-size list,
// physically PhysicalType::ARRAY (types.cpp) rather than LIST. Its wire
// shape was reverse-engineered by hex-dumping a live response (no
// documentation exists for this custom protocol): field103 is the
// flattened child vector's total element count (count*arraySize, no
// per-row list_entry_t table since the stride is static), field104 wraps
// that one child vector.
func TestQuackQueryArrayColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(), "SELECT [1,2,3]::INTEGER[3] AS a;")
	require.NoError(t, err)
	require.Equal(t, []string{"a"}, cols)
	require.Equal(t, []string{"INTEGER[3]"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, []any{int64(1), int64(2), int64(3)}, rows[0]["a"])
}

// TestQuackQueryArrayColumnEdgeCases covers the cases that broke the first
// implementation attempt: a NULL array row (validity, not a wire shape
// difference), and a nested ARRAY-of-ARRAY — which revealed that field103
// (see decodeQuackArrayVector) always restates lt.arraySize alone, never
// count*arraySize, even when the child itself is an ARRAY whose own
// "count" is the outer array's flattened element total rather than 1.
func TestQuackQueryArrayColumnEdgeCases(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(),
		"SELECT NULL::INTEGER[3] AS null_arr, [[1,2],[3,4]]::INTEGER[2][2] AS nested;")
	require.NoError(t, err)
	require.Equal(t, []string{"null_arr", "nested"}, cols)
	require.Equal(t, []string{"INTEGER[3]", "INTEGER[2][2]"}, types)
	require.Len(t, rows, 1)
	require.Nil(t, rows[0]["null_arr"])
	require.Equal(t, []any{
		[]any{int64(1), int64(2)},
		[]any{int64(3), int64(4)},
	}, rows[0]["nested"])
}

// TestQuackQueryUnionColumn covers Union(107): physically PhysicalType::
// STRUCT (types.cpp) with a hidden field 0 (name "", type UTINYINT) tag
// selecting which member is valid for a given row (LogicalType::UNION,
// types.cpp). Rather than exposing every member (mostly nil) plus the raw
// tag, this decodes into a single-entry map{selected member's name: its
// value} — matching DuckDB's own JSON rendering of a UNION value
// (confirmed against a real binary: `SELECT union_value(a := 1)` renders
// as {"a": 1}, not the full struct).
func TestQuackQueryUnionColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	quackAddr := freeLocalAddr(t)
	quackProc, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &quackAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = quackProc.close(t.Context()) })

	cols, types, rows, err := quackProc.query(t.Context(),
		"SELECT union_value(a := 1) AS u, union_value(b := 'hi') AS u2, "+
			"NULL::UNION(a INTEGER, b VARCHAR) AS u3;")
	require.NoError(t, err)
	require.Equal(t, []string{"u", "u2", "u3"}, cols)
	require.Equal(t, []string{"UNION(a INTEGER)", "UNION(b VARCHAR)", "UNION(a INTEGER, b VARCHAR)"}, types)
	require.Len(t, rows, 1)
	require.Equal(t, map[string]any{"a": int64(1)}, rows[0]["u"])
	require.Equal(t, map[string]any{"b": "hi"}, rows[0]["u2"])
	require.Nil(t, rows[0]["u3"])
}
