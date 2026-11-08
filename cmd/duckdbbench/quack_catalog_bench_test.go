package duckdbbench

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// This file adds a fifth catalog mode to the appender-based benchmarks
// alongside file (openAppenderBenchDB's default), Postgres
// (requirePostgresCatalogBench), in-memory (inmemory_bench_test.go), and
// SQLite (sqlite_bench_test.go): a DuckLake catalog backed by a *remote*
// DuckDB instance — its metadata living in that instance's own memory, not
// on this process's disk at all — accessed over the quack wire protocol.
// DuckLake ships this as a first-class metadata backend
// (pkg/db/duckdb/process.go's DuckLakeOptions.QuackCatalogAddr/
// QuackCatalogToken; see ducklake_quack_test.go for the driver-level
// confirmation this actually works, including its optional server-side
// commit fast path when the remote instance also has ducklake loaded).
//
// Unlike Postgres, this needs no external server process outside our own
// control — the "remote" metadata instance is just a second duckdb
// subprocess this file spawns itself (openQuackCatalogMetadataServer),
// exactly the same binary/extensions as everything else here.

// selfBuiltQuackEnv holds DUCKLAKE_SELFBUILT_DUCKDB/QUACK/HTTPFS — the same
// three env vars postgres_catalog.go's requirePostgresCatalogBench reads,
// without that function's additional postgres_scanner/live-postgres
// requirements, since a quack-catalog run needs none of that. See
// requireSelfBuiltQuack's doc comment for why this exists at all: the
// standard pinned duckdb's published quack extension has a known wire-
// protocol concurrency bug (root-caused this session; fixed on quack commit
// f4328c5333), which surfaces here as soon as the metadata server sees
// genuinely concurrent DuckLake metadata queries — not a new limitation,
// the same one FINDINGS/ducklake-postgres-catalog-100k.md documents.
type selfBuiltQuackEnv struct {
	binaryPath    string
	quackExtPath  string
	httpfsExtPath string
}

// localExtensionPaths is openAppenderBenchDBQuackCatalogWithConns/
// openQuackCatalogMetadataServer's Options.LocalExtensionPaths when built
// from a selfBuiltQuackEnv — no "postgres" entry, unlike
// postgresCatalogEnv's, since a quack catalog never touches Postgres.
func (env selfBuiltQuackEnv) localExtensionPaths() map[string]string {
	return map[string]string{
		"ducklake": "ducklake",
		"quack":    env.quackExtPath,
		"httpfs":   env.httpfsExtPath,
	}
}

// requireSelfBuiltQuack reports whether this run should use the self-built,
// concurrency-bug-fixed duckdb/quack/httpfs stack (DUCKLAKE_SELFBUILT_DUCKDB/
// QUACK/HTTPFS) for both the quack-catalog metadata server and the main,
// attaching subprocess, instead of the ordinary pinned duckdb binary and its
// published quack extension. All three env vars must be set (and exist on
// disk), or this returns false and every caller falls back to the standard
// binary — this switch is opt-in, so BenchmarkAppenderWritesQuackCatalog100k
// (concurrency=1, unaffected by the bug) keeps working out of the box with
// no self-built binaries required.
func requireSelfBuiltQuack(tb testing.TB) (selfBuiltQuackEnv, bool) {
	tb.Helper()

	env := selfBuiltQuackEnv{
		binaryPath:    os.Getenv("DUCKLAKE_SELFBUILT_DUCKDB"),
		quackExtPath:  os.Getenv("DUCKLAKE_SELFBUILT_QUACK"),
		httpfsExtPath: os.Getenv("DUCKLAKE_SELFBUILT_HTTPFS"),
	}
	if env.binaryPath == "" || env.quackExtPath == "" || env.httpfsExtPath == "" {
		return selfBuiltQuackEnv{}, false
	}
	for _, p := range []string{env.binaryPath, env.quackExtPath, env.httpfsExtPath} {
		if _, err := os.Stat(p); err != nil {
			tb.Fatalf("DUCKLAKE_SELFBUILT_* path does not exist: %v", err)
		}
	}
	return env, true
}

// openQuackCatalogMetadataServer spawns a bare (no DuckLake attach of its
// own) quack-serving subprocess for openAppenderBenchDBQuackCatalog to
// attach a DuckLake catalog against as its remote metadata store — an
// in-memory database, so its metadata never touches disk at all. Loads the
// ducklake extension (without attaching anything) so
// QuackMetadataManager's server-side commit fast path is exercised, not
// just the minimal client-driven-commit path
// pkg/db/duckdb/ducklake_quack_test.go's TestDuckLakeQuackCatalogAttaches
// already covers. binPath/allowUnsigned/localExtensionPaths mirror
// whichever stack the caller resolved (requireDuckDBBinary's standard one,
// or requireSelfBuiltQuack's) — both sides of the quack connection (this
// server and the main subprocess in openAppenderBenchDBQuackCatalogWithConns)
// must agree, since quack's wire protocol is versioned to the exact build.
func openQuackCatalogMetadataServer(tb testing.TB, binPath string, allowUnsigned bool, localExtensionPaths map[string]string) (addr, token string) {
	tb.Helper()

	addr = freeLocalAddr(tb)
	token = "bench-quack-catalog-" + uuid.NewString()
	db, err := driver.Open(tb.Context(), driver.Options{
		BinaryPath:              binPath,
		DBFile:                  ":memory:",
		QuackAddr:               &addr,
		QuackServeToken:         token,
		AllowUnsignedExtensions: allowUnsigned,
		LocalExtensionPaths:     localExtensionPaths,
	})
	if err != nil {
		tb.Fatalf("opening quack catalog metadata server: %v", err)
	}
	tb.Cleanup(func() { _ = db.Close() })

	ducklakeStmt := "INSTALL ducklake; LOAD ducklake;"
	if localExtensionPaths != nil {
		ducklakeStmt = "LOAD 'ducklake';"
	}
	if _, err := db.ExecContext(tb.Context(), ducklakeStmt); err != nil {
		tb.Fatalf("loading ducklake on quack catalog metadata server: %v", err)
	}
	return addr, token
}

