package duckdbbench

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// This file is an alternative to merge_bench_test.go's tick-based
// generateInterleavedBatches/generatePartitionAlignedBatches: instead of
// spacing a run's queued/started/ended writes a fixed benchPipelineDepth
// ticks apart and sweeping partition cardinality (benchPartitionCounts),
// it scatters totalRuns runs' queued_at times uniformly at random across a
// real wall-clock window (benchTimeRanges), models each run's
// queued-to-started and queued-to-ended gaps with a random,
// heavy-tailed-but-mostly-short duration (sampleExpMedian), and applies
// every run's three writes to the database in the order they'd actually
// arrive — not grouped by lifecycle stage or by tick. This is a more
// realistic traffic shape than the tick-based generators' deterministic
// interleave, at the cost of losing their compile-time guarantee that no
// batch ever contains two events for the same run_id (see
// applyEventsInTimeOrderedBatches' doc comment) — recovered instead via a
// SQL-side dedup (QuackMergeConfig.DedupKeys/DedupOrderBy,
// merge_appender_bench_test.go's mergeAppenderBatch) rather than any
// client-side collision avoidance. Partition alignment
// (generatePartitionAlignedBatchesParallel's counterpart) is deliberately
// not modeled here yet — every run's account_id is still assigned
// round-robin over accountIDs, exactly like generateInterleavedBatchesParallel.

// benchArrivalStageMedian is the median of the random, per-stage duration
// sampleExpMedian draws for both a run's queued-to-started and
// queued-to-ended gap (see generateRandomArrivalEvents) — chosen so half of
// all runs reach a given lifecycle stage within 5 minutes of being queued,
// the other half taking longer (a long, thin tail out past that), rather
// than every run taking the exact same fixed offset the tick-based
// generators use.
const benchArrivalStageMedian = 5 * time.Minute

// sampleExpMedian draws one duration from an exponential distribution
// rescaled so its median equals median (exponential, not just "random", to
// get the right shape for a real duration: mostly-short with a long tail of
// occasional slow runs, and memoryless — no built-in upper bound). rand.
// ExpFloat64 already draws from Exp(rate=1), whose median is math.Ln2; the
// rescale factor median/math.Ln2 converts that to Exp(rate=math.Ln2/median),
// whose median is exactly median.
func sampleExpMedian(median time.Duration) time.Duration {
	return time.Duration(rand.ExpFloat64() * float64(median) / math.Ln2)
}

// timedRunEvent pairs a runEvent with the real time it fires at (queued_at
// for a queued-stage event, started_at for a started-stage one, and so on)
// — generateRandomArrivalEvents' output, sorted and then consumed by
// applyEventsInTimeOrderedBatches, which needs that real time to reconstruct
// the actual arrival order across every run's interleaved lifecycle.
type timedRunEvent struct {
	at time.Time
	ev runEvent
}

