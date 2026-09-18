package duckdbbench

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// This file adds a third catalog mode to the appender-based benchmarks
// alongside file (openAppenderBenchDB's default) and Postgres
// (requirePostgresCatalogBench): no DuckLake catalog at all — a plain,
// native DuckDB :memory: database (Options.DuckLake left nil). It's a
// baseline: how much of DuckLake's write cost (its own metadata
// bookkeeping, Parquet flushes, optimistic-concurrency conflict handling)
// disappears once there's no lakehouse catalog in the loop at all, just
// DuckDB's own native in-memory tables.
//
// account_id partitioning (ALTER TABLE ... SET PARTITIONED BY, DuckLake-only
// syntax) is skipped entirely here — there's no DuckLake catalog to
// partition, and a native in-memory table has no file layout for it to
// affect anyway.

// openAppenderBenchDBInMemory is openAppenderBenchDB's in-memory-catalog
// counterpart: same appenderBenchTables shape, quack transport, and
// benchQuackConns concurrency, but Options.DuckLake is left nil — writes
// land directly in a native DuckDB table under the default catalog's "main"
// schema, not a DuckLake-managed one.
func openAppenderBenchDBInMemory(tb testing.TB) *sql.DB {
	tb.Helper()
	return openAppenderBenchDBInMemoryWithConns(tb, benchQuackConns)
}

// openAppenderBenchDBInMemoryWithConns is openAppenderBenchDBInMemory with
// an explicit Options.QuackConns instead of the fixed benchQuackConns —
// BenchmarkAppenderWritesInMemoryByConcurrency's one caller, sweeping
// concurrency levels the same way the other catalog modes' *WithConns
// helpers do.
func openAppenderBenchDBInMemoryWithConns(tb testing.TB, conns int) *sql.DB {
	tb.Helper()

	binPath := requireDuckDBBinary(tb)
	requireQuackExtension(tb, binPath)
	addr := freeLocalAddr(tb)
	opts := driver.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		QuackAddr:  &addr,
		QuackConns: conns,
	}

	db, err := driver.Open(tb.Context(), opts)
	if err != nil {
		tb.Fatalf("opening subprocess db: %v", err)
	}
	tb.Cleanup(func() { _ = db.Close() })

	for _, table := range appenderBenchTables {
		schema := fmt.Sprintf(`CREATE TABLE main.%s (
			account_id UUID NOT NULL,
			run_id VARCHAR NOT NULL,
			status VARCHAR NOT NULL,
			queued_at TIMESTAMP_MS NOT NULL,
			started_at TIMESTAMP_MS,
			ended_at TIMESTAMP_MS,
			attributes JSON NOT NULL,
			updated_at TIMESTAMP_MS NOT NULL
		);`, table)
		if _, err := db.ExecContext(tb.Context(), schema); err != nil {
			tb.Fatalf("creating %s: %v", table, err)
		}
	}
	return db
}

// flatInsertAppenderBatchInMemory is flatInsertAppenderBatch's in-memory-
// catalog counterpart: identical write path, except catalog is "" (no
// DuckLake catalog to USE — the appender operates against the connection's
// already-active default catalog) instead of driver.DuckLakeAlias.
func flatInsertAppenderBatchInMemory(ctx context.Context, db *sql.DB, events []runEvent) error {
	if len(events) == 0 {
		return nil
	}
	appender, err := driver.NewQuackAppender(ctx, db, "", "main", "bench_runs_flat_appender", runQuackColumnKinds)
	if err != nil {
		return err
	}
	for _, e := range events {
		if err := appender.AppendRow(runEventArgs(e)...); err != nil {
			return err
		}
	}
	return appender.Close(ctx)
}

// mergeAppenderBatchInMemory is mergeAppenderBatch's in-memory-catalog
// counterpart — see flatInsertAppenderBatchInMemory's doc comment for the
// catalog difference; DedupKeys/DedupOrderBy are still needed here for the
// same reason mergeAppenderBatch sets them (see its doc comment).
func mergeAppenderBatchInMemory(ctx context.Context, db *sql.DB, table, attributesSet string, events []runEvent) error {
	if len(events) == 0 {
		return nil
	}
	appender, err := driver.NewQuackMergeAppender(ctx, db, "", "main", table,
		driver.QuackMergeConfig{
			On:           "t.account_id = s.account_id AND t.run_id = s.run_id",
			UpdateSet:    fmt.Sprintf("status = s.status, started_at = s.started_at, ended_at = s.ended_at, attributes = %s, updated_at = s.updated_at", attributesSet),
			DedupKeys:    []string{"account_id", "run_id"},
			DedupOrderBy: "updated_at",
		},
		runQuackMergeColumns)
	if err != nil {
		return err
	}
	for _, e := range events {
		if err := appender.AppendRow(runEventArgs(e)...); err != nil {
			return err
		}
	}
	return appender.Close(ctx)
}

