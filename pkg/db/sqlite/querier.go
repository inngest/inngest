package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/db"
	"github.com/oklog/ulid/v2"
)

var _ db.Querier = (*sqliteQuerier)(nil)

type sqliteQuerier struct {
	q sqlc.Querier
}

// bytesToNullString preserves nil-vs-empty semantics while adapting db-layer
// []byte JSON payloads to the generated SQLite insert params.
func bytesToNullString(b []byte) sql.NullString {
	if len(b) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

// --- Apps ---

func (sq *sqliteQuerier) DeleteApp(ctx context.Context, id uuid.UUID) error {
	return sq.q.DeleteApp(ctx, id)
}

func (sq *sqliteQuerier) GetAllApps(ctx context.Context) ([]*db.App, error) {
	rows, err := sq.q.GetAllApps(ctx)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, appFromSQLite), nil
}

func (sq *sqliteQuerier) GetApp(ctx context.Context, id uuid.UUID) (*db.App, error) {
	r, err := sq.q.GetApp(ctx, id)
	if err != nil {
		return nil, err
	}
	return appFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetAppByChecksum(ctx context.Context, checksum string) (*db.App, error) {
	r, err := sq.q.GetAppByChecksum(ctx, checksum)
	if err != nil {
		return nil, err
	}
	return appFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetAppByID(ctx context.Context, id uuid.UUID) (*db.App, error) {
	r, err := sq.q.GetAppByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return appFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetAppByName(ctx context.Context, name string) (*db.App, error) {
	r, err := sq.q.GetAppByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return appFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetAppByURL(ctx context.Context, url string) (*db.App, error) {
	r, err := sq.q.GetAppByURL(ctx, url)
	if err != nil {
		return nil, err
	}
	return appFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetApps(ctx context.Context) ([]*db.App, error) {
	rows, err := sq.q.GetApps(ctx)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, appFromSQLite), nil
}

func (sq *sqliteQuerier) UpsertApp(ctx context.Context, arg db.UpsertAppParams) (*db.App, error) {
	r, err := sq.q.UpsertApp(ctx, sqlc.UpsertAppParams{
		ID: arg.ID, Name: arg.Name, SdkLanguage: arg.SdkLanguage,
		SdkVersion: arg.SdkVersion, Framework: arg.Framework, Metadata: arg.Metadata,
		Status: arg.Status, Error: arg.Error, Checksum: arg.Checksum,
		Url: arg.Url, Method: arg.Method, AppVersion: arg.AppVersion,
	})
	if err != nil {
		return nil, err
	}
	return appFromSQLite(r), nil
}

func (sq *sqliteQuerier) UpdateAppError(ctx context.Context, arg db.UpdateAppErrorParams) (*db.App, error) {
	r, err := sq.q.UpdateAppError(ctx, sqlc.UpdateAppErrorParams{Error: arg.Error, ID: arg.ID})
	if err != nil {
		return nil, err
	}
	return appFromSQLite(r), nil
}

func (sq *sqliteQuerier) UpdateAppURL(ctx context.Context, arg db.UpdateAppURLParams) (*db.App, error) {
	r, err := sq.q.UpdateAppURL(ctx, sqlc.UpdateAppURLParams{Url: arg.Url, ID: arg.ID})
	if err != nil {
		return nil, err
	}
	return appFromSQLite(r), nil
}

// --- Functions ---

func (sq *sqliteQuerier) DeleteFunctionsByAppID(ctx context.Context, appID uuid.UUID) error {
	return sq.q.DeleteFunctionsByAppID(ctx, appID)
}

func (sq *sqliteQuerier) DeleteFunctionsByIDs(ctx context.Context, ids []uuid.UUID) error {
	return sq.q.DeleteFunctionsByIDs(ctx, ids)
}

func (sq *sqliteQuerier) GetAppFunctions(ctx context.Context, appID uuid.UUID) ([]*db.Function, error) {
	rows, err := sq.q.GetAppFunctions(ctx, appID)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, functionFromSQLite), nil
}

func (sq *sqliteQuerier) GetAppFunctionsBySlug(ctx context.Context, name string) ([]*db.Function, error) {
	rows, err := sq.q.GetAppFunctionsBySlug(ctx, name)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, functionFromSQLite), nil
}

func (sq *sqliteQuerier) GetFunctionByID(ctx context.Context, id uuid.UUID) (*db.Function, error) {
	r, err := sq.q.GetFunctionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return functionFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetFunctionBySlug(ctx context.Context, slug string) (*db.Function, error) {
	r, err := sq.q.GetFunctionBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return functionFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetFunctionByAppNameAndSlug(ctx context.Context, appName string, slug string) (*db.Function, error) {
	r, err := sq.q.GetFunctionByAppNameAndSlug(ctx, sqlc.GetFunctionByAppNameAndSlugParams{
		Name: appName,
		Slug: slug,
	})
	if err != nil {
		return nil, err
	}
	return functionFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetFunctions(ctx context.Context) ([]*db.Function, error) {
	rows, err := sq.q.GetFunctions(ctx)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, functionFromSQLite), nil
}

func (sq *sqliteQuerier) UpsertFunction(ctx context.Context, arg db.UpsertFunctionParams) (*db.Function, error) {
	r, err := sq.q.UpsertFunction(ctx, sqlc.UpsertFunctionParams{
		ID: arg.ID, AppID: arg.AppID, Name: arg.Name,
		Slug: arg.Slug, Config: arg.Config, CreatedAt: arg.CreatedAt,
	})
	if err != nil {
		return nil, err
	}
	return functionFromSQLite(r), nil
}

func (sq *sqliteQuerier) UpdateFunctionConfig(ctx context.Context, arg db.UpdateFunctionConfigParams) (*db.Function, error) {
	r, err := sq.q.UpdateFunctionConfig(ctx, sqlc.UpdateFunctionConfigParams{Config: arg.Config, ID: arg.ID})
	if err != nil {
		return nil, err
	}
	return functionFromSQLite(r), nil
}

// --- Events ---

func (sq *sqliteQuerier) InsertEvent(ctx context.Context, arg db.InsertEventParams) error {
	return sq.q.InsertEvent(ctx, sqlc.InsertEventParams{
		InternalID: arg.InternalID, ReceivedAt: arg.ReceivedAt,
		EventID: arg.EventID, EventName: arg.EventName,
		EventData: arg.EventData, EventUser: arg.EventUser,
		EventV: arg.EventV, EventTs: arg.EventTs,
	})
}

func (sq *sqliteQuerier) GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*db.Event, error) {
	r, err := sq.q.GetEventByInternalID(ctx, internalID)
	if err != nil {
		return nil, err
	}
	return eventFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*db.Event, error) {
	rows, err := sq.q.GetEventsByInternalIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, eventFromSQLite), nil
}

func (sq *sqliteQuerier) GetEventsIDbound(ctx context.Context, arg db.GetEventsIDboundParams) ([]*db.Event, error) {
	rows, err := sq.q.GetEventsIDbound(ctx, sqlc.GetEventsIDboundParams{
		After: arg.After, Before: arg.Before,
		IncludeInternal: arg.IncludeInternal, Limit: arg.Limit,
	})
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, eventFromSQLite), nil
}

// --- Event Batches ---

func (sq *sqliteQuerier) InsertEventBatch(ctx context.Context, arg db.InsertEventBatchParams) error {
	return sq.q.InsertEventBatch(ctx, sqlc.InsertEventBatchParams{
		ID: arg.ID, AccountID: arg.AccountID, WorkspaceID: arg.WorkspaceID,
		AppID: arg.AppID, WorkflowID: arg.WorkflowID, RunID: arg.RunID,
		StartedAt: arg.StartedAt, ExecutedAt: arg.ExecutedAt, EventIds: arg.EventIds,
	})
}

func (sq *sqliteQuerier) GetEventBatchByRunID(ctx context.Context, runID ulid.ULID) (*db.EventBatch, error) {
	r, err := sq.q.GetEventBatchByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	return eventBatchFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetEventBatchesByEventID(ctx context.Context, instr string) ([]*db.EventBatch, error) {
	rows, err := sq.q.GetEventBatchesByEventID(ctx, instr)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, eventBatchFromSQLite), nil
}

// --- Function Runs ---

func (sq *sqliteQuerier) InsertFunctionRun(ctx context.Context, arg db.InsertFunctionRunParams) error {
	return sq.q.InsertFunctionRun(ctx, sqlc.InsertFunctionRunParams{
		RunID: arg.RunID, RunStartedAt: arg.RunStartedAt,
		FunctionID: arg.FunctionID, FunctionVersion: arg.FunctionVersion,
		TriggerType: arg.TriggerType, EventID: arg.EventID,
		BatchID: arg.BatchID, OriginalRunID: arg.OriginalRunID,
		Cron: arg.Cron, WorkspaceID: arg.WorkspaceID,
	})
}

func (sq *sqliteQuerier) InsertFunctionFinish(ctx context.Context, arg db.InsertFunctionFinishParams) error {
	return sq.q.InsertFunctionFinish(ctx, sqlc.InsertFunctionFinishParams{
		RunID: arg.RunID, Status: arg.Status, Output: arg.Output,
		CompletedStepCount: arg.CompletedStepCount, CreatedAt: arg.CreatedAt,
	})
}

func (sq *sqliteQuerier) GetFunctionRun(ctx context.Context, runID ulid.ULID) (*db.FunctionRunRow, error) {
	r, err := sq.q.GetFunctionRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	return functionRunRowFromSQLite(&r.FunctionRun, &r.FunctionFinish), nil
}

func (sq *sqliteQuerier) GetFunctionRuns(ctx context.Context) ([]*db.FunctionRunRow, error) {
	rows, err := sq.q.GetFunctionRuns(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*db.FunctionRunRow, len(rows))
	for i, r := range rows {
		out[i] = functionRunRowFromSQLite(&r.FunctionRun, &r.FunctionFinish)
	}
	return out, nil
}

func (sq *sqliteQuerier) GetFunctionRunsFromEvents(ctx context.Context, eventIds []ulid.ULID) ([]*db.FunctionRunRow, error) {
	rows, err := sq.q.GetFunctionRunsFromEvents(ctx, eventIds)
	if err != nil {
		return nil, err
	}
	out := make([]*db.FunctionRunRow, len(rows))
	for i, r := range rows {
		out[i] = functionRunRowFromSQLite(&r.FunctionRun, &r.FunctionFinish)
	}
	return out, nil
}

func (sq *sqliteQuerier) GetFunctionRunsTimebound(ctx context.Context, arg db.GetFunctionRunsTimeboundParams) ([]*db.FunctionRunRow, error) {
	rows, err := sq.q.GetFunctionRunsTimebound(ctx, sqlc.GetFunctionRunsTimeboundParams{
		After: arg.After, Before: arg.Before, Limit: arg.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*db.FunctionRunRow, len(rows))
	for i, r := range rows {
		out[i] = functionRunRowFromSQLite(&r.FunctionRun, &r.FunctionFinish)
	}
	return out, nil
}

func (sq *sqliteQuerier) GetFunctionRunFinishesByRunIDs(ctx context.Context, runIds []ulid.ULID) ([]*db.FunctionFinish, error) {
	rows, err := sq.q.GetFunctionRunFinishesByRunIDs(ctx, runIds)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, functionFinishFromSQLite), nil
}

// --- History ---

func (sq *sqliteQuerier) InsertHistory(ctx context.Context, arg db.InsertHistoryParams) error {
	return sq.q.InsertHistory(ctx, sqlc.InsertHistoryParams{
		ID: arg.ID, CreatedAt: arg.CreatedAt, RunStartedAt: arg.RunStartedAt,
		FunctionID: arg.FunctionID, FunctionVersion: arg.FunctionVersion,
		RunID: arg.RunID, EventID: arg.EventID, BatchID: arg.BatchID,
		GroupID: arg.GroupID, IdempotencyKey: arg.IdempotencyKey,
		Type: arg.Type, Attempt: arg.Attempt, LatencyMs: arg.LatencyMs,
		StepName: arg.StepName, StepID: arg.StepID, StepType: arg.StepType,
		Url: arg.Url, CancelRequest: arg.CancelRequest, Sleep: arg.Sleep,
		WaitForEvent: arg.WaitForEvent, WaitResult: arg.WaitResult,
		InvokeFunction: arg.InvokeFunction, InvokeFunctionResult: arg.InvokeFunctionResult,
		Result: arg.Result,
	})
}

func (sq *sqliteQuerier) GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*db.History, error) {
	rows, err := sq.q.GetFunctionRunHistory(ctx, runID)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, historyFromSQLite), nil
}

