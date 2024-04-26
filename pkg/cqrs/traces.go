package cqrs

import (
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type Trace struct {
	Timestamp          time.Time         `json:"ts"`
	TraceID            string            `json:"trace_id"`
	SpanID             string            `json:"span_id"`
	ParentSpanID       *string           `json:"parent_span_id,omitempty"`
	TraceState         *string           `json:"trace_state,omitempty"`
	SpanName           string            `json:"span_name"`
	SpanKind           string            `json:"span_kind"`
	ServiceName        string            `json:"service_name"`
	ResourceAttributes map[string]string `json:"resource_attributes"`
	ScopeName          string            `json:"scope_name"`
	ScopeVersion       string            `json:"scope_version"`
	SpanAttributes     map[string]string `json:"span_attributes"`
	Duration           time.Duration     `json:"duration"`
	StatusCode         string            `json:"status_code"`
	StatusMessage      *string           `json:"status_message"`
	RunID              *ulid.ULID        `json:"run_id"`
}

type TraceRun struct {
	AccountID   uuid.UUID     `json:"account_id"`
	WorkspaceID uuid.UUID     `json:"workspace_id"`
	AppID       uuid.UUID     `json:"app_id"`
	FunctionID  uuid.UUID     `json:"function_id"`
	TraceID     string        `json:"trace_id"`
	RunID       ulid.ULID     `json:"run_id"`
	QueuedAt    time.Time     `json:"queued_at"`
	StartedAt   time.Time     `json:"started_at,omitempty"`
	EndedAt     time.Time     `json:"ended_at,omitempty"`
	Duration    time.Duration `json:"duration"`
	Status      string        `json:"status"`
	IsBatch     bool          `json:"is_batch"`
	IsDebounce  bool          `json:"is_debounce"`
}
