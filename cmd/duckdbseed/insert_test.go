package main

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/stretchr/testify/require"
)

// TestSortRunsMatchesTheRunsTableSortKey pins the exact key schema.go
// declares for inngest.runs: SORTED BY (year(queued_at), month(queued_at),
// account_id, env_id, run_id, queued_at) — tenant grouping outranks
// chronological order within the same year/month, so a later-queued run in
// an alphabetically-earlier account must still sort first.
func TestSortRunsMatchesTheRunsTableSortKey(t *testing.T) {
	sameMonth := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	accountA := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	accountB := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	envID := uuid.New()

	rows := []RunRow{
		{RunID: "b-run", AccountID: accountB, EnvID: envID, QueuedAt: sameMonth.Add(-time.Hour)}, // earlier in time, later account
		{RunID: "a-run", AccountID: accountA, EnvID: envID, QueuedAt: sameMonth},                 // later in time, earlier account
	}

	sortRuns(rows)

	require.Equal(t, []string{"a-run", "b-run"}, []string{rows[0].RunID, rows[1].RunID},
		"account_id must outrank queued_at within the same year/month")
}

// TestSortSpansMatchesTheSpansTableSortKey pins run_trace_spans' key:
// SORTED BY (year(run_queued_at), month(run_queued_at), account_id, env_id,
// run_id, start_time, end_time).
func TestSortSpansMatchesTheSpansTableSortKey(t *testing.T) {
	sameMonth := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	runA := uuid.MustParse("00000000-0000-0000-0000-000000000001").String()
	runB := uuid.MustParse("00000000-0000-0000-0000-000000000002").String()
	accountID, envID := uuid.New(), uuid.New()

	rows := []SpanRow{
		{RunID: runB, AccountID: accountID, EnvID: envID, RunQueuedAt: sameMonth, StartTime: sameMonth.Add(-time.Hour), EndTime: sameMonth.Add(-time.Hour)},
		{RunID: runA, AccountID: accountID, EnvID: envID, RunQueuedAt: sameMonth, StartTime: sameMonth, EndTime: sameMonth},
	}

	sortSpans(rows)

	require.Equal(t, []string{runA, runB}, []string{rows[0].RunID, rows[1].RunID},
		"run_id must outrank start_time within the same year/month/tenant")
}

// TestSortEventsMatchesTheEventsTableSortKey pins events' key:
// SORTED BY (year(received_at), month(received_at), account_id, env_id,
// internal_id, received_at).
func TestSortEventsMatchesTheEventsTableSortKey(t *testing.T) {
	sameMonth := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	accountID, envID := uuid.New(), uuid.New()

	rows := []EventRow{
		{InternalID: "01B", AccountID: accountID, EnvID: envID, ReceivedAt: sameMonth.Add(-time.Hour)},
		{InternalID: "01A", AccountID: accountID, EnvID: envID, ReceivedAt: sameMonth},
	}

	sortEvents(rows)

	require.Equal(t, []string{"01A", "01B"}, []string{rows[0].InternalID, rows[1].InternalID},
		"internal_id must outrank received_at within the same year/month/tenant")
}

