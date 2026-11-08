package duckdbbench

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// This file adds a fourth catalog mode to the appender-based benchmarks
// alongside file (openAppenderBenchDB's default), Postgres
// (requirePostgresCatalogBench), and in-memory (inmemory_bench_test.go): a
// real DuckLake catalog backed by a SQLite file instead of DuckDB's own
// catalog file format or a Postgres server. Unlike Postgres, this needs
// nothing beyond the standard pinned duckdb binary and its sqlite
// extension — no self-built duckdb/ducklake/quack, no external server — see
// pkg/db/duckdb's ducklake_sqlite_test.go, which confirms the standard
// pinned build already supports it. It's otherwise a full DuckLake catalog
// (account_id partitioning included), so this reuses appenderInsertScenarios
// and driver.DuckLakeAlias directly, unlike the in-memory mode's separate
// no-catalog scenario set.

// openAppenderBenchDBSQLite is openAppenderBenchDB's SQLite-catalog
// counterpart: same appenderBenchTables shape (schema and account_id
// partitioning included), quack transport, and benchQuackConns concurrency,
// but Options.DuckLake.SQLiteCatalogPath instead of CatalogPath or
// PostgresCatalogURI.
func openAppenderBenchDBSQLite(tb testing.TB) *sql.DB {
	tb.Helper()
	return openAppenderBenchDBSQLiteWithConns(tb, benchQuackConns)
}

// openAppenderBenchDBSQLiteWithConns is openAppenderBenchDBSQLite with an
// explicit Options.QuackConns instead of the fixed benchQuackConns —
// BenchmarkAppenderWritesSQLiteByConcurrency's one caller, sweeping
// concurrency levels against a SQLite catalog specifically to see how its
// single-writer file lock (DuckLake's SQLiteMetadataManager treats
// "database is locked" as a retryable commit error — see
// sqlite_metadata_manager.cpp) affects throughput as concurrent writers
// increase.
func openAppenderBenchDBSQLiteWithConns(tb testing.TB, conns int) *sql.DB {
	tb.Helper()

	binPath := requireDuckDBBinary(tb)
	requireQuackExtension(tb, binPath)
	dir := tb.TempDir()
	addr := freeLocalAddr(tb)
	opts := driver.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &driver.DuckLakeOptions{
			SQLiteCatalogPath: filepath.Join(dir, "catalog.sqlite"),
			DataPath:          filepath.Join(dir, "data"),
			// Matches file/postgres modes' convention (duckLakeOptions/
			// postgresCatalogEnv.duckLakeOptions): -1 explicitly disables
			// inlining, so every write flushes straight to Parquet rather
			// than staying inlined in the catalog.
			DataInliningRowLimit: -1,
		},
		QuackAddr:  &addr,
		QuackConns: conns,
	}

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

// BenchmarkAppenderWritesSQLite100k is BenchmarkAppenderWrites100k's
// SQLite-catalog counterpart: the same 100,000-row, single-threaded pass
// over every appenderInsertScenarios entry, against a DuckLake catalog
// backed by a SQLite file instead of a DuckDB one. Run with:
//
//	cd cmd/duckdbbench && go test -bench BenchmarkAppenderWritesSQLite100k -run '^$' -benchtime=1x .
func BenchmarkAppenderWritesSQLite100k(b *testing.B) {
	const totalRows = 100_000
	totalRuns := totalRows / eventsPerRun
	accountID := uuid.New()

	for _, sc := range appenderInsertScenarios {
		b.Run(sc.name, func(b *testing.B) {
			for b.Loop() {
				db := openAppenderBenchDBSQLite(b)
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

// benchConcurrencyLevels sweeps how many concurrent shards
// BenchmarkAppenderWritesSQLiteByConcurrency splits a fixed 100k-row write
// into, from effectively single-threaded up past the benchQuackConns
// default this file's other benchmarks use — wide enough to show whether
// (and how sharply) throughput plateaus or regresses once SQLite's
// single-writer file lock becomes the bottleneck rather than benchQuackConns
// happening to already be past that point.
var benchConcurrencyLevels = []int{1, 2, 4, 8, 16, 32}

// BenchmarkAppenderWritesSQLiteByConcurrency holds row count (100k) and
// account_id cardinality fixed and instead sweeps concurrency: how many
// shards generateInterleavedBatchesParallelN splits the write into, each on
// its own quack connection (Options.QuackConns matched to the same count via
// openAppenderBenchDBSQLiteWithConns). "interleaved" account_id assignment
// (round-robin over 50 accounts, not partition-aligned) is used throughout,
// since the point is to measure catalog-commit contention specifically, not
// how partition alignment changes it — see
// generateInterleavedBatchesParallelN's doc comment for why interleaved
// writers already contend on DuckLake's own optimistic-concurrency conflicts
// even with disjoint run_id ranges; a SQLite catalog adds a second,
// coarser-grained contention point on top (its own file lock, serializing
// every commit regardless of which table/partition it touches).
//
// Every concurrency level writes into a fresh SQLite catalog file
// (openAppenderBenchDBSQLiteWithConns's tb.TempDir()), so results are
// comparable across levels rather than compounding on a growing catalog.
// Run with:
//
//	cd cmd/duckdbbench && go test -bench BenchmarkAppenderWritesSQLiteByConcurrency -run '^$' -benchtime=1x -timeout 30m .
func BenchmarkAppenderWritesSQLiteByConcurrency(b *testing.B) {
	const totalRows = 100_000
	totalRuns := totalRows / eventsPerRun
	accountIDs := generateAccountIDs(50)

	for _, conns := range benchConcurrencyLevels {
		b.Run(fmt.Sprintf("concurrency=%d", conns), func(b *testing.B) {
			for _, sc := range appenderInsertScenarios {
				b.Run(sc.name, func(b *testing.B) {
					iter := 0
					for b.Loop() {
						db := openAppenderBenchDBSQLiteWithConns(b, conns)
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
