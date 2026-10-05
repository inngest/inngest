package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
	// Output and Input are nil when the span carries none (stored as NULL).
	Output json.RawMessage
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
// it's up to the Inserter how (DuckDBInserter materializes inngest.runs from
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
// multi-row INSERT per batch into inngest.<table>, plus materializeRuns for
// span batches.
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
	"start_time", "end_time", "attributes", "output", "input",
}

func (d *DuckDBInserter) InsertSpans(ctx context.Context, spans []Span) error {
	if len(spans) == 0 {
		return nil
	}
	rows := make([][]any, len(spans))
	for i, s := range spans {
		rows[i] = []any{
			s.AccountID, s.EnvID, s.AppID, s.AppName, s.FunctionID, s.FunctionSlug,
			s.RunID, s.RunQueuedAt, s.TraceID, s.SpanID, s.ParentSpanID, s.Name,
			s.StartTime, s.EndTime, s.Attributes, nullableJSON(s.Output), nullableJSON(s.Input),
		}
	}
	if err := d.insert(ctx, "inngest.run_trace_spans", spanColumns, rows); err != nil {
		return err
	}
	return materializeRuns(ctx, d.db, spans)
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

	var sb strings.Builder
	fmt.Fprintf(&sb, "INSERT INTO %s (%s) VALUES ", table, strings.Join(cols, ", "))

	placeholders := "(" + strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", ") + ")"
	args := make([]any, 0, len(rows)*len(cols))
	for i, row := range rows {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(placeholders)
		args = append(args, row...)
	}
	sb.WriteString(";")

	_, err := d.db.ExecContext(ctx, sb.String(), args...)
	return err
}

// nullableJSON maps an absent JSON value to SQL NULL rather than an empty
// (invalid) JSON literal.
func nullableJSON(v json.RawMessage) any {
	if len(v) == 0 {
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