func TestInsertGeneratedRunsWritesRunsSpansAndEvents(t *testing.T) {
	connector, db := newTestDB(t, 1)
	ctx := t.Context()

	tmpl := testTemplates()
	cfg := GenerateConfig{RunCount: 3, Window: time.Hour, Now: time.Now().UTC()}
	generated, err := GenerateRuns(rand.New(rand.NewSource(1)), tmpl, cfg)
	require.NoError(t, err)

	summary, err := InsertGeneratedRuns(ctx, connector, generated, 10, 1, nil)
	require.NoError(t, err)
	require.Equal(t, 3, summary.Runs)

	wantSpans, wantEvents := 0, 0
	for _, g := range generated {
		wantSpans += len(g.Spans)
		wantEvents += len(g.Events)
	}
	require.Equal(t, wantSpans, summary.Spans)
	require.Equal(t, wantEvents, summary.Events)

	var runCount int
	row := db.QueryRowContext(ctx, "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".runs;")
	require.NoError(t, row.Scan(&runCount))
	require.Equal(t, 3, runCount)

	var spanCount int
	row = db.QueryRowContext(ctx, "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".run_trace_spans;")
	require.NoError(t, row.Scan(&spanCount))
	require.Equal(t, wantSpans, spanCount)

	var eventCount int
	row = db.QueryRowContext(ctx, "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".events;")
	require.NoError(t, row.Scan(&eventCount))
	require.Equal(t, wantEvents, eventCount)
}

func TestInsertGeneratedRunsWithNoRunsIsANoop(t *testing.T) {
	connector, _ := newTestDB(t, 1)

	summary, err := InsertGeneratedRuns(t.Context(), connector, nil, 10, 1, nil)
	require.NoError(t, err)
	require.Equal(t, Summary{}, summary)
}

func TestInsertGeneratedRunsReportsProgressAfterEachBatch(t *testing.T) {
	connector, _ := newTestDB(t, 1)
	ctx := t.Context()

	tmpl := testTemplates()
	cfg := GenerateConfig{RunCount: 3, Window: time.Hour, Now: time.Now().UTC()}
	generated, err := GenerateRuns(rand.New(rand.NewSource(1)), tmpl, cfg)
	require.NoError(t, err)

	var mu sync.Mutex
	var progress [][4]int
	onProgress := func(batchIndex, totalBatches, done, total int) {
		mu.Lock()
		defer mu.Unlock()
		progress = append(progress, [4]int{batchIndex, totalBatches, done, total})
	}

	// batchSize=1 and parallelism=1 means one flush per run, strictly in
	// order — the same granularity and ordering as before batching existed.
	_, err = InsertGeneratedRuns(ctx, connector, generated, 1, 1, onProgress)
	require.NoError(t, err)

	require.Equal(t, [][4]int{{1, 3, 1, 3}, {2, 3, 2, 3}, {3, 3, 3, 3}}, progress)
}

func TestInsertGeneratedRunsBatchesAcrossMultipleRunsPerFlush(t *testing.T) {
	connector, db := newTestDB(t, 1)
	ctx := t.Context()

	tmpl := testTemplates()
	cfg := GenerateConfig{RunCount: 5, Window: time.Hour, Now: time.Now().UTC()}
	generated, err := GenerateRuns(rand.New(rand.NewSource(2)), tmpl, cfg)
	require.NoError(t, err)

	var mu sync.Mutex
	var progress [][4]int
	onProgress := func(batchIndex, totalBatches, done, total int) {
		mu.Lock()
		defer mu.Unlock()
		progress = append(progress, [4]int{batchIndex, totalBatches, done, total})
	}

	summary, err := InsertGeneratedRuns(ctx, connector, generated, 2, 1, onProgress)
	require.NoError(t, err)

	require.Equal(t, [][4]int{{1, 3, 2, 5}, {2, 3, 4, 5}, {3, 3, 5, 5}}, progress,
		"flushes every 2 runs, plus a final partial batch; batchIndex/totalBatches reflect the real 3-batch chunking")

	wantSpans, wantEvents := 0, 0
	for _, g := range generated {
		wantSpans += len(g.Spans)
		wantEvents += len(g.Events)
	}
	require.Equal(t, 5, summary.Runs)
	require.Equal(t, wantSpans, summary.Spans)
	require.Equal(t, wantEvents, summary.Events)

	var runCount int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&runCount))
	require.Equal(t, 5, runCount)
}

// TestInsertGeneratedRunsProgressBatchIndicesAreDistinctUnderConcurrency
// pins that batchIndex is the batch's real, stable identity — not derived
// from the running done count, which under concurrency can no longer be
// trusted to line up with any particular batch. Every batch must still
// report a distinct index in [1, totalBatches], and parallelism > 1 must
// still write every row exactly once (proving the per-worker appenderSet
// design doesn't lose or duplicate rows).
func TestInsertGeneratedRunsProgressBatchIndicesAreDistinctUnderConcurrency(t *testing.T) {
	connector, db := newTestDB(t, 4)
	ctx := t.Context()

	tmpl := testTemplates()
	cfg := GenerateConfig{RunCount: 40, Window: time.Hour, Now: time.Now().UTC()}
	generated, err := GenerateRuns(rand.New(rand.NewSource(4)), tmpl, cfg)
	require.NoError(t, err)

	var mu sync.Mutex
	seen := map[int]bool{}
	var totalBatchesSeen int
	onProgress := func(batchIndex, totalBatches, done, total int) {
		mu.Lock()
		defer mu.Unlock()
		seen[batchIndex] = true
		totalBatchesSeen = totalBatches
	}

	summary, err := InsertGeneratedRuns(ctx, connector, generated, 5, 4, onProgress)
	require.NoError(t, err)

	require.Equal(t, 8, totalBatchesSeen)
	require.Len(t, seen, 8)
	for i := 1; i <= 8; i++ {
		require.True(t, seen[i], "batch index %d must have been reported exactly once", i)
	}

	require.Equal(t, 40, summary.Runs)
	var runCount, distinctRunIDs int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&runCount))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(DISTINCT run_id) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&distinctRunIDs))
	require.Equal(t, 40, runCount)
	require.Equal(t, 40, distinctRunIDs, "no run should have been written more than once")
}

