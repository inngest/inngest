package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	duckdbdriver "github.com/inngest/inngest/pkg/duckdb/driver"
	"golang.org/x/sync/errgroup"
)

// Summary reports how many rows were written per table.
type Summary struct {
	Runs     int
	Spans    int
	Events   int
	Metadata int
}

// Timings reports how much time GenerateAndInsert's two stages actually
// spent working. Generate is the generator goroutine's time and Insert the
// sum across every table writer, not wall-clock: generation overlaps
// inserts, and the tables (and writer sets) write concurrently, so their
// sum can exceed the real elapsed time.
type Timings struct {
	Generate time.Duration
	Insert   time.Duration
	// Tables has one entry per lake table, in seedTables order.
	Tables []TableTimings
}

// TableTimings is one table's writers' cumulative work.
type TableTimings struct {
	Table   string
	Rows    int
	Flushes int
	// Busy is the summed time this table's writers spent building rows,
	// sending them and committing.
	Busy time.Duration
}

// Column kind lists mirror each table's column order exactly, as created by
// pkg/db/duckdb/migrations (000001_baseline.sql). duckdbdriver.QuackAppender's wire
// protocol carries no column names — a row's values are matched
// positionally against the table's own full column list — so there is no
// way to scope the appender to a subset of columns: defaulted columns such
// as inngest.runs' is_deferred must be supplied explicitly
// too (see runRowValues). A schema change to any of these tables must be
// mirrored here; insert_test.go exercises every table against the real
// migrations.
var (
	runColumns = []duckdbdriver.QuackColumnKind{
		duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnTimestampTZ, // account_id, env_id, run_id, queued_at
		duckdbdriver.QuackColumnTimestampTZ, duckdbdriver.QuackColumnTimestampTZ, duckdbdriver.QuackColumnTimestampTZ, // scheduled_at, started_at, ended_at
		duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnVarchar, // app_id, app_name, function_id, function_slug
		duckdbdriver.QuackColumnVarchar,                                                                   // status
		duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, // attributes, inputs, output
		duckdbdriver.QuackColumnVarchar,     // event_ids — VARCHAR[] via array-literal text, see runRowValues' doc comment
		duckdbdriver.QuackColumnVarchar,     // sessions — STRUCT(key VARCHAR, id VARCHAR)[] via JSON-literal text, see sessionsLiteral's doc comment
		duckdbdriver.QuackColumnVarchar,     // is_deferred (BOOLEAN, VARCHAR-cast) — this tool never generates deferred runs
		duckdbdriver.QuackColumnVarchar,     // defer_parent_fn_slug
		duckdbdriver.QuackColumnVarchar,     // defer_parent_run_ids — VARCHAR[], same array-literal text as event_ids
		duckdbdriver.QuackColumnVarchar,     // trace_id
		duckdbdriver.QuackColumnTimestampTZ, // bucket_at
	}
	spanColumns = []duckdbdriver.QuackColumnKind{
		duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnTimestampTZ, // account_id, env_id, run_id, run_queued_at
		duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnVarchar, // app_id, app_name, function_id, function_slug
		duckdbdriver.QuackColumnVarchar,                                          // name
		duckdbdriver.QuackColumnTimestampTZ, duckdbdriver.QuackColumnTimestampTZ, // start_time, end_time
		duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, // trace_id, span_id, parent_span_id
		duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, // attributes, output, input
		duckdbdriver.QuackColumnTimestampTZ, // bucket_at
	}
	eventColumns = []duckdbdriver.QuackColumnKind{
		duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnTimestampTZ, // account_id, env_id, internal_id, received_at
		duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, // source, source_id, event_id, event_name
		duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnTimestampTZ, duckdbdriver.QuackColumnVarchar, // event_data, event_v, event_ts, event_meta
		duckdbdriver.QuackColumnTimestampTZ, // bucket_at
	}
	// metadataColumns' step_index/step_attempt (INTEGER) and is_user
	// (BOOLEAN) have no native duckdbdriver.QuackColumnKind — see that type's own
	// doc comment: only UUID/VARCHAR/JSON/TIMESTAMPTZ exist — so they're
	// sent as QuackColumnVarchar text (see metadataRowValues) and rely on
	// DuckDB's Appender doing implicit VARCHAR->INTEGER/BOOLEAN casting, the
	// same mechanism already relied on (and verified against a real
	// duckdb-quack server) for event_ids' VARCHAR->LIST casting above.
	metadataColumns = []duckdbdriver.QuackColumnKind{
		duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnUUID, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnTimestampTZ, // account_id, env_id, run_id, run_queued_at
		duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, // span_id, scope
		duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, // step_id, step_index, step_attempt
		duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnVarchar, // kind, is_user
		duckdbdriver.QuackColumnVarchar, duckdbdriver.QuackColumnTimestampTZ, // values, created_at
		duckdbdriver.QuackColumnTimestampTZ, // bucket_at
	}
)

