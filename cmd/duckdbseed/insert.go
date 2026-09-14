package main

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
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
// spent working. Generate and Insert are each a sum across every goroutine
// that did that kind of work — the single generator goroutine for Generate,
// every parallel Appender worker for Insert — not wall-clock, since
// GenerateAndInsert deliberately overlaps batch N+1's generation with batch
// N's insert (see its own doc comment). Their sum can therefore exceed the
// stage's real elapsed time; a caller wanting that overlap benefit
// quantified should compare Generate+Insert against its own wall-clock
// measurement around the call.
type Timings struct {
	Generate time.Duration
	Insert   time.Duration
}

// Column kind lists mirror pkg/db/duckdb/migrations/000001_baseline.sql's
// (and, for runs, 000002_runs_is_deferred.sql's appended column) column
// order for each table exactly. duckdb.QuackAppender's wire protocol carries
// no column names — a row's values are matched positionally against the
// table's own full column list — so, unlike the earlier duckdb-go-based
// version of this tool, there is no way to scope the appender to a subset
// of columns: inngest.runs' inserted_at (DEFAULT current_timestamp) must be
// supplied explicitly too (see runRowValues), even though nothing else in
// this codebase ever needs to read it back.
var (
	runColumns = []duckdb.QuackColumnKind{
		duckdb.QuackColumnUUID, duckdb.QuackColumnUUID, duckdb.QuackColumnVarchar, duckdb.QuackColumnTimestampMS, // account_id, env_id, run_id, queued_at
		duckdb.QuackColumnTimestampMS, duckdb.QuackColumnTimestampMS, duckdb.QuackColumnTimestampMS, // scheduled_at, started_at, ended_at
		duckdb.QuackColumnUUID, duckdb.QuackColumnUUID, duckdb.QuackColumnVarchar, // app_id, function_id, status
		duckdb.QuackColumnJSON, duckdb.QuackColumnJSON, // inputs, output
		duckdb.QuackColumnVarchar,     // event_ids — VARCHAR[] via array-literal text, see runRowValues' doc comment
		duckdb.QuackColumnVarchar,     // sessions — STRUCT(key VARCHAR, id VARCHAR)[] via JSON-literal text, see sessionsLiteral's doc comment
		duckdb.QuackColumnTimestampMS, // inserted_at
		duckdb.QuackColumnVarchar,     // is_deferred (BOOLEAN, VARCHAR-cast) — this tool never generates deferred runs, so always NULL
	}
	spanColumns = []duckdb.QuackColumnKind{
		duckdb.QuackColumnUUID, duckdb.QuackColumnUUID, duckdb.QuackColumnVarchar, duckdb.QuackColumnTimestampMS, // account_id, env_id, run_id, run_queued_at
		duckdb.QuackColumnUUID, duckdb.QuackColumnUUID, duckdb.QuackColumnVarchar, // app_id, function_id, name
		duckdb.QuackColumnTimestampMS, duckdb.QuackColumnTimestampMS, // start_time, end_time
		duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, // trace_id, span_id, parent_span_id
		duckdb.QuackColumnJSON, duckdb.QuackColumnJSON, duckdb.QuackColumnJSON, duckdb.QuackColumnJSON, // attributes, links, output, input
	}
	eventColumns = []duckdb.QuackColumnKind{
		duckdb.QuackColumnUUID, duckdb.QuackColumnUUID, duckdb.QuackColumnVarchar, duckdb.QuackColumnTimestampMS, // account_id, env_id, internal_id, received_at
		duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, // source, source_id, event_id, event_name
		duckdb.QuackColumnJSON, duckdb.QuackColumnVarchar, duckdb.QuackColumnTimestampMS, duckdb.QuackColumnJSON, // event_data, event_v, event_ts, event_meta
	}
	// metadataColumns' step_index/step_attempt (INTEGER) and is_user
	// (BOOLEAN) have no native duckdb.QuackColumnKind — see that type's own
	// doc comment: only UUID/VARCHAR/JSON/TIMESTAMP_MS exist — so they're
	// sent as QuackColumnVarchar text (see metadataRowValues) and rely on
	// DuckDB's Appender doing implicit VARCHAR->INTEGER/BOOLEAN casting, the
	// same mechanism already relied on (and verified against a real
	// duckdb-quack server) for event_ids' VARCHAR->LIST casting above.
	metadataColumns = []duckdb.QuackColumnKind{
		duckdb.QuackColumnUUID, duckdb.QuackColumnUUID, duckdb.QuackColumnVarchar, duckdb.QuackColumnTimestampMS, // account_id, env_id, run_id, run_queued_at
		duckdb.QuackColumnUUID, duckdb.QuackColumnUUID, duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, // app_id, function_id, span_id, scope
		duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, // step_id, step_index, step_attempt
		duckdb.QuackColumnVarchar, duckdb.QuackColumnVarchar, // kind, is_user
		duckdb.QuackColumnJSON, duckdb.QuackColumnTimestampMS, // values, created_at
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

// appenderSet holds one dedicated connection's three table Appenders,
// created once and reused across every batch a worker processes — DuckDB
// Appenders are not safe for concurrent use, so each parallel worker must
// own its own connection (obtained via Connector.Connect directly, bypassing
// *sql.DB's pool — see duckdb.OpenConnector's doc comment for why) and its
// own set.
type appenderSet struct {
	conn     driver.Conn
	runs     *duckdb.QuackAppender
	spans    *duckdb.QuackAppender
	events   *duckdb.QuackAppender
	metadata *duckdb.QuackAppender
}

func newAppenderSet(ctx context.Context, connector *duckdb.Connector) (*appenderSet, error) {
	conn, err := connector.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("duckdbseed: opening appender connection: %w", err)
	}

	runs, err := duckdb.NewQuackAppenderFromConn(ctx, conn, duckdb.DuckLakeAlias, "main", "runs", runColumns)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("duckdbseed: creating runs appender: %w", err)
	}
	spans, err := duckdb.NewQuackAppenderFromConn(ctx, conn, duckdb.DuckLakeAlias, "main", "run_trace_spans", spanColumns)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("duckdbseed: creating run_trace_spans appender: %w", err)
	}
	events, err := duckdb.NewQuackAppenderFromConn(ctx, conn, duckdb.DuckLakeAlias, "main", "events", eventColumns)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("duckdbseed: creating events appender: %w", err)
	}
	metadata, err := duckdb.NewQuackAppenderFromConn(ctx, conn, duckdb.DuckLakeAlias, "main", "run_metadata", metadataColumns)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("duckdbseed: creating run_metadata appender: %w", err)
	}

	return &appenderSet{conn: conn, runs: runs, spans: spans, events: events, metadata: metadata}, nil
}

