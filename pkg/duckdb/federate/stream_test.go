package federate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/schema"
	"github.com/stretchr/testify/require"
)

// newTestDB opens a real duckdb subprocess over quack with DuckLake attached
// and the POC schema migrated.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	binPath := driver.RequireDuckDBBinary(t)
	dir := t.TempDir()
	addr := driver.EphemeralQuackAddr
	db, err := driver.Open(t.Context(), driver.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &driver.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.ducklake"),
			DataPath:    filepath.Join(dir, "data"),
		},
		QuackAddr: &addr,
	})
	if err != nil {
		t.Skipf("duckdb with quack unavailable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, duckdb.Migrate(t.Context(), db, true))
	return db
}

// streamDelta runs query over a delta streamed from r as the CTE "delta",
// as the executor reads one.
func streamDelta(t *testing.T, conn *sql.Conn, r BatchReader, rowCap int, query string) (*sql.Rows, error) {
	t.Helper()
	st, def, err := deltaStream("t", r, rowCap)
	require.NoError(t, err)
	def = strings.Replace(def, quoteIdent(deltaCTEName("t")), "delta", 1)
	return conn.QueryContext(driver.WithQuackStreams(context.Background(), st), "WITH "+def+" "+query)
}

func TestDeltaStreamCarriesEveryType(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	schema := []Column{
		{"id", schema.TypeUUID}, {"name", schema.TypeVarchar}, {"attrs", schema.TypeJSON}, {"ts", schema.TypeTimestamp},
		{"ok", schema.TypeBool}, {"n", schema.TypeBigint}, {"tags", schema.TypeVarcharList},
	}
	id := uuid.New()
	ts := time.UnixMilli(1_700_000_000_123).UTC()
	b1 := &Batch{Schema: schema, Columns: [][]any{
		{id}, {"a"}, {`{"k":1}`}, {ts}, {true}, {int64(42)}, {[]string{"x", "y,z"}},
	}}
	b2 := &Batch{Schema: schema, Columns: [][]any{
		{id.String()}, {nil}, {nil}, {nil}, {nil}, {nil}, {nil},
	}}

	rows, err := streamDelta(t, conn, NewSliceReader(schema, b1, b2), 0, `SELECT id::VARCHAR, name, attrs->>'k', ts, ok, n, tags::VARCHAR,
		name IS NULL AND attrs IS NULL AND ts IS NULL AND ok IS NULL AND n IS NULL AND tags IS NULL
		FROM delta ORDER BY name NULLS LAST`)
	require.NoError(t, err)
	defer rows.Close()

	var gotID string
	var gotName, gotK, gotTags sql.NullString
	var gotTS sql.NullTime
	var gotOK, allNull sql.NullBool
	var gotN sql.NullInt64
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&gotID, &gotName, &gotK, &gotTS, &gotOK, &gotN, &gotTags, &allNull))
	require.Equal(t, id.String(), gotID)
	require.Equal(t, "a", gotName.String)
	require.Equal(t, "1", gotK.String)
	require.True(t, ts.Equal(gotTS.Time), "got %v want %v", gotTS.Time, ts)
	require.True(t, gotOK.Bool)
	require.Equal(t, int64(42), gotN.Int64)
	require.Equal(t, "[x, 'y,z']", gotTags.String)
	require.False(t, allNull.Bool)

	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&gotID, &gotName, &gotK, &gotTS, &gotOK, &gotN, &gotTags, &allNull))
	require.Equal(t, id.String(), gotID, "a UUID given as text")
	require.True(t, allNull.Bool, "NULLs of every type")
	require.False(t, rows.Next())
	require.NoError(t, rows.Err())
}

func TestDeltaStreamEnforcesRowCap(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	schema := []Column{{"name", schema.TypeVarchar}}
	b := &Batch{Schema: schema, Columns: [][]any{{"a", "b", "c"}}}
	rows, err := streamDelta(t, conn, NewSliceReader(schema, b), 2, "SELECT * FROM delta")
	if err == nil {
		for rows.Next() {
		}
		err = rows.Err()
		_ = rows.Close()
	}
	require.True(t, errors.Is(err, ErrRowCapExceeded), "got %v", err)
}

func TestToWireEncodesEmptyListsAsEmpty(t *testing.T) {
	schema := []Column{{Name: "ids", Type: schema.TypeVarcharList}}
	for _, v := range []any{[]string(nil), []string{}} {
		got, err := toWire(schema, []any{v})
		require.NoError(t, err)
		require.Equal(t, []any{"[]"}, got)
	}
}

// TestListLiteralRoundTripsThroughDuckDB checks AppendListLiteral's text casts
// back to the same list in DuckDB, for the characters JSON-style escaping got
// wrong ('<' became "u003c", a newline became "n").
func TestListLiteralRoundTripsThroughDuckDB(t *testing.T) {
	db := newTestDB(t)
	cases := [][]string{
		{},
		{""},
		{"evt-1", "evt-2"},
		{`q"uote`, `back\slash`, `both\"`, "<>&", "line1\nline2", "tab\tx", "é漢", "[x]", "a,b", "'single'"},
	}
	for _, vals := range cases {
		want, err := json.Marshal(vals)
		require.NoError(t, err)
		var same bool
		err = db.QueryRowContext(context.Background(),
			`SELECT CAST(?::VARCHAR AS VARCHAR[]) = CAST(?::JSON AS VARCHAR[]);`,
			string(AppendListLiteral(nil, vals)), string(want)).Scan(&same)
		require.NoError(t, err, "%q", vals)
		require.True(t, same, "%q round-trips", vals)
	}
}
