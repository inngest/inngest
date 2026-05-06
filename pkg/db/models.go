// Package db defines database-agnostic domain model types and adapter interfaces
// for the CQRS layer.
//
// These types are the canonical representation of all database entities. Each database
// adapter (SQLite, PostgreSQL, MySQL, etc.) converts its dialect-specific sqlc-generated
// types into these domain types. Consumers of the CQRS layer should only depend on these
// types, never on dialect-specific models.
//
// Design decisions:
//   - Nullable fields use database/sql null types (sql.NullString, sql.NullTime, etc.)
//   - Integer fields use int64 consistently (widest common type across dialects)
//   - JSON/binary fields use []byte
//   - IDs use uuid.UUID or ulid.ULID as appropriate
package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// App represents a registered SDK application.
type App struct {
	ID          uuid.UUID
	Name        string
	SdkLanguage string
	SdkVersion  string
	Framework   sql.NullString
	Metadata    string
	Status      string
	Error       sql.NullString
	Checksum    string
	CreatedAt   time.Time
	ArchivedAt  sql.NullTime
	Url         string
	Method      string
	AppVersion  sql.NullString
}

// Event represents an ingested event.
type Event struct {
	InternalID  ulid.ULID
	AccountID   sql.NullString
	WorkspaceID sql.NullString
	Source      sql.NullString
	SourceID    sql.NullString
	ReceivedAt  time.Time
	EventID     string
	EventName   string
	EventData   string
	EventUser   string
	EventV      sql.NullString
	EventTs     time.Time
}

// EventBatch represents a batch of events processed together.
type EventBatch struct {
	ID          ulid.ULID
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       uuid.UUID
	WorkflowID  uuid.UUID
	RunID       ulid.ULID
	StartedAt   time.Time
	ExecutedAt  time.Time
	EventIds    []byte
}

// Function represents a workflow function definition.
type Function struct {
	ID         uuid.UUID
	AppID      uuid.UUID
	Name       string
	Slug       string
	Config     string
	CreatedAt  time.Time
	ArchivedAt sql.NullTime
}

// FunctionFinish records the completion of a function run.
type FunctionFinish struct {
	RunID              ulid.ULID
	Status             sql.NullString
	Output             sql.NullString
	CompletedStepCount sql.NullInt64
	CreatedAt          sql.NullTime
}

// FunctionRun represents an individual function execution.
type FunctionRun struct {
	RunID           ulid.ULID
	RunStartedAt    time.Time
	FunctionID      uuid.UUID
	FunctionVersion int64
	TriggerType     string
	EventID         ulid.ULID
	BatchID         ulid.ULID
	OriginalRunID   ulid.ULID
	Cron            sql.NullString
	WorkspaceID     uuid.UUID
}

// History records execution state machine transitions.
type History struct {
	ID                   ulid.ULID
	CreatedAt            time.Time
	RunStartedAt         time.Time
	FunctionID           uuid.UUID
	FunctionVersion      int64
	RunID                ulid.ULID
	EventID              ulid.ULID
	BatchID              ulid.ULID
	GroupID              sql.NullString
	IdempotencyKey       string
	Type                 string
	Attempt              int64
	LatencyMs            sql.NullInt64
	StepName             sql.NullString
	StepID               sql.NullString
	StepType             sql.NullString
	Url                  sql.NullString
	CancelRequest        sql.NullString
	Sleep                sql.NullString
	WaitForEvent         sql.NullString
	WaitResult           sql.NullString
	InvokeFunction       sql.NullString
	InvokeFunctionResult sql.NullString
	Result               sql.NullString
}

// QueueSnapshotChunk stores a chunk of serialized queue state.
type QueueSnapshotChunk struct {
	SnapshotID string
	ChunkID    int64
	Data       []byte
}

