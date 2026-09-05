// Command duckdbseed seeds a DuckDB dual-write database
// (inngest.runs/run_trace_spans/events, per
// pkg/db/duckdb/migrations/000001_baseline.sql) with synthetic test data
// shaped after a real dev database, for exercising pkg/cqrs/duckdbquery and
// the trace UI without running real workloads through `inngest dev`.
//
// It writes via pkg/db/duckdb's quack Appender (duckdb.QuackAppender) over
// the same subprocess/HTTP transport dual-write itself uses — not
// duckdb-go/v2's embedded cgo driver, which an earlier version of this tool
// used. That means the real `duckdb` CLI binary must be on PATH (see
// pkg/db/duckdb/process.go's Open), same as `inngest dev --duckdb`.
//
// Run with: go run ./cmd/duckdbseed [flags]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
)

// runConfig parameterizes run — the seeding pipeline main wires up from
// flags. Now defaults to time.Now().UTC() (in run) when left zero, so tests
// can pin it for determinism.
type runConfig struct {
	SourceDir   string
	TargetDir   string
	RunCount    int
	Window      time.Duration
	SampleLimit int
	Seed        int64
	// Now anchors the generated time range; defaults to time.Now().UTC() in
	// run when left zero, so tests can pin it for determinism.
	Now time.Time
	// BatchSize is the number of runs grouped into one Appender flush (see
	// InsertGeneratedRuns). Defaults to 10000 in run when left <= 0.
	BatchSize int
	// Parallelism is how many Appenders (each its own quack connection to
	// the same subprocess) write concurrently. Defaults to 1 in run when
	// left <= 0. Values > 1 require the target's connection to have been
	// opened with more than one quack connection available — see
	// openDuckDB, which always does this.
	Parallelism int
	// DryRun, when true, samples and generates exactly as a real run would
	// and returns the Summary it would have written, but never opens,
	// creates, or writes to TargetDir.
	DryRun bool
	// Log, if non-nil, receives progress messages as run proceeds through
	// each stage (opening databases, sampling, generating, inserting).
	// Left nil in tests that don't care about log output; main wires it to
	// a real logger.
	Log func(format string, args ...any)
}

// logf calls cfg.Log if set, so every call site below stays a one-liner
// regardless of whether logging is configured.
func (cfg runConfig) logf(format string, args ...any) {
	if cfg.Log != nil {
		cfg.Log(format, args...)
	}
}

