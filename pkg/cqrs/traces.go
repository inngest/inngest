package cqrs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

type SpanStatus int

const (
	SpanStatusUnknown = iota
	SpanStatusQueued
	SpanStatusOk
	SpanStatusError
)

// Raw otel span
type RawOtelSpan struct {
	Name         string         `json:"name"`
	SpanID       string         `json:"span_id"`
	TraceID      string         `json:"trace_id"`
	ParentSpanID *string        `json:"parent_span_id,omitempty"`
	StartTime    time.Time      `json:"start_time"`
	EndTime      time.Time      `json:"end_time"`
	Attributes   map[string]any `json:"attributes,omitempty"`
}

type OtelSpan struct {
	RawOtelSpan

	Status   enums.StepStatus `json:"status"`
	OutputID *string          `json:"output_id,omitempty,omitzero"`

	// Parsed attributes from the span
	Attributes *meta.ExtractedValues `json:"attributes,omitempty,omitzero"`

	// A span may be marked as dropped following idempotency or that we intend
	// to hide it (e.g. discovery steps).
	MarkedAsDropped bool `json:"marked_as_dropped,omitempty,omitzero"`

	RunID      ulid.ULID `json:"run_id,omitempty,omitzero"`
	AppID      uuid.UUID `json:"app_id,omitempty,omitzero"`
	FunctionID uuid.UUID `json:"function_id,omitempty,omitzero"`

	DebugRunID     ulid.ULID `json:"debug_run_id,omitempty,omitzero"`
	DebugSessionID ulid.ULID `json:"debug_session_id,omitempty,omitzero"`

	Children []*OtelSpan `json:"children,omitempty,omitzero"`
}

func (s *OtelSpan) GetAppID() uuid.UUID {
	return s.AppID
}

func (s *OtelSpan) GetFunctionID() uuid.UUID {
	return s.FunctionID
}

func (s *OtelSpan) GetRunID() ulid.ULID {
	return s.RunID
}

func (s *OtelSpan) GetDebugRunID() ulid.ULID {
	return s.DebugRunID
}

func (s *OtelSpan) GetDebugSessionID() ulid.ULID {
	return s.DebugSessionID
}

func (s *OtelSpan) GetSpanID() string {
	return s.SpanID
}

func (s *OtelSpan) GetTraceID() string {
	return s.TraceID
}

func (s *OtelSpan) GetStepName() string {
	if dn := s.Attributes.StepName; dn != nil {
		return *dn
	}

	return s.Name
}

func (s *OtelSpan) GetOutputID() *string {
	if s.OutputID == nil || *s.OutputID == "" {
		return nil
	}

	return s.OutputID
}

// TODO is this max?
func (s *OtelSpan) GetAttempts() int {
	if attempts := s.Attributes.StepAttempt; attempts != nil {
		return *attempts
	}

	return 0
}

func (s *OtelSpan) GetParentSpanID() *string {
	if s.ParentSpanID == nil || *s.ParentSpanID == "" {
		return nil
	}

	return s.ParentSpanID
}

func (s *OtelSpan) GetIsRoot() bool {
	parentSpanID := s.GetParentSpanID()

	return parentSpanID == nil || *parentSpanID == "" || *parentSpanID == "0000000000000000"
}

// Get the time that the span was "queued". This will always be present. If a
// value cannot be found internally (i.e. we haven't explicitly set the moment
// this span was queued), then the time will match the span's start time in
// order to show no queued time in the UI.
func (s *OtelSpan) GetQueuedAtTime() time.Time {
	if s.Attributes != nil && s.Attributes.QueuedAt != nil {
		return *s.Attributes.QueuedAt
	}

	// This should always be a value, so if we don't have one, just use when
	// the span was created.
	return s.StartTime
}

// Get the time that the span started. Note that this is not necessarily when
// the span created, as it may be dynamic.
func (s *OtelSpan) GetStartedAtTime() *time.Time {
	if s.Attributes == nil {
		return nil
	}
	return s.Attributes.StartedAt
}

// Get the time that the span ended. Note that this is not necessarily when the
// span was persisted, as it may be dynamic.
func (s *OtelSpan) GetEndedAtTime() *time.Time {
	if s.Attributes == nil {
		return nil
	}
	return s.Attributes.EndedAt
}

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

	// Children is a virtual field used for reconstructing the trace tree.
	// This field is not expected to be stored in the DB
	Children []*Span `json:"spans"`
}

// IsUserland checks if the span is a userland span, meaning it was created in
// client-side code, outside of the Executor.
func (s *Span) IsUserland() bool {
	_, isUserland := s.SpanAttributes[consts.OtelScopeUserland]

	return isUserland
}