func (a *appenderSet) close(ctx context.Context) error {
	return errors.Join(a.runs.Close(ctx), a.spans.Close(ctx), a.events.Close(ctx), a.metadata.Close(ctx), a.conn.Close())
}

// insertBatch flattens one chunk of generated runs into its four tables'
// rows and appends them — in the run/event/span/metadata order the tool
// has always used (metadata appended last, after spans) — flushing each
// appender once the batch is fully appended.
func (a *appenderSet) insertBatch(ctx context.Context, batch []GeneratedRun) (Summary, error) {
	var runs []RunRow
	var spans []SpanRow
	var events []EventRow
	var metadataRows []MetadataRow
	for _, g := range batch {
		runs = append(runs, g.Run)
		spans = append(spans, g.Spans...)
		events = append(events, g.Events...)
		metadataRows = append(metadataRows, g.Metadata...)
	}

	now := time.Now().UTC()
	for _, r := range runs {
		if err := a.runs.AppendRow(runRowValues(r, now)...); err != nil {
			return Summary{}, fmt.Errorf("duckdbseed: appending run %s: %w", r.RunID, err)
		}
	}
	for _, e := range events {
		if err := a.events.AppendRow(eventRowValues(e)...); err != nil {
			return Summary{}, fmt.Errorf("duckdbseed: appending event %s: %w", e.InternalID, err)
		}
	}
	for _, s := range spans {
		if err := a.spans.AppendRow(spanRowValues(s)...); err != nil {
			return Summary{}, fmt.Errorf("duckdbseed: appending span %s: %w", s.SpanID, err)
		}
	}
	for _, md := range metadataRows {
		if err := a.metadata.AppendRow(metadataRowValues(md)...); err != nil {
			return Summary{}, fmt.Errorf("duckdbseed: appending metadata for span %s: %w", md.SpanID, err)
		}
	}

	if err := a.runs.Flush(ctx); err != nil {
		return Summary{}, fmt.Errorf("duckdbseed: flushing runs appender: %w", err)
	}
	if err := a.events.Flush(ctx); err != nil {
		return Summary{}, fmt.Errorf("duckdbseed: flushing events appender: %w", err)
	}
	if err := a.spans.Flush(ctx); err != nil {
		return Summary{}, fmt.Errorf("duckdbseed: flushing spans appender: %w", err)
	}
	if err := a.metadata.Flush(ctx); err != nil {
		return Summary{}, fmt.Errorf("duckdbseed: flushing metadata appender: %w", err)
	}

	return Summary{Runs: len(runs), Spans: len(spans), Events: len(events), Metadata: len(metadataRows)}, nil
}

