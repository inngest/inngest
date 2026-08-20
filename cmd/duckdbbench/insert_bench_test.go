// Package duckdbbench benchmarks INSERT throughput into
// inngest.run_trace_spans (pkg/db/duckdb/migrations/000001_baseline.sql)
// across four DuckDB access paths — see go.mod's doc comment for why this
// lives in its own module.
package duckdbbench

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	duckdbgo "github.com/duckdb/duckdb-go/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
)

// spanColumns mirrors cmd/duckdbseed/insert.go's spanColumns — the column
// order inngest.run_trace_spans' migration declares.
var spanColumns = []string{
	"account_id", "env_id", "run_id", "run_queued_at", "app_id", "function_id",
	"name", "start_time", "end_time", "trace_id", "span_id", "parent_span_id",
	"attributes", "links", "output", "input",
}

// spanRow is a synthetic inngest.run_trace_spans row for benchmarking.
type spanRow struct {
	accountID, envID, appID, functionID uuid.UUID
	runID                               string
	runQueuedAt                         time.Time
	name                                string
	startTime, endTime                  time.Time
	traceID, spanID                     string
	attributes                          string
}

// generateSpanBatch returns n synthetic spans, uniquely identified by
// (batchSeed, i) so repeated benchmark iterations never collide on
// run_id/trace_id/span_id.
func generateSpanBatch(n, batchSeed int) []spanRow {
	tenant := uuid.New()
	now := time.Now()
	rows := make([]spanRow, n)
	for i := range n {
		runID := fmt.Sprintf("run-%d-%d", batchSeed, i)
		start := now.Add(time.Duration(i) * time.Millisecond)
		rows[i] = spanRow{
			accountID:   tenant,
			envID:       tenant,
			appID:       tenant,
			functionID:  tenant,
			runID:       runID,
			runQueuedAt: start,
			name:        "step.discovered",
			startTime:   start,
			endTime:     start.Add(time.Millisecond),
			traceID:     fmt.Sprintf("trace-%d-%d", batchSeed, i),
			spanID:      fmt.Sprintf("span-%d-%d", batchSeed, i),
			attributes:  `{"job.group.id":"grp","step.id":"step"}`,
		}
	}
	return rows
}

// batchInsertQuery builds "INSERT INTO table (cols...) VALUES (?,...), ...;"
// for rowCount rows of len(columns) values each — mirrors
// cmd/duckdbseed/insert.go's helper of the same name.
func batchInsertQuery(table string, columns []string, rowCount int) string {
	placeholderRow := "(" + strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", ") + ")"
	rows := make([]string, rowCount)
	for i := range rows {
		rows[i] = placeholderRow
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s;", table, strings.Join(columns, ", "), strings.Join(rows, ", "))
}

// spanInserter abstracts over how a batch of spans reaches the table, so
// BenchmarkInsertSpans can drive the three INSERT-statement transports
// (jsonlines, quack, embedded) and the Appender-API path through the same
// loop. Each insertBatch call is one full flush — one multi-row INSERT, or
// one create-append*N-close Appender lifecycle — mirroring dual-write's own
// per-flush unit of work.
type spanInserter interface {
	insertBatch(ctx context.Context, rows []spanRow) error
}

// dbInserter drives the three transports that only ever speak SQL text:
// jsonlines and quack (both via pkg/db/duckdb's database/sql driver) and the
// embedded connection's own INSERT INTO ... VALUES (...) path.
type dbInserter struct{ db *sql.DB }

func (d dbInserter) insertBatch(ctx context.Context, rows []spanRow) error {
	if len(rows) == 0 {
		return nil
	}
	args := make([]any, 0, len(rows)*len(spanColumns))
	for _, s := range rows {
		args = append(args,
			s.accountID.String(), s.envID.String(), s.runID, s.runQueuedAt,
			s.appID.String(), s.functionID.String(), s.name, s.startTime, s.endTime,
			s.traceID, s.spanID, nil, s.attributes, nil, nil, nil,
		)
	}
	query := batchInsertQuery(duckdb.DuckLakeAlias+".run_trace_spans", spanColumns, len(rows))
	_, err := d.db.ExecContext(ctx, query, args...)
	return err
}

