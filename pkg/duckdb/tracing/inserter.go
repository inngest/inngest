package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/oklog/ulid/v2"
)

// Span is one run trace span, as the listener's hooks produce it (via this
// listener's own TracerProvider and spanExportRow). It is the source for
// both inngest.run_trace_spans and, for run-level span names, inngest.runs.
type Span struct {
	AccountID    string
	EnvID        string
	AppID        string
	AppName      string
	FunctionID   string
	FunctionSlug string
	RunID        string
	RunQueuedAt  time.Time

	TraceID      string
	SpanID       string
	ParentSpanID string
	Name         string
	StartTime    time.Time
	EndTime      time.Time

	// Attributes is always set (a JSON object).
	Attributes json.RawMessage
	// Output and Error are the span's output envelope split
	// (cqrs.SplitSpanOutputEnvelope): Output is its data value and Error its
	// error value, nil unless the span failed. Output, Error and Input are
	// nil when the span carries none (stored as NULL).
	Output json.RawMessage
	Error  json.RawMessage
	Input  json.RawMessage
}

// Event is one received event, as OnEventReceived produces it.
type Event struct {
	AccountID  uuid.UUID
	EnvID      uuid.UUID
	InternalID ulid.ULID
	ReceivedAt time.Time
	Source     string
	// SourceID is nil when the event has no ingest source ID (stored as NULL).
	SourceID  *string
	EventID   string
	EventName string
	EventTS   time.Time
	// EventData and EventMeta are always set (EventData defaults to {}).
	EventData json.RawMessage
	EventMeta json.RawMessage
	EventV    string
}

// RunMetadata is one metadata emission, as OnMetadataEntry produces it.
type RunMetadata struct {
	AccountID   string
	EnvID       string
	RunID       string
	RunQueuedAt time.Time
	SpanID      string
	Scope       string
	Kind        string
	IsUser      bool
	Values      json.RawMessage
	CreatedAt   time.Time
	// Step fields are nil for run-scoped metadata.
	StepID      *string
	StepIndex   *int
	StepAttempt *int
}

// Inserter persists the entities the listener produces. The listener buffers
// each entity type in its own batcher (batch.go) and calls the matching
// method once per flushed batch, so an Inserter only does the write. Runs are
// not an entity of their own here: they're derived from run-level spans, and
// it's up to the Inserter how (DuckDBInserter builds inngest.runs rows from
// each span batch; a ClickHouse-backed one would rely on a materialized view).
//
// Each entity's batcher runs on its own goroutine, so implementations must
// be safe for concurrent calls across methods. A returned error drops that
// batch after logging, except driver.ErrDisabled, which stops every batcher
// sharing the listener's disabled state. If an Inserter also implements
// interface{ Close() error }, the listener's Close calls it after every
// batcher has drained.
type Inserter interface {
	InsertSpans(ctx context.Context, spans []Span) error
	InsertEvents(ctx context.Context, events []Event) error
	InsertRunMetadata(ctx context.Context, entries []RunMetadata) error
}

// DuckDBInserter is the Inserter for the DuckDB dual-write: one literal
// multi-row INSERT per batch into inngest.<table> (span batches also
// write their runs rows, see InsertSpans).
type DuckDBInserter struct {
	db *sql.DB
}

// NewDuckDBInserter returns a DuckDBInserter writing to db. Its Close closes
// db.
func NewDuckDBInserter(db *sql.DB) *DuckDBInserter {
	return &DuckDBInserter{db: db}
}

var spanColumns = []string{
	"account_id", "env_id", "app_id", "app_name", "function_id", "function_slug",
	"run_id", "run_queued_at", "trace_id", "span_id", "parent_span_id", "name",
	"start_time", "end_time", "attributes", "output", "error", "input",
}

// spanBatchTable is the per-connection temp table InsertSpans stages each
// span batch in.
const spanBatchTable = "temp.main.dualwrite_span_batch"

// bucketAtSQL is inngest.run_trace_spans.bucket_at's column default (see
// 000001_baseline.sql). InsertSpans stamps it once while staging, so a span
// and the runs row built from it share one bucket_at, same as when runs were
// read back from the stored span.
const bucketAtSQL = "make_timestamp_ms(epoch_ms(current_timestamp))"

// rollbackTimeout bounds the ROLLBACK after a failed span batch.
const rollbackTimeout = 5 * time.Second

