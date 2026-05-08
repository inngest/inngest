package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// UpsertAppParams are the parameters for creating or updating an app.
type UpsertAppParams struct {
	ID          uuid.UUID
	Name        string
	SdkLanguage string
	SdkVersion  string
	Framework   sql.NullString
	Metadata    string
	Status      string
	Error       sql.NullString
	Checksum    string
	Url         string
	Method      string
	AppVersion  sql.NullString
}

// UpdateAppErrorParams are the parameters for updating an app's error field.
type UpdateAppErrorParams struct {
	Error sql.NullString
	ID    uuid.UUID
}

// UpdateAppURLParams are the parameters for updating an app's URL.
type UpdateAppURLParams struct {
	Url string
	ID  uuid.UUID
}

// UpsertFunctionParams are the parameters for inserting or refreshing a
// function definition. On id conflict, the row's app_id, name, slug, config
// are overwritten and archived_at is cleared.
type UpsertFunctionParams struct {
	ID        uuid.UUID
	AppID     uuid.UUID
	Name      string
	Slug      string
	Config    string
	CreatedAt time.Time
}

// UpdateFunctionConfigParams are the parameters for updating a function's config.
type UpdateFunctionConfigParams struct {
	Config string
	ID     uuid.UUID
}

// InsertEventParams are the parameters for inserting an event.
type InsertEventParams struct {
	InternalID ulid.ULID
	ReceivedAt time.Time
	EventID    string
	EventName  string
	EventData  string
	EventUser  string
	EventV     sql.NullString
	EventTs    time.Time
}

// GetEventsIDboundParams are the parameters for querying events by ID bounds.
type GetEventsIDboundParams struct {
	After           ulid.ULID
	Before          ulid.ULID
	IncludeInternal string
	Limit           int64
}

// InsertEventBatchParams are the parameters for inserting an event batch.
type InsertEventBatchParams struct {
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

// InsertFunctionRunParams are the parameters for inserting a function run.
type InsertFunctionRunParams struct {
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

// InsertFunctionFinishParams are the parameters for inserting a function completion record.
type InsertFunctionFinishParams struct {
	RunID              ulid.ULID
	Status             sql.NullString
	Output             sql.NullString
	CompletedStepCount sql.NullInt64
	CreatedAt          sql.NullTime
}

// GetFunctionRunsTimeboundParams are the parameters for querying function runs by time range.
type GetFunctionRunsTimeboundParams struct {
	After  time.Time
	Before time.Time
	Limit  int64
}

// InsertHistoryParams are the parameters for inserting a history record.
type InsertHistoryParams struct {
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

// InsertQueueSnapshotChunkParams are the parameters for inserting a queue snapshot chunk.
type InsertQueueSnapshotChunkParams struct {
	SnapshotID string
	ChunkID    int64
	Data       []byte
}

// InsertSpanParams are the parameters for inserting a span.
type InsertSpanParams struct {
	SpanID         string
	TraceID        string
	ParentSpanID   sql.NullString
	Name           string
	StartTime      time.Time
	EndTime        time.Time
	RunID          string
	AccountID      string
	AppID          string
	FunctionID     string
	EnvID          string
	DynamicSpanID  sql.NullString
	Attributes     []byte
	Links          []byte
	Output         []byte
	Input          []byte
	DebugRunID     sql.NullString
	DebugSessionID sql.NullString
	Status         sql.NullString
	EventIds       []byte
}

// InsertTraceParams are the parameters for inserting a trace.
type InsertTraceParams struct {
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

// InsertTraceRunParams are the parameters for inserting/upserting a trace run.
type InsertTraceRunParams struct {
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
	BatchID      ulid.ULID
	IsDebounce   bool
	CronSchedule sql.NullString
	HasAi        bool
}

// InsertWorkerConnectionParams are the parameters for inserting/upserting a worker connection.
type InsertWorkerConnectionParams struct {
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

// GetWorkerConnectionParams are the parameters for querying a worker connection.
type GetWorkerConnectionParams struct {
	AccountID    uuid.UUID
	WorkspaceID  uuid.UUID
	ConnectionID ulid.ULID
}

// GetTraceSpansParams are the parameters for querying trace spans.
type GetTraceSpansParams struct {
	TraceID string
	RunID   ulid.ULID
}

// GetTraceSpanOutputParams are the parameters for querying trace span output.
type GetTraceSpanOutputParams struct {
	TraceID string
	SpanID  string
}

// GetRunSpanByRunIDParams are the parameters for querying the run-level span.
type GetRunSpanByRunIDParams struct {
	RunID     string
	AccountID string
}

// GetSpanBySpanIDParams are the parameters for querying a span by its ID.
type GetSpanBySpanIDParams struct {
	RunID     string
	SpanID    string
	AccountID string
}

// GetStepSpanByStepIDParams are the parameters for querying a step span.
type GetStepSpanByStepIDParams struct {
	RunID     string
	AccountID string
	StepID    string
}

// GetExecutionSpanByStepIDAndAttemptParams are the parameters for querying an execution span.
type GetExecutionSpanByStepIDAndAttemptParams struct {
	RunID       string
	AccountID   string
	StepID      string
	StepAttempt int64
}

// GetLatestExecutionSpanByStepIDParams are the parameters for querying the latest execution span.
type GetLatestExecutionSpanByStepIDParams struct {
	RunID     string
	AccountID string
	StepID    string
}