// chunkedInserter wraps another spanInserter, splitting one logical batch
// into chunkSize-sized pieces and issuing one sequential insertBatch call per
// piece over the same underlying session/connection — trading a single large
// statement for several smaller ones. This exists to test whether it avoids
// quack's disproportionate PrepareRequest cost for one giant statement (every
// quack exec goes through a real SQL PREPARE — see quack_session.go's exec —
// and binding a single INSERT's multi-thousand-row literal VALUES list scales
// worse than linearly there, confirmed empirically: jsonlines holds ~42µs/row
// from batch=1000 to 10000, quack goes from ~9µs/row-marginal to ~860µs/row).
type chunkedInserter struct {
	inner     spanInserter
	chunkSize int
}

func (c chunkedInserter) insertBatch(ctx context.Context, rows []spanRow) error {
	for start := 0; start < len(rows); start += c.chunkSize {
		end := min(start+c.chunkSize, len(rows))
		if err := c.inner.insertBatch(ctx, rows[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// quackAppendInserter drives quack's native AppendRequest message
// (pkg/db/duckdb's QuackAppender, added specifically to answer the "why is
// quack so slow" investigation this benchmark started) — a binary DataChunk
// sent directly to the table, no SQL parsing/binding at all, unlike every
// other quack path here which goes through PrepareRequest.
type quackAppendInserter struct{ appender *duckdb.QuackAppender }

func (q quackAppendInserter) insertBatch(ctx context.Context, rows []spanRow) error {
	if len(rows) == 0 {
		return nil
	}
	for _, s := range rows {
		if err := q.appender.AppendRow(
			s.accountID.String(), s.envID.String(), s.runID, s.runQueuedAt,
			s.appID.String(), s.functionID.String(), s.name, s.startTime, s.endTime,
			s.traceID, s.spanID, nil, s.attributes, nil, nil, nil,
		); err != nil {
			return fmt.Errorf("appending row: %w", err)
		}
	}
	return q.appender.Flush(ctx)
}

// appenderInserter drives DuckDB's native Appender API (duckdb-go/v2 only —
// the C-level appender has no equivalent over the jsonlines/quack SQL-text
// transports), DuckDB's recommended bulk-load path in place of INSERT
// statements. conn is a dedicated driver.Conn opened directly against the
// same duckdbgo.Connector openEmbeddedConnector used for schema setup, not a
// connection borrowed from *sql.DB's pool — the Appender API needs to own its
// connection for its lifetime.
type appenderInserter struct{ conn driver.Conn }

func (a appenderInserter) insertBatch(ctx context.Context, rows []spanRow) error {
	if len(rows) == 0 {
		return nil
	}
	appender, err := duckdbgo.NewAppender(a.conn, duckdb.DuckLakeAlias, "main", "run_trace_spans")
	if err != nil {
		return fmt.Errorf("creating appender: %w", err)
	}
	for _, s := range rows {
		// Column order matches spanColumns. attributes is wrapped in
		// json.RawMessage so the appender's internal json.Marshal emits the
		// JSON text verbatim instead of re-encoding it as a quoted string —
		// see setJSON in duckdb-go/v2's vector_setters.go.
		if err := appender.AppendRow(
			duckdbgo.UUID(s.accountID), duckdbgo.UUID(s.envID), s.runID, s.runQueuedAt,
			duckdbgo.UUID(s.appID), duckdbgo.UUID(s.functionID), s.name, s.startTime, s.endTime,
			s.traceID, s.spanID, nil, json.RawMessage(s.attributes), nil, nil, nil,
		); err != nil {
			_ = appender.Close()
			return fmt.Errorf("appending row: %w", err)
		}
	}
	if err := appender.Close(); err != nil {
		return fmt.Errorf("flushing appender: %w", err)
	}
	return nil
}

// duckLakeOptions returns fresh DuckLakeOptions rooted under tb.TempDir(), so
// each opened database gets its own catalog/data directories. rowLimit sets
// DataInliningRowLimit — see BenchmarkInsertSpans' doc comment for why this
// is swept: writes at or below it stay inlined in the catalog, writes above
// it get flushed to a Parquet file, and that split has real throughput
// consequences batch size alone doesn't capture.
func duckLakeOptions(tb testing.TB, rowLimit int) *duckdb.DuckLakeOptions {
	dir := tb.TempDir()
	return &duckdb.DuckLakeOptions{
		CatalogPath:          filepath.Join(dir, "catalog.ducklake"),
		DataPath:             filepath.Join(dir, "data"),
		DataInliningRowLimit: rowLimit,
	}
}

// requireDuckDBBinary skips the benchmark if the duckdb CLI isn't on PATH —
// mirrors pkg/db/duckdb's requireDuckDBBinary skip-don't-fail convention.
func requireDuckDBBinary(tb testing.TB) string {
	tb.Helper()
	path, err := exec.LookPath("duckdb")
	if err != nil {
		tb.Skip("duckdb binary not found on PATH; skipping subprocess benchmark")
	}
	return path
}

// requireQuackExtension skips the benchmark if the quack extension can't be
// installed (no network, or an old duckdb build) — mirrors
// pkg/db/duckdb/quack_testutil_test.go's helper of the same name.
func requireQuackExtension(tb testing.TB, binPath string) {
	tb.Helper()
	out, err := exec.Command(binPath, ":memory:", "-c", "INSTALL quack; LOAD quack; SELECT 1;").CombinedOutput()
	if err != nil {
		tb.Skipf("quack extension unavailable (no network access or unsupported duckdb version); output: %s, err: %v", out, err)
	}
}

// freeLocalAddr resolves an ephemeral local port for quack_serve to bind.
func freeLocalAddr(tb testing.TB) string {
	tb.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("resolving free local address: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return fmt.Sprintf("127.0.0.1:%d", port)
}

// openJSONLines returns an open func for the default -jsonlines stdio
// transport, with the real inngest.run_trace_spans schema migrated in and
// DuckLake attached at rowLimit.
func openJSONLines(rowLimit int) func(testing.TB) spanInserter {
	return func(tb testing.TB) spanInserter {
		tb.Helper()
		binPath := requireDuckDBBinary(tb)
		db, err := duckdb.Open(tb.Context(), duckdb.Options{
			BinaryPath: binPath,
			DBFile:     ":memory:",
			DuckLake:   duckLakeOptions(tb, rowLimit),
		})
		if err != nil {
			tb.Fatalf("opening jsonlines db: %v", err)
		}
		tb.Cleanup(func() { _ = db.Close() })
		if err := duckdb.Migrate(tb.Context(), db); err != nil {
			tb.Fatalf("migrating jsonlines db: %v", err)
		}
		return dbInserter{db: db}
	}
}

// openQuack returns an open func for the quack HTTP transport, with the real
// inngest.run_trace_spans schema migrated in and DuckLake attached at
// rowLimit.
func openQuack(rowLimit int) func(testing.TB) spanInserter {
	return func(tb testing.TB) spanInserter {
		tb.Helper()
		binPath := requireDuckDBBinary(tb)
		requireQuackExtension(tb, binPath)
		addr := freeLocalAddr(tb)
		db, err := duckdb.Open(tb.Context(), duckdb.Options{
			BinaryPath: binPath,
			DBFile:     ":memory:",
			DuckLake:   duckLakeOptions(tb, rowLimit),
			QuackAddr:  &addr,
		})
		if err != nil {
			tb.Fatalf("opening quack db: %v", err)
		}
		tb.Cleanup(func() { _ = db.Close() })
		if err := duckdb.Migrate(tb.Context(), db); err != nil {
			tb.Fatalf("migrating quack db: %v", err)
		}
		return dbInserter{db: db}
	}
}

// openQuackChunked returns an open func like openQuack, but wraps the quack
// session in a chunkedInserter that splits each logical batch into
// chunkSize-sized sequential INSERTs — see chunkedInserter's doc comment for
// why (quack's per-request PrepareRequest cost scales worse than linearly
// with statement size).
func openQuackChunked(chunkSize, rowLimit int) func(testing.TB) spanInserter {
	return func(tb testing.TB) spanInserter {
		return chunkedInserter{inner: openQuack(rowLimit)(tb), chunkSize: chunkSize}
	}
}

// quackAppendColumns matches spanColumns' order and inngest.run_trace_spans'
// declared types exactly (pkg/db/duckdb/migrations/000001_baseline.sql) —
// AppendRequest has no wire-level column-name matching, only positional
// types, so this must stay in lockstep with spanColumns.
var quackAppendColumns = []duckdb.QuackColumnKind{
	duckdb.QuackColumnUUID, duckdb.QuackColumnUUID, duckdb.QuackColumnVarchar, duckdb.QuackColumnTimestampMS, // account_id, env_id, run_id, run_queued_at
	duckdb.QuackColumnUUID, duckdb.QuackColumnUUID, duckdb.QuackColumnVarchar, duckdb.QuackColumnTimestampMS, duckdb.QuackColumnTimestampMS, // app_id, function_id, name, start_time, end_time
	duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, // trace_id, span_id, parent_span_id
	duckdb.QuackColumnJSON, duckdb.QuackColumnJSON, duckdb.QuackColumnJSON, duckdb.QuackColumnJSON, // attributes, links, output, input
}

// openQuackAppend returns an open func for the quack transport (schema
// migrated in, same as openQuack) driving the AppendRequest path instead of
// PrepareRequest/INSERT, with DuckLake attached at rowLimit.
func openQuackAppend(rowLimit int) func(testing.TB) spanInserter {
	return func(tb testing.TB) spanInserter {
		tb.Helper()
		binPath := requireDuckDBBinary(tb)
		requireQuackExtension(tb, binPath)
		addr := freeLocalAddr(tb)
		db, err := duckdb.Open(tb.Context(), duckdb.Options{
			BinaryPath: binPath,
			DBFile:     ":memory:",
			DuckLake:   duckLakeOptions(tb, rowLimit),
			QuackAddr:  &addr,
		})
		if err != nil {
			tb.Fatalf("opening quack db: %v", err)
		}
		tb.Cleanup(func() { _ = db.Close() })
		if err := duckdb.Migrate(tb.Context(), db); err != nil {
			tb.Fatalf("migrating quack db: %v", err)
		}

		appender, err := duckdb.NewQuackAppender(tb.Context(), db, duckdb.DuckLakeAlias, "main", "run_trace_spans", quackAppendColumns)
		if err != nil {
			tb.Fatalf("creating quack appender: %v", err)
		}
		return quackAppendInserter{appender: appender}
	}
}

// generateQuackBenchToken and duckLakeAttachStmts mirror the exact
// INSTALL/LOAD/ATTACH sequence pkg/db/duckdb/process.go's
// bootstrapDuckLakeLocked runs, so the embedded path attaches the same way
// the subprocess paths do — same extension, same DATA_INLINING_ROW_LIMIT
// zero-value fallback (opts.DataInliningRowLimit <= 0 uses
// duckdb.DefaultDataInliningRowLimit, matching bootstrapDuckLakeLocked
// exactly).
func duckLakeAttachStmts(opts *duckdb.DuckLakeOptions) []string {
	rowLimit := opts.DataInliningRowLimit
	if rowLimit <= 0 {
		rowLimit = duckdb.DefaultDataInliningRowLimit
	}
	return []string{
		"INSTALL ducklake;",
		"LOAD ducklake;",
		fmt.Sprintf(
			"ATTACH IF NOT EXISTS 'ducklake:%s' AS %s (DATA_PATH '%s/', DATA_INLINING_ROW_LIMIT %d);",
			opts.CatalogPath, duckdb.DuckLakeAlias, strings.TrimSuffix(opts.DataPath, "/"), rowLimit,
		),
	}
}

// openEmbeddedConnector opens an in-process (no subprocess, no IPC) DuckDB
// instance via duckdb-go/v2's own Connector, with the real
// inngest.run_trace_spans schema migrated in. It returns both the pooled
// *sql.DB (for ATTACH/Migrate/INSERT) and the Connector itself: a caller that
// also needs a raw driver.Conn for the Appender API (openAppender) must open
// it via connector.Connect against this exact same Connector, not a second,
// independent one — only that guarantees the same underlying in-memory
// database and catalog/DuckLake attachment, the way duckdb-go/v2's own
// examples/appender opens both from one Connector.
func openEmbeddedConnector(tb testing.TB, rowLimit int) (*sql.DB, *duckdbgo.Connector) {
	tb.Helper()
	connector, err := duckdbgo.NewConnector(":memory:", nil)
	if err != nil {
		tb.Fatalf("creating embedded connector: %v", err)
	}
	db := sql.OpenDB(connector)
	tb.Cleanup(func() { _ = db.Close() })

	opts := duckLakeOptions(tb, rowLimit)
	if err := ensureDir(opts.DataPath); err != nil {
		tb.Fatalf("creating DuckLake data path: %v", err)
	}
	for _, stmt := range duckLakeAttachStmts(opts) {
		if _, err := db.ExecContext(tb.Context(), stmt); err != nil {
			tb.Fatalf("DuckLake bootstrap failed on %q: %v", stmt, err)
		}
	}

	if err := duckdb.Migrate(tb.Context(), db); err != nil {
		tb.Fatalf("migrating embedded db: %v", err)
	}
	return db, connector
}

// openEmbedded drives the embedded connection with plain INSERT INTO ...
// VALUES (...) statements, for a like-for-like comparison against the
// subprocess transports (openJSONLines/openQuack), which can only ever speak
// SQL text.
func openEmbedded(rowLimit int) func(testing.TB) spanInserter {
	return func(tb testing.TB) spanInserter {
		db, _ := openEmbeddedConnector(tb, rowLimit)
		return dbInserter{db: db}
	}
}

// openAppender drives the same embedded connection via DuckDB's native
// Appender API instead of INSERT statements — DuckDB's own recommended path
// for bulk loads, only reachable through duckdb-go/v2's cgo bindings, not
// over the jsonlines/quack SQL-text transports.
func openAppender(rowLimit int) func(testing.TB) spanInserter {
	return func(tb testing.TB) spanInserter {
		tb.Helper()
		_, connector := openEmbeddedConnector(tb, rowLimit)
		conn, err := connector.Connect(tb.Context())
		if err != nil {
			tb.Fatalf("opening raw appender connection: %v", err)
		}
		tb.Cleanup(func() { _ = conn.Close() })
		return appenderInserter{conn: conn}
	}
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// BenchmarkInsertSpans compares INSERT throughput into
// inngest.run_trace_spans across the -jsonlines subprocess transport, the
// quack HTTP subprocess transport (raw, chunked, and native AppendRequest),
// an embedded (no subprocess) duckdb-go/v2 connection using plain INSERT
// statements, and that same embedded connection using DuckDB's native
// Appender API, at a sweep of batch sizes and DuckLake
// DATA_INLINING_ROW_LIMIT values.
//
// The row limit matters independently of batch size: DuckLake inlines a
// write of at most that many rows directly in the catalog, and flushes
// anything larger to a standalone Parquet file instead (see
// pkg/db/duckdb.DuckLakeOptions.DataInliningRowLimit's doc comment) — so the
// same batch size can land on either side of that cost boundary depending on
// the limit, and the sweep below is what actually shows that transition
// rather than assuming a fixed default throughout.
//
// Run with:
//
//	cd cmd/duckdbbench && go test -bench=. -run='^$' ./...
func BenchmarkInsertSpans(b *testing.B) {
	transports := []struct {
		name string
		open func(rowLimit int) func(testing.TB) spanInserter
	}{
		{"jsonlines", openJSONLines},
		{"quack", openQuack},
		{"quack-append", openQuackAppend},
		{"quack-chunk100", func(rowLimit int) func(testing.TB) spanInserter { return openQuackChunked(100, rowLimit) }},
		{"quack-chunk500", func(rowLimit int) func(testing.TB) spanInserter { return openQuackChunked(500, rowLimit) }},
		{"embedded", openEmbedded},
		{"appender", openAppender},
	}
	rowLimits := []int{10, 100, 1000, 10000}
	batchSizes := []int{1, 100, 1000, 10000, 50000, 100000}
	// Plain "quack" drives every batch through one literal-VALUES
	// PrepareRequest, which Finding 1 established scales quadratically
	// with row count (~860µs/row marginal cost at batch=10000). Extrapolated
	// to 50000/100000 rows that's on the order of tens of minutes per
	// rowLimit sweep for a result we already know is catastrophic, so plain
	// quack is capped at the smaller sizes; every other transport (which
	// chunks, or avoids PrepareRequest altogether) runs the full sweep.
	quackBatchSizes := []int{1, 100, 1000, 10000}

	for _, tr := range transports {
		b.Run(tr.name, func(b *testing.B) {
			sizes := batchSizes
			if tr.name == "quack" {
				sizes = quackBatchSizes
			}
			for _, rowLimit := range rowLimits {
				b.Run(fmt.Sprintf("rowlimit=%d", rowLimit), func(b *testing.B) {
					for _, n := range sizes {
						b.Run(fmt.Sprintf("batch=%d", n), func(b *testing.B) {
							inserter := tr.open(rowLimit)(b)
							ctx := b.Context()

							i := 0
							for b.Loop() {
								rows := generateSpanBatch(n, i)
								if err := inserter.insertBatch(ctx, rows); err != nil {
									b.Fatalf("inserting batch of %d: %v", n, err)
								}
								i++
							}

							b.ReportMetric(float64(n)*float64(b.N)/b.Elapsed().Seconds(), "rows/sec")
						})
					}
				})
			}
		})
	}
}

// TestAppenderStoresIdenticalRowToSQLInsert guards the Appender path's
// value encoding — in particular, that wrapping attributes in
// json.RawMessage (appenderInserter.insertBatch) avoids the double-encoding
// setJSON's json.Marshal would otherwise produce for a bare Go string. It
// inserts the same synthetic row via both dbInserter (SQL INSERT) and
// appenderInserter (Appender API) against one embedded connection and
// asserts every column reads back identical.
func TestAppenderStoresIdenticalRowToSQLInsert(t *testing.T) {
	db, connector := openEmbeddedConnector(t, duckdb.DefaultDataInliningRowLimit)
	conn, err := connector.Connect(t.Context())
	if err != nil {
		t.Fatalf("opening raw appender connection: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	sqlRow := generateSpanBatch(1, 0)
	appenderRow := generateSpanBatch(1, 1)

	if err := (dbInserter{db: db}).insertBatch(t.Context(), sqlRow); err != nil {
		t.Fatalf("SQL insert: %v", err)
	}
	if err := (appenderInserter{conn: conn}).insertBatch(t.Context(), appenderRow); err != nil {
		t.Fatalf("appender insert: %v", err)
	}

	// duckdb-go/v2 decodes the JSON column natively into a Go
	// map[string]any (unlike the jsonlines/quack transports, which surface
	// it as raw JSON text) — scan into `any` and compare structurally rather
	// than expecting a particular Go type back.
	fetch := func(runID string) (name string, attributes any) {
		row := db.QueryRowContext(t.Context(),
			"SELECT name, attributes FROM "+duckdb.DuckLakeAlias+".run_trace_spans WHERE run_id = ?;", runID)
		if err := row.Scan(&name, &attributes); err != nil {
			t.Fatalf("querying row for run_id %q: %v", runID, err)
		}
		return name, attributes
	}

	sqlName, sqlAttrs := fetch(sqlRow[0].runID)
	appenderName, appenderAttrs := fetch(appenderRow[0].runID)

	if sqlName != appenderName {
		t.Errorf("name mismatch: sql=%q appender=%q", sqlName, appenderName)
	}
	if !reflect.DeepEqual(sqlAttrs, appenderAttrs) {
		t.Errorf("attributes mismatch: sql=%#v appender=%#v", sqlAttrs, appenderAttrs)
	}

	var want map[string]any
	if err := json.Unmarshal([]byte(sqlRow[0].attributes), &want); err != nil {
		t.Fatalf("unmarshaling source attributes: %v", err)
	}
	if !reflect.DeepEqual(appenderAttrs, want) {
		t.Errorf("appender attributes diverged from the source row: got %#v, want %#v", appenderAttrs, want)
	}
}