func (sq *sqliteQuerier) GetRunDeferOpcodes(ctx context.Context, runID ulid.ULID, stepTypes []string) ([]*db.RunDeferOpcode, error) {
	nullStepTypes := make([]sql.NullString, len(stepTypes))
	for i, s := range stepTypes {
		nullStepTypes[i] = sql.NullString{String: s, Valid: true}
	}
	rows, err := sq.q.GetRunDeferOpcodes(ctx, sqlc.GetRunDeferOpcodesParams{
		RunID:     runID,
		StepTypes: nullStepTypes,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*db.RunDeferOpcode, len(rows))
	for i, r := range rows {
		out[i] = &db.RunDeferOpcode{ID: r.ID, Result: r.Result}
	}
	return out, nil
}

func (sq *sqliteQuerier) GetRunsByUserEventIDs(ctx context.Context, eventIDs []string) ([]*db.RunWithUserEventID, error) {
	rows, err := sq.q.GetRunsByUserEventIDs(ctx, eventIDs)
	if err != nil {
		return nil, err
	}
	out := make([]*db.RunWithUserEventID, len(rows))
	for i, r := range rows {
		out[i] = &db.RunWithUserEventID{
			UserEventID:    r.UserEventID,
			FunctionRun:    *functionRunFromSQLite(&r.FunctionRun),
			FunctionFinish: *functionFinishFromSQLite(&r.FunctionFinish),
		}
	}
	return out, nil
}

func (sq *sqliteQuerier) GetRunDeferredFromEvent(ctx context.Context, runID ulid.ULID, eventName string) (string, error) {
	return sq.q.GetRunDeferredFromEvent(ctx, sqlc.GetRunDeferredFromEventParams{
		RunID:     runID,
		EventName: eventName,
	})
}

func (sq *sqliteQuerier) GetHistoryItem(ctx context.Context, id ulid.ULID) (*db.History, error) {
	r, err := sq.q.GetHistoryItem(ctx, id)
	if err != nil {
		return nil, err
	}
	return historyFromSQLite(r), nil
}

func (sq *sqliteQuerier) HistoryCountRuns(ctx context.Context) (int64, error) {
	return sq.q.HistoryCountRuns(ctx)
}

// --- Queue Snapshots ---

func (sq *sqliteQuerier) InsertQueueSnapshotChunk(ctx context.Context, arg db.InsertQueueSnapshotChunkParams) error {
	return sq.q.InsertQueueSnapshotChunk(ctx, sqlc.InsertQueueSnapshotChunkParams{
		SnapshotID: arg.SnapshotID, ChunkID: arg.ChunkID, Data: arg.Data,
	})
}

func (sq *sqliteQuerier) DeleteOldQueueSnapshots(ctx context.Context, limit int64) (int64, error) {
	return sq.q.DeleteOldQueueSnapshots(ctx, limit)
}

func (sq *sqliteQuerier) GetQueueSnapshotChunks(ctx context.Context, snapshotID string) ([]*db.QueueSnapshotChunkRow, error) {
	rows, err := sq.q.GetQueueSnapshotChunks(ctx, snapshotID)
	if err != nil {
		return nil, err
	}
	out := make([]*db.QueueSnapshotChunkRow, len(rows))
	for i, r := range rows {
		out[i] = &db.QueueSnapshotChunkRow{ChunkID: r.ChunkID, Data: r.Data}
	}
	return out, nil
}

func (sq *sqliteQuerier) GetLatestQueueSnapshotChunks(ctx context.Context) ([]*db.QueueSnapshotChunkRow, error) {
	rows, err := sq.q.GetLatestQueueSnapshotChunks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*db.QueueSnapshotChunkRow, len(rows))
	for i, r := range rows {
		out[i] = &db.QueueSnapshotChunkRow{ChunkID: r.ChunkID, Data: r.Data}
	}
	return out, nil
}