// Span represents an execution span in the new spans table.
type Span struct {
	SpanID         string
	TraceID        string
	ParentSpanID   sql.NullString
	Name           string
	StartTime      time.Time
	EndTime        time.Time
	Attributes     []byte
	Links          []byte
	DynamicSpanID  sql.NullString
	AccountID      string
	AppID          string
	FunctionID     string
	RunID          string
	EnvID          string
	Output         []byte
	Input          []byte
	DebugRunID     sql.NullString
	DebugSessionID sql.NullString
	Status         sql.NullString
	EventIds       []byte
}

// Trace represents an OpenTelemetry trace span.
type Trace struct {
	Timestamp          time.Time
	TimestampUnixMs    int64
	TraceID            string
	SpanID             string
	ParentSpanID       sql.NullString
	TraceState         sql.NullString
	SpanName           string
	SpanKind           string
	ServiceName        string
	ResourceAttributes []byte
	ScopeName          string
	ScopeVersion       string
	SpanAttributes     []byte
	Duration           int64
	StatusCode         string
	StatusMessage      sql.NullString
	Events             []byte
	Links              []byte
	RunID              ulid.ULID
}

// TraceRun holds trace metadata for a function run.
type TraceRun struct {
	RunID        ulid.ULID
	AccountID    uuid.UUID
	WorkspaceID  uuid.UUID
	AppID        uuid.UUID
	FunctionID   uuid.UUID
	TraceID      []byte
	QueuedAt     int64
	StartedAt    int64
	EndedAt      int64
	Status       int64
	SourceID     string
	TriggerIds   []byte
	Output       []byte
	IsDebounce   bool
	BatchID      ulid.ULID
	CronSchedule sql.NullString
	HasAi        bool
}

// WorkerConnection represents a connected worker instance.
type WorkerConnection struct {
	AccountID            uuid.UUID
	WorkspaceID          uuid.UUID
	AppName              string
	AppID                *uuid.UUID
	ID                   ulid.ULID
	GatewayID            ulid.ULID
	InstanceID           string
	Status               int64
	WorkerIp             string
	MaxWorkerConcurrency int64
	ConnectedAt          int64
	LastHeartbeatAt      sql.NullInt64
	DisconnectedAt       sql.NullInt64
	RecordedAt           int64
	InsertedAt           int64
	DisconnectReason     sql.NullString
	GroupHash            []byte
	SdkLang              string
	SdkVersion           string
	SdkPlatform          string
	SyncID               *uuid.UUID
	AppVersion           sql.NullString
	FunctionCount        int64
	CpuCores             int64
	MemBytes             int64
	Os                   string
}

// FunctionRunRow is the joined result of a function run with its optional finish record.
type FunctionRunRow struct {
	FunctionRun    FunctionRun
	FunctionFinish FunctionFinish
}

// RunWithUserEventID is a FunctionRunRow paired with the user-facing event_id
// (text) that triggered it. Returned by lookups that resolve user-facing event
// IDs to their child runs.
type RunWithUserEventID struct {
	UserEventID    string
	FunctionRun    FunctionRun
	FunctionFinish FunctionFinish
}

// RunDeferOpcode is the projection of history rows GetRunDeferOpcodes returns.
// Only id and result are read; the result blob holds the marshaled opcode the
// OnDefer lifecycle listener wrote.
type RunDeferOpcode struct {
	ID     ulid.ULID
	Result sql.NullString
}

// SpanRow is the common shape returned by span queries that group by dynamic_span_id.
type SpanRow struct {
	RunID          string
	TraceID        string
	DynamicSpanID  sql.NullString
	StartTime      time.Time
	EndTime        time.Time
	ParentSpanID   sql.NullString
	SpanFragments  []byte
	DebugRunID     sql.NullString
	DebugSessionID sql.NullString
}

// QueueSnapshotChunkRow is the result of reading snapshot chunk data.
type QueueSnapshotChunkRow struct {
	ChunkID int64
	Data    []byte
}

// SpanOutputRow is the result of reading span input/output data.
type SpanOutputRow struct {
	Input  []byte
	Output []byte
}