// generateRandomArrivalEvents returns every lifecycle event (3 per run, like
// generateRunEvent) for totalRuns runs, indexed [indexOffset,
// indexOffset+totalRuns) globally (see generateInterleavedBatchesParallel's
// doc comment for why a global, not shard-local, index matters for
// account_id assignment) — unsorted; the caller sorts by .at.
//
// Per run: queued_at is base plus a uniform-random offset somewhere in
// [0, timeRange) ("insert randomly over the range" — every run's arrival
// time is equally likely anywhere in the window, unlike the tick-based
// generators' evenly-spaced arrivals). started_at and ended_at are each
// independently sampled as base+queued_at-relative offsets via
// sampleExpMedian, then sorted so the smaller becomes started_at and the
// larger ended_at — both are measured relative to queued_at specifically
// (not chained started->ended), per the modeling requirement that "started"
// and "ended" are each their own random gap since the run was queued, not a
// compounding pair of gaps.
func generateRandomArrivalEvents(batchSeed, indexOffset, totalRuns int, timeRange time.Duration, accountIDs []uuid.UUID) []timedRunEvent {
	base := time.Now()
	events := make([]timedRunEvent, 0, totalRuns*eventsPerRun)
	for i := range totalRuns {
		g := indexOffset + i
		accountID := accountIDs[g%len(accountIDs)]
		runID := fmt.Sprintf("run-%d-%d", batchSeed, g)

		queuedAt := base.Add(time.Duration(rand.Float64() * float64(timeRange)))
		startedOffset, endedOffset := sampleExpMedian(benchArrivalStageMedian), sampleExpMedian(benchArrivalStageMedian)
		if startedOffset > endedOffset {
			startedOffset, endedOffset = endedOffset, startedOffset
		}
		startedAt := queuedAt.Add(startedOffset)
		endedAt := queuedAt.Add(endedOffset)

		events = append(events,
			timedRunEvent{at: queuedAt, ev: runEvent{
				accountID: accountID, runID: runID, status: statusBySeq[0],
				queuedAt: queuedAt, attributes: attributesBySeq[0], updatedAt: queuedAt,
			}},
			timedRunEvent{at: startedAt, ev: runEvent{
				accountID: accountID, runID: runID, status: statusBySeq[1],
				queuedAt: queuedAt, startedAt: sql.NullTime{Time: startedAt, Valid: true},
				attributes: attributesBySeq[1], updatedAt: startedAt,
			}},
			timedRunEvent{at: endedAt, ev: runEvent{
				accountID: accountID, runID: runID, status: statusBySeq[2],
				queuedAt: queuedAt, startedAt: sql.NullTime{Time: startedAt, Valid: true},
				endedAt:    sql.NullTime{Time: endedAt, Valid: true},
				attributes: attributesBySeq[2], updatedAt: endedAt,
			}},
		)
	}
	return events
}

// applyEventsInTimeOrderedBatches sorts events by .at and calls apply once
// per fixed, batchSize-sized chunk of the sorted list, in arrival order.
//
// Unlike generateInterleavedBatches' fixed pipelineDepth spacing (which
// guarantees no two events for the same run_id ever land in one batch),
// arrival times here are random, so a run's own started_at could in
// principle land arbitrarily close to its queued_at (sampleExpMedian has no
// lower bound), with nothing else scheduled to fall between them — this
// function makes no attempt to prevent that. A chunk with two same-key rows
// is harmless for flatInsertAppenderBatch (a plain INSERT; every row lands,
// same-run or not), but would be unsafe for a literal, undeduplicated MERGE
// INTO (WHEN NOT MATCHED evaluates against the target's state as of the
// statement's start, not row-by-row, so two same-key unmatched source rows
// would both take the INSERT branch and duplicate that row). The
// QuackMergeAppender-based scenarios this generator feeds
// (mergeUpsertAppenderBatch/mergeJSONPatchAppenderBatch, see
// merge_appender_bench_test.go) handle that with QuackMergeConfig's
// DedupKeys/DedupOrderBy instead — a SQL-side QUALIFY dedup on the MERGE's
// own USING subquery — rather than this generator pre-avoiding collisions
// itself.
func applyEventsInTimeOrderedBatches(events []timedRunEvent, batchSize int, apply func(batch []runEvent) error) error {
	sort.Slice(events, func(i, j int) bool { return events[i].at.Before(events[j].at) })

	for start := 0; start < len(events); start += batchSize {
		end := min(start+batchSize, len(events))
		batch := make([]runEvent, end-start)
		for i, te := range events[start:end] {
			batch[i] = te.ev
		}
		if err := apply(batch); err != nil {
			return err
		}
	}
	return nil
}

// generateRandomArrivalBatches is generateInterleavedBatches' random-arrival
// counterpart: totalRuns runs (globally indexed from indexOffset, see
// generateRandomArrivalEvents), scattered across timeRange and batched via
// applyEventsInTimeOrderedBatches instead of a fixed tick sweep.
func generateRandomArrivalBatches(batchSeed, indexOffset, totalRuns int, timeRange time.Duration, batchSize int, accountIDs []uuid.UUID, apply func(batch []runEvent) error) error {
	events := generateRandomArrivalEvents(batchSeed, indexOffset, totalRuns, timeRange, accountIDs)
	return applyEventsInTimeOrderedBatches(events, batchSize, apply)
}