// --- Spans ---

func (sq *sqliteQuerier) InsertSpan(ctx context.Context, arg db.InsertSpanParams) error {
	startTime := arg.StartTime.Round(0).UTC()
	endTime := arg.EndTime.Round(0).UTC()

	return sq.q.InsertSpan(ctx, sqlc.InsertSpanParams{
		SpanID: arg.SpanID, TraceID: arg.TraceID, ParentSpanID: arg.ParentSpanID,
		Name: arg.Name, StartTime: startTime, EndTime: endTime,
		RunID: arg.RunID, AccountID: arg.AccountID, AppID: arg.AppID,
		FunctionID: arg.FunctionID, EnvID: arg.EnvID,
		DynamicSpanID:  arg.DynamicSpanID,
		Attributes:     bytesToNullString(arg.Attributes),
		Links:          bytesToNullString(arg.Links),
		Output:         bytesToNullString(arg.Output),
		Input:          bytesToNullString(arg.Input),
		DebugRunID:     arg.DebugRunID,
		DebugSessionID: arg.DebugSessionID,
		Status:         arg.Status,
		EventIds:       bytesToNullString(arg.EventIds),
	})
}

func (sq *sqliteQuerier) GetSpansByRunID(ctx context.Context, runID string) ([]*db.SpanRow, error) {
	rows, err := sq.q.GetSpansByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	out := make([]*db.SpanRow, len(rows))
	for i, r := range rows {
		out[i] = spanRowFromSQLiteRunID(r)
	}
	return out, nil
}

