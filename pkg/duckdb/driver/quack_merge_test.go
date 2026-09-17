package driver

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestQuackMergeAppenderMergesRowsIntoRealTable is the MERGE analogue of
// quack_append_test.go's TestQuackAppenderWritesRowsIntoRealTable: it drives
// NewQuackMergeAppender/AppendRow/Close against a real duckdb subprocess with
// quack loaded, pushing source rows over SEND_DATA_REQUEST (not literal SQL
// VALUES) into a MERGE INTO ... USING scan_data_from_quack_client(...)
// statement, then reads the target table back to prove both WHEN MATCHED
// UPDATE and WHEN NOT MATCHED INSERT actually ran against a real server.
func TestQuackMergeAppenderMergesRowsIntoRealTable(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
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

	_, err = db.ExecContext(t.Context(), "CREATE TABLE "+DuckLakeAlias+".merge_t (k VARCHAR, v VARCHAR);")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO "+DuckLakeAlias+".merge_t VALUES ('a', 'unchanged'), ('b', 'stale');")
	require.NoError(t, err)

	appender, err := NewQuackMergeAppender(t.Context(), db, DuckLakeAlias, "main", "merge_t",
		QuackMergeConfig{
			On:        "t.k = s.k",
			UpdateSet: "v = s.v",
		},
		[]QuackMergeColumn{
			{Name: "k", Kind: QuackColumnVarchar},
			{Name: "v", Kind: QuackColumnVarchar},
		})
	require.NoError(t, err)

	require.NoError(t, appender.AppendRow("b", "updated"))
	require.NoError(t, appender.AppendRow("c", "inserted"))
	require.NoError(t, appender.Close(t.Context()))

	rows, err := db.QueryContext(t.Context(), "SELECT k, v FROM "+DuckLakeAlias+".merge_t ORDER BY k;")
	require.NoError(t, err)
	defer rows.Close()

	var got []map[string]any
	for rows.Next() {
		row, err := ScanRowByName(rows)
		require.NoError(t, err)
		got = append(got, row)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []map[string]any{
		{"k": "a", "v": "unchanged"}, // untouched: no source row matched or targeted it
		{"k": "b", "v": "updated"},   // WHEN MATCHED: updated in place
		{"k": "c", "v": "inserted"},  // WHEN NOT MATCHED: inserted
	}, got)
}

// TestQuackMergeAppenderFlushesMoreThanOneVectorWorthOfRows is the MERGE
// analogue of quack_append_test.go's identically-named test: proves
// rowsToQuackChunks' STANDARD_VECTOR_SIZE splitting (shared with
// QuackAppender via sendDataDrive) holds up through the MERGE path against a
// real server too, not just the INSERT path already covered there.
func TestQuackMergeAppenderFlushesMoreThanOneVectorWorthOfRows(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
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

	_, err = db.ExecContext(t.Context(), "CREATE TABLE "+DuckLakeAlias+".many_rows_merge_t (k VARCHAR, v VARCHAR);")
	require.NoError(t, err)

	appender, err := NewQuackMergeAppender(t.Context(), db, DuckLakeAlias, "main", "many_rows_merge_t",
		QuackMergeConfig{On: "t.k = s.k", UpdateSet: "v = s.v"},
		[]QuackMergeColumn{
			{Name: "k", Kind: QuackColumnVarchar},
			{Name: "v", Kind: QuackColumnVarchar},
		})
	require.NoError(t, err)

	const wantRows = 2*quackStandardVectorSize + 500
	for i := range wantRows {
		require.NoError(t, appender.AppendRow(fmt.Sprintf("row-%d", i), fmt.Sprintf("v-%d", i)))
	}
	require.NoError(t, appender.Close(t.Context()))

	row := db.QueryRowContext(t.Context(), "SELECT count(*) FROM "+DuckLakeAlias+".many_rows_merge_t;")
	var gotCount int
	require.NoError(t, row.Scan(&gotCount))
	require.Equal(t, wantRows, gotCount)

	last := db.QueryRowContext(t.Context(),
		"SELECT v FROM "+DuckLakeAlias+".many_rows_merge_t WHERE k = 'row-"+fmt.Sprint(wantRows-1)+"';")
	var gotV string
	require.NoError(t, last.Scan(&gotV))
	require.Equal(t, fmt.Sprintf("v-%d", wantRows-1), gotV)
}

