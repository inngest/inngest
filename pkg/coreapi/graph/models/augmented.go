package models

import (
	"time"

	"github.com/google/uuid"
	ulid "github.com/oklog/ulid/v2"
)

type WorkerConnectionsConnection struct {
	Edges    []*ConnectV1WorkerConnectionEdge `json:"edges"`
	PageInfo *PageInfo                        `json:"pageInfo"`

	After   *string
	Filter  ConnectV1WorkerConnectionsFilter
	OrderBy []*ConnectV1WorkerConnectionsOrderBy
}

type RunsV2Connection struct {
	Edges    []*FunctionRunV2Edge `json:"edges"`
	PageInfo *PageInfo            `json:"pageInfo"`

	After   *string
	Filter  RunsFilterV2
	OrderBy []*RunsV2OrderBy
}

type RunTraceSpan struct {
	AppID             uuid.UUID                 `json:"appID"`
	FunctionID        uuid.UUID                 `json:"functionID"`
	RunID             ulid.ULID                 `json:"runID"`
	Run               *FunctionRun              `json:"run"`
	SpanID            string                    `json:"spanID"`
	TraceID           string                    `json:"traceID"`
	Name              string                    `json:"name"`
	Status            RunTraceSpanStatus        `json:"status"`
	Attempts          *int                      `json:"attempts,omitempty"`
	Duration          *int                      `json:"duration,omitempty"`
	OutputID          *string                   `json:"outputID,omitempty"`
	QueuedAt          time.Time                 `json:"queuedAt"`
	StartedAt         *time.Time                `json:"startedAt,omitempty"`
	EndedAt           *time.Time                `json:"endedAt,omitempty"`
	ChildrenSpans     []*RunTraceSpan           `json:"childrenSpans"`
	StepOp            *StepOp                   `json:"stepOp,omitempty"`
	StepID            *string                   `json:"stepID,omitempty"`
	StepInfo          StepInfo                  `json:"stepInfo,omitempty"`
	StepType          string                    `json:"stepType"`
	IsRoot            bool                      `json:"isRoot"`
	ParentSpanID      *string                   `json:"parentSpanID,omitempty"`
	ParentSpan        *RunTraceSpan             `json:"parentSpan,omitempty"`
	IsUserland        bool                      `json:"isUserland"`
	UserlandSpan      *UserlandSpan             `json:"userlandSpan,omitempty"`
	DebugRunID        *ulid.ULID                `json:"debugRunID,omitempty"`
	DebugSessionID    *ulid.ULID                `json:"debugSessionID,omitempty"`
	DebugPaused       bool                      `json:"debugPaused"`
	SkipReason        *string                   `json:"skipReason,omitempty"`
	SkipExistingRunID *string                   `json:"skipExistingRunID,omitempty"`
	Metadata          []*SpanMetadata           `json:"metadata,omitempty"`
	Response          *RunTraceSpanResponseInfo `json:"response,omitempty"`

	// Internal fields not exposed over GraphQL.
	SpanTypeName string
	Omit         bool
}

func RunTraceEnded(s RunTraceSpanStatus) bool {
	return s == RunTraceSpanStatusCompleted || s == RunTraceSpanStatusCancelled || s == RunTraceSpanStatusFailed || s == RunTraceSpanStatusSkipped
}

// RunDeferredFrom carries a parent run's ID and function slug. Function and
// Run are resolved lazily so list views that only need the parent's function
// metadata don't pay for a full TraceRun fetch per linkage entry.
type RunDeferredFrom struct {
	RunID  ulid.ULID `json:"runID"`
	FnSlug string    `json:"-"`
}

// RunDefer is a single defer attached to a parent run. Function and Run are
// resolved lazily — FnSlug (always set) drives the function lookup, RunID
// drives the run lookup when the child has been scheduled.
type RunDefer struct {
	HashedDeferID   string         `json:"hashedDeferID"`
	UserlandDeferID string         `json:"userlandDeferID"`
	FnSlug          string         `json:"fnSlug"`
	Status          RunDeferStatus `json:"status"`
	// RunID is nil when the defer hasn't been linked to a scheduled child
	// yet (parent still running, aborted, or rejected).
	RunID *ulid.ULID `json:"runID,omitempty"`
}