func (sq *sqliteQuerier) GetSpansByDebugRunID(ctx context.Context, debugRunID sql.NullString) ([]*db.SpanRow, error) {
	rows, err := sq.q.GetSpansByDebugRunID(ctx, debugRunID)
	if err != nil {
		return nil, err
	}
	out := make([]*db.SpanRow, len(rows))
	for i, r := range rows {
		out[i] = &db.SpanRow{
			RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
			StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
			SpanFragments: toBytes(r.SpanFragments), DebugSessionID: r.DebugSessionID,
			DebugRunID: debugRunID,
		}
	}
	return out, nil
}

func (sq *sqliteQuerier) GetSpansByDebugSessionID(ctx context.Context, debugSessionID sql.NullString) ([]*db.SpanRow, error) {
	rows, err := sq.q.GetSpansByDebugSessionID(ctx, debugSessionID)
	if err != nil {
		return nil, err
	}
	out := make([]*db.SpanRow, len(rows))
	for i, r := range rows {
		out[i] = &db.SpanRow{
			RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
			StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
			SpanFragments: toBytes(r.SpanFragments), DebugRunID: r.DebugRunID,
			DebugSessionID: debugSessionID,
		}
	}
	return out, nil
}

func (sq *sqliteQuerier) GetRunSpanByRunID(ctx context.Context, arg db.GetRunSpanByRunIDParams) (*db.SpanRow, error) {
	r, err := sq.q.GetRunSpanByRunID(ctx, sqlc.GetRunSpanByRunIDParams{RunID: arg.RunID, AccountID: arg.AccountID})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: toBytes(r.SpanFragments),
	}, nil
}