// generateRandomArrivalBatchesParallel is
// generateInterleavedBatchesParallel's random-arrival counterpart: same
// benchQuackConns-way sharding by disjoint, globally-indexed run ranges (so
// account_id assignment still spans the full accountIDs range per shard —
// see generateInterleavedBatchesParallel's doc comment for why that matters),
// each shard scattering its own runs across the same timeRange independently
// and applying with conflict retry.
func generateRandomArrivalBatchesParallel(batchSeed, totalRuns int, timeRange time.Duration, batchSize int, accountIDs []uuid.UUID, apply func(batch []runEvent) error) error {
	shards := min(benchQuackConns, totalRuns)
	if shards <= 1 {
		return generateRandomArrivalBatches(batchSeed, 0, totalRuns, timeRange, batchSize, accountIDs, apply)
	}

	retryingApply := func(batch []runEvent) error { return applyWithConflictRetry(apply, batch) }

	var g errgroup.Group
	base := totalRuns / shards
	extra := totalRuns % shards
	offset := 0
	for s := range shards {
		chunk := base
		if s < extra {
			chunk++
		}
		shardOffset, shardRuns := offset, chunk
		g.Go(func() error {
			return generateRandomArrivalBatches(batchSeed, shardOffset, shardRuns, timeRange, batchSize, accountIDs, retryingApply)
		})
		offset += chunk
	}
	return g.Wait()
}

// benchMonth approximates a calendar month as a fixed duration (30 days) —
// time.Duration has no calendar-aware unit, and benchTimeRanges only needs a
// representative order of magnitude, not calendar precision.
const benchMonth = 30 * 24 * time.Hour

// benchTimeRanges sweeps how spread out (vs. bursty) totalRuns runs' arrival
// times are, holding row count and partition count fixed — the random-
// arrival analogue of benchPartitionCounts. A short range packs the same
// million writes into a narrow window, so many runs' lifecycles overlap in
// time (closer to a real traffic spike); a long range spreads them thin.
var benchTimeRanges = []time.Duration{
	1 * time.Hour,
	24 * time.Hour,
	7 * 24 * time.Hour,
	benchMonth,
	6 * benchMonth,
}

