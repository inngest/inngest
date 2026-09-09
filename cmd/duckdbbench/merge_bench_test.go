// BenchmarkRunsInsert and BenchmarkRunsQuery compare a MERGE INTO-upserted
// "runs" table against a flat, append-only one, against a real DuckLake
// catalog. Both tables model inngest.runs' real lifecycle (queued -> started
// -> ended, pkg/db/duckdb/migrations/000001_baseline.sql), partitioned by
// account_id the same way inngest.runs itself is, and the flat table's
// "latest row per run_id" read mirrors pkg/cqrs/duckdbquery/runs.go's
// QUALIFY ROW_NUMBER() collapse — the exact query-side cost a MERGE INTO
// table (one row per run_id, updated in place) would remove.
package duckdbbench

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	duckdbgo "github.com/duckdb/duckdb-go/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
)

// eventsPerRun is the number of lifecycle writes a single run produces —
// queued, started, ended — matching runCommonFields' three real dual-write
// hooks (pkg/execution/dualwrite/listener.go).
const eventsPerRun = 3

var statusBySeq = [eventsPerRun]string{"queued", "running", "completed"}

// attributesBySeq gives each lifecycle stage its own distinct JSON fields,
// mirroring how real metadata accrues incrementally (scheduler info at
// queue time, worker assignment at start, result/duration at end) rather
// than one stage rewriting the same keys. mergeJSONPatchBatch exercises the
// case this shape is meant to test: merging each stage's fragment into the
// accumulated JSON in place, instead of one stage's write clobbering the
// fields an earlier stage set.
var attributesBySeq = [eventsPerRun]string{
	`{"queued_by":"scheduler","priority":1}`,
	`{"worker":"w-1"}`,
	`{"duration_ms":1234,"result":"ok"}`,
}

// benchNumPartitions is how many distinct account_ids synthetic runs are
// spread across. openMergeBenchDB DuckLake-partitions every benchmark table
// on account_id, mirroring inngest.runs' own
// `SET PARTITIONED BY (..., account_id)` — so writes land across
// benchNumPartitions separate partition directories instead of one, and
// every MERGE INTO's target-row lookup has to do it across partition
// boundaries the way it would in production, rather than against a single
// unpartitioned file set.
const benchNumPartitions = 200

var benchAccountIDs = generateAccountIDs(benchNumPartitions)

func generateAccountIDs(n int) []uuid.UUID {
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.New()
	}
	return ids
}

// runEvent is one lifecycle write for a single run: the flat table appends
// it as a new row, the merge tables upsert it into the row already keyed on
// (account_id, run_id).
type runEvent struct {
	accountID  uuid.UUID
	runID      string
	status     string
	queuedAt   time.Time
	startedAt  sql.NullTime
	endedAt    sql.NullTime
	attributes string
	updatedAt  time.Time
}

// generateRunEvent returns the seq'th lifecycle write (0=queued, 1=started,
// 2=ended) for run (batchSeed, i) with the given account_id, uniquely
// identified so repeated benchmark iterations never collide on run_id.
func generateRunEvent(batchSeed, i, seq int, accountID uuid.UUID) runEvent {
	now := time.Now()
	ts := now.Add(time.Duration(seq) * time.Second)
	e := runEvent{
		accountID:  accountID,
		runID:      fmt.Sprintf("run-%d-%d", batchSeed, i),
		status:     statusBySeq[seq],
		queuedAt:   now,
		attributes: attributesBySeq[seq],
		updatedAt:  ts,
	}
	if seq >= 1 {
		e.startedAt = sql.NullTime{Time: ts, Valid: true}
	}
	if seq >= 2 {
		e.endedAt = sql.NullTime{Time: ts, Valid: true}
	}
	return e
}

// runColumns is every benchmark table's column order (bench_runs_flat,
// bench_runs_merged, bench_runs_merged_jsonpatch all share a schema — only
// the write path differs).
var runColumns = []string{"account_id", "run_id", "status", "queued_at", "started_at", "ended_at", "attributes", "updated_at"}

func runEventArgs(e runEvent) []any {
	return []any{e.accountID.String(), e.runID, e.status, e.queuedAt, nullTimeArg(e.startedAt), nullTimeArg(e.endedAt), e.attributes, e.updatedAt}
}