func (sq *sqliteQuerier) GetSpanBySpanID(ctx context.Context, arg db.GetSpanBySpanIDParams) (*db.SpanRow, error) {
	r, err := sq.q.GetSpanBySpanID(ctx, sqlc.GetSpanBySpanIDParams{RunID: arg.RunID, SpanID: arg.SpanID, AccountID: arg.AccountID})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: toBytes(r.SpanFragments),
	}, nil
}

func (sq *sqliteQuerier) GetStepSpanByStepID(ctx context.Context, arg db.GetStepSpanByStepIDParams) (*db.SpanRow, error) {
	r, err := sq.q.GetStepSpanByStepID(ctx, sqlc.GetStepSpanByStepIDParams{RunID: arg.RunID, AccountID: arg.AccountID, StepID: arg.StepID})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: toBytes(r.SpanFragments),
	}, nil
}

func (sq *sqliteQuerier) GetSpanOutput(ctx context.Context, ids []string) ([]*db.SpanOutputRow, error) {
	rows, err := sq.q.GetSpanOutput(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]*db.SpanOutputRow, len(rows))
	for i, r := range rows {
		out[i] = &db.SpanOutputRow{Input: toBytes(r.Input), Output: toBytes(r.Output)}
	}
	return out, nil
}

func (sq *sqliteQuerier) GetExecutionSpanByStepIDAndAttempt(ctx context.Context, arg db.GetExecutionSpanByStepIDAndAttemptParams) (*db.SpanRow, error) {
	r, err := sq.q.GetExecutionSpanByStepIDAndAttempt(ctx, sqlc.GetExecutionSpanByStepIDAndAttemptParams{
		RunID: arg.RunID, AccountID: arg.AccountID, StepID: arg.StepID, StepAttempt: arg.StepAttempt,
	})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: toBytes(r.SpanFragments),
	}, nil
}

