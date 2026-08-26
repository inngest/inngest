package duckdb

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestQuackEncodeUUIDMatchesPinnedWireBytes reuses the exact wire-byte pair
// quack_protocol_test.go's TestQuackDecodeChunkUUIDDecodesToStandardString
// already pins (confirmed against a real duckdb subprocess), proving
// quackEncodeUUID is the true inverse of the decode side's transform rather
// than just self-consistent.
func TestQuackEncodeUUIDMatchesPinnedWireBytes(t *testing.T) {
	want := []byte{137, 103, 245, 228, 211, 194, 177, 160, 137, 71, 246, 229, 212, 195, 178, 33}
	id := uuid.MustParse("a1b2c3d4-e5f6-4789-a0b1-c2d3e4f56789")

	got := quackEncodeUUID(id)
	require.Equal(t, want, got[:])
}

// TestQuackEncodeDataChunkDecodesBackToOriginalValues round-trips a
// multi-column, multi-row chunk (every QuackColumnKind, plus a NULL in each
// column) purely through this client's own encode/decode pair — no
// subprocess needed, since the wire format is symmetric and the decode side
// is already verified against a real duckdb subprocess elsewhere
// (quack_protocol_test.go).
func TestQuackEncodeDataChunkDecodesBackToOriginalValues(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	ts1 := time.UnixMilli(1_700_000_000_123).UTC()
	ts2 := time.UnixMilli(1_800_000_000_456).UTC()

	cols := []QuackColumnKind{QuackColumnUUID, QuackColumnVarchar, QuackColumnJSON, QuackColumnTimestampMS}
	rows := [][]any{
		{id1.String(), "span-a", `{"k":"v","n":1}`, ts1},
		{nil, nil, nil, nil},
		{id2.String(), "span-b", `{"arr":[1,2,3]}`, ts2},
	}

	chunk, err := encodeQuackDataChunk(cols, rows)
	require.NoError(t, err)

	r := newQuackReader(chunk)
	c, err := decodeQuackDataChunk(r)
	require.NoError(t, err)
	require.Equal(t, 3, c.rowCount)
	require.Len(t, c.columns, 4)

	uuidVals, err := c.columns[0].values()
	require.NoError(t, err)
	require.Equal(t, []any{id1.String(), nil, id2.String()}, uuidVals)

	varcharVals, err := c.columns[1].values()
	require.NoError(t, err)
	require.Equal(t, []any{"span-a", nil, "span-b"}, varcharVals)

	require.Equal(t, "JSON", c.columns[2].alias)
	jsonVals, err := c.columns[2].values()
	require.NoError(t, err)
	require.Equal(t, []any{
		map[string]any{"k": "v", "n": float64(1)},
		nil,
		map[string]any{"arr": []any{float64(1), float64(2), float64(3)}},
	}, jsonVals)

	tsVals, err := c.columns[3].values()
	require.NoError(t, err)
	require.Len(t, tsVals, 3)
	require.True(t, ts1.Equal(tsVals[0].(time.Time)))
	require.Nil(t, tsVals[1])
	require.True(t, ts2.Equal(tsVals[2].(time.Time)))
}

// TestQuackAppenderWritesRowsIntoRealTable is the load-bearing check: it
// drives NewQuackAppender/AppendRow/Close against a real duckdb subprocess
// with quack loaded, then reads the rows back through the ordinary
// PrepareRequest exec path (SELECT) to prove AppendRequest's wire shape —
// including the field3 append_chunk presence-byte-then-object encoding this
// client's own protocol source didn't make certain in isolation (see
// quack_append.go's package doc comment) — is actually accepted by a real
// duckdb-quack server, not just self-consistent with this client's decoder.
func TestQuackAppenderWritesRowsIntoRealTable(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	dir := t.TempDir()
	addr := freeLocalAddr(t)
	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.ducklake"),
			DataPath:    filepath.Join(dir, "data"),
		},
		QuackAddr: &addr,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE "+DuckLakeAlias+".append_t "+
		"(id UUID, name VARCHAR, attrs JSON, ts TIMESTAMP_MS);")
	require.NoError(t, err)

	id := uuid.New()
	ts := time.UnixMilli(1_700_000_000_000).UTC()

	appender, err := NewQuackAppender(t.Context(), db, DuckLakeAlias, "main", "append_t",
		[]QuackColumnKind{QuackColumnUUID, QuackColumnVarchar, QuackColumnJSON, QuackColumnTimestampMS})
	require.NoError(t, err)

	require.NoError(t, appender.AppendRow(id.String(), "hello", `{"a":1}`, ts))
	require.NoError(t, appender.AppendRow(nil, nil, nil, nil))
	require.NoError(t, appender.Close(t.Context()))

	rows, err := db.QueryContext(t.Context(),
		"SELECT id, name, attrs, ts FROM "+DuckLakeAlias+".append_t ORDER BY name NULLS LAST;")
	require.NoError(t, err)
	defer rows.Close()

	var got []map[string]any
	for rows.Next() {
		row, err := ScanRowByName(rows)
		require.NoError(t, err)
		got = append(got, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, got, 2)

	require.Equal(t, id.String(), got[0]["id"])
	require.Equal(t, "hello", got[0]["name"])
	require.Equal(t, map[string]any{"a": float64(1)}, got[0]["attrs"])
	gotTS, err := AsTimestamp(got[0]["ts"])
	require.NoError(t, err)
	require.True(t, ts.Equal(gotTS), "got %v want %v", gotTS, ts)

	require.Nil(t, got[1]["id"])
	require.Nil(t, got[1]["name"])
}