func TestInsertGeneratedRunsClampsNonPositiveBatchSizeToOne(t *testing.T) {
	connector, _ := newTestDB(t, 1)
	ctx := t.Context()

	tmpl := testTemplates()
	cfg := GenerateConfig{RunCount: 2, Window: time.Hour, Now: time.Now().UTC()}
	generated, err := GenerateRuns(rand.New(rand.NewSource(3)), tmpl, cfg)
	require.NoError(t, err)

	summary, err := InsertGeneratedRuns(ctx, connector, generated, 0, 1, nil)
	require.NoError(t, err)
	require.Equal(t, 2, summary.Runs)
}

// TestGenerateAndInsertMatchesPreGeneratingThenInserting is the core
// equivalence proof for interleaved generation: GenerateRuns' output only
// ever depends on the shared rng's current state and per-call cfg — never
// on how many runs a prior call asked for — so calling it once per batch
// (interleaved with each batch's insert) must produce byte-identical rows,
// in the same order, as generating the whole run in one call up front. Both
// paths seed a fresh rng from the same source seed and write into separate
// DBs so the comparison is apples to apples.
func TestGenerateAndInsertMatchesPreGeneratingThenInserting(t *testing.T) {
	ctx := t.Context()
	tmpl := testTemplates()
	genCfg := GenerateConfig{RunCount: 11, Window: time.Hour, Now: time.Now().UTC()}

	preGenerated, err := GenerateRuns(rand.New(rand.NewSource(9)), tmpl, genCfg)
	require.NoError(t, err)
	preGenConnector, preGenDB := newTestDB(t, 1)
	wantSummary, err := InsertGeneratedRuns(ctx, preGenConnector, preGenerated, 4, 1, nil)
	require.NoError(t, err)

	interleavedConnector, interleavedDB := newTestDB(t, 1)
	gotSummary, err := GenerateAndInsert(ctx, interleavedConnector, rand.New(rand.NewSource(9)), tmpl, genCfg, 4, 1, nil)
	require.NoError(t, err)

	require.Equal(t, wantSummary, gotSummary)

	wantRunIDs := runIDsInDB(t, preGenDB)
	gotRunIDs := runIDsInDB(t, interleavedDB)
	require.Equal(t, wantRunIDs, gotRunIDs, "interleaved generation must produce the exact same run IDs as pre-generating everything")
}

func TestGenerateAndInsertReportsProgressPerBatch(t *testing.T) {
	connector, _ := newTestDB(t, 1)
	tmpl := testTemplates()
	genCfg := GenerateConfig{RunCount: 5, Window: time.Hour, Now: time.Now().UTC()}

	var mu sync.Mutex
	var progress [][4]int
	onProgress := func(batchIndex, totalBatches, done, total int) {
		mu.Lock()
		defer mu.Unlock()
		progress = append(progress, [4]int{batchIndex, totalBatches, done, total})
	}

	summary, err := GenerateAndInsert(t.Context(), connector, rand.New(rand.NewSource(1)), tmpl, genCfg, 2, 1, onProgress)
	require.NoError(t, err)
	require.Equal(t, 5, summary.Runs)

	require.Equal(t, [][4]int{{1, 3, 2, 5}, {2, 3, 4, 5}, {3, 3, 5, 5}}, progress)
}

// TestGenerateAndInsertPropagatesGenerationErrors proves a generation
// failure (here: no tenants to attribute runs to) surfaces as a real error
// rather than silently inserting nothing.
func TestGenerateAndInsertPropagatesGenerationErrors(t *testing.T) {
	connector, _ := newTestDB(t, 1)
	tmpl := testTemplates()
	tmpl.Tenants = nil
	genCfg := GenerateConfig{RunCount: 5, Window: time.Hour, Now: time.Now().UTC()}

	_, err := GenerateAndInsert(t.Context(), connector, rand.New(rand.NewSource(1)), tmpl, genCfg, 2, 1, nil)
	require.Error(t, err)
}

// TestGenerateAndInsertWithParallelism proves the interleaved
// generate+insert path also writes every row exactly once when multiple
// Appender workers run concurrently.
func TestGenerateAndInsertWithParallelism(t *testing.T) {
	connector, db := newTestDB(t, 4)
	tmpl := testTemplates()
	genCfg := GenerateConfig{RunCount: 40, Window: time.Hour, Now: time.Now().UTC()}

	summary, err := GenerateAndInsert(t.Context(), connector, rand.New(rand.NewSource(5)), tmpl, genCfg, 5, 4, nil)
	require.NoError(t, err)
	require.Equal(t, 40, summary.Runs)

	var runCount, distinctRunIDs int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&runCount))
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(DISTINCT run_id) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&distinctRunIDs))
	require.Equal(t, 40, runCount)
	require.Equal(t, 40, distinctRunIDs)
}