func (sq *sqliteQuerier) GetLatestExecutionSpanByStepID(ctx context.Context, arg db.GetLatestExecutionSpanByStepIDParams) (*db.SpanRow, error) {
	r, err := sq.q.GetLatestExecutionSpanByStepID(ctx, sqlc.GetLatestExecutionSpanByStepIDParams{
		RunID: arg.RunID, AccountID: arg.AccountID, StepID: arg.StepID,
	})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: toBytes(r.SpanFragments),
	}, nil
}

// --- Traces ---

func (sq *sqliteQuerier) InsertTrace(ctx context.Context, arg db.InsertTraceParams) error {
	return sq.q.InsertTrace(ctx, sqlc.InsertTraceParams{
		Timestamp: arg.Timestamp, TimestampUnixMs: arg.TimestampUnixMs,
		TraceID: arg.TraceID, SpanID: arg.SpanID, ParentSpanID: arg.ParentSpanID,
		TraceState: arg.TraceState, SpanName: arg.SpanName, SpanKind: arg.SpanKind,
		ServiceName: arg.ServiceName, ResourceAttributes: arg.ResourceAttributes,
		ScopeName: arg.ScopeName, ScopeVersion: arg.ScopeVersion,
		SpanAttributes: arg.SpanAttributes, Duration: arg.Duration,
		StatusCode: arg.StatusCode, StatusMessage: arg.StatusMessage,
		Events: arg.Events, Links: arg.Links, RunID: arg.RunID,
	})
}

func (sq *sqliteQuerier) InsertTraceRun(ctx context.Context, arg db.InsertTraceRunParams) error {
	return sq.q.InsertTraceRun(ctx, sqlc.InsertTraceRunParams{
		RunID: arg.RunID, AccountID: arg.AccountID, WorkspaceID: arg.WorkspaceID,
		AppID: arg.AppID, FunctionID: arg.FunctionID, TraceID: arg.TraceID,
		QueuedAt: arg.QueuedAt, StartedAt: arg.StartedAt, EndedAt: arg.EndedAt,
		Status: arg.Status, SourceID: arg.SourceID, TriggerIds: arg.TriggerIds,
		Output: arg.Output, BatchID: arg.BatchID, IsDebounce: arg.IsDebounce,
		CronSchedule: arg.CronSchedule, HasAi: arg.HasAi,
	})
}