// runRowValues matches runColumns' order exactly. event_ids (VARCHAR[]) is
// sent as array-literal text (e.g. `["a","b"]`, `"[]"` for empty, nil for
// NULL) rather than a native list value: DuckDB's Appender does implicit
// VARCHAR->LIST casting from that text, which sidesteps a real server-side
// gap — duckdb-quack's AppendRequest handler crashes (empty-bodied HTTP
// 500) on any native LIST-typed column, verified down to the minimal case;
// see pkg/db/duckdb/quack_append.go's wireID doc comment for the full
// writeup. now stands in for inserted_at's DEFAULT current_timestamp, which
// this appender (having no column-name info on the wire) cannot leave
// unsupplied — see runColumns' doc comment.
func runRowValues(r RunRow, now time.Time) []any {
	return []any{
		r.AccountID, r.EnvID, r.RunID, r.QueuedAt,
		nullableTime(r.ScheduledAt), nullableTime(r.StartedAt), nullableTime(r.EndedAt),
		r.AppID, r.FunctionID, r.Status,
		r.Inputs, r.Output, eventIDsLiteral(r.EventIDs), sessionsLiteral(r.Sessions),
		now,
		nil, // is_deferred — this tool never generates deferred runs
	}
}

func spanRowValues(s SpanRow) []any {
	var parent any
	if s.ParentSpanID != nil {
		parent = *s.ParentSpanID
	}
	return []any{
		s.AccountID, s.EnvID, s.RunID, s.RunQueuedAt,
		s.AppID, s.FunctionID, s.Name, s.StartTime, s.EndTime,
		s.TraceID, s.SpanID, parent, s.Attributes, nil, nullableJSON(s.Output), nullableJSON(s.Input),
	}
}

// metadataRowValues matches metadataColumns' order exactly. step_index/
// step_attempt/is_user go through the same VARCHAR-text-plus-implicit-cast
// trick as event_ids above (see metadataColumns' doc comment) since
// duckdb.QuackColumnKind has no native INTEGER/BOOLEAN wire type.
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
		m.AppID, m.FunctionID, m.SpanID, m.Scope,
		stepID, stepIndex, stepAttempt,
		m.Kind, strconv.FormatBool(m.IsUser),
		m.Values, m.CreatedAt,
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
	}
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// nullableJSON substitutes SQL NULL for an empty JSON-column value --
// jsonText (sample.go) returns "" for a sampled column that was itself SQL
// NULL, and an empty string is not valid JSON, so writing it verbatim into
// a JSON-typed column (output/input) fails at commit time ("Malformed
// JSON... input length is 0") rather than faithfully reproducing "this
// span/run had no output/input".
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
		return nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return string(b)
}

// sessionsLiteral renders pairs as DuckDB list-of-struct-literal text (e.g.
// `[{"key":"a","id":"b"}]`), or nil for NULL when pairs is empty — the same
// VARCHAR->LIST(STRUCT) implicit-casting trick eventIDsLiteral uses for
// event_ids' VARCHAR->LIST casting, verified empirically (real `duckdb`
// CLI, v1.5.2) that a JSON-object-shaped VARCHAR string casts cleanly into
// a STRUCT(key VARCHAR, id VARCHAR)[] column, the same as DuckDB's own
// single-quoted struct-literal syntax. SessionPair's json tags ("key"/"id")
// match the column's struct field names exactly.
func sessionsLiteral(pairs []SessionPair) any {
	if len(pairs) == 0 {
		return nil
	}
	b, err := json.Marshal(pairs)
	if err != nil {
		return nil
	}
	return string(b)
}

