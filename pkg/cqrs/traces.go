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
	"github.com/oklog/ulid/v2"
)

type SpanStatus int

const (
	SpanStatusUnknown = iota
	SpanStatusQueued
	SpanStatusOk
	SpanStatusError
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

	// Children is a virtual field used for reconstructing the trace tree.
	// This field is not expected to be stored in the DB
	Children []*Span `json:"spans"`
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

type TraceWriterDev interface {
	// FindOrCreateTraceRun will return a TraceRun by runID, or create a new one if it doesn't exists
	FindOrBuildTraceRun(ctx context.Context, opts FindOrCreateTraceRunOpt) (*TraceRun, error)
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
	// GetSpanOutput retrieves the output for the specified span
	GetSpanOutput(ctx context.Context, id SpanIdentifier) (*SpanOutput, error)
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