func mergeUpsertAppenderBatchInMemory(ctx context.Context, db *sql.DB, events []runEvent) error {
	return mergeAppenderBatchInMemory(ctx, db, "bench_runs_merged_appender", "s.attributes", events)
}

func mergeJSONPatchAppenderBatchInMemory(ctx context.Context, db *sql.DB, events []runEvent) error {
	return mergeAppenderBatchInMemory(ctx, db, "bench_runs_merged_jsonpatch_appender", "json_merge_patch(t.attributes, s.attributes)", events)
}

// inMemoryInsertScenarios is appenderInsertScenarios' in-memory-catalog
// counterpart.
var inMemoryInsertScenarios = []struct {
	name  string
	apply func(ctx context.Context, db *sql.DB, events []runEvent) error
}{
	{"flat", flatInsertAppenderBatchInMemory},
	{"merge", mergeUpsertAppenderBatchInMemory},
	{"merge-json-patch", mergeJSONPatchAppenderBatchInMemory},
}

// BenchmarkAppenderWritesInMemory100k is BenchmarkAppenderWrites100k's
// in-memory-catalog counterpart: the same 100,000-row, single-threaded pass
// over every scenario, but against a plain native DuckDB :memory: database
// instead of a DuckLake-managed one (file or Postgres) — a baseline for how
// much of DuckLake's write cost disappears with no lakehouse catalog in the
// loop at all. Run with:
//
//	cd cmd/duckdbbench && go test -bench BenchmarkAppenderWritesInMemory100k -run '^$' -benchtime=1x .
func BenchmarkAppenderWritesInMemory100k(b *testing.B) {
	const totalRows = 100_000
	totalRuns := totalRows / eventsPerRun
	accountID := uuid.New()

	for _, sc := range inMemoryInsertScenarios {
		b.Run(sc.name, func(b *testing.B) {
			for b.Loop() {
				db := openAppenderBenchDBInMemory(b)
				ctx := b.Context()
				err := generateInterleavedBatches(0, totalRuns, benchPipelineDepth, benchBatchSize, 0,
					func(int) uuid.UUID { return accountID },
					func(batch []runEvent) error { return sc.apply(ctx, db, batch) })
				if err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(totalRuns*eventsPerRun)*float64(b.N)/b.Elapsed().Seconds(), "rows/sec")
		})
	}
}

// BenchmarkAppenderWritesInMemoryByConcurrency is
// BenchmarkAppenderWritesSQLiteByConcurrency's in-memory-catalog
// counterpart: same benchConcurrencyLevels sweep, fixed 100k rows and 50
// accounts, interleaved (round-robin) account_id assignment via
// generateInterleavedBatchesParallelN, against inMemoryInsertScenarios
// instead of appenderInsertScenarios (no DuckLake catalog, no
// driver.DuckLakeAlias). A baseline reference point: no catalog at all
// means no DuckLake optimistic-concurrency conflicts and no catalog-level
// lock, so this is expected to scale roughly linearly with concurrency
// rather than degrade the way every DuckLake-backed catalog mode does. Run
// with:
//
//	cd cmd/duckdbbench && go test -bench BenchmarkAppenderWritesInMemoryByConcurrency -run '^$' -benchtime=1x -timeout 30m .
func BenchmarkAppenderWritesInMemoryByConcurrency(b *testing.B) {
	const totalRows = 100_000
	totalRuns := totalRows / eventsPerRun
	accountIDs := generateAccountIDs(50)

	for _, conns := range benchConcurrencyLevels {
		b.Run(fmt.Sprintf("concurrency=%d", conns), func(b *testing.B) {
			for _, sc := range inMemoryInsertScenarios {
				b.Run(sc.name, func(b *testing.B) {
					iter := 0
					for b.Loop() {
						db := openAppenderBenchDBInMemoryWithConns(b, conns)
						ctx := b.Context()
						err := generateInterleavedBatchesParallelN(iter, totalRuns, benchPipelineDepth, benchBatchSize, conns, accountIDs, func(batch []runEvent) error {
							return sc.apply(ctx, db, batch)
						})
						if err != nil {
							b.Fatal(err)
						}
						iter++
					}
					b.ReportMetric(float64(totalRuns*eventsPerRun)*float64(b.N)/b.Elapsed().Seconds(), "rows/sec")
				})
			}
		})
	}
}