// InsertGeneratedRuns writes every generated run, its span tree, and its
// trigger events via parallelism concurrent Appenders (each its own
// connection to connector's embedded database), batchSize runs per flush.
//
// onProgress, if non-nil, is called once after each batch is flushed, with
// that batch's own 1-based index and the total batch count (its real
// identity — stable regardless of completion order), plus the cumulative
// number of runs written so far and the overall total. Calls are serialized
// regardless of parallelism. Safe to leave nil.
func InsertGeneratedRuns(ctx context.Context, connector *duckdb.Connector, generated []GeneratedRun, batchSize, parallelism int, onProgress func(batchIndex, totalBatches, done, total int)) (Summary, error) {
	batchSize = max(batchSize, 1)
	parallelism = max(parallelism, 1)

	var chunks [][]GeneratedRun
	for start := 0; start < len(generated); start += batchSize {
		chunks = append(chunks, generated[start:min(start+batchSize, len(generated))])
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

	summary, _, err := drainAndInsert(ctx, connector, work, parallelism, len(generated), len(chunks), onProgress, nil)
	return summary, err
}

// GenerateAndInsert generates cfg.RunCount runs in batchSize-sized chunks
// and inserts each chunk as soon as it's generated, instead of building the
// entire run set in memory before any insert begins. Generation runs on a
// single dedicated goroutine — math/rand.Rand is not safe for concurrent
// use — feeding the same worker pool InsertGeneratedRuns uses, so batch N+1
// can be generated while batch N is still being appended.
//
// batchSize, parallelism, and onProgress mean exactly what they do on
// InsertGeneratedRuns. The returned Timings' Generate/Insert are cumulative
// across the generator goroutine and every insert worker respectively — see
// Timings' doc comment for why that isn't the same as wall-clock.
func GenerateAndInsert(ctx context.Context, connector *duckdb.Connector, rng *rand.Rand, tmpl Templates, cfg GenerateConfig, batchSize, parallelism int, onProgress func(batchIndex, totalBatches, done, total int)) (Summary, Timings, error) {
	batchSize = max(batchSize, 1)
	parallelism = max(parallelism, 1)
	total := cfg.RunCount
	totalBatches := (total + batchSize - 1) / batchSize

	var genNanos atomic.Int64
	work := make(chan indexedBatch)
	var genErr error
	go func() {
		defer close(work)
		remaining := total
		for i := 0; remaining > 0; i++ {
			n := min(batchSize, remaining)
			start := time.Now()
			rows, err := GenerateRuns(rng, tmpl, GenerateConfig{RunCount: n, Window: cfg.Window, Now: cfg.Now})
			genNanos.Add(int64(time.Since(start)))
			if err != nil {
				genErr = err
				return
			}
			select {
			case work <- indexedBatch{index: i + 1, rows: rows}:
			case <-ctx.Done():
				return
			}
			remaining -= n
		}
	}()

	summary, insertNanos, err := drainAndInsert(ctx, connector, work, parallelism, total, totalBatches, onProgress, new(atomic.Int64))
	timings := Timings{Generate: time.Duration(genNanos.Load()), Insert: time.Duration(insertNanos)}
	if err != nil {
		return summary, timings, err
	}
	if genErr != nil {
		return summary, timings, genErr
	}
	return summary, timings, nil
}

// drainAndInsert runs exactly parallelism persistent worker goroutines,
// each with its own appenderSet, draining work until the channel closes.
// Shared by InsertGeneratedRuns (work is a pre-chunked slice) and
// GenerateAndInsert (work is generated just-in-time) so both get identical
// batching/parallelism/progress semantics from one implementation.
//
// insertNanos, if non-nil, accumulates every worker's own insertBatch
// timing (see Timings' doc comment); InsertGeneratedRuns has no use for it
// and passes nil. The second return value is always insertNanos' final
// total (0 when insertNanos was nil), returned by value since the caller's
// own accumulator may go out of scope before it can be read otherwise.
func drainAndInsert(ctx context.Context, connector *duckdb.Connector, work <-chan indexedBatch, parallelism, total, totalBatches int, onProgress func(batchIndex, totalBatches, done, total int), insertNanos *atomic.Int64) (Summary, int64, error) {
	var (
		mu      sync.Mutex
		summary Summary
	)

	g, gctx := errgroup.WithContext(ctx)
	for range parallelism {
		g.Go(func() error {
			appenders, err := newAppenderSet(gctx, connector)
			if err != nil {
				return err
			}
			defer appenders.close(gctx)

			for b := range work {
				start := time.Now()
				batchSummary, err := appenders.insertBatch(gctx, b.rows)
				if insertNanos != nil {
					insertNanos.Add(int64(time.Since(start)))
				}
				if err != nil {
					return err
				}

				mu.Lock()
				summary.Runs += batchSummary.Runs
				summary.Spans += batchSummary.Spans
				summary.Events += batchSummary.Events
				summary.Metadata += batchSummary.Metadata
				if onProgress != nil {
					onProgress(b.index, totalBatches, summary.Runs, total)
				}
				mu.Unlock()
			}
			return nil
		})
	}

	waitErr := g.Wait()
	var total64 int64
	if insertNanos != nil {
		total64 = insertNanos.Load()
	}
	if waitErr != nil {
		return summary, total64, waitErr
	}
	return summary, total64, nil
}

// sortRuns, sortSpans, and sortEvents sort a batch's rows in place to match
// each table's declared SORTED BY key exactly
// (pkg/db/duckdb/migrations/000001_baseline.sql) — GenerateRuns produces
// rows in arbitrary RNG order, so without this, writes would land out of
// the order DuckLake expects for zonemap pruning. Each key is built as a
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