func nullTimeArg(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return t.Time
}

func prefixColumns(alias string, cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = alias + "." + c
	}
	return out
}

// flatInsertBatch appends events as new rows into bench_runs_flat — the
// current production pattern (dual-write's batcher, see batch.go), which
// never mutates a prior row.
func flatInsertBatch(ctx context.Context, db *sql.DB, events []runEvent) error {
	if len(events) == 0 {
		return nil
	}
	args := make([]any, 0, len(events)*len(runColumns))
	for _, e := range events {
		args = append(args, runEventArgs(e)...)
	}
	query := batchInsertQuery(duckdb.DuckLakeAlias+".bench_runs_flat", runColumns, len(events))
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// mergeBatch upserts events into the named table keyed on (account_id,
// run_id): a run's first event inserts its row, later events
// (started/ended) update it in place, so the table always holds exactly one
// row per run_id. account_id is included in the join condition (redundant
// with run_id alone, which is already globally unique) so the MERGE only
// has to match within the source rows' own partitions rather than across
// all benchNumPartitions of them. attributesSet is the UPDATE SET
// expression for the attributes column — a plain overwrite for
// mergeUpsertBatch, or a json_merge_patch fold for mergeJSONPatchBatch.
// queued_at is deliberately left out of the UPDATE SET list — it's set once
// at insert and never revised, mirroring inngest.runs' own queued_at
// semantics.
func mergeBatch(ctx context.Context, db *sql.DB, table, attributesSet string, events []runEvent) error {
	if len(events) == 0 {
		return nil
	}
	placeholderRow := "(" + strings.TrimSuffix(strings.Repeat("?, ", len(runColumns)), ", ") + ")"
	rows := make([]string, len(events))
	args := make([]any, 0, len(events)*len(runColumns))
	for i, e := range events {
		rows[i] = placeholderRow
		args = append(args, runEventArgs(e)...)
	}
	query := fmt.Sprintf(`
MERGE INTO %[1]s.%[2]s AS t
USING (VALUES %[3]s) AS s(%[4]s)
ON t.account_id = s.account_id AND t.run_id = s.run_id
WHEN MATCHED THEN UPDATE SET status = s.status, started_at = s.started_at, ended_at = s.ended_at, attributes = %[5]s, updated_at = s.updated_at
WHEN NOT MATCHED THEN INSERT (%[4]s) VALUES (%[6]s);`,
		duckdb.DuckLakeAlias, table, strings.Join(rows, ", "), strings.Join(runColumns, ", "), attributesSet, strings.Join(prefixColumns("s", runColumns), ", "))
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func mergeUpsertBatch(ctx context.Context, db *sql.DB, events []runEvent) error {
	return mergeBatch(ctx, db, "bench_runs_merged", "s.attributes", events)
}

// mergeJSONPatchBatch upserts events the same way mergeUpsertBatch does,
// except attributes is folded via json_merge_patch instead of overwritten:
// each lifecycle stage's JSON fragment (attributesBySeq) merges into the
// row's accumulated attributes rather than replacing them outright, so
// bench_runs_merged_jsonpatch ends up holding every stage's fields, not
// just the last stage's — the more realistic (and more expensive, per-row
// JSON computation) shape a real incremental-metadata upsert would need.
func mergeJSONPatchBatch(ctx context.Context, db *sql.DB, events []runEvent) error {
	return mergeBatch(ctx, db, "bench_runs_merged_jsonpatch", "json_merge_patch(t.attributes, s.attributes)", events)
}

// benchTables are every table openMergeBenchDB creates: bench_runs_flat
// (flatInsertBatch), bench_runs_merged (mergeUpsertBatch), and
// bench_runs_merged_jsonpatch (mergeJSONPatchBatch) — all three share the
// same column shape and account_id partitioning, only the write path
// differs.
var benchTables = []string{"bench_runs_flat", "bench_runs_merged", "bench_runs_merged_jsonpatch"}

// openMergeBenchDB opens an embedded (no subprocess) DuckDB instance with a
// fresh DuckLake catalog attached under tb.TempDir(), and creates
// benchTables, each partitioned by account_id — mirrors
// insert_bench_test.go's openEmbeddedConnector, minus the real inngest.*
// migration set, which this benchmark doesn't need.
func openMergeBenchDB(tb testing.TB) *sql.DB {
	tb.Helper()
	connector, err := duckdbgo.NewConnector(":memory:", nil)
	if err != nil {
		tb.Fatalf("creating embedded connector: %v", err)
	}
	db := sql.OpenDB(connector)
	tb.Cleanup(func() { _ = db.Close() })

	opts := duckLakeOptions(tb, duckdb.DefaultDataInliningRowLimit)
	if err := ensureDir(opts.DataPath); err != nil {
		tb.Fatalf("creating DuckLake data path: %v", err)
	}
	for _, stmt := range duckLakeAttachStmts(opts) {
		if _, err := db.ExecContext(tb.Context(), stmt); err != nil {
			tb.Fatalf("DuckLake bootstrap failed on %q: %v", stmt, err)
		}
	}

	for _, table := range benchTables {
		schema := fmt.Sprintf(`CREATE TABLE %s.%s (
			account_id UUID NOT NULL,
			run_id VARCHAR NOT NULL,
			status VARCHAR NOT NULL,
			queued_at TIMESTAMP_MS NOT NULL,
			started_at TIMESTAMP_MS,
			ended_at TIMESTAMP_MS,
			attributes JSON NOT NULL,
			updated_at TIMESTAMP_MS NOT NULL
		);`, duckdb.DuckLakeAlias, table)
		if _, err := db.ExecContext(tb.Context(), schema); err != nil {
			tb.Fatalf("creating %s: %v", table, err)
		}
		partition := fmt.Sprintf("ALTER TABLE %s.%s SET PARTITIONED BY (account_id);", duckdb.DuckLakeAlias, table)
		if _, err := db.ExecContext(tb.Context(), partition); err != nil {
			tb.Fatalf("partitioning %s: %v", table, err)
		}
	}
	return db
}

// benchPipelineDepth is how many ticks apart a run's queued/started/ended
// writes land when generateInterleavedBatches replays its lifecycle. This
// models runs actually progressing through their lifecycle concurrently,
// instead of one lifecycle stage sweeping the whole run range before the
// next stage starts: at any tick past 2*pipelineDepth, a batch mixes a
// fresh run's queued insert, an older run's started update, and an
// even-older run's ended update — three different run_ids, pipelineDepth
// apart, so a batch never contains two events for the same run (MERGE INTO
// can't apply two source rows to the same target row in one statement). This
// is the mix dual-write's real batcher actually sees in production — runs at
// every stage of their lifecycle writing concurrently — not the
// lifecycle-sorted, one-stage-at-a-time traffic three homogeneous sweeps
// would produce.
const benchPipelineDepth = 5000

// generateInterleavedBatches replays totalRuns runs' full lifecycle,
// batchSize ticks at a time, calling apply once per batch — see
// benchPipelineDepth's doc comment for the interleaving this produces. Each
// run's account_id is accountIDs[i % len(accountIDs)] — the same value for
// all of that run's events, since a run's account never changes
// mid-lifecycle — so len(accountIDs) is the partition cardinality writes
// spread across. A run's own three writes are still generated in tick order
// (queued at tick i, started at i+pipelineDepth, ended at i+2*pipelineDepth)
// and batches are applied in increasing tick order, so per-run write order
// is preserved; only the interleaving across different runs' concurrent
// lifecycles changes.
//
// Requires pipelineDepth >= batchSize: a batch spans batchSize consecutive
// ticks, and two ticks t1 < t2 within it only ever produce events for the
// same run_id if t2-t1 equals pipelineDepth (queued vs. started) or
// pipelineDepth again (started vs. ended) — impossible once the maximum
// in-batch tick distance (batchSize-1) is less than pipelineDepth. Violating
// this lets one batch carry two events for the same not-yet-existing run;
// MERGE INTO checks WHEN NOT MATCHED against the target's state at the
// statement's start, not row-by-row, so both would take the INSERT branch
// and silently produce two rows for one run_id instead of one.
func generateInterleavedBatches(batchSeed, totalRuns, pipelineDepth, batchSize int, accountIDs []uuid.UUID, apply func(batch []runEvent) error) error {
	if pipelineDepth < batchSize {
		return fmt.Errorf("pipelineDepth (%d) must be >= batchSize (%d)", pipelineDepth, batchSize)
	}
	totalTicks := totalRuns + 2*pipelineDepth
	for start := 0; start < totalTicks; start += batchSize {
		end := min(start+batchSize, totalTicks)
		batch := make([]runEvent, 0, 3*(end-start))
		for t := start; t < end; t++ {
			if t < totalRuns {
				batch = append(batch, generateRunEvent(batchSeed, t, 0, accountIDs[t%len(accountIDs)]))
			}
			if i := t - pipelineDepth; i >= 0 && i < totalRuns {
				batch = append(batch, generateRunEvent(batchSeed, i, 1, accountIDs[i%len(accountIDs)]))
			}
			if i := t - 2*pipelineDepth; i >= 0 && i < totalRuns {
				batch = append(batch, generateRunEvent(batchSeed, i, 2, accountIDs[i%len(accountIDs)]))
			}
		}
		if err := apply(batch); err != nil {
			return fmt.Errorf("batch [%d:%d): %w", start, end, err)
		}
	}
	return nil
}

// seedRunsData writes totalRuns runs' full lifecycle (eventsPerRun events
// each) into every benchmark table, interleaved via
// generateInterleavedBatches, so the merge tables see a realistic mix of
// inserts and UPDATE-in-place upserts within each batch rather than
// homogeneous sweeps.
func seedRunsData(tb testing.TB, ctx context.Context, db *sql.DB, totalRuns, pipelineDepth, batchSize int, accountIDs []uuid.UUID) {
	tb.Helper()
	err := generateInterleavedBatches(0, totalRuns, pipelineDepth, batchSize, accountIDs, func(batch []runEvent) error {
		if err := flatInsertBatch(ctx, db, batch); err != nil {
			return fmt.Errorf("seeding bench_runs_flat: %w", err)
		}
		if err := mergeUpsertBatch(ctx, db, batch); err != nil {
			return fmt.Errorf("seeding bench_runs_merged: %w", err)
		}
		if err := mergeJSONPatchBatch(ctx, db, batch); err != nil {
			return fmt.Errorf("seeding bench_runs_merged_jsonpatch: %w", err)
		}
		return nil
	})
	if err != nil {
		tb.Fatal(err)
	}
}

// benchRowCounts sweeps two scales: the originally requested 1,000,000
// row-writes, and 10,000,000 to see whether either insertion strategy's
// relative advantage changes at 10x scale — e.g. bench_runs_flat growing
// large enough that DuckLake's per-partition file bookkeeping starts to cost
// differently than MERGE INTO's per-row target lookup does.
var benchRowCounts = []int{1_000_000, 10_000_000}

const benchBatchSize = 2000

// insertScenarios are the three write strategies BenchmarkRunsInsert and
// BenchmarkRunsInsertByPartitions both sweep.
var insertScenarios = []struct {
	name  string
	apply func(ctx context.Context, db *sql.DB, events []runEvent) error
}{
	{"flat", flatInsertBatch},
	{"merge", mergeUpsertBatch},
	{"merge-json-patch", mergeJSONPatchBatch},
}

// runInsertScenarios times each of insertScenarios writing totalRuns runs'
// full interleaved lifecycle against accountIDs' partition cardinality,
// reporting rows/sec for each so the three insertion strategies are directly
// comparable at whatever (totalRuns, accountIDs) the caller is sweeping.
func runInsertScenarios(b *testing.B, totalRuns int, accountIDs []uuid.UUID) {
	for _, sc := range insertScenarios {
		b.Run(sc.name, func(b *testing.B) {
			iter := 0
			for b.Loop() {
				db := openMergeBenchDB(b)
				ctx := b.Context()
				err := generateInterleavedBatches(iter, totalRuns, benchPipelineDepth, benchBatchSize, accountIDs, func(batch []runEvent) error {
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

// BenchmarkRunsInsert times writing every run's full lifecycle (totalRuns
// runs * eventsPerRun events, for each of benchRowCounts) into a fresh
// DuckLake-backed table partitioned across benchNumPartitions account_ids —
// see BenchmarkRunsInsertByPartitions for the effect of varying that
// partition count instead of the row count. Writes are interleaved
// (generateInterleavedBatches), not lifecycle-sorted — every batch mixes
// inserts and updates the way production traffic would.
func BenchmarkRunsInsert(b *testing.B) {
	for _, totalWrites := range benchRowCounts {
		totalRuns := totalWrites / eventsPerRun
		b.Run(fmt.Sprintf("rows=%d", totalWrites), func(b *testing.B) {
			runInsertScenarios(b, totalRuns, benchAccountIDs)
		})
	}
}

// benchPartitionCounts sweeps partition cardinality at a fixed 1,000,000
// row-writes, isolating how much of MERGE INTO's cost (vs. flat INSERT)
// comes from partition count specifically. Compare against
// BenchmarkRunsInsert's rows=1000000 numbers at benchNumPartitions (200)
// partitions: going from an unpartitioned, lifecycle-sorted benchmark to an
// interleaved, 200-partition one flipped MERGE INTO from ~13% faster than
// flat INSERT to roughly half its throughput — this sweep shows how much of
// that reversal is the partitioning itself versus the interleaving.
var benchPartitionCounts = []int{1, 10, 50, 100, 200}

const benchPartitionSweepRows = 1_000_000

// BenchmarkRunsInsertByPartitions times the same three insert strategies as
// BenchmarkRunsInsert, holding row count fixed at benchPartitionSweepRows
// and instead sweeping how many distinct account_ids (i.e. DuckLake
// partitions) those rows spread across.
func BenchmarkRunsInsertByPartitions(b *testing.B) {
	totalRuns := benchPartitionSweepRows / eventsPerRun
	for _, numPartitions := range benchPartitionCounts {
		accountIDs := generateAccountIDs(numPartitions)
		b.Run(fmt.Sprintf("partitions=%d", numPartitions), func(b *testing.B) {
			runInsertScenarios(b, totalRuns, accountIDs)
		})
	}
}

// BenchmarkRunsQuery seeds every table once per benchRowCounts scale with
// totalRuns runs' full interleaved lifecycle, then times representative read
// patterns against each: a full "current state of every run" scan, a single
// run_id point lookup, and a status-grouped count. The flat/* queries pay
// the QUALIFY ROW_NUMBER()-per-run_id collapse pkg/cqrs/duckdbquery/runs.go's
// real queries already require; the merge/* queries are plain reads against
// the already-collapsed table. bench_runs_merged_jsonpatch isn't queried
// here separately — it has the same one-row-per-run_id shape and row count
// as bench_runs_merged, so its read cost is identical; json_merge_patch's
// extra cost only shows up in BenchmarkRunsInsert.
func BenchmarkRunsQuery(b *testing.B) {
	for _, totalWrites := range benchRowCounts {
		totalRuns := totalWrites / eventsPerRun
		b.Run(fmt.Sprintf("rows=%d", totalWrites), func(b *testing.B) {
			db := openMergeBenchDB(b)
			ctx := b.Context()
			seedRunsData(b, ctx, db, totalRuns, benchPipelineDepth, benchBatchSize, benchAccountIDs)

			flatLatestSubquery := `
				SELECT run_id, status, queued_at, started_at, ended_at
				FROM ` + duckdb.DuckLakeAlias + `.bench_runs_flat
				QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id ORDER BY updated_at DESC) = 1`

			lookupRunID := fmt.Sprintf("run-0-%d", totalRuns/2)

			queries := []struct {
				name string
				sql  string
				args []any
			}{
				{"flat/latest-per-run", flatLatestSubquery, nil},
				{"merge/all-rows", `SELECT run_id, status, queued_at, started_at, ended_at FROM ` + duckdb.DuckLakeAlias + `.bench_runs_merged`, nil},
				{
					"flat/point-lookup",
					`SELECT status FROM ` + duckdb.DuckLakeAlias + `.bench_runs_flat WHERE run_id = ?
					 QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id ORDER BY updated_at DESC) = 1`,
					[]any{lookupRunID},
				},
				{"merge/point-lookup", `SELECT status FROM ` + duckdb.DuckLakeAlias + `.bench_runs_merged WHERE run_id = ?`, []any{lookupRunID}},
				{"flat/status-count", `SELECT status, count(*) FROM (` + flatLatestSubquery + `) GROUP BY status`, nil},
				{"merge/status-count", `SELECT status, count(*) FROM ` + duckdb.DuckLakeAlias + `.bench_runs_merged GROUP BY status`, nil},
			}

			for _, q := range queries {
				b.Run(q.name, func(b *testing.B) {
					runQueryBenchmark(b, db, ctx, q.name, q.sql, q.args)
				})
			}
		})
	}
}

// runQueryBenchmark times query (with args) against db, draining every row
// of every result set so the timing reflects a full scan/fetch rather than
// just planning — shared by BenchmarkRunsQuery and BenchmarkRunsCompaction.
func runQueryBenchmark(b *testing.B, db *sql.DB, ctx context.Context, name, query string, args []any) {
	for b.Loop() {
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			b.Fatalf("query %s: %v", name, err)
		}
		n := 0
		for rows.Next() {
			n++
		}
		closeErr := rows.Close()
		if err := rows.Err(); err != nil {
			b.Fatalf("query %s: %v", name, err)
		}
		if closeErr != nil {
			b.Fatalf("query %s: closing rows: %v", name, closeErr)
		}
	}
}

// benchCompactionQueries are the two representative reads
// BenchmarkRunsCompaction times before and after compaction — the same
// flat/latest-per-run collapse and merge/all-rows scan BenchmarkRunsQuery
// already uses, since those are the queries file fragmentation from
// interleaved writes (Finding 2/3) would actually slow down.
func benchCompactionQueries() []struct {
	name string
	sql  string
} {
	flatLatestSubquery := `
		SELECT run_id, status, queued_at, started_at, ended_at
		FROM ` + duckdb.DuckLakeAlias + `.bench_runs_flat
		QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id ORDER BY updated_at DESC) = 1`
	return []struct {
		name string
		sql  string
	}{
		{"flat/latest-per-run", flatLatestSubquery},
		{"merge/all-rows", `SELECT run_id, status, queued_at, started_at, ended_at FROM ` + duckdb.DuckLakeAlias + `.bench_runs_merged`},
	}
}

// BenchmarkRunsCompaction seeds every benchmark table at
// benchPartitionSweepRows (interleaved) for each of benchPartitionCounts,
// times the two benchCompactionQueries reads before compaction (with
// however many small Parquet files interleaved writes at that partition
// count produced), times DuckLake's own `ducklake_merge_adjacent_files`
// maintenance call itself, then times the same two reads again after
// compaction (against whatever single/few files it produced) — showing
// whether the write-side file fragmentation Finding 2/3 measured is also a
// read-side cost, and how expensive fixing it is at each partition count.
func BenchmarkRunsCompaction(b *testing.B) {
	totalRuns := benchPartitionSweepRows / eventsPerRun
	queries := benchCompactionQueries()

	for _, numPartitions := range benchPartitionCounts {
		accountIDs := generateAccountIDs(numPartitions)
		b.Run(fmt.Sprintf("partitions=%d", numPartitions), func(b *testing.B) {
			db := openMergeBenchDB(b)
			ctx := b.Context()
			seedRunsData(b, ctx, db, totalRuns, benchPipelineDepth, benchBatchSize, accountIDs)

			for _, q := range queries {
				b.Run(q.name+"/before-compaction", func(b *testing.B) {
					runQueryBenchmark(b, db, ctx, q.name, q.sql, nil)
				})
			}

			b.Run("compact", func(b *testing.B) {
				for b.Loop() {
					if _, err := db.ExecContext(ctx, "CALL ducklake_merge_adjacent_files('"+duckdb.DuckLakeAlias+"');"); err != nil {
						b.Fatalf("compacting: %v", err)
					}
				}
			})

			for _, q := range queries {
				b.Run(q.name+"/after-compaction", func(b *testing.B) {
					runQueryBenchmark(b, db, ctx, q.name, q.sql, nil)
				})
			}
		})
	}
}

// TestRunsMergeTablesMatchFlatLatest guards the write-path semantics both
// benchmarks assume, against a small dataset (not millions of rows — this
// only needs to prove correctness, not perf) and a short pipeline depth (so
// the interleave actually kicks in at this scale): after one run's full
// lifecycle, bench_runs_merged must hold exactly one row per run with the
// flat table's latest-row-per-run_id state, and bench_runs_merged_jsonpatch
// must hold that same state but with attributes accumulated from every
// lifecycle stage instead of only the last one.
func TestRunsMergeTablesMatchFlatLatest(t *testing.T) {
	const numRuns = 50

	db := openMergeBenchDB(t)
	ctx := t.Context()
	seedRunsData(t, ctx, db, numRuns, 7, 7, benchAccountIDs) // odd batch size to exercise a partial final batch; pipelineDepth == batchSize is the minimum safe value

	for _, table := range []string{"bench_runs_merged", "bench_runs_merged_jsonpatch"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+duckdb.DuckLakeAlias+"."+table).Scan(&count); err != nil {
			t.Fatalf("counting %s: %v", table, err)
		}
		if count != numRuns {
			t.Errorf("%s: got %d rows, want %d (one per run_id)", table, count, numRuns)
		}
	}

	flatLatest := `SELECT status, started_at, ended_at FROM ` + duckdb.DuckLakeAlias + `.bench_runs_flat
		WHERE run_id = ? QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id ORDER BY updated_at DESC) = 1`

	for i := range numRuns {
		runID := fmt.Sprintf("run-0-%d", i)

		var wantStatus string
		var wantStarted, wantEnded sql.NullTime
		if err := db.QueryRowContext(ctx, flatLatest, runID).Scan(&wantStatus, &wantStarted, &wantEnded); err != nil {
			t.Fatalf("reading flat latest for %s: %v", runID, err)
		}
		if wantStatus != "completed" {
			t.Fatalf("flat latest for %s: got status %q, want %q (test setup bug)", runID, wantStatus, "completed")
		}

		var gotStatus string
		if err := db.QueryRowContext(ctx, "SELECT status FROM "+duckdb.DuckLakeAlias+".bench_runs_merged WHERE run_id = ?", runID).Scan(&gotStatus); err != nil {
			t.Fatalf("reading bench_runs_merged for %s: %v", runID, err)
		}
		if gotStatus != wantStatus {
			t.Errorf("bench_runs_merged %s: got status %q, want %q", runID, gotStatus, wantStatus)
		}

		// duckdb-go/v2 decodes the JSON column natively into a Go
		// map[string]any (see insert_bench_test.go's
		// TestAppenderStoresIdenticalRowToSQLInsert) rather than surfacing raw
		// JSON text, so scan into `any` and inspect the map.
		var attributes any
		if err := db.QueryRowContext(ctx, "SELECT attributes FROM "+duckdb.DuckLakeAlias+".bench_runs_merged_jsonpatch WHERE run_id = ?", runID).Scan(&attributes); err != nil {
			t.Fatalf("reading bench_runs_merged_jsonpatch for %s: %v", runID, err)
		}
		attrMap, ok := attributes.(map[string]any)
		if !ok {
			t.Fatalf("bench_runs_merged_jsonpatch %s: attributes decoded as %T, want map[string]any", runID, attributes)
		}
		for _, want := range []string{"queued_by", "worker", "duration_ms", "result"} {
			if _, ok := attrMap[want]; !ok {
				t.Errorf("bench_runs_merged_jsonpatch %s: attributes %#v missing key %q from an earlier lifecycle stage", runID, attrMap, want)
			}
		}
	}
}

// TestRunsPartitionedAcrossAccountIDs guards openMergeBenchDB's ALTER
// TABLE ... SET PARTITIONED BY: with more than benchNumPartitions runs
// written, every benchmark table must have laid its data out into multiple
// account_id partition directories, not one, and DuckLake's own accounting
// (ducklake_table_info) must reflect data files, not everything still
// sitting inlined in the catalog.
func TestRunsPartitionedAcrossAccountIDs(t *testing.T) {
	db := openMergeBenchDB(t)
	ctx := t.Context()
	// > DefaultDataInliningRowLimit so every table is forced to flush real
	// partitioned Parquet files instead of staying inlined in the catalog.
	seedRunsData(t, ctx, db, 4*benchNumPartitions, 500, 500, benchAccountIDs)

	for _, table := range benchTables {
		var count int
		if err := db.QueryRowContext(ctx,
			"SELECT count(DISTINCT account_id) FROM "+duckdb.DuckLakeAlias+"."+table,
		).Scan(&count); err != nil {
			t.Fatalf("counting distinct account_id in %s: %v", table, err)
		}
		if count != benchNumPartitions {
			t.Errorf("%s: got %d distinct account_id values, want %d", table, count, benchNumPartitions)
		}
	}
}