// TestQuackMergeAppenderOmitsUpdateWhenUpdateSetEmpty proves
// QuackMergeConfig.UpdateSet's documented zero-value behavior: an empty
// UpdateSet must omit the WHEN MATCHED clause from the generated SQL
// entirely, so a source row matching an existing target row leaves it
// untouched (an insert-only merge) rather than DuckDB rejecting an empty
// UPDATE SET list outright.
func TestQuackMergeAppenderOmitsUpdateWhenUpdateSetEmpty(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
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

	_, err = db.ExecContext(t.Context(), "CREATE TABLE "+DuckLakeAlias+".insert_only_merge_t (k VARCHAR, v VARCHAR);")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO "+DuckLakeAlias+".insert_only_merge_t VALUES ('a', 'original');")
	require.NoError(t, err)

	appender, err := NewQuackMergeAppender(t.Context(), db, DuckLakeAlias, "main", "insert_only_merge_t",
		QuackMergeConfig{On: "t.k = s.k"}, // UpdateSet deliberately left empty
		[]QuackMergeColumn{
			{Name: "k", Kind: QuackColumnVarchar},
			{Name: "v", Kind: QuackColumnVarchar},
		})
	require.NoError(t, err)

	require.NoError(t, appender.AppendRow("a", "would-be-overwritten"))
	require.NoError(t, appender.AppendRow("b", "inserted"))
	require.NoError(t, appender.Close(t.Context()))

	rows, err := db.QueryContext(t.Context(), "SELECT k, v FROM "+DuckLakeAlias+".insert_only_merge_t ORDER BY k;")
	require.NoError(t, err)
	defer rows.Close()

	var got []map[string]any
	for rows.Next() {
		row, err := ScanRowByName(rows)
		require.NoError(t, err)
		got = append(got, row)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []map[string]any{
		{"k": "a", "v": "original"}, // matched, but left untouched: no WHEN MATCHED clause
		{"k": "b", "v": "inserted"}, // not matched: inserted
	}, got)
}

// TestQuackMergeAppenderDedupKeysKeepsLastRowPerKey proves QuackMergeConfig.
// DedupKeys/DedupOrderBy's documented behavior: a single Flush's batch may
// contain more than one row for the same key, and only the row with the
// greatest DedupOrderBy value per DedupKeys group survives — for both an
// insert (no prior target row: two same-key rows would otherwise both take
// the WHEN NOT MATCHED branch and produce two target rows) and an update (a
// prior target row already exists: only the winning row's values should
// land, not some undefined mix of the two).
func TestQuackMergeAppenderDedupKeysKeepsLastRowPerKey(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
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

	_, err = db.ExecContext(t.Context(), "CREATE TABLE "+DuckLakeAlias+".dedup_merge_t (k VARCHAR, seq VARCHAR, v VARCHAR);")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO "+DuckLakeAlias+".dedup_merge_t VALUES ('existing', '000', 'stale');")
	require.NoError(t, err)

	appender, err := NewQuackMergeAppender(t.Context(), db, DuckLakeAlias, "main", "dedup_merge_t",
		QuackMergeConfig{
			On:           "t.k = s.k",
			UpdateSet:    "seq = s.seq, v = s.v",
			DedupKeys:    []string{"k"},
			DedupOrderBy: "seq",
		},
		[]QuackMergeColumn{
			{Name: "k", Kind: QuackColumnVarchar},
			{Name: "seq", Kind: QuackColumnVarchar}, // zero-padded so lexicographic (SQL string) order matches numeric order
			{Name: "v", Kind: QuackColumnVarchar},
		})
	require.NoError(t, err)

	// "existing" is matched (an update, three same-key rows in the batch)
	// and "new" is unmatched (an insert, two same-key rows) — both must end
	// up with only the highest-seq row's v, not "-first"/"-mid" and not two
	// rows.
	require.NoError(t, appender.AppendRow("existing", "001", "existing-first"))
	require.NoError(t, appender.AppendRow("existing", "003", "existing-last"))
	require.NoError(t, appender.AppendRow("existing", "002", "existing-mid"))
	require.NoError(t, appender.AppendRow("new", "002", "new-first"))
	require.NoError(t, appender.AppendRow("new", "005", "new-last"))
	require.NoError(t, appender.Close(t.Context()))

	rows, err := db.QueryContext(t.Context(), "SELECT k, seq, v FROM "+DuckLakeAlias+".dedup_merge_t ORDER BY k;")
	require.NoError(t, err)
	defer rows.Close()

	var got []map[string]any
	for rows.Next() {
		row, err := ScanRowByName(rows)
		require.NoError(t, err)
		got = append(got, row)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []map[string]any{
		{"k": "existing", "seq": "003", "v": "existing-last"},
		{"k": "new", "seq": "005", "v": "new-last"},
	}, got)
}