// indexedBatch carries a batch's real, stable identity (its position in
// generation/chunking order) alongside its rows — batches can finish
// inserting out of order once parallelism > 1, so this is what onProgress
// reports instead of a derived approximation.
type indexedBatch struct {
	index int
	rows  []GeneratedRun
}

// seedTable is one lake table this tool writes, w/ how to pull its rows out
// of a batch of generated runs.
type seedTable struct {
	name    string
	columns []duckdbdriver.QuackColumnKind
	// rows appends each row's values for this table to dst.
	rows func(dst [][]any, batch []GeneratedRun) [][]any
}

// seedTables are written in this order, but each by its own writers (see
// drainAndInsert), so they flush independently.
var seedTables = []seedTable{
	{name: "runs", columns: runColumns, rows: func(dst [][]any, batch []GeneratedRun) [][]any {
		for _, g := range batch {
			dst = append(dst, runRowValues(g.Run))
		}
		return dst
	}},
	{name: "run_trace_spans", columns: spanColumns, rows: func(dst [][]any, batch []GeneratedRun) [][]any {
		for _, g := range batch {
			for _, s := range g.Spans {
				dst = append(dst, spanRowValues(s))
			}
		}
		return dst
	}},
	{name: "events", columns: eventColumns, rows: func(dst [][]any, batch []GeneratedRun) [][]any {
		for _, g := range batch {
			for _, e := range g.Events {
				dst = append(dst, eventRowValues(e))
			}
		}
		return dst
	}},
	{name: "run_metadata", columns: metadataColumns, rows: func(dst [][]any, batch []GeneratedRun) [][]any {
		for _, g := range batch {
			for _, m := range g.Metadata {
				dst = append(dst, metadataRowValues(m))
			}
		}
		return dst
	}},
}

// InsertOptions tunes how generated runs are written.
type InsertOptions struct {
	// BatchSize is how many runs (and their spans/events/metadata) go into
	// one flush per table. Clamped to >= 1.
	BatchSize int
	// Parallelism is how many writer sets run, each w/ one writer (its own
	// quack connection + appender) per table. Clamped to >= 1. Values > 1
	// mean concurrent Iceberg commits to the same table, which conflict and
	// retry.
	Parallelism int
	// OnProgress, if non-nil, is called once each batch is flushed to every
	// table, w/ that batch's own 1-based index and the total batch count
	// (stable regardless of completion order), plus the cumulative number of
	// runs written so far and the overall total. Calls are serialized.
	OnProgress func(batchIndex, totalBatches, done, total int)
}

func (o InsertOptions) normalized() InsertOptions {
	o.BatchSize = max(o.BatchSize, 1)
	o.Parallelism = max(o.Parallelism, 1)
	return o
}