// UserlandChildren returns any children of the span that are userland spans.
// This is used to reconstruct the trace tree for userland spans.
func (s *Span) UserlandChildren() []*Span {
	// If we're already a userland span, return our children
	if s.IsUserland() {
		return s.Children
	}

	// If we're not a userland span, but our first child is, return its
	// children.
	//
	// We do this because userland spans are always underneath an
	// `"inngest.execution"` span created by an SDK. So in this case, we're
	// checking that we have `Executor span -> inngest.exection span ->
	// userland`.
	//
	// Critically, this means we're also completely ignoring the
	// `"inngest.execution"` span itself, since we never want to display it to
	// the user.
	if len(s.Children) > 0 && s.Children[0].IsUserland() && len(s.Children[0].Children) > 0 {
		return s.Children[0].Children
	}

	return nil
}

func (s *Span) GroupID() *string {
	if groupID, ok := s.SpanAttributes[consts.OtelSysStepGroupID]; ok {
		return &groupID
	}
	return nil
}

func (s *Span) AccountID() *uuid.UUID {
	if str, ok := s.SpanAttributes[consts.OtelSysAccountID]; ok {
		if id, err := uuid.Parse(str); err == nil {
			return &id
		}
	}
	return nil
}

func (s *Span) WorkspaceID() *uuid.UUID {
	if str, ok := s.SpanAttributes[consts.OtelSysWorkspaceID]; ok {
		if id, err := uuid.Parse(str); err == nil {
			return &id
		}
	}
	return nil
}

func (s *Span) AppID() *uuid.UUID {
	if str, ok := s.SpanAttributes[consts.OtelSysAppID]; ok {
		if id, err := uuid.Parse(str); err == nil {
			return &id
		}
	}
	return nil
}

func (s *Span) FunctionID() *uuid.UUID {
	if str, ok := s.SpanAttributes[consts.OtelSysFunctionID]; ok {
		if id, err := uuid.Parse(str); err == nil {
			return &id
		}
	}
	return nil
}

func (s *Span) StepDisplayName() *string {
	if name, ok := s.SpanAttributes[consts.OtelSysStepDisplayName]; ok {
		return &name
	}
	return nil
}

func (s *Span) Status() SpanStatus {
	switch strings.ToUpper(s.StatusCode) {
	case "OK", "STATUS_CODE_OK":
		return SpanStatusOk
	case "ERROR", "STATUS_CODE_ERROR":
		return SpanStatusError
	case "QUEUED": // virtual status
		return SpanStatusQueued
	}

	return SpanStatusUnknown
}

func (s *Span) FunctionStatus() enums.RunStatus {
	if str, ok := s.SpanAttributes[consts.OtelSysFunctionStatusCode]; ok {
		if code, err := strconv.ParseInt(str, 10, 64); err == nil {
			return enums.RunCodeToStatus(code)
		}
	}
	return enums.RunStatusUnknown
}

func (s *Span) StepOpCode() enums.Opcode {
	if op, ok := s.SpanAttributes[consts.OtelSysStepOpcode]; ok {
		switch op {
		case enums.OpcodeStep.String(), enums.OpcodeStepRun.String(), enums.OpcodeStepError.String():
			return enums.OpcodeStepRun
		case enums.OpcodeSleep.String():
			return enums.OpcodeSleep
		case enums.OpcodeInvokeFunction.String():
			return enums.OpcodeInvokeFunction
		case enums.OpcodeWaitForEvent.String():
			return enums.OpcodeWaitForEvent
		case enums.OpcodeStepPlanned.String():
			return enums.OpcodeStepPlanned
		case enums.OpcodeAIGateway.String():
			return enums.OpcodeAIGateway
		case enums.OpcodeWaitForSignal.String():
			return enums.OpcodeWaitForSignal
		}
	}

	return enums.OpcodeNone
}

func (s *Span) DurationMS() int64 {
	return int64(s.Duration / time.Millisecond)
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
	AccountID    uuid.UUID       `json:"account_id"`
	WorkspaceID  uuid.UUID       `json:"workspace_id"`
	AppID        uuid.UUID       `json:"app_id"`
	FunctionID   uuid.UUID       `json:"function_id"`
	TraceID      string          `json:"trace_id"`
	RunID        string          `json:"run_id"`
	QueuedAt     time.Time       `json:"queued_at"`
	StartedAt    time.Time       `json:"started_at,omitempty"`
	EndedAt      time.Time       `json:"ended_at,omitempty"`
	Duration     time.Duration   `json:"duration"`
	SourceID     string          `json:"source_id,omitempty"`
	TriggerIDs   []string        `json:"trigger_ids"`
	Triggers     [][]byte        `json:"triggers"`
	Output       []byte          `json:"output,omitempty"`
	Status       enums.RunStatus `json:"status"`
	IsBatch      bool            `json:"is_batch"`
	IsDebounce   bool            `json:"is_debounce"`
	HasAI        bool            `json:"has_ai"`
	BatchID      *ulid.ULID      `json:"batch_id,omitempty"`
	CronSchedule *string         `json:"cron_schedule,omitempty"`
	// Cursor is a composite cursor used for pagination
	Cursor string `json:"cursor"`
}

