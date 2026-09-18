package duckdbbench

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// appenderBenchTables are every table openAppenderBenchDB creates — the
// appender-only analogues of merge_bench_test.go's benchTables, each backed
// by an appender-based write path instead of a literal-SQL one:
// bench_runs_flat_appender (flatInsertAppenderBatch, QuackAppender),
// bench_runs_merged_appender (mergeUpsertAppenderBatch, QuackMergeAppender),
// bench_runs_merged_jsonpatch_appender (mergeJSONPatchAppenderBatch,
// QuackMergeAppender with a json_merge_patch UPDATE SET).
var appenderBenchTables = []string{
	"bench_runs_flat_appender",
	"bench_runs_merged_appender",
	"bench_runs_merged_jsonpatch_appender",
}

// openAppenderBenchDB opens a DuckDB instance the same way openMergeBenchDB
// does — including the same Postgres-catalog switch (see
// requirePostgresCatalogBench/postgres_catalog.go) — but creates
// appenderBenchTables instead of benchTables: this file's benchmark only
// exercises the appender-based write paths (QuackAppender/
// QuackMergeAppender), never the literal-interpolated-SQL ones
// merge_bench_test.go's scenarios use.
func openAppenderBenchDB(tb testing.TB) *sql.DB {
	tb.Helper()
	return openAppenderBenchDBWithConns(tb, benchQuackConns)
}

// openAppenderBenchDBWithConns is openAppenderBenchDB with an explicit
// Options.QuackConns instead of the fixed benchQuackConns —
// BenchmarkAppenderWritesByConcurrency's one caller, sweeping concurrency
// levels the same way sqlite_bench_test.go's
// openAppenderBenchDBSQLiteWithConns and quack_catalog_bench_test.go's
// openAppenderBenchDBQuackCatalogWithConns do for their catalogs. Selects
// file or Postgres the same way openAppenderBenchDB always has —
// whichever this run's requirePostgresCatalogBench env vars resolve to.
func openAppenderBenchDBWithConns(tb testing.TB, conns int) *sql.DB {
	tb.Helper()

	var opts driver.Options
	if pg, ok := requirePostgresCatalogBench(tb); ok {
		opts = driver.Options{
			BinaryPath:              pg.binaryPath,
			DBFile:                  ":memory:",
			DuckLake:                pg.duckLakeOptions(filepath.Join(tb.TempDir(), "data")),
			AllowUnsignedExtensions: true,
			LocalExtensionPaths:     pg.localExtensionPaths(),
			QuackConns:              conns,
		}
	} else {
		binPath := requireDuckDBBinary(tb)
		requireQuackExtension(tb, binPath)
		opts = driver.Options{
			BinaryPath: binPath,
			DBFile:     ":memory:",
			DuckLake:   duckLakeOptions(tb, -1),
			QuackConns: conns,
		}
	}
	addr := freeLocalAddr(tb)
	opts.QuackAddr = &addr

	db, err := driver.Open(tb.Context(), opts)
	if err != nil {
		tb.Fatalf("opening subprocess db: %v", err)
	}
	tb.Cleanup(func() { _ = db.Close() })

	for _, table := range appenderBenchTables {
		schema := fmt.Sprintf(`CREATE TABLE %s.%s (
			account_id UUID NOT NULL,
			run_id VARCHAR NOT NULL,
			status VARCHAR NOT NULL,
			queued_at TIMESTAMP_MS NOT NULL,
			started_at TIMESTAMP_MS,
			ended_at TIMESTAMP_MS,
			attributes JSON NOT NULL,
			updated_at TIMESTAMP_MS NOT NULL
		);`, driver.DuckLakeAlias, table)
		if _, err := db.ExecContext(tb.Context(), schema); err != nil {
			tb.Fatalf("creating %s: %v", table, err)
		}
		partition := fmt.Sprintf("ALTER TABLE %s.%s SET PARTITIONED BY (account_id);", driver.DuckLakeAlias, table)
		if _, err := db.ExecContext(tb.Context(), partition); err != nil {
			tb.Fatalf("partitioning %s: %v", table, err)
		}
	}
	return db
}