// runRowValues matches runColumns' order exactly. event_ids (VARCHAR[]) is
// sent as array-literal text (e.g. `["a","b"]`, `"[]"` for empty, nil for
// NULL) rather than a native list value: DuckDB's Appender does implicit
// VARCHAR->LIST casting from that text, which sidesteps a real server-side
// gap — duckdb-quack's AppendRequest handler crashes (empty-bodied HTTP
// 500) on any native LIST-typed column, verified down to the minimal case;
// see pkg/duckdb/driver/internal/quack/append.go's ColumnKind.wireID doc comment for the full
// writeup. scheduled_at is NOT NULL: a run with none falls back to its
// queued_at, as the dual-write path's materializeRuns does. bucket_at is
// when the run's final row would have reached the buffer (runBucketAt).
func runRowValues(r RunRow) []any {
	scheduledAt := r.QueuedAt
	if r.ScheduledAt != nil {
		scheduledAt = *r.ScheduledAt
	}
	return []any{
		r.AccountID, r.EnvID, r.RunID, r.QueuedAt,
		scheduledAt, nullableTime(r.StartedAt), nullableTime(r.EndedAt),
		r.AppID, r.AppName, r.FunctionID, r.FunctionSlug, r.Status,
		nonEmptyJSON(r.Attributes), r.Inputs, r.Output, eventIDsLiteral(r.EventIDs), sessionsLiteral(r.Sessions),
		"false", // is_deferred — this tool never generates deferred runs
		nil,     // defer_parent_fn_slug
		"[]",    // defer_parent_run_ids
		r.TraceID,
		runBucketAt(r),
	}
}

func spanRowValues(s SpanRow) []any {
	var parent any
	if s.ParentSpanID != nil {
		parent = *s.ParentSpanID
	}
	return []any{
		s.AccountID, s.EnvID, s.RunID, s.RunQueuedAt,
		s.AppID, s.AppName, s.FunctionID, s.FunctionSlug, s.Name, s.StartTime, s.EndTime,
		s.TraceID, s.SpanID, parent, s.Attributes, nullableJSON(s.Output), nullableJSON(s.Input),
		s.EndTime, // bucket_at: a span is written when it ends
	}
}

// metadataRowValues matches metadataColumns' order exactly. step_index/
// step_attempt/is_user go through the same VARCHAR-text-plus-implicit-cast
// trick as event_ids above (see metadataColumns' doc comment) since
// duckdbdriver.QuackColumnKind has no native INTEGER/BOOLEAN wire type.
func metadataRowValues(m MetadataRow) []any {
	var stepID, stepIndex, stepAttempt any
	if m.StepID != nil {
		stepID = *m.StepID
	}
	if m.StepIndex != nil {
		stepIndex = strconv.Itoa(*m.StepIndex)
	}
	if m.StepAttempt != nil {
		stepAttempt = strconv.Itoa(*m.StepAttempt)
	}
	return []any{
		m.AccountID, m.EnvID, m.RunID, m.RunQueuedAt,
		m.SpanID, m.Scope,
		stepID, stepIndex, stepAttempt,
		m.Kind, strconv.FormatBool(m.IsUser),
		m.Values, m.CreatedAt,
		m.CreatedAt, // bucket_at: an emission is written when it's recorded
	}
}

func eventRowValues(e EventRow) []any {
	var sourceID any
	if e.SourceID != nil {
		sourceID = *e.SourceID
	}
	return []any{
		e.AccountID, e.EnvID, e.InternalID, e.ReceivedAt,
		e.Source, sourceID, e.EventID, e.EventName,
		e.EventData, e.EventV, e.EventTS, e.EventMeta,
		e.ReceivedAt, // bucket_at
	}
}