// TestQuackAppenderWritesVarcharArrayLiteralIntoRealArrayColumn proves the
// working path for a VARCHAR[]-typed column such as
// cmd/duckdbseed's inngest.runs.event_ids: DuckDB's Appender does implicit
// VARCHAR->LIST casting from array-literal text (e.g. `["a","b"]`), so
// QuackColumnVarchar reaches a VARCHAR[] column correctly without needing a
// native LIST wire type at all.
//
// A native LIST QuackColumnKind was prototyped and reverted: the
// client-side wire encoding round-tripped correctly through this file's own
// decoder, but the real duckdb-quack server (v1.5-variegata branch) returns
// an empty-bodied HTTP 500 for any AppendRequest containing a LIST-typed
// column — verified down to the minimal single-row, single-element case,
// and confirmed via duckdb-quack's own server source (quack_server.cpp)
// that the crash happens before the append handler's try/catch (which does
// gracefully convert a std::exception to an ErrorResponse), most likely
// during DuckDB core's DataChunk::Deserialize for the LIST vector.
// duckdb-quack's own test suite has no LIST/array coverage for the
// append/DML path either — this is a server-side gap, not fixable from this
// client alone. The array-literal-text approach below sidesteps it
// entirely.
func TestQuackAppenderWritesVarcharArrayLiteralIntoRealArrayColumn(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	dir := t.TempDir()
	addr := freeLocalAddr(t)
	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.ducklake"),
			DataPath:    filepath.Join(dir, "data"),
		},
		QuackAddr: &addr,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE "+DuckLakeAlias+".array_t (id VARCHAR, tags VARCHAR[]);")
	require.NoError(t, err)

	appender, err := NewQuackAppender(t.Context(), db, DuckLakeAlias, "main", "array_t",
		[]QuackColumnKind{QuackColumnVarchar, QuackColumnVarchar})
	require.NoError(t, err)

	require.NoError(t, appender.AppendRow("multi", `["evt-1","evt-2"]`))
	require.NoError(t, appender.AppendRow("empty", "[]"))
	require.NoError(t, appender.AppendRow("null-list", nil))
	require.NoError(t, appender.Close(t.Context()))

	rows, err := db.QueryContext(t.Context(), "SELECT id, tags FROM "+DuckLakeAlias+".array_t ORDER BY id;")
	require.NoError(t, err)
	defer rows.Close()

	var got []map[string]any
	for rows.Next() {
		row, err := ScanRowByName(rows)
		require.NoError(t, err)
		got = append(got, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, got, 3)

	require.Equal(t, "empty", got[0]["id"])
	require.Equal(t, []any{}, got[0]["tags"])

	require.Equal(t, "multi", got[1]["id"])
	require.Equal(t, []any{"evt-1", "evt-2"}, got[1]["tags"])

	require.Equal(t, "null-list", got[2]["id"])
	require.Nil(t, got[2]["tags"])
}

// TestQuackAppenderFromConnWritesConcurrentlyWithoutLoss proves the actual
// mechanism a caller needing genuinely parallel Appenders relies on:
// OpenConnector's *Connector, called repeatedly for its own raw connections
// (bypassing *sql.DB's pool — see OpenConnector and
// NewQuackAppenderFromConn's doc comments), each backed by an independent
// quackSession that appends concurrently with the others into the same
// table without any row landing more than once, and without any two
// workers ending up sharing a connection.
func TestQuackAppenderFromConnWritesConcurrentlyWithoutLoss(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	dir := t.TempDir()
	addr := freeLocalAddr(t)
	const workers = 4
	const rowsPerWorker = 25
	connector, db, err := OpenConnector(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.ducklake"),
			DataPath:    filepath.Join(dir, "data"),
		},
		QuackAddr:  &addr,
		QuackConns: workers,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE "+DuckLakeAlias+".concurrent_t (id VARCHAR);")
	require.NoError(t, err)

	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			conn, err := connector.Connect(t.Context())
			require.NoError(t, err)
			defer conn.Close()

			appender, err := NewQuackAppenderFromConn(t.Context(), conn, DuckLakeAlias, "main", "concurrent_t", []QuackColumnKind{QuackColumnVarchar})
			require.NoError(t, err)
			for i := range rowsPerWorker {
				require.NoError(t, appender.AppendRow(fmt.Sprintf("worker-%d-row-%d-%s", worker, i, uuid.New())))
			}
			require.NoError(t, appender.Close(t.Context()))
		}(w)
	}
	wg.Wait()

	var count, distinct int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM "+DuckLakeAlias+".concurrent_t;").Scan(&count))
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(DISTINCT id) FROM "+DuckLakeAlias+".concurrent_t;").Scan(&distinct))
	require.Equal(t, workers*rowsPerWorker, count)
	require.Equal(t, workers*rowsPerWorker, distinct, "no row should have been lost or duplicated across concurrent appenders")
}

// TestQuackAppenderRejectsNonQuackConnection pins the guard: appending
// requires a quack-transport *sql.DB (Options.QuackAddr set), not the
// default jsonlines transport, since AppendRequest has no jsonlines
// equivalent.
func TestQuackAppenderRejectsNonQuackConnection(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = NewQuackAppender(t.Context(), db, "", "main", "t", []QuackColumnKind{QuackColumnVarchar})
	require.Error(t, err)
}