// flatInsertAppenderBatch is flatInsertBatch's QuackAppender counterpart:
// appends events as new rows into bench_runs_flat_appender via quack's
// SEND_DATA_REQUEST mechanism instead of a literal-interpolated INSERT.
func flatInsertAppenderBatch(ctx context.Context, db *sql.DB, events []runEvent) error {
	if len(events) == 0 {
		return nil
	}
	appender, err := driver.NewQuackAppender(ctx, db, driver.DuckLakeAlias, "main", "bench_runs_flat_appender", runQuackColumnKinds)
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

// runQuackColumnKinds is runColumns' physical wire type, in the same order —
// shared by every appender-based write path in this file.
var runQuackColumnKinds = []driver.QuackColumnKind{
	driver.QuackColumnUUID,        // account_id
	driver.QuackColumnVarchar,     // run_id
	driver.QuackColumnVarchar,     // status
	driver.QuackColumnTimestampMS, // queued_at
	driver.QuackColumnTimestampMS, // started_at
	driver.QuackColumnTimestampMS, // ended_at
	driver.QuackColumnJSON,        // attributes
	driver.QuackColumnTimestampMS, // updated_at
}

// runQuackMergeColumns is runColumns named for QuackMergeConfig's On/
// UpdateSet SQL to reference via "s.<name>" — shared by every
// QuackMergeAppender-based write path in this file.
var runQuackMergeColumns = []driver.QuackMergeColumn{
	{Name: "account_id", Kind: driver.QuackColumnUUID},
	{Name: "run_id", Kind: driver.QuackColumnVarchar},
	{Name: "status", Kind: driver.QuackColumnVarchar},
	{Name: "queued_at", Kind: driver.QuackColumnTimestampMS},
	{Name: "started_at", Kind: driver.QuackColumnTimestampMS},
	{Name: "ended_at", Kind: driver.QuackColumnTimestampMS},
	{Name: "attributes", Kind: driver.QuackColumnJSON},
	{Name: "updated_at", Kind: driver.QuackColumnTimestampMS},
}

// mergeAppenderBatch upserts events into table via QuackMergeAppender —
// shared by mergeUpsertAppenderBatch and mergeJSONPatchAppenderBatch, which
// only differ in attributesSet (a plain overwrite vs. a json_merge_patch
// fold), exactly mirroring merge_bench_test.go's mergeBatch/
// mergeUpsertBatch/mergeJSONPatchBatch split.
//
// DedupKeys/DedupOrderBy are set unconditionally, even though the
// tick-based generators (generateInterleavedBatches et al.) never produce a
// batch with two events for the same (account_id, run_id): the random-
// arrival generator (arrival_bench_test.go) can, and this function is
// shared by both, so it's simplest and cheapest to always let the MERGE
// itself dedup on (account_id, run_id) — keeping whichever same-key row has
// the greatest updated_at — rather than have callers reason about which
// generator they're paired with.
func mergeAppenderBatch(ctx context.Context, db *sql.DB, table, attributesSet string, events []runEvent) error {
	if len(events) == 0 {
		return nil
	}
	appender, err := driver.NewQuackMergeAppender(ctx, db, driver.DuckLakeAlias, "main", table,
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

func mergeUpsertAppenderBatch(ctx context.Context, db *sql.DB, events []runEvent) error {
	return mergeAppenderBatch(ctx, db, "bench_runs_merged_appender", "s.attributes", events)
}

func mergeJSONPatchAppenderBatch(ctx context.Context, db *sql.DB, events []runEvent) error {
	return mergeAppenderBatch(ctx, db, "bench_runs_merged_jsonpatch_appender", "json_merge_patch(t.attributes, s.attributes)", events)
}

// appenderInsertScenarios are the appender-based analogues of
// merge_bench_test.go's insertScenarios: same three write shapes (flat,
// merge, merge-json-patch), but every one goes through an appender
// (QuackAppender/QuackMergeAppender) instead of a literal-interpolated SQL
// statement.
var appenderInsertScenarios = []struct {
	name  string
	apply func(ctx context.Context, db *sql.DB, events []runEvent) error
}{
	{"flat", flatInsertAppenderBatch},
	{"merge", mergeUpsertAppenderBatch},
	{"merge-json-patch", mergeJSONPatchAppenderBatch},
}

// BenchmarkAppenderWrites100k is a basic, single-threaded (no parallel
// shards, no partition sweep) 100,000-row pass over every appenderInsertScenarios
// entry — the appender-only counterpart to merge_bench_test.go's
// BenchmarkRunsInsert, deliberately excluding every literal-SQL write path.
// Run with:
//
//	cd cmd/duckdbbench && go test -bench BenchmarkAppenderWrites100k -run '^$' -benchtime=1x .            # file catalog
//	DUCKLAKE_SELFBUILT_DUCKDB=<path> DUCKLAKE_SELFBUILT_POSTGRES_SCANNER=<path> \
//	DUCKLAKE_SELFBUILT_QUACK=<path> DUCKLAKE_SELFBUILT_HTTPFS=<path> \
//	DUCKLAKE_BENCH_POSTGRES_URI=postgres://user:pass@host:5432/db \
//	go test -bench BenchmarkAppenderWrites100k -run '^$' -benchtime=1x .                                   # postgres catalog
func BenchmarkAppenderWrites100k(b *testing.B) {
	const totalRows = 100_000
	totalRuns := totalRows / eventsPerRun
	accountID := uuid.New()

	for _, sc := range appenderInsertScenarios {
		b.Run(sc.name, func(b *testing.B) {
			for b.Loop() {
				db := openAppenderBenchDB(b)
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

// runAppenderInsertScenarios is merge_bench_test.go's runInsertScenarios,
// appender-only: same sweep shape (one b.Run per appenderInsertScenarios
// entry, same genFn-driven concurrent shards), against openDB's tables
// instead of openMergeBenchDB's — openAppenderBenchDB for the file/postgres
// sweep (BenchmarkAppenderInsertByPartitions),
// openAppenderBenchDBSQLite for the SQLite-catalog one
// (BenchmarkAppenderInsertByPartitionsSQLite). genFn's own conflict-retry
// wrapping (generateInterleavedBatchesParallel/
// generatePartitionAlignedBatchesParallel's applyWithConflictRetry) applies
// unchanged — DuckLake's optimistic-concurrency conflicts are a property of
// concurrent commits against the same table, not of which wire mechanism
// produced them or which catalog backs it.
func runAppenderInsertScenarios(b *testing.B, totalRuns int, accountIDs []uuid.UUID, genFn batchGenerator, openDB func(testing.TB) *sql.DB) {
	for _, sc := range appenderInsertScenarios {
		b.Run(sc.name, func(b *testing.B) {
			iter := 0
			for b.Loop() {
				db := openDB(b)
				ctx := b.Context()
				err := genFn(iter, totalRuns, benchPipelineDepth, benchBatchSize, accountIDs, func(batch []runEvent) error {
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
}

// BenchmarkAppenderInsertByPartitions is merge_bench_test.go's
// BenchmarkRunsInsertByPartitions, appender-only: the same insertModes x
// benchPartitionCounts sweep, rows held fixed at benchPartitionSweepRows,
// with every scenario going through an appender (QuackAppender/
// QuackMergeAppender, appenderInsertScenarios) instead of literal SQL. Run
// the same way as BenchmarkAppenderWrites100k (see its doc comment) —
// substitute this benchmark's name and, for a 100k-row sweep matching
// FINDINGS/ducklake-postgres-catalog-100k.md's convention, temporarily set
// benchPartitionSweepRows to 100_000.
func BenchmarkAppenderInsertByPartitions(b *testing.B) {
	totalRuns := benchPartitionSweepRows / eventsPerRun
	for _, mode := range insertModes {
		b.Run(mode.name, func(b *testing.B) {
			for _, numPartitions := range benchPartitionCounts {
				accountIDs := generateAccountIDs(numPartitions)
				b.Run(fmt.Sprintf("partitions=%d", numPartitions), func(b *testing.B) {
					runAppenderInsertScenarios(b, totalRuns, accountIDs, mode.gen, openAppenderBenchDB)
				})
			}
		})
	}
}

// BenchmarkAppenderInsertByPartitionsSQLite is BenchmarkAppenderInsertByPartitions'
// SQLite-catalog counterpart — see sqlite_bench_test.go's package doc
// comment for what that catalog mode is. Run the same way as
// BenchmarkAppenderWrites100k (see its doc comment), substituting this
// benchmark's name.
func BenchmarkAppenderInsertByPartitionsSQLite(b *testing.B) {
	totalRuns := benchPartitionSweepRows / eventsPerRun
	for _, mode := range insertModes {
		b.Run(mode.name, func(b *testing.B) {
			for _, numPartitions := range benchPartitionCounts {
				accountIDs := generateAccountIDs(numPartitions)
				b.Run(fmt.Sprintf("partitions=%d", numPartitions), func(b *testing.B) {
					runAppenderInsertScenarios(b, totalRuns, accountIDs, mode.gen, openAppenderBenchDBSQLite)
				})
			}
		})
	}
}

// BenchmarkAppenderWritesByConcurrency is
// BenchmarkAppenderWritesSQLiteByConcurrency's file/Postgres-catalog
// counterpart: same benchConcurrencyLevels sweep, fixed 100k rows and 50
// accounts, interleaved (round-robin, not partition-aligned) account_id
// assignment via generateInterleavedBatchesParallelN — against whichever
// catalog openAppenderBenchDBWithConns resolves to (file by default,
// Postgres when requirePostgresCatalogBench's env vars are set — run this
// benchmark once each way, the same convention every other file/Postgres
// benchmark in this package already uses). A reference point for the
// SQLite/quack-catalog concurrency sweeps: unlike either of those, neither
// file nor Postgres has a single-writer-style lock, so this is expected to
// degrade only from DuckLake's ordinary optimistic-concurrency conflicts
// (retried via applyWithConflictRetry), not fail outright at low
// concurrency. Run with:
//
//	cd cmd/duckdbbench && go test -bench BenchmarkAppenderWritesByConcurrency -run '^$' -benchtime=1x -timeout 30m .
func BenchmarkAppenderWritesByConcurrency(b *testing.B) {
	const totalRows = 100_000
	totalRuns := totalRows / eventsPerRun
	accountIDs := generateAccountIDs(50)

	for _, conns := range benchConcurrencyLevels {
		b.Run(fmt.Sprintf("concurrency=%d", conns), func(b *testing.B) {
			for _, sc := range appenderInsertScenarios {
				b.Run(sc.name, func(b *testing.B) {
					iter := 0
					for b.Loop() {
						db := openAppenderBenchDBWithConns(b, conns)
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