func (sq *sqliteQuerier) GetTraceRun(ctx context.Context, runID ulid.ULID) (*db.TraceRun, error) {
	r, err := sq.q.GetTraceRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	return traceRunFromSQLite(r), nil
}

func (sq *sqliteQuerier) GetTraceRunsByTriggerId(ctx context.Context, eventID string) ([]*db.TraceRun, error) {
	rows, err := sq.q.GetTraceRunsByTriggerId(ctx, eventID)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, traceRunFromSQLite), nil
}

func (sq *sqliteQuerier) GetTraceSpans(ctx context.Context, arg db.GetTraceSpansParams) ([]*db.Trace, error) {
	rows, err := sq.q.GetTraceSpans(ctx, sqlc.GetTraceSpansParams{TraceID: arg.TraceID, RunID: arg.RunID})
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, traceFromSQLite), nil
}

func (sq *sqliteQuerier) GetTraceSpanOutput(ctx context.Context, arg db.GetTraceSpanOutputParams) ([]*db.Trace, error) {
	rows, err := sq.q.GetTraceSpanOutput(ctx, sqlc.GetTraceSpanOutputParams{TraceID: arg.TraceID, SpanID: arg.SpanID})
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, traceFromSQLite), nil
}

// --- Worker Connections ---

func (sq *sqliteQuerier) InsertWorkerConnection(ctx context.Context, arg db.InsertWorkerConnectionParams) error {
	return sq.q.InsertWorkerConnection(ctx, sqlc.InsertWorkerConnectionParams{
		AccountID: arg.AccountID, WorkspaceID: arg.WorkspaceID,
		AppName: arg.AppName, AppID: arg.AppID, ID: arg.ID,
		GatewayID: arg.GatewayID, InstanceID: arg.InstanceID,
		Status: arg.Status, WorkerIp: arg.WorkerIp,
		MaxWorkerConcurrency: arg.MaxWorkerConcurrency,
		ConnectedAt:          arg.ConnectedAt, LastHeartbeatAt: arg.LastHeartbeatAt,
		DisconnectedAt: arg.DisconnectedAt, RecordedAt: arg.RecordedAt,
		InsertedAt: arg.InsertedAt, DisconnectReason: arg.DisconnectReason,
		GroupHash: arg.GroupHash, SdkLang: arg.SdkLang, SdkVersion: arg.SdkVersion,
		SdkPlatform: arg.SdkPlatform, SyncID: arg.SyncID, AppVersion: arg.AppVersion,
		FunctionCount: arg.FunctionCount, CpuCores: arg.CpuCores,
		MemBytes: arg.MemBytes, Os: arg.Os,
	})
}

func (sq *sqliteQuerier) GetWorkerConnection(ctx context.Context, arg db.GetWorkerConnectionParams) (*db.WorkerConnection, error) {
	r, err := sq.q.GetWorkerConnection(ctx, sqlc.GetWorkerConnectionParams{
		AccountID: arg.AccountID, WorkspaceID: arg.WorkspaceID, ConnectionID: arg.ConnectionID,
	})
	if err != nil {
		return nil, err
	}
	return workerConnectionFromSQLite(r), nil
}

// --- helpers ---

func convertSlice[S any, D any](src []*S, fn func(*S) *D) []*D {
	out := make([]*D, len(src))
	for i, s := range src {
		out[i] = fn(s)
	}
	return out
}

func toBytes(v interface{}) []byte {
	if v == nil {
		return nil
	}
	switch b := v.(type) {
	case []byte:
		return b
	case json.RawMessage:
		return []byte(b)
	case string:
		return []byte(b)
	default:
		return nil
	}
}