// InsertSpans stages the batch once in a temp table (spanBatchTable) and
// inserts both inngest.run_trace_spans and inngest.runs from it (see
// runsInsertSQL), so building the batch's runs rows never reads the lake.
//
// On a quack connection the whole thing runs as one transaction, so a
// failure can't leave the batch's spans w/o their runs rows (nothing ever
// rebuilds runs from spans, so that run row would be lost for good). DuckLake
// commits multi-table transactions atomically. NOTE: an Iceberg catalog
// can't commit 2 tables atomically unless it advertises
// /v1/transactions/commit, so there this degrades to 2 commits; the
// batcher's retries (retryableFlushError) cover most of that gap. The
// jsonlines transport (tests) can't recover from a failed statement inside a
// transaction (see driver.SupportsTransactions), so it runs the same
// statements w/o one.
func (d *DuckDBInserter) InsertSpans(ctx context.Context, spans []Span) error {
	if len(spans) == 0 {
		return nil
	}
	rows := make([][]any, len(spans))
	for i, s := range spans {
		rows[i] = []any{
			s.AccountID, s.EnvID, s.AppID, s.AppName, s.FunctionID, s.FunctionSlug,
			s.RunID, s.RunQueuedAt, s.TraceID, s.SpanID, nullableString(s.ParentSpanID), s.Name,
			s.StartTime, s.EndTime, s.Attributes, nullableJSON(s.Output), nullableJSON(s.Error), nullableJSON(s.Input),
		}
	}

	cols := strings.Join(spanColumns, ", ") + ", bucket_at"
	stageSQL, args := insertSQL(spanBatchTable, spanColumns, rows, [2]string{"bucket_at", bucketAtSQL})
	stmts := []string{
		// Clones run_trace_spans' column types (VARIANT etc.) so the runs
		// projection sees exactly what it would reading the stored span.
		// LIMIT 0 is planned as an empty result, so this doesn't scan the
		// lake.
		"CREATE OR REPLACE TEMP TABLE " + spanBatchTable + " AS SELECT * FROM inngest.run_trace_spans LIMIT 0;",
		stageSQL,
		fmt.Sprintf("INSERT INTO inngest.run_trace_spans (%s) SELECT %s FROM %s;", cols, cols, spanBatchTable),
	}
	if slices.ContainsFunc(spans, func(s Span) bool { return slices.Contains(runsMaterializingSpanNames, s.Name) }) {
		stmts = append(stmts, runsInsertSQL(spanBatchTable))
	}
	stmts = append(stmts, "DROP TABLE "+spanBatchTable+";")

	conn, err := d.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if !driver.SupportsTransactions(conn) {
		_, err = conn.ExecContext(ctx, strings.Join(stmts, "\n"), args...)
		return err
	}

	script := "BEGIN TRANSACTION;\n" + strings.Join(stmts, "\n") + "\nCOMMIT;"
	if _, err := conn.ExecContext(ctx, script, args...); err != nil {
		// A failed statement leaves the transaction aborted on this session
		// (a failed COMMIT has already rolled back, and this just errors).
		// Roll back so the pooled conn is usable again.
		rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
		_, _ = conn.ExecContext(rctx, "ROLLBACK;")
		cancel()
		return err
	}
	return nil
}

var eventColumns = []string{
	"account_id", "env_id", "internal_id", "received_at", "source", "source_id", "event_id",
	"event_name", "event_ts", "event_data", "event_v", "event_meta",
}

func (d *DuckDBInserter) InsertEvents(ctx context.Context, events []Event) error {
	rows := make([][]any, len(events))
	for i, e := range events {
		rows[i] = []any{
			e.AccountID, e.EnvID, e.InternalID, e.ReceivedAt, e.Source, nullable(e.SourceID), e.EventID,
			e.EventName, e.EventTS, e.EventData, e.EventV, e.EventMeta,
		}
	}
	return d.insert(ctx, "inngest.events", eventColumns, rows)
}

var runMetadataColumns = []string{
	"account_id", "env_id", "run_id", "run_queued_at", "span_id", "scope", "kind",
	"is_user", "values", "created_at", "step_id", "step_index", "step_attempt",
}

func (d *DuckDBInserter) InsertRunMetadata(ctx context.Context, entries []RunMetadata) error {
	rows := make([][]any, len(entries))
	for i, m := range entries {
		rows[i] = []any{
			m.AccountID, m.EnvID, m.RunID, m.RunQueuedAt, m.SpanID, m.Scope, m.Kind,
			m.IsUser, m.Values, m.CreatedAt, nullable(m.StepID), nullable(m.StepIndex), nullable(m.StepAttempt),
		}
	}
	return d.insert(ctx, "inngest.run_metadata", runMetadataColumns, rows)
}

// Close closes the db passed to NewDuckDBInserter.
func (d *DuckDBInserter) Close() error {
	return d.db.Close()
}

// insert writes rows (each in cols order) into table with one literal
// multi-row INSERT.
func (d *DuckDBInserter) insert(ctx context.Context, table string, cols []string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	query, args := insertSQL(table, cols, rows)
	_, err := d.db.ExecContext(ctx, query, args...)
	return err
}

// insertSQL builds one literal multi-row INSERT of rows (each in cols order)
// into table, w/ a ? per value. extra are {column, SQL expression} pairs
// appended to every row after cols (InsertSpans passes bucket_at's).
func insertSQL(table string, cols []string, rows [][]any, extra ...[2]string) (string, []any) {
	var sb strings.Builder
	colList := strings.Join(cols, ", ")
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", ")
	for _, e := range extra {
		colList += ", " + e[0]
		placeholders += ", " + e[1]
	}
	fmt.Fprintf(&sb, "INSERT INTO %s (%s) VALUES ", table, colList)

	placeholders = "(" + placeholders + ")"
	args := make([]any, 0, len(rows)*len(cols))
	for i, row := range rows {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(placeholders)
		args = append(args, row...)
	}
	sb.WriteString(";")
	return sb.String(), args
}

// nullableJSON maps an absent JSON value to SQL NULL rather than an empty
// (invalid) JSON literal.
func nullableJSON(v json.RawMessage) any {
	if len(v) == 0 {
		return nil
	}
	return v
}

// nullableString maps "" to SQL NULL, so a root span's parent_span_id is
// NULL rather than empty.
func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// nullable maps a nil pointer to SQL NULL and a set one to its value.
func nullable[T any](v *T) any {
	if v == nil {
		return nil
	}
	return *v
}