// runBucketAt is when a run's latest row would have reached the buffer:
// when it ended, else started, else was queued.
func runBucketAt(r RunRow) time.Time {
	switch {
	case r.EndedAt != nil:
		return *r.EndedAt
	case r.StartedAt != nil:
		return *r.StartedAt
	}
	return r.QueuedAt
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// nonEmptyJSON supplies "{}" for a NOT NULL JSON column with no value.
func nonEmptyJSON(s string) string {
	if s == "" {
		return `{}`
	}
	return s
}

// nullableJSON substitutes SQL NULL for an empty JSON-column value --
// jsonText (sample.go) returns "" for a sampled column that was itself SQL
// NULL, and an empty string is not valid JSON, so writing it verbatim into
// a JSON text column (output/input) would leave readers an unparseable
// value rather than faithfully reproducing "this span/run had no
// output/input".
func nullableJSON(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// eventIDsLiteral renders s as DuckDB array-literal text (e.g.
// `["a","b"]` — JSON array syntax is valid DuckDB list-literal syntax too,
// verified empirically), or nil for NULL when s is empty — see
// runRowValues' doc comment for why this goes through VARCHAR->LIST casting
// instead of a native list value. ULIDs (the only values ever in
// RunRow.EventIDs) never contain characters JSON needs to escape, but
// json.Marshal is used regardless rather than hand-building the literal.
func eventIDsLiteral(s []string) any {
	if len(s) == 0 {
		return "[]"
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return string(b)
}

// sessionsLiteral renders pairs as DuckDB list-of-struct-literal text (e.g.
// `[{"key":"a","id":"b"}]`), or "[]" when pairs is empty — the same
// VARCHAR->LIST(STRUCT) implicit-casting trick eventIDsLiteral uses for
// event_ids' VARCHAR->LIST casting, verified empirically (real `duckdb`
// CLI, v1.5.2) that a JSON-object-shaped VARCHAR string casts cleanly into
// a STRUCT(key VARCHAR, id VARCHAR)[] column, the same as DuckDB's own
// single-quoted struct-literal syntax. SessionPair's json tags ("key"/"id")
// match the column's struct field names exactly.
func sessionsLiteral(pairs []SessionPair) any {
	if len(pairs) == 0 {
		return "[]"
	}
	b, err := json.Marshal(pairs)
	if err != nil {
		return nil
	}
	return string(b)
}

// InsertGeneratedRuns writes every generated run, its span tree, its
// trigger events and its metadata, opts.BatchSize runs per flush (see
// InsertOptions).
func InsertGeneratedRuns(ctx context.Context, connector *duckdbdriver.Connector, generated []GeneratedRun, opts InsertOptions) (Summary, error) {
	opts = opts.normalized()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var chunks [][]GeneratedRun
	for start := 0; start < len(generated); start += opts.BatchSize {
		chunks = append(chunks, generated[start:min(start+opts.BatchSize, len(generated))])
	}

	work := make(chan indexedBatch)
	go func() {
		defer close(work)
		for i, c := range chunks {
			select {
			case work <- indexedBatch{index: i + 1, rows: c}:
			case <-ctx.Done():
				return
			}
		}
	}()

	summary, _, err := drainAndInsert(ctx, connector, work, opts, len(generated), len(chunks))
	return summary, err
}

// GenerateAndInsert generates cfg.RunCount runs in opts.BatchSize-sized
// chunks and inserts each chunk as soon as it's generated, instead of
// building the entire run set in memory before any insert begins.
// Generation runs on a single dedicated goroutine (math/rand.Rand is not
// safe for concurrent use) feeding the same writers InsertGeneratedRuns
// uses, so batch N+1 can be generated while batch N is still being written.
func GenerateAndInsert(ctx context.Context, connector *duckdbdriver.Connector, rng *rand.Rand, tmpl Templates, cfg GenerateConfig, opts InsertOptions) (Summary, Timings, error) {
	opts = opts.normalized()
	total := cfg.RunCount
	totalBatches := (total + opts.BatchSize - 1) / opts.BatchSize

	// Cancelled once the writers are done, so a write error can't leave the
	// generator blocked on work forever.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var genNanos atomic.Int64
	work := make(chan indexedBatch)
	genDone := make(chan error, 1)
	go func() {
		defer close(work)
		remaining := total
		for i := 0; remaining > 0; i++ {
			n := min(opts.BatchSize, remaining)
			start := time.Now()
			rows, err := GenerateRuns(rng, tmpl, GenerateConfig{RunCount: n, Window: cfg.Window, Now: cfg.Now})
			genNanos.Add(int64(time.Since(start)))
			if err != nil {
				genDone <- err
				return
			}
			select {
			case work <- indexedBatch{index: i + 1, rows: rows}:
			case <-ctx.Done():
				genDone <- nil
				return
			}
			remaining -= n
		}
		genDone <- nil
	}()

	summary, timings, err := drainAndInsert(ctx, connector, work, opts, total, totalBatches)
	cancel()
	genErr := <-genDone
	timings.Generate = time.Duration(genNanos.Load())
	if err != nil {
		return summary, timings, err
	}
	return summary, timings, genErr
}

// batchState tracks one batch across its per-table writers.
type batchState struct {
	index int
	runs  int
	// remaining counts tables that haven't flushed this batch yet.
	remaining atomic.Int32
}

type tableWork struct {
	batch *batchState
	rows  []GeneratedRun
}

// drainAndInsert writes every batch from work until it closes, w/
// opts.Parallelism writers per table, each its own quack connection and
// appender. A dispatcher hands each batch to every table's queue, so the
// tables flush in parallel and a slow table (spans) only holds back its own
// writers until its queue fills. Iceberg commits to different tables don't
// conflict, so w/ Parallelism 1 nothing ever retries.
//
// A batch is reported to opts.OnProgress once its last table has flushed.
func drainAndInsert(ctx context.Context, connector *duckdbdriver.Connector, work <-chan indexedBatch, opts InsertOptions, total, totalBatches int) (Summary, Timings, error) {
	var (
		mu       sync.Mutex
		summary  Summary
		timings  Timings
		runsDone int
	)
	timings.Tables = make([]TableTimings, len(seedTables))
	for i, t := range seedTables {
		timings.Tables[i].Table = t.name
	}

	g, gctx := errgroup.WithContext(ctx)

	queues := make([]chan tableWork, len(seedTables))
	for i := range queues {
		queues[i] = make(chan tableWork, 1)
	}
	g.Go(func() error {
		defer func() {
			for _, q := range queues {
				close(q)
			}
		}()
		for {
			var b indexedBatch
			var ok bool
			select {
			case b, ok = <-work:
			case <-gctx.Done():
				return nil
			}
			if !ok {
				return nil
			}
			st := &batchState{index: b.index, runs: len(b.rows)}
			st.remaining.Store(int32(len(seedTables)))
			for _, q := range queues {
				select {
				case q <- tableWork{batch: st, rows: b.rows}:
				case <-gctx.Done():
					return nil
				}
			}
		}
	})

	for ti, table := range seedTables {
		for range opts.Parallelism {
			g.Go(func() error {
				conn, err := connector.Connect(gctx)
				if err != nil {
					return fmt.Errorf("duckdbseed: opening %s connection: %w", table.name, err)
				}
				defer conn.Close()
				appender, err := duckdbdriver.NewQuackAppenderFromConn(gctx, conn, duckdbdriver.LakeAlias, "main", table.name, table.columns)
				if err != nil {
					return fmt.Errorf("duckdbseed: creating %s appender: %w", table.name, err)
				}
				defer appender.Close(gctx)

				var rows [][]any
				for w := range queues[ti] {
					start := time.Now()
					rows = table.rows(rows[:0], w.rows)
					for _, r := range rows {
						if err := appender.AppendRow(r...); err != nil {
							return fmt.Errorf("duckdbseed: appending %s row: %w", table.name, err)
						}
					}
					if err := appender.Flush(gctx); err != nil {
						return fmt.Errorf("duckdbseed: flushing %s batch %d: %w", table.name, w.batch.index, err)
					}
					busy := time.Since(start)

					mu.Lock()
					tt := &timings.Tables[ti]
					tt.Rows += len(rows)
					tt.Flushes++
					tt.Busy += busy
					timings.Insert += busy
					if w.batch.remaining.Add(-1) == 0 {
						runsDone += w.batch.runs
						if opts.OnProgress != nil {
							opts.OnProgress(w.batch.index, totalBatches, runsDone, total)
						}
					}
					mu.Unlock()
				}
				return nil
			})
		}
	}

	err := g.Wait()
	summary.Runs = timings.Tables[0].Rows
	summary.Spans = timings.Tables[1].Rows
	summary.Events = timings.Tables[2].Rows
	summary.Metadata = timings.Tables[3].Rows
	return summary, timings, err
}

// sortRuns, sortSpans, and sortEvents sort a batch's rows in place to match
// each table's declared SORTED BY key exactly
// (pkg/db/duckdb/migrations/000001_baseline.sql) — GenerateRuns produces
// rows in arbitrary RNG order, so without this, writes would land out of
// the order the lake sorts by for min/max pruning. Each key is built as a
// single sortable string — year and month zero-padded, timestamps as
// zero-padded UnixNano — so a plain lexicographic string comparison
// reproduces the exact multi-column ORDER BY the schema declares, including
// that tenant/natural-key columns outrank the trailing timestamp column(s)
// within the same year/month.

func sortRuns(rows []RunRow) {
	sort.Slice(rows, func(i, j int) bool { return runSortKey(rows[i]) < runSortKey(rows[j]) })
}

// runSortKey mirrors inngest.runs' SORTED BY (year(queued_at),
// month(queued_at), account_id, env_id, run_id, queued_at).
func runSortKey(r RunRow) string {
	return fmt.Sprintf("%04d-%02d-%s-%s-%s-%020d",
		r.QueuedAt.Year(), r.QueuedAt.Month(), r.AccountID, r.EnvID, r.RunID, r.QueuedAt.UnixNano())
}

func sortSpans(rows []SpanRow) {
	sort.Slice(rows, func(i, j int) bool { return spanSortKey(rows[i]) < spanSortKey(rows[j]) })
}

// spanSortKey mirrors inngest.run_trace_spans' SORTED BY
// (year(run_queued_at), month(run_queued_at), account_id, env_id, run_id,
// start_time, end_time).
func spanSortKey(s SpanRow) string {
	return fmt.Sprintf("%04d-%02d-%s-%s-%s-%020d-%020d",
		s.RunQueuedAt.Year(), s.RunQueuedAt.Month(), s.AccountID, s.EnvID, s.RunID,
		s.StartTime.UnixNano(), s.EndTime.UnixNano())
}

func sortEvents(rows []EventRow) {
	sort.Slice(rows, func(i, j int) bool { return eventSortKey(rows[i]) < eventSortKey(rows[j]) })
}

// eventSortKey mirrors inngest.events' SORTED BY (year(received_at),
// month(received_at), account_id, env_id, internal_id, received_at).
func eventSortKey(e EventRow) string {
	return fmt.Sprintf("%04d-%02d-%s-%s-%s-%020d",
		e.ReceivedAt.Year(), e.ReceivedAt.Month(), e.AccountID, e.EnvID, e.InternalID, e.ReceivedAt.UnixNano())
}

func sortMetadata(rows []MetadataRow) {
	sort.Slice(rows, func(i, j int) bool { return metadataSortKey(rows[i]) < metadataSortKey(rows[j]) })
}

// metadataSortKey mirrors inngest.run_metadata's SORTED BY
// (year(run_queued_at), month(run_queued_at), account_id, env_id, run_id,
// scope, step_id, step_index, step_attempt, span_id, kind). step_index/
// step_attempt sort as -1 when NULL (a run-scoped row) — only meant to keep
// output deterministic, not to reproduce DuckDB's own NULL-ordering rules.
func metadataSortKey(m MetadataRow) string {
	var stepID string
	if m.StepID != nil {
		stepID = *m.StepID
	}
	stepIndex, stepAttempt := -1, -1
	if m.StepIndex != nil {
		stepIndex = *m.StepIndex
	}
	if m.StepAttempt != nil {
		stepAttempt = *m.StepAttempt
	}
	return fmt.Sprintf("%04d-%02d-%s-%s-%s-%s-%s-%020d-%020d-%s-%s",
		m.RunQueuedAt.Year(), m.RunQueuedAt.Month(), m.AccountID, m.EnvID, m.RunID,
		m.Scope, stepID, stepIndex, stepAttempt, m.SpanID, m.Kind)
}