// openAppenderBenchDBQuackCatalog is openAppenderBenchDB's remote-quack-
// catalog counterpart: same appenderBenchTables shape (schema and
// account_id partitioning included), quack transport, and benchQuackConns
// concurrency for the data-plane connection, but
// Options.DuckLake.QuackCatalogAddr/QuackCatalogToken instead of CatalogPath,
// PostgresCatalogURI, or SQLiteCatalogPath — a second, separate subprocess
// (openQuackCatalogMetadataServer) holds the metadata instead.
func openAppenderBenchDBQuackCatalog(tb testing.TB) *sql.DB {
	tb.Helper()
	return openAppenderBenchDBQuackCatalogWithConns(tb, benchQuackConns)
}

// openAppenderBenchDBQuackCatalogWithConns is
// openAppenderBenchDBQuackCatalog with an explicit Options.QuackConns for
// the main (data-attaching) subprocess, instead of the fixed benchQuackConns
// — BenchmarkAppenderWritesQuackCatalogByConcurrency's one caller, sweeping
// concurrency levels the same way sqlite_bench_test.go's
// openAppenderBenchDBSQLiteWithConns does for the SQLite catalog. The
// metadata server's own connection count is untouched (openQuackCatalogMetadataServer
// never sets QuackConns): concurrent commits there serialize on the remote
// instance's own in-memory DuckDB MVCC, not a coarse file lock the way
// SQLite's catalog does, so there's no equivalent bottleneck to sweep on
// that side.
func openAppenderBenchDBQuackCatalogWithConns(tb testing.TB, conns int) *sql.DB {
	tb.Helper()

	var (
		binPath             string
		allowUnsigned       bool
		localExtensionPaths map[string]string
	)
	if env, ok := requireSelfBuiltQuack(tb); ok {
		binPath = env.binaryPath
		allowUnsigned = true
		localExtensionPaths = env.localExtensionPaths()
	} else {
		binPath = requireDuckDBBinary(tb)
		requireQuackExtension(tb, binPath)
	}
	metadataAddr, metadataToken := openQuackCatalogMetadataServer(tb, binPath, allowUnsigned, localExtensionPaths)

	dir := tb.TempDir()
	addr := freeLocalAddr(tb)
	opts := driver.Options{
		BinaryPath:              binPath,
		DBFile:                  ":memory:",
		AllowUnsignedExtensions: allowUnsigned,
		LocalExtensionPaths:     localExtensionPaths,
		DuckLake: &driver.DuckLakeOptions{
			QuackCatalogAddr:  metadataAddr,
			QuackCatalogToken: metadataToken,
			DataPath:          filepath.Join(dir, "data"),
			// Matches file/postgres/sqlite modes' convention: -1 explicitly
			// disables inlining, so every write flushes straight to Parquet
			// rather than staying inlined in the (remote, in-memory)
			// catalog.
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

// BenchmarkAppenderWritesQuackCatalog100k is BenchmarkAppenderWrites100k's
// remote-quack-catalog counterpart: the same 100,000-row, single-threaded
// pass over every appenderInsertScenarios entry, against a DuckLake catalog
// whose metadata lives in a second, separate in-memory duckdb subprocess
// instead of a local file, Postgres, or SQLite one. Run with:
//
//	cd cmd/duckdbbench && go test -bench BenchmarkAppenderWritesQuackCatalog100k -run '^$' -benchtime=1x .
func BenchmarkAppenderWritesQuackCatalog100k(b *testing.B) {
	const totalRows = 100_000
	totalRuns := totalRows / eventsPerRun
	accountID := uuid.New()

	for _, sc := range appenderInsertScenarios {
		b.Run(sc.name, func(b *testing.B) {
			for b.Loop() {
				db := openAppenderBenchDBQuackCatalog(b)
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

// BenchmarkAppenderWritesQuackCatalogByConcurrency is
// BenchmarkAppenderWritesSQLiteByConcurrency's remote-quack-catalog
// counterpart: same benchConcurrencyLevels sweep, fixed 100k rows and 50
// accounts, interleaved (round-robin, not partition-aligned) account_id
// assignment via generateInterleavedBatchesParallelN — but against a
// DuckLake catalog whose metadata lives in a second in-memory duckdb
// subprocess instead of a SQLite file, to see whether that catalog's
// concurrent-commit behavior looks like SQLite's single-writer file-lock
// wall (BenchmarkAppenderWritesSQLiteByConcurrency: works at concurrency=1,
// fails outright at 2+) or more like DuckLake's ordinary optimistic-
// concurrency conflicts (degrades but keeps succeeding, retried by
// applyWithConflictRetry). Run with:
//
//	cd cmd/duckdbbench && go test -bench BenchmarkAppenderWritesQuackCatalogByConcurrency -run '^$' -benchtime=1x -timeout 30m .
func BenchmarkAppenderWritesQuackCatalogByConcurrency(b *testing.B) {
	const totalRows = 100_000
	totalRuns := totalRows / eventsPerRun
	accountIDs := generateAccountIDs(50)

	for _, conns := range benchConcurrencyLevels {
		b.Run(fmt.Sprintf("concurrency=%d", conns), func(b *testing.B) {
			for _, sc := range appenderInsertScenarios {
				b.Run(sc.name, func(b *testing.B) {
					iter := 0
					for b.Loop() {
						db := openAppenderBenchDBQuackCatalogWithConns(b, conns)
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