// TestQuackMergeAppenderDedupKeysRejectsUnknownColumn guards
// buildQuackSendDataMergeSQL's validation: a DedupKeys entry that doesn't
// name one of the columns passed to NewQuackMergeAppender must fail clearly
// up front, rather than reaching the server as a "column not found" SQL
// error against a QUALIFY clause the caller never wrote themselves.
func TestQuackMergeAppenderDedupKeysRejectsUnknownColumn(t *testing.T) {
	_, err := buildQuackSendDataMergeSQL("main", "t", QuackMergeConfig{
		On:           "t.k = s.k",
		DedupKeys:    []string{"nope"},
		DedupOrderBy: "seq",
	}, []QuackMergeColumn{{Name: "k", Kind: QuackColumnVarchar}}, "stream-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "nope")
}

// TestQuackMergeAppenderDedupKeysRequiresDedupOrderBy guards the other
// buildQuackSendDataMergeSQL validation: DedupKeys with no DedupOrderBy has
// no way to choose which same-key row survives, so it must fail clearly up
// front rather than building a QUALIFY with an empty ORDER BY.
func TestQuackMergeAppenderDedupKeysRequiresDedupOrderBy(t *testing.T) {
	_, err := buildQuackSendDataMergeSQL("main", "t", QuackMergeConfig{
		On:        "t.k = s.k",
		DedupKeys: []string{"k"},
	}, []QuackMergeColumn{{Name: "k", Kind: QuackColumnVarchar}}, "stream-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "DedupOrderBy")
}

// TestQuackMergeAppenderConcurrentPooledUseDoesNotSuperseded is the MERGE
// analogue of quack_append_test.go's identically-named test: proves
// NewQuackMergeAppender's pooled *sql.Conn is held open for the appender's
// whole lifetime (not released back to db's pool the moment the underlying
// quackSession is extracted), so many concurrent, short-lived appenders
// sharing a small connection pool never have one appender's in-flight
// stream superseded by another's PrepareRequest landing on the same
// connection mid-flight.
func TestQuackMergeAppenderConcurrentPooledUseDoesNotSuperseded(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	dir := t.TempDir()
	addr := freeLocalAddr(t)
	const quackConns = 4
	const workers = 20
	const appendersPerWorker = 5
	const rowsPerAppender = 5
	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.ducklake"),
			DataPath:    filepath.Join(dir, "data"),
		},
		QuackAddr:  &addr,
		QuackConns: quackConns,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE "+DuckLakeAlias+".concurrent_pooled_merge_t (k VARCHAR, v VARCHAR);")
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make([]error, workers)
	for w := range workers {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for a := range appendersPerWorker {
				appender, err := NewQuackMergeAppender(t.Context(), db, DuckLakeAlias, "main", "concurrent_pooled_merge_t",
					QuackMergeConfig{On: "t.k = s.k", UpdateSet: "v = s.v"},
					[]QuackMergeColumn{
						{Name: "k", Kind: QuackColumnVarchar},
						{Name: "v", Kind: QuackColumnVarchar},
					})
				if err != nil {
					errs[worker] = err
					return
				}
				for i := range rowsPerAppender {
					k := fmt.Sprintf("worker-%d-appender-%d-row-%d-%s", worker, a, i, uuid.New())
					if err := appender.AppendRow(k, "v0"); err != nil {
						errs[worker] = err
						return
					}
				}
				if err := appender.Close(t.Context()); err != nil {
					errs[worker] = err
					return
				}
			}
		}(w)
	}
	wg.Wait()
	for w, err := range errs {
		require.NoError(t, err, "worker %d", w)
	}

	const wantRows = workers * appendersPerWorker * rowsPerAppender
	var count, distinct int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM "+DuckLakeAlias+".concurrent_pooled_merge_t;").Scan(&count))
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(DISTINCT k) FROM "+DuckLakeAlias+".concurrent_pooled_merge_t;").Scan(&distinct))
	require.Equal(t, wantRows, count)
	require.Equal(t, wantRows, distinct, "no row should have been lost or duplicated across concurrent pooled merge appenders")
}
