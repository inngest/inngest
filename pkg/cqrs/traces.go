package cqrs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// Span represents an distributed span in a function execution flow
type Span struct {
	Timestamp          time.Time         `json:"ts"`
	TraceID            string            `json:"trace_id"`
	SpanID             string            `json:"span_id"`
	ParentSpanID       *string           `json:"parent_span_id,omitempty"`
	TraceState         *string           `json:"trace_state,omitempty"`
	SpanName           string            `json:"span_name"`
	SpanKind           string            `json:"span_kind"`
	ServiceName        string            `json:"service_name"`
	ResourceAttributes map[string]string `json:"rattr"`
	ScopeName          string            `json:"scope_name"`
	ScopeVersion       string            `json:"scope_version"`
	SpanAttributes     map[string]string `json:"sattr"`
	Duration           time.Duration     `json:"duration"`
	StatusCode         string            `json:"status_code"`
	StatusMessage      *string           `json:"status_message"`
	Events             []SpanEvent       `json:"events"`
	Links              []SpanLink        `json:"links"`
	RunID              *ulid.ULID        `json:"run_id"`
}

type SpanEvent struct {
	Timestamp  time.Time         `json:"ts"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attr"`
}

type SpanLink struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	TraceState string            `json:"trace_state"`
	Attributes map[string]string `json:"attr"`
}

// TraceRun represents a function run backed by a trace
type TraceRun struct {
	AccountID   uuid.UUID       `json:"account_id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	AppID       uuid.UUID       `json:"app_id"`
	FunctionID  uuid.UUID       `json:"function_id"`
	TraceID     string          `json:"trace_id"`
	RunID       ulid.ULID       `json:"run_id"`
	QueuedAt    time.Time       `json:"queued_at"`
	StartedAt   time.Time       `json:"started_at,omitempty"`
	EndedAt     time.Time       `json:"ended_at,omitempty"`
	Duration    time.Duration   `json:"duration"`
	SourceID    string          `json:"source_id,omitempty"`
	TriggerIDs  []ulid.ULID     `json:"trigger_ids"`
	Triggers    [][]byte        `json:"triggers"`
	Output      []byte          `json:"output,omitempty"`
	Status      enums.RunStatus `json:"status"`
	IsBatch     bool            `json:"is_batch"`
	IsDebounce  bool            `json:"is_debounce"`
	// Cursor is a composite cursor used for pagination
	Cursor string `json:"cursor"`
}

type TraceReadWriter interface {
	TraceWriter
	TraceReader
}

type TraceWriter interface {
	// InsertSpan writes a trace span into the backing store
	InsertSpan(ctx context.Context, span *Span) error
	// InsertTraceRun writes a trace based function run into the backing store
	InsertTraceRun(ctx context.Context, run *TraceRun) error
}

type TraceReader interface {
	// GetSpansByTraceIDAndRunID retrieves spans based on their traceID and runID
	GetSpansByTraceIDAndRunID(ctx context.Context, tid string, runID ulid.ULID) ([]*Span, error)
	// GetTraceRuns retrieves a list of TraceRun based on the options specified
	GetTraceRuns(ctx context.Context, opt GetTraceRunOpt) ([]*TraceRun, error)
}

type GetTraceRunOpt struct {
	Filter GetTraceRunFilter
	Order  []GetTraceRunOrder
	Cursor string
	Items  uint
}

type GetTraceRunFilter struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       []uuid.UUID
	FunctionID  []uuid.UUID
	TimeField   enums.TraceRunTime
	From        time.Time
	Until       time.Time
	Status      []enums.RunStatus
	CEL         string
}

type GetTraceRunOrder struct {
	Field     enums.TraceRunTime
	Direction enums.TraceRunOrder
}

// TracePageCursor represents the composite cursor used to handle pagination
type TracePageCursor struct {
	ID      string                 `json:"id"`
	Cursors map[string]TraceCursor `json:"c"`
}

func (c *TracePageCursor) IsEmpty() bool {
	return c.Cursors == nil || len(c.Cursors) == 0
}

// Find finds a cusor with the provided name
func (c *TracePageCursor) Find(field string) *TraceCursor {
	if c.IsEmpty() {
		return nil
	}

	f := strings.ToLower(field)
	if v, ok := c.Cursors[f]; ok {
		return &v
	}
	return nil
}

func (c *TracePageCursor) Add(field string) {
	f := strings.ToLower(field)
	if _, ok := c.Cursors[f]; !ok {
		c.Cursors[f] = TraceCursor{Field: f}
	}
}

func (c *TracePageCursor) Encode() (string, error) {
	byt, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(byt), nil
}

func (c *TracePageCursor) Decode(val string) error {
	if c.Cursors == nil {
		c.Cursors = map[string]TraceCursor{}
	}
	byt, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(byt, c)
}

// TraceCursor represents a cursor that is used as part of the pagination cursor
type TraceCursor struct {
	// Field represents the field used for this cursor
	Field string `json:"f"`
	// Value represents the value used for this cursor
	Value int64 `json:"v"`
}