func main() {
	source := flag.String("source", filepath.Join(".inngest", "duckdb"), "directory containing the DuckDB catalog/data to sample real rows from as templates")
	target := flag.String("target", "", "directory to write seeded data into (defaults to -source)")
	runs := flag.Int("runs", 50, "number of synthetic runs to generate")
	window := flag.Duration("window", 7*24*time.Hour, "time range to spread generated runs' queued_at over, ending now")
	sampleLimit := flag.Int("sample-limit", 200, "max distinct runs to sample from -source as generation templates")
	seed := flag.Int64("seed", time.Now().UnixNano(), "RNG seed; pass a fixed value for reproducible output")
	batchSize := flag.Int("batch-size", 10_000, "number of runs (and their spans/events) grouped into one Appender flush")
	parallelism := flag.Int("parallelism", 1, "number of Appenders (quack connections) writing concurrently")
	dryRun := flag.Bool("dry-run", false, "sample and generate as usual, but report counts without writing to -target")
	flag.Parse()

	targetDir := *target
	if targetDir == "" {
		targetDir = *source
	}

	stderrLog := log.New(os.Stderr, "duckdbseed: ", log.LstdFlags)

	summary, err := run(context.Background(), runConfig{
		SourceDir:   *source,
		TargetDir:   targetDir,
		RunCount:    *runs,
		Window:      *window,
		SampleLimit: *sampleLimit,
		Seed:        *seed,
		BatchSize:   *batchSize,
		Parallelism: *parallelism,
		DryRun:      *dryRun,
		Log:         stderrLog.Printf,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	verb := "seeded"
	if *dryRun {
		verb = "would seed"
	}
	fmt.Printf("%s %d runs, %d spans, %d events into %s\n", verb, summary.Runs, summary.Spans, summary.Events, targetDir)
}

// formatRate renders count/d as a "%.0f" items-per-second figure for log
// output, or "n/a" when d isn't positive (too short to measure, or a
// cumulative timing that never accumulated any time — e.g. Timings.Insert
// on a dry run) rather than dividing by zero.
func formatRate(count int, d time.Duration) string {
	if d <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.0f", float64(count)/d.Seconds())
}

// run samples templates from cfg.SourceDir, generates cfg.RunCount synthetic
// runs from them, and inserts the result into cfg.TargetDir. When both
// resolve to the same directory, a single connector/subprocess is reused
// for both sampling and inserting — DuckDB's single-writer-per-catalog-file
// constraint means opening the same catalog twice, from two independent
// subprocesses, would otherwise fail.
func run(ctx context.Context, cfg runConfig) (Summary, error) {
	if cfg.Now.IsZero() {
		cfg.Now = time.Now().UTC()
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10_000
	}
	if cfg.Parallelism <= 0 {
		cfg.Parallelism = 1
	}

	cfg.logf("opening source %s", cfg.SourceDir)
	sourceConnector, sourceDB, err := openDuckDB(ctx, cfg.SourceDir, cfg.Parallelism)
	if err != nil {
		return Summary{}, fmt.Errorf("duckdbseed: opening source %q: %w", cfg.SourceDir, err)
	}
	defer sourceConnector.Close()
	defer sourceDB.Close()

	cfg.logf("sampling up to %d distinct run(s) from source", cfg.SampleLimit)
	sampleStart := time.Now()
	tmpl, sampled, err := BuildTemplates(ctx, sourceDB, cfg.SampleLimit)
	sampleDur := time.Since(sampleStart)
	if err != nil {
		return Summary{}, fmt.Errorf("duckdbseed: sampling templates from %q: %w", cfg.SourceDir, err)
	}
	if sampled {
		cfg.logf("sampled %d tenant(s), %d trace(s), %d event profile(s) from source", len(tmpl.Tenants), len(tmpl.Traces), len(tmpl.EventProfiles))
	} else {
		cfg.logf("source has no existing runs; using built-in default templates")
	}

	rng := rand.New(rand.NewSource(cfg.Seed))
	genCfg := GenerateConfig{RunCount: cfg.RunCount, Window: cfg.Window, Now: cfg.Now}

	if cfg.DryRun {
		cfg.logf("dry run: generating %d run(s) (seed=%d, window=%s) without opening or writing to target %s", cfg.RunCount, cfg.Seed, cfg.Window, cfg.TargetDir)
		genStart := time.Now()
		generated, err := GenerateRuns(rng, tmpl, genCfg)
		genDur := time.Since(genStart)
		cfg.logf("sampling %s, generation %s (%s runs/sec)", sampleDur, genDur, formatRate(cfg.RunCount, genDur))
		if err != nil {
			return Summary{}, err
		}
		return summarizeGeneratedRuns(generated), nil
	}

	targetConnector := sourceConnector
	if sameDir(cfg.SourceDir, cfg.TargetDir) {
		cfg.logf("target is the same directory as source; reusing connection")
	} else {
		cfg.logf("opening target %s", cfg.TargetDir)
		var targetDB *sql.DB
		targetConnector, targetDB, err = openDuckDB(ctx, cfg.TargetDir, cfg.Parallelism)
		if err != nil {
			return Summary{}, fmt.Errorf("duckdbseed: opening target %q: %w", cfg.TargetDir, err)
		}
		defer targetConnector.Close()
		defer targetDB.Close()
	}

	cfg.logf("generating and inserting %d run(s) (seed=%d, window=%s, batch size %d, parallelism %d)",
		cfg.RunCount, cfg.Seed, cfg.Window, cfg.BatchSize, cfg.Parallelism)
	onProgress := progressLogger(cfg.logf)
	wallStart := time.Now()
	summary, timings, err := GenerateAndInsert(ctx, targetConnector, rng, tmpl, genCfg, cfg.BatchSize, cfg.Parallelism, onProgress)
	wall := time.Since(wallStart)
	totalRows := summary.Runs + summary.Spans + summary.Events
	cfg.logf("sampling %s, generation %s (cumulative, %s runs/sec), insertion %s (cumulative, %s rows/sec), wall %s (%s runs/sec overall)",
		sampleDur,
		timings.Generate, formatRate(cfg.RunCount, timings.Generate),
		timings.Insert, formatRate(totalRows, timings.Insert),
		wall, formatRate(cfg.RunCount, wall))
	return summary, err
}

// summarizeGeneratedRuns reports the Summary InsertGeneratedRuns would have
// returned for generated, without touching any database — used by dry run.
func summarizeGeneratedRuns(generated []GeneratedRun) Summary {
	var summary Summary
	for _, g := range generated {
		summary.Runs++
		summary.Spans += len(g.Spans)
		summary.Events += len(g.Events)
	}
	return summary
}

// progressLogger logs one line per batch flush, using InsertGeneratedRuns'
// own batchIndex/totalBatches — a batch's real, stable identity, not
// derived from the running done count (which under parallelism > 1 no
// longer lines up with any particular batch once completions interleave).
// No throttling: a batch flush is already the natural progress unit, so
// even the default batch size of 10000 produces at most a few dozen lines
// for a large -runs value.
func progressLogger(logf func(format string, args ...any)) func(batchIndex, totalBatches, done, total int) {
	return func(batchIndex, totalBatches, done, total int) {
		logf("flushed batch %d/%d (%d/%d runs)", batchIndex, totalBatches, done, total)
	}
}

// openDuckDB opens dir's DuckDB catalog (creating it on first use) with
// DuckLake attached under duckdb.DuckLakeAlias, quack always enabled (this
// tool's Appender-based writes require it — see insert.go), and the schema
// migrated (duckdb.Migrate — the real dual-write migrations, not a local
// copy), mirroring pkg/devserver/dualwrite.go's setupDualWrite layout
// (<dir>/main.duckdb, <dir>/catalog.duckdb, <dir>/data/) so this tool reads
// and writes the same catalog `inngest dev --duckdb` does.
//
// parallelism is passed straight through as Options.QuackConns: values > 1
// let InsertGeneratedRuns's worker pool open that many additional,
// genuinely independent quack connections directly against the returned
// *duckdb.Connector (see newAppenderSet), on top of the one *sql.DB uses
// for ordinary querying (sampling, schema checks).
func openDuckDB(ctx context.Context, dir string, parallelism int) (*duckdb.Connector, *sql.DB, error) {
	binPath, err := lookDuckDBBinary()
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("creating %q: %w", dir, err)
	}

	addr, err := freeLocalQuackAddr()
	if err != nil {
		return nil, nil, fmt.Errorf("allocating a local port for the quack listener: %w", err)
	}

	connector, db, err := duckdb.OpenConnector(ctx, duckdb.Options{
		BinaryPath: binPath,
		DBFile:     filepath.Join(dir, "main.duckdb"),
		DuckLake: &duckdb.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
		QuackAddr:  &addr,
		QuackConns: parallelism,
	})
	if err != nil {
		return nil, nil, err
	}

	if err := duckdb.Migrate(ctx, db); err != nil {
		_ = db.Close()
		_ = connector.Close()
		return nil, nil, err
	}
	return connector, db, nil
}

// freeLocalQuackAddr resolves an ephemeral loopback port for
// duckdb.Options.QuackAddr, mirroring pkg/devserver/dualwrite.go's helper
// of the same purpose (unexported there too, so duplicated rather than
// imported across packages for one small function). Closing the listener
// immediately before duckdb binds it is a small TOCTOU race, acceptable for
// this dev tool.
func freeLocalQuackAddr() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := l.Addr().(*net.TCPAddr)
	if err := l.Close(); err != nil {
		return "", err
	}
	return fmt.Sprintf("127.0.0.1:%d", addr.Port), nil
}

func lookDuckDBBinary() (string, error) {
	path, err := exec.LookPath("duckdb")
	if err != nil {
		return "", fmt.Errorf("duckdb binary not found on PATH: %w", err)
	}
	return path, nil
}

func sameDir(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return absA == absB
}