// runAppenderInsertScenariosByTimeRange is runAppenderInsertScenarios'
// random-arrival counterpart: same per-appenderInsertScenarios sweep and
// rows/sec reporting, but driven by generateRandomArrivalBatchesParallel
// (timeRange, fixed accountIDs) instead of a batchGenerator mode. openDB
// selects the catalog under test — openAppenderBenchDB for
// BenchmarkAppenderInsertByTimeRange, openAppenderBenchDBSQLite for
// BenchmarkAppenderInsertByTimeRangeSQLite.
func runAppenderInsertScenariosByTimeRange(b *testing.B, totalRuns int, timeRange time.Duration, accountIDs []uuid.UUID, openDB func(testing.TB) *sql.DB) {
	for _, sc := range appenderInsertScenarios {
		b.Run(sc.name, func(b *testing.B) {
			iter := 0
			for b.Loop() {
				db := openDB(b)
				ctx := b.Context()
				err := generateRandomArrivalBatchesParallel(iter, totalRuns, timeRange, benchBatchSize, accountIDs, func(batch []runEvent) error {
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

// BenchmarkAppenderInsertByTimeRange is BenchmarkAppenderInsertByPartitions'
// random-arrival counterpart: same fixed row count
// (benchPartitionSweepRows) and fixed account_id set (benchAccountIDs,
// round-robin — partition alignment isn't modeled for random arrival yet),
// but sweeps benchTimeRanges instead of benchPartitionCounts x insertModes.
// Run the same way as BenchmarkAppenderWrites100k (see its doc comment),
// substituting this benchmark's name.
func BenchmarkAppenderInsertByTimeRange(b *testing.B) {
	totalRuns := benchPartitionSweepRows / eventsPerRun
	for _, tr := range benchTimeRanges {
		b.Run(fmt.Sprintf("range=%s", tr), func(b *testing.B) {
			runAppenderInsertScenariosByTimeRange(b, totalRuns, tr, benchAccountIDs, openAppenderBenchDB)
		})
	}
}

// BenchmarkAppenderInsertByTimeRangeSQLite is BenchmarkAppenderInsertByTimeRange's
// SQLite-catalog counterpart — see sqlite_bench_test.go's package doc
// comment for what that catalog mode is.
func BenchmarkAppenderInsertByTimeRangeSQLite(b *testing.B) {
	totalRuns := benchPartitionSweepRows / eventsPerRun
	for _, tr := range benchTimeRanges {
		b.Run(fmt.Sprintf("range=%s", tr), func(b *testing.B) {
			runAppenderInsertScenariosByTimeRange(b, totalRuns, tr, benchAccountIDs, openAppenderBenchDBSQLite)
		})
	}
}

// seedBatchSeed is the batchSeed runAppenderInsertScenariosByTimeRangeIntoSeededTable
// uses to pre-populate a table before timing: distinct from every batchSeed
// (0, 1, 2, ...) the timed loop itself uses, so the seed data's run_ids
// (run--1-*) can never collide with a timed iteration's run_ids (run-0-*,
// run-1-*, ...) — both sets must coexist afterward as genuinely separate
// rows, not overwrite one another.
const seedBatchSeed = -1

// runAppenderInsertScenariosByTimeRangeIntoSeededTable is
// runAppenderInsertScenariosByTimeRange's counterpart for measuring the
// marginal cost of inserting into a table that already holds data, rather
// than an empty one: before the timed loop starts, it writes totalRuns runs
// (the same random-arrival shape and size as the timed insert itself — "an
// already populated uniformly distributed data set of the same size") into
// each scenario's table, untimed, then times inserting a second, equally
// sized batch of *new* runs (disjoint run_ids — see seedBatchSeed) on top of
// that. This exercises whatever the seeded rows changed about the target
// table's on-disk shape (existing Parquet files/partitions a MERGE INTO now
// has to search, existing delete files, etc.) that an empty-table run like
// runAppenderInsertScenariosByTimeRange never encounters.
//
// The seed write and the timed write share the exact same timeRange window
// (both scattered uniformly across it — see generateRandomArrivalEvents),
// so the seeded table looks like "this same workload, already having run
// once" rather than a differently-shaped dataset.
func runAppenderInsertScenariosByTimeRangeIntoSeededTable(b *testing.B, totalRuns int, timeRange time.Duration, accountIDs []uuid.UUID) {
	for _, sc := range appenderInsertScenarios {
		b.Run(sc.name, func(b *testing.B) {
			db := openAppenderBenchDB(b)
			ctx := b.Context()

			seedErr := generateRandomArrivalBatchesParallel(seedBatchSeed, totalRuns, timeRange, benchBatchSize, accountIDs, func(batch []runEvent) error {
				return sc.apply(ctx, db, batch)
			})
			if seedErr != nil {
				b.Fatalf("seeding: %v", seedErr)
			}

			iter := 0
			for b.Loop() {
				err := generateRandomArrivalBatchesParallel(iter, totalRuns, timeRange, benchBatchSize, accountIDs, func(batch []runEvent) error {
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

// BenchmarkAppenderInsertByTimeRangeIntoSeededTable is
// BenchmarkAppenderInsertByTimeRange's counterpart against an already-
// populated table — see runAppenderInsertScenariosByTimeRangeIntoSeededTable's
// doc comment. Run the same way as BenchmarkAppenderWrites100k (see its doc
// comment), substituting this benchmark's name.
func BenchmarkAppenderInsertByTimeRangeIntoSeededTable(b *testing.B) {
	totalRuns := benchPartitionSweepRows / eventsPerRun
	for _, tr := range benchTimeRanges {
		b.Run(fmt.Sprintf("range=%s", tr), func(b *testing.B) {
			runAppenderInsertScenariosByTimeRangeIntoSeededTable(b, totalRuns, tr, benchAccountIDs)
		})
	}
}

// TestAppenderInsertIntoSeededTableKeepsSeedAndNewRowsDisjoint guards
// runAppenderInsertScenariosByTimeRangeIntoSeededTable's core premise:
// seedBatchSeed keeps the untimed seed write's run_ids disjoint from the
// timed write's (batchSeed 0), so writing totalRuns runs into an
// already-seeded totalRuns-row table ends with exactly 2*totalRuns distinct
// run_ids — neither overwriting the seed data nor colliding with it — not
// still totalRuns (which a real run_id collision would silently produce for
// a MERGE-based table, upserting the timed batch on top of the seed batch
// instead of adding new rows).
func TestAppenderInsertIntoSeededTableKeepsSeedAndNewRowsDisjoint(t *testing.T) {
	const totalRuns = 100
	const batchSize = 11
	accountIDs := generateAccountIDs(5)

	db := openAppenderBenchDB(t)
	ctx := t.Context()

	seedErr := generateRandomArrivalBatchesParallel(seedBatchSeed, totalRuns, time.Minute, batchSize, accountIDs, func(batch []runEvent) error {
		return mergeUpsertAppenderBatch(ctx, db, batch)
	})
	if seedErr != nil {
		t.Fatalf("seeding: %v", seedErr)
	}

	if err := generateRandomArrivalBatchesParallel(0, totalRuns, time.Minute, batchSize, accountIDs, func(batch []runEvent) error {
		return mergeUpsertAppenderBatch(ctx, db, batch)
	}); err != nil {
		t.Fatalf("inserting into seeded table: %v", err)
	}

	const want = 2 * totalRuns
	var count, distinct int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+driver.DuckLakeAlias+".bench_runs_merged_appender;").Scan(&count); err != nil {
		t.Fatalf("counting rows: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT count(DISTINCT run_id) FROM "+driver.DuckLakeAlias+".bench_runs_merged_appender;").Scan(&distinct); err != nil {
		t.Fatalf("counting distinct run_ids: %v", err)
	}
	if count != want {
		t.Errorf("got %d rows, want %d (seed + new, one row per run_id)", count, want)
	}
	if distinct != want {
		t.Errorf("got %d distinct run_ids, want %d", distinct, want)
	}
}

// TestApplyEventsInTimeOrderedBatchesCoversEveryEventExactlyOnce guards
// applyEventsInTimeOrderedBatches' only real invariant now that same-batch
// run_id collisions are handled downstream (QuackMergeConfig.DedupKeys) and
// not here: every event handed in must be applied exactly once, split into
// batches of at most batchSize.
func TestApplyEventsInTimeOrderedBatchesCoversEveryEventExactlyOnce(t *testing.T) {
	const totalRuns = 500
	const batchSize = 7 // deliberately not a divisor of totalRuns*eventsPerRun, so the final batch is a genuine partial one
	accountIDs := generateAccountIDs(5)

	events := generateRandomArrivalEvents(0, 0, totalRuns, time.Minute, accountIDs)
	if len(events) != totalRuns*eventsPerRun {
		t.Fatalf("got %d events, want %d", len(events), totalRuns*eventsPerRun)
	}

	var totalApplied int
	err := applyEventsInTimeOrderedBatches(events, batchSize, func(batch []runEvent) error {
		if len(batch) == 0 || len(batch) > batchSize {
			t.Errorf("got batch of %d events, want between 1 and %d", len(batch), batchSize)
		}
		totalApplied += len(batch)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if totalApplied != len(events) {
		t.Errorf("got %d events applied across all batches, want %d", totalApplied, len(events))
	}
}

// TestGenerateRandomArrivalEventsStagesAfterQueuedAt guards
// generateRandomArrivalEvents' timing invariant: every run's started_at and
// ended_at must fall at or after its own queued_at (started_at <= ended_at
// too, since generateRandomArrivalEvents sorts the two sampled offsets
// before assigning them) — sampleExpMedian only ever returns non-negative
// durations, but this pins that the offsets are actually applied atop
// queued_at consistently, not compared against some other run's queued_at.
func TestGenerateRandomArrivalEventsStagesAfterQueuedAt(t *testing.T) {
	accountIDs := generateAccountIDs(3)
	events := generateRandomArrivalEvents(0, 0, 200, time.Hour, accountIDs)

	byRun := make(map[string][]runEvent, 200)
	for _, te := range events {
		byRun[te.ev.runID] = append(byRun[te.ev.runID], te.ev)
	}
	for runID, evs := range byRun {
		if len(evs) != eventsPerRun {
			t.Fatalf("run_id %s: got %d events, want %d", runID, len(evs), eventsPerRun)
		}
		queuedAt := evs[0].queuedAt
		for _, e := range evs {
			if e.startedAt.Valid && e.startedAt.Time.Before(queuedAt) {
				t.Errorf("run_id %s: started_at %v before queued_at %v", runID, e.startedAt.Time, queuedAt)
			}
			if e.endedAt.Valid && e.endedAt.Time.Before(queuedAt) {
				t.Errorf("run_id %s: ended_at %v before queued_at %v", runID, e.endedAt.Time, queuedAt)
			}
			if e.startedAt.Valid && e.endedAt.Valid && e.endedAt.Time.Before(e.startedAt.Time) {
				t.Errorf("run_id %s: ended_at %v before started_at %v", runID, e.endedAt.Time, e.startedAt.Time)
			}
		}
	}
}

// TestRandomArrivalMergeTablesMatchFlatLatest is
// TestRunsMergeTablesMatchFlatLatest's random-arrival counterpart: after
// writing a small dataset via generateRandomArrivalBatchesParallel,
// bench_runs_merged_appender must hold exactly one row per run with the
// flat table's latest-row-per-run_id state — proving
// mergeUpsertAppenderBatch's QuackMergeConfig.DedupKeys correctly resolves
// whatever same-run_id collisions this generator's random arrival times
// produce within a batch (unlike the tick-based generators, this one makes
// no attempt to avoid them — see applyEventsInTimeOrderedBatches' doc
// comment), rather than silently duplicating or losing rows.
//
// This uses the appender-based tables/writers (openAppenderBenchDB,
// flatInsertAppenderBatch, mergeUpsertAppenderBatch), not
// merge_bench_test.go's literal-SQL ones: the literal-SQL mergeUpsertBatch
// has no dedup mechanism at all and would be unsafe against a batch with
// same-run_id collisions, which this generator can genuinely produce.
func TestRandomArrivalMergeTablesMatchFlatLatest(t *testing.T) {
	const numRuns = 100
	accountIDs := generateAccountIDs(5)

	db := openAppenderBenchDB(t)
	ctx := t.Context()
	err := generateRandomArrivalBatchesParallel(0, numRuns, time.Minute, 11, accountIDs, func(batch []runEvent) error {
		if err := flatInsertAppenderBatch(ctx, db, batch); err != nil {
			return fmt.Errorf("seeding bench_runs_flat_appender: %w", err)
		}
		return mergeUpsertAppenderBatch(ctx, db, batch)
	})
	if err != nil {
		t.Fatal(err)
	}

	var mergedCount int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+driver.DuckLakeAlias+".bench_runs_merged_appender").Scan(&mergedCount); err != nil {
		t.Fatalf("counting bench_runs_merged_appender: %v", err)
	}
	if mergedCount != numRuns {
		t.Errorf("bench_runs_merged_appender: got %d rows, want %d (one per run_id)", mergedCount, numRuns)
	}

	flatLatest := `SELECT status FROM ` + driver.DuckLakeAlias + `.bench_runs_flat_appender
		WHERE run_id = ? QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id ORDER BY updated_at DESC) = 1`
	for i := range numRuns {
		runID := fmt.Sprintf("run-0-%d", i)

		var wantStatus string
		if err := db.QueryRowContext(ctx, flatLatest, runID).Scan(&wantStatus); err != nil {
			t.Fatalf("reading flat latest for %s: %v", runID, err)
		}
		if wantStatus != "completed" {
			t.Fatalf("flat latest for %s: got status %q, want %q (test setup bug)", runID, wantStatus, "completed")
		}

		var gotStatus string
		if err := db.QueryRowContext(ctx, "SELECT status FROM "+driver.DuckLakeAlias+".bench_runs_merged_appender WHERE run_id = ?", runID).Scan(&gotStatus); err != nil {
			t.Fatalf("reading bench_runs_merged_appender for %s: %v", runID, err)
		}
		if gotStatus != wantStatus {
			t.Errorf("bench_runs_merged_appender %s: got status %q, want %q", runID, gotStatus, wantStatus)
		}
	}
}
