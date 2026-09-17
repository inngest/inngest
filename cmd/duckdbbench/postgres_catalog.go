package duckdbbench

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/oklog/ulid/v2"

	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// postgresCatalogEnv names the environment variables that switch every
// openMergeBenchDB call in this run from a file-based DuckLake catalog to a
// Postgres one — see requirePostgresCatalogBench's doc comment for why this
// needs a self-built duckdb rather than the ordinary community-extension
// duckdb these benchmarks otherwise use.
type postgresCatalogEnv struct {
	binaryPath             string
	postgresScannerExtPath string
	quackExtPath           string
	httpfsExtPath          string
	uri                    string
}

// requirePostgresCatalogBench reports whether this run should exercise a
// Postgres DuckLake catalog instead of the default file-based one, and
// returns the config to do it. All five env vars must be set (and the
// binary/extension paths must exist), or every openMergeBenchDB call falls
// straight back through to the ordinary file-based catalog — this switch is
// opt-in and every existing invocation of these benchmarks is completely
// unaffected by its absence.
//
// A self-built duckdb is required, not the ordinary community-extension one
// requireDuckDBBinary resolves: as of this writing every published DuckDB
// nightly's ducklake build rejects its own batched Postgres metadata SQL
// with "cannot insert multiple commands into a prepared statement" on the
// very first CREATE TABLE (a regression between DuckDB v1.5.5 and current
// nightlies, distinct from the already-handled ducklake#1175 JSON-inlining
// bug) — already fixed on ducklake's own main branch
// (postgres_metadata_manager.cpp's SplitBatchStatements, from ducklake#1407)
// but not yet published under any resolvable nightly build. See
// FINDINGS/notes.md for the investigation and pkg/db/duckdb's
// ducklake_selfbuilt_test.go for the same build requirement at the driver
// level.
func requirePostgresCatalogBench(tb testing.TB) (postgresCatalogEnv, bool) {
	tb.Helper()

	env := postgresCatalogEnv{
		binaryPath:             os.Getenv("DUCKLAKE_SELFBUILT_DUCKDB"),
		postgresScannerExtPath: os.Getenv("DUCKLAKE_SELFBUILT_POSTGRES_SCANNER"),
		quackExtPath:           os.Getenv("DUCKLAKE_SELFBUILT_QUACK"),
		httpfsExtPath:          os.Getenv("DUCKLAKE_SELFBUILT_HTTPFS"),
		uri:                    os.Getenv("DUCKLAKE_BENCH_POSTGRES_URI"),
	}
	if env.binaryPath == "" || env.postgresScannerExtPath == "" || env.quackExtPath == "" || env.httpfsExtPath == "" || env.uri == "" {
		return postgresCatalogEnv{}, false
	}
	for _, p := range []string{env.binaryPath, env.postgresScannerExtPath, env.quackExtPath, env.httpfsExtPath} {
		if _, err := os.Stat(p); err != nil {
			tb.Fatalf("DUCKLAKE_SELFBUILT_* path does not exist: %v", err)
		}
	}
	db, err := sql.Open("pgx", env.uri)
	if err != nil {
		tb.Fatalf("opening DUCKLAKE_BENCH_POSTGRES_URI: %v", err)
	}
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		tb.Fatalf("DUCKLAKE_BENCH_POSTGRES_URI not reachable: %v", err)
	}
	return env, true
}

// duckLakeOptionsFor builds the DuckLakeOptions and matching driver.Options
// overrides for one openMergeBenchDB call: a fresh MetadataSchema every
// time, so repeated calls within (and across) a benchmark run against the
// same long-lived Postgres database never collide on each other's tables or
// DATA_PATH — the postgres-catalog analogue of the fresh tb.TempDir() a file
// catalog already gets per call.
func (env postgresCatalogEnv) duckLakeOptions(dataPath string) *driver.DuckLakeOptions {
	return &driver.DuckLakeOptions{
		PostgresCatalogURI: env.uri,
		DataPath:           dataPath,
		MetadataSchema:     "bench_" + ulid.Make().String(),
		// Explicitly disabled to match file-mode's -1 (see duckLakeOptions
		// in insert_bench_test.go) -- forced to 0 either way for a postgres
		// catalog (ducklake#1175), spelled out here for clarity rather than
		// relying on that override.
		DataInliningRowLimit: -1,
	}
}

// localExtensionPaths returns the Options.LocalExtensionPaths this env
// needs: ducklake is statically linked into env.binaryPath (see
// ducklake_selfbuilt_test.go's identical "ducklake": "ducklake" pattern),
// postgres/quack are loadable files built from the exact same duckdb
// commit, and httpfs is only a prerequisite so quack_serve gets a real
// (not read-only) crypto engine on this from-scratch self-build.
func (env postgresCatalogEnv) localExtensionPaths() map[string]string {
	return map[string]string{
		"ducklake": "ducklake",
		"postgres": env.postgresScannerExtPath,
		"quack":    env.quackExtPath,
		"httpfs":   env.httpfsExtPath,
	}
}
