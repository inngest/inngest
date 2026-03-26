package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// Querier defines the complete set of database operations using domain types.
//
// This is the target interface that all database adapters must implement.
// It mirrors the existing sqlc-generated Querier but returns domain types
// from this package instead of dialect-specific sqlc types.
type Querier interface {
	// Apps
	DeleteApp(ctx context.Context, id uuid.UUID) error
	GetAllApps(ctx context.Context) ([]*App, error)
	GetApp(ctx context.Context, id uuid.UUID) (*App, error)
	GetAppByChecksum(ctx context.Context, checksum string) (*App, error)
	GetAppByID(ctx context.Context, id uuid.UUID) (*App, error)
	GetAppByName(ctx context.Context, name string) (*App, error)
	GetAppByURL(ctx context.Context, url string) (*App, error)
	GetApps(ctx context.Context) ([]*App, error)
	UpsertApp(ctx context.Context, arg UpsertAppParams) (*App, error)
	UpdateAppError(ctx context.Context, arg UpdateAppErrorParams) (*App, error)
	UpdateAppURL(ctx context.Context, arg UpdateAppURLParams) (*App, error)

	// Functions
	DeleteFunctionsByAppID(ctx context.Context, appID uuid.UUID) error
	DeleteFunctionsByIDs(ctx context.Context, ids []uuid.UUID) error
	GetAppFunctions(ctx context.Context, appID uuid.UUID) ([]*Function, error)
	GetAppFunctionsBySlug(ctx context.Context, name string) ([]*Function, error)
	GetFunctionByID(ctx context.Context, id uuid.UUID) (*Function, error)
	GetFunctionBySlug(ctx context.Context, slug string) (*Function, error)
	GetFunctions(ctx context.Context) ([]*Function, error)
	InsertFunction(ctx context.Context, arg InsertFunctionParams) (*Function, error)
	UpdateFunctionConfig(ctx context.Context, arg UpdateFunctionConfigParams) (*Function, error)

	// Events
	InsertEvent(ctx context.Context, arg InsertEventParams) error
	GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*Event, error)
	GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*Event, error)
	GetEventsIDbound(ctx context.Context, arg GetEventsIDboundParams) ([]*Event, error)

	// Event Batches
	InsertEventBatch(ctx context.Context, arg InsertEventBatchParams) error
	GetEventBatchByRunID(ctx context.Context, runID ulid.ULID) (*EventBatch, error)
	GetEventBatchesByEventID(ctx context.Context, instr string) ([]*EventBatch, error)

	// Function Runs
	InsertFunctionRun(ctx context.Context, arg InsertFunctionRunParams) error
	InsertFunctionFinish(ctx context.Context, arg InsertFunctionFinishParams) error
	GetFunctionRun(ctx context.Context, runID ulid.ULID) (*FunctionRunRow, error)
	GetFunctionRuns(ctx context.Context) ([]*FunctionRunRow, error)
	GetFunctionRunsFromEvents(ctx context.Context, eventIds []ulid.ULID) ([]*FunctionRunRow, error)
	GetFunctionRunsTimebound(ctx context.Context, arg GetFunctionRunsTimeboundParams) ([]*FunctionRunRow, error)
	GetFunctionRunFinishesByRunIDs(ctx context.Context, runIds []ulid.ULID) ([]*FunctionFinish, error)

	// History
	InsertHistory(ctx context.Context, arg InsertHistoryParams) error
	GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*History, error)
	GetHistoryItem(ctx context.Context, id ulid.ULID) (*History, error)
	HistoryCountRuns(ctx context.Context) (int64, error)

	// Queue Snapshots
	InsertQueueSnapshotChunk(ctx context.Context, arg InsertQueueSnapshotChunkParams) error
	DeleteOldQueueSnapshots(ctx context.Context, limit int64) (int64, error)
	GetQueueSnapshotChunks(ctx context.Context, snapshotID string) ([]*QueueSnapshotChunkRow, error)
	GetLatestQueueSnapshotChunks(ctx context.Context) ([]*QueueSnapshotChunkRow, error)

	// Spans (new tracing)
	InsertSpan(ctx context.Context, arg InsertSpanParams) error
	GetSpansByRunID(ctx context.Context, runID string) ([]*SpanRow, error)
	GetSpansByDebugRunID(ctx context.Context, debugRunID sql.NullString) ([]*SpanRow, error)
	GetSpansByDebugSessionID(ctx context.Context, debugSessionID sql.NullString) ([]*SpanRow, error)
	GetRunSpanByRunID(ctx context.Context, arg GetRunSpanByRunIDParams) (*SpanRow, error)
	GetSpanBySpanID(ctx context.Context, arg GetSpanBySpanIDParams) (*SpanRow, error)
	GetStepSpanByStepID(ctx context.Context, arg GetStepSpanByStepIDParams) (*SpanRow, error)
	GetSpanOutput(ctx context.Context, ids []string) ([]*SpanOutputRow, error)
	GetExecutionSpanByStepIDAndAttempt(ctx context.Context, arg GetExecutionSpanByStepIDAndAttemptParams) (*SpanRow, error)
	GetLatestExecutionSpanByStepID(ctx context.Context, arg GetLatestExecutionSpanByStepIDParams) (*SpanRow, error)

	// Traces (OpenTelemetry)
	InsertTrace(ctx context.Context, arg InsertTraceParams) error
	InsertTraceRun(ctx context.Context, arg InsertTraceRunParams) error
	GetTraceRun(ctx context.Context, runID ulid.ULID) (*TraceRun, error)
	GetTraceRunsByTriggerId(ctx context.Context, eventID string) ([]*TraceRun, error)
	GetTraceSpans(ctx context.Context, arg GetTraceSpansParams) ([]*Trace, error)
	GetTraceSpanOutput(ctx context.Context, arg GetTraceSpanOutputParams) ([]*Trace, error)

	// Worker Connections
	InsertWorkerConnection(ctx context.Context, arg InsertWorkerConnectionParams) error
	GetWorkerConnection(ctx context.Context, arg GetWorkerConnectionParams) (*WorkerConnection, error)
}