type SpanOutput struct {
	Input            []byte
	Data             []byte
	Timestamp        time.Time
	Attributes       map[string]string
	IsError          bool
	IsFunctionOutput bool
	IsStepOutput     bool
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

type TraceReadWriterDev interface {
	// FindOrCreateTraceRun will return a TraceRun by runID, or create a new one if it doesn't exists
	FindOrBuildTraceRun(ctx context.Context, opts FindOrCreateTraceRunOpt) (*TraceRun, error)
	// Returns a list of TraceRun triggered by triggerID
	GetTraceRunsByTriggerID(ctx context.Context, triggerID ulid.ULID) ([]*TraceRun, error)
}

type TraceReader interface {
	// GetTraceRuns retrieves a list of TraceRun based on the options specified
	GetTraceRuns(ctx context.Context, opt GetTraceRunOpt) ([]*TraceRun, error)
	// GetTraceRunsCount returns the total number of items applicable to the specified filter
	GetTraceRunsCount(ctx context.Context, opt GetTraceRunOpt) (int, error)
	// GetTraceRun retrieve the specified run
	GetTraceRun(ctx context.Context, id TraceRunIdentifier) (*TraceRun, error)
	// GetTraceSpansByRun retrieves all the spans related to the trace
	GetTraceSpansByRun(ctx context.Context, id TraceRunIdentifier) ([]*Span, error)
	// LegacyGetSpanOutput retrieves the output for the specified span
	LegacyGetSpanOutput(ctx context.Context, id SpanIdentifier) (*SpanOutput, error)
	// GetSpanStack retrieves the step stack for the specified span
	GetSpanStack(ctx context.Context, id SpanIdentifier) ([]string, error)
	// GetSpansByRunID retrieves all spans related to the specified run
	GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*OtelSpan, error)
	// GetSpansByDebugRunID retrieves all spans related to the specified debug run
	GetSpansByDebugRunID(ctx context.Context, debugRunID ulid.ULID) ([]*OtelSpan, error)
	// GetSpansByDebugSessionID retrieves all spans related to the specified debug session
	GetSpansByDebugSessionID(ctx context.Context, debugSessionID ulid.ULID) ([]*OtelSpan, error)
	GetSpanOutput(ctx context.Context, id SpanIdentifier) (*SpanOutput, error)
	// TODO move to dedicated entitlement interface once that is implemented properly
	// for both oss & cloud
	OtelTracesEnabled(ctx context.Context, accountID uuid.UUID) (bool, error)
	// GetEventRuns returns the runs that were triggered by an event.
	GetEventRuns(ctx context.Context, eventID ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) ([]*FunctionRun, error)
	// GetRun returns a single function run.
	GetRun(ctx context.Context, runID ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) (*FunctionRun, error)
	// GetEvent returns a single event.
	GetEvent(ctx context.Context, id ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) (*Event, error)
	// GetEvents returns a list of latest events.
	GetEvents(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, opts *WorkspaceEventsOpts) ([]*Event, error)
}

type GetTraceRunOpt struct {
	Filter GetTraceRunFilter
	Order  []GetTraceRunOrder
	Cursor string
	Items  uint
}

type FindOrCreateTraceRunOpt struct {
	// Only runID is used for search, others fields are considered default values
	// if the trace run doesn't exists
	RunID       ulid.ULID
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       uuid.UUID
	FunctionID  uuid.UUID
	TraceID     string
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

type TraceRunIdentifier struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       uuid.UUID
	FunctionID  uuid.UUID
	TraceID     string
	RunID       ulid.ULID
}

type SpanIdentifier struct {
	AccountID   uuid.UUID `json:"acctID"`
	WorkspaceID uuid.UUID `json:"wsID"`
	AppID       uuid.UUID `json:"appID"`
	FunctionID  uuid.UUID `json:"fnID"`
	TraceID     string    `json:"tid"`
	SpanID      string    `json:"sid"`

	// Whether the output should direct to the tracing preview stores
	Preview *bool `json:"preview,omitempty,omitzero"`
}

func (si *SpanIdentifier) Encode() (string, error) {
	byt, err := json.Marshal(si)
	if err != nil {
		return "", fmt.Errorf("error encoding span identifier: %w", err)
	}
	return base64.StdEncoding.EncodeToString(byt), nil
}

func (si *SpanIdentifier) Decode(data string) error {
	byt, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(byt, si)
}

// TracePageCursor represents the composite cursor used to handle pagination
type TracePageCursor struct {
	ID      string                 `json:"id"`
	Cursors map[string]TraceCursor `json:"c"`
}

func (c *TracePageCursor) IsEmpty() bool {
	return len(c.Cursors) == 0
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
