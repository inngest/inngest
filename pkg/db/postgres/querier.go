package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	"github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/db/postgres/sqltypes"
	"github.com/oklog/ulid/v2"
)

var _ db.Querier = (*pgQuerier)(nil)

type pgQuerier struct {
	db sqlc.DBTX
	q  *sqlc.Queries
}

// --- Apps ---

func (pq *pgQuerier) DeleteApp(ctx context.Context, id uuid.UUID) error {
	return pq.q.DeleteApp(ctx, id)
}

func (pq *pgQuerier) GetAllApps(ctx context.Context) ([]*db.App, error) {
	rows, err := pq.q.GetAllApps(ctx)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, appFromPG), nil
}

func (pq *pgQuerier) GetApp(ctx context.Context, id uuid.UUID) (*db.App, error) {
	r, err := pq.q.GetApp(ctx, id)
	if err != nil {
		return nil, err
	}
	return appFromPG(r), nil
}

func (pq *pgQuerier) GetAppByChecksum(ctx context.Context, checksum string) (*db.App, error) {
	r, err := pq.q.GetAppByChecksum(ctx, checksum)
	if err != nil {
		return nil, err
	}
	return appFromPG(r), nil
}

func (pq *pgQuerier) GetAppByID(ctx context.Context, id uuid.UUID) (*db.App, error) {
	r, err := pq.q.GetAppByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return appFromPG(r), nil
}

func (pq *pgQuerier) GetAppByName(ctx context.Context, name string) (*db.App, error) {
	r, err := pq.q.GetAppByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return appFromPG(r), nil
}

func (pq *pgQuerier) GetAppByURL(ctx context.Context, url string) (*db.App, error) {
	r, err := pq.q.GetAppByURL(ctx, url)
	if err != nil {
		return nil, err
	}
	return appFromPG(r), nil
}

func (pq *pgQuerier) GetApps(ctx context.Context) ([]*db.App, error) {
	rows, err := pq.q.GetApps(ctx)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, appFromPG), nil
}

func (pq *pgQuerier) UpsertApp(ctx context.Context, arg db.UpsertAppParams) (*db.App, error) {
	r, err := pq.q.UpsertApp(ctx, sqlc.UpsertAppParams{
		ID: arg.ID, Name: arg.Name, SdkLanguage: arg.SdkLanguage,
		SdkVersion: arg.SdkVersion, Framework: arg.Framework, Metadata: arg.Metadata,
		Status: arg.Status, Error: arg.Error, Checksum: arg.Checksum,
		Url: arg.Url, Method: arg.Method, AppVersion: arg.AppVersion,
	})
	if err != nil {
		return nil, err
	}
	return appFromPG(r), nil
}

func (pq *pgQuerier) UpdateAppError(ctx context.Context, arg db.UpdateAppErrorParams) (*db.App, error) {
	r, err := pq.q.UpdateAppError(ctx, sqlc.UpdateAppErrorParams{Error: arg.Error, ID: arg.ID})
	if err != nil {
		return nil, err
	}
	return appFromPG(r), nil
}

func (pq *pgQuerier) UpdateAppURL(ctx context.Context, arg db.UpdateAppURLParams) (*db.App, error) {
	r, err := pq.q.UpdateAppURL(ctx, sqlc.UpdateAppURLParams{Url: arg.Url, ID: arg.ID})
	if err != nil {
		return nil, err
	}
	return appFromPG(r), nil
}

// --- Functions ---

func (pq *pgQuerier) DeleteFunctionsByAppID(ctx context.Context, appID uuid.UUID) error {
	return pq.q.DeleteFunctionsByAppID(ctx, appID)
}

func (pq *pgQuerier) DeleteFunctionsByIDs(ctx context.Context, ids []uuid.UUID) error {
	// Postgres sqlc expects []string for this query.
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	return pq.q.DeleteFunctionsByIDs(ctx, strs)
}

func (pq *pgQuerier) GetAppFunctions(ctx context.Context, appID uuid.UUID) ([]*db.Function, error) {
	rows, err := pq.q.GetAppFunctions(ctx, appID)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, functionFromPG), nil
}

func (pq *pgQuerier) GetAppFunctionsBySlug(ctx context.Context, name string) ([]*db.Function, error) {
	rows, err := pq.q.GetAppFunctionsBySlug(ctx, name)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, functionFromPG), nil
}

func (pq *pgQuerier) GetFunctionByID(ctx context.Context, id uuid.UUID) (*db.Function, error) {
	r, err := pq.q.GetFunctionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return functionFromPG(r), nil
}

func (pq *pgQuerier) GetFunctionBySlug(ctx context.Context, slug string) (*db.Function, error) {
	r, err := pq.q.GetFunctionBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return functionFromPG(r), nil
}

func (pq *pgQuerier) GetFunctions(ctx context.Context) ([]*db.Function, error) {
	rows, err := pq.q.GetFunctions(ctx)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, functionFromPG), nil
}

func (pq *pgQuerier) InsertFunction(ctx context.Context, arg db.InsertFunctionParams) (*db.Function, error) {
	r, err := pq.q.InsertFunction(ctx, sqlc.InsertFunctionParams{
		ID: arg.ID, AppID: arg.AppID, Name: arg.Name,
		Slug: arg.Slug, Config: arg.Config, CreatedAt: arg.CreatedAt,
	})
	if err != nil {
		return nil, err
	}
	return functionFromPG(r), nil
}

func (pq *pgQuerier) UpdateFunctionConfig(ctx context.Context, arg db.UpdateFunctionConfigParams) (*db.Function, error) {
	r, err := pq.q.UpdateFunctionConfig(ctx, sqlc.UpdateFunctionConfigParams{Config: arg.Config, ID: arg.ID})
	if err != nil {
		return nil, err
	}
	return functionFromPG(r), nil
}

// --- Events ---

func (pq *pgQuerier) InsertEvent(ctx context.Context, arg db.InsertEventParams) error {
	return pq.q.InsertEvent(ctx, sqlc.InsertEventParams{
		InternalID: arg.InternalID, ReceivedAt: arg.ReceivedAt,
		EventID: arg.EventID, EventName: arg.EventName,
		EventData: arg.EventData, EventUser: arg.EventUser,
		EventV: arg.EventV, EventTs: arg.EventTs,
	})
}

func (pq *pgQuerier) GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*db.Event, error) {
	r, err := pq.q.GetEventByInternalID(ctx, internalID)
	if err != nil {
		return nil, err
	}
	return eventFromPG(r), nil
}

func (pq *pgQuerier) GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*db.Event, error) {
	// Postgres sqlc expects [][]byte for this query.
	byteIDs := make([][]byte, len(ids))
	for i, id := range ids {
		byteIDs[i] = id[:]
	}
	rows, err := pq.q.GetEventsByInternalIDs(ctx, byteIDs)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, eventFromPG), nil
}

func (pq *pgQuerier) GetEventsIDbound(ctx context.Context, arg db.GetEventsIDboundParams) ([]*db.Event, error) {
	rows, err := pq.q.GetEventsIDbound(ctx, sqlc.GetEventsIDboundParams{
		InternalID: arg.After, InternalID_2: arg.Before,
		EventName: arg.IncludeInternal, Limit: int32(arg.Limit),
	})
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, eventFromPG), nil
}

// --- Event Batches ---

func (pq *pgQuerier) InsertEventBatch(ctx context.Context, arg db.InsertEventBatchParams) error {
	return pq.q.InsertEventBatch(ctx, sqlc.InsertEventBatchParams{
		ID:          sqltypes.FromULID(arg.ID),
		AccountID:   arg.AccountID,
		WorkspaceID: arg.WorkspaceID,
		AppID:       arg.AppID,
		WorkflowID:  arg.WorkflowID,
		RunID:       sqltypes.FromULID(arg.RunID),
		StartedAt:   arg.StartedAt,
		ExecutedAt:  arg.ExecutedAt,
		EventIds:    arg.EventIds,
	})
}

func (pq *pgQuerier) GetEventBatchByRunID(ctx context.Context, runID ulid.ULID) (*db.EventBatch, error) {
	// Postgres sqlc expects string for this query.
	r, err := pq.q.GetEventBatchByRunID(ctx, runID.String())
	if err != nil {
		return nil, err
	}
	return eventBatchFromPG(r), nil
}

func (pq *pgQuerier) GetEventBatchesByEventID(ctx context.Context, instr string) ([]*db.EventBatch, error) {
	rows, err := pq.q.GetEventBatchesByEventID(ctx, instr)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, eventBatchFromPG), nil
}

// --- Function Runs ---

func (pq *pgQuerier) InsertFunctionRun(ctx context.Context, arg db.InsertFunctionRunParams) error {
	return pq.q.InsertFunctionRun(ctx, sqlc.InsertFunctionRunParams{
		RunID: arg.RunID, RunStartedAt: arg.RunStartedAt,
		FunctionID: arg.FunctionID, FunctionVersion: int32(arg.FunctionVersion),
		TriggerType: arg.TriggerType, EventID: arg.EventID,
		BatchID: arg.BatchID, OriginalRunID: arg.OriginalRunID,
		Cron: arg.Cron,
		// Postgres InsertFunctionRunParams doesn't have WorkspaceID.
	})
}

func (pq *pgQuerier) InsertFunctionFinish(ctx context.Context, arg db.InsertFunctionFinishParams) error {
	// Postgres schema: status VARCHAR NOT NULL, output VARCHAR NOT NULL DEFAULT '{}',
	// completed_step_count INT NOT NULL DEFAULT 1, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP.
	// Use schema defaults when the nullable domain field is not set.
	status := arg.Status.String
	if !arg.Status.Valid {
		return fmt.Errorf("InsertFunctionFinish: status is required")
	}
	output := "{}"
	if arg.Output.Valid {
		output = arg.Output.String
	}
	var completedStepCount int32 = 1
	if arg.CompletedStepCount.Valid {
		completedStepCount = int32(arg.CompletedStepCount.Int64)
	}
	createdAt := time.Now()
	if arg.CreatedAt.Valid {
		createdAt = arg.CreatedAt.Time
	}

	return pq.q.InsertFunctionFinish(ctx, sqlc.InsertFunctionFinishParams{
		RunID:              arg.RunID,
		Status:             status,
		Output:             output,
		CompletedStepCount: completedStepCount,
		CreatedAt:          createdAt,
	})
}

func (pq *pgQuerier) GetFunctionRun(ctx context.Context, runID ulid.ULID) (*db.FunctionRunRow, error) {
	r, err := pq.q.GetFunctionRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	return functionRunRowFromPG(&r.FunctionRun, &r.FunctionFinish), nil
}

func (pq *pgQuerier) GetFunctionRuns(ctx context.Context) ([]*db.FunctionRunRow, error) {
	rows, err := pq.q.GetFunctionRuns(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*db.FunctionRunRow, len(rows))
	for i, r := range rows {
		out[i] = functionRunRowFromPG(&r.FunctionRun, &r.FunctionFinish)
	}
	return out, nil
}

func (pq *pgQuerier) GetFunctionRunsFromEvents(ctx context.Context, eventIds []ulid.ULID) ([]*db.FunctionRunRow, error) {
	// Postgres sqlc expects [][]byte for this query.
	byteIDs := make([][]byte, len(eventIds))
	for i, id := range eventIds {
		byteIDs[i] = id[:]
	}
	rows, err := pq.q.GetFunctionRunsFromEvents(ctx, byteIDs)
	if err != nil {
		return nil, err
	}
	// GetFunctionRunsFromEventsRow has flattened finish fields instead of embedded struct.
	out := make([]*db.FunctionRunRow, len(rows))
	for i, r := range rows {
		out[i] = &db.FunctionRunRow{
			FunctionRun: *functionRunFromPG(&r.FunctionRun),
			FunctionFinish: db.FunctionFinish{
				RunID:              r.FunctionRun.RunID,
				Status:             sql.NullString{String: r.FinishStatus, Valid: r.FinishStatus != ""},
				Output:             sql.NullString{String: r.FinishOutput, Valid: r.FinishOutput != ""},
				CompletedStepCount: sql.NullInt64{Int64: int64(r.FinishCompletedStepCount), Valid: true},
				CreatedAt:          sql.NullTime{Time: r.FinishCreatedAt, Valid: !r.FinishCreatedAt.IsZero()},
			},
		}
	}
	return out, nil
}

func (pq *pgQuerier) GetFunctionRunsTimebound(ctx context.Context, arg db.GetFunctionRunsTimeboundParams) ([]*db.FunctionRunRow, error) {
	rows, err := pq.q.GetFunctionRunsTimebound(ctx, sqlc.GetFunctionRunsTimeboundParams{
		RunStartedAt: arg.After, RunStartedAt_2: arg.Before, Limit: int32(arg.Limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*db.FunctionRunRow, len(rows))
	for i, r := range rows {
		out[i] = functionRunRowFromPG(&r.FunctionRun, &r.FunctionFinish)
	}
	return out, nil
}

func (pq *pgQuerier) GetFunctionRunFinishesByRunIDs(ctx context.Context, runIds []ulid.ULID) ([]*db.FunctionFinish, error) {
	rows, err := pq.q.GetFunctionRunFinishesByRunIDs(ctx, runIds)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, functionFinishFromPG), nil
}

// --- History ---

func (pq *pgQuerier) InsertHistory(ctx context.Context, arg db.InsertHistoryParams) error {
	return pq.q.InsertHistory(ctx, sqlc.InsertHistoryParams{
		ID: arg.ID, CreatedAt: arg.CreatedAt, RunStartedAt: arg.RunStartedAt,
		FunctionID: arg.FunctionID, FunctionVersion: int32(arg.FunctionVersion),
		RunID: arg.RunID, EventID: arg.EventID, BatchID: arg.BatchID,
		GroupID: arg.GroupID, IdempotencyKey: arg.IdempotencyKey,
		Type: arg.Type, Attempt: int32(arg.Attempt),
		LatencyMs: sql.NullInt32{Int32: int32(arg.LatencyMs.Int64), Valid: arg.LatencyMs.Valid},
		StepName:  arg.StepName, StepID: arg.StepID, StepType: arg.StepType,
		Url: arg.Url, CancelRequest: arg.CancelRequest, Sleep: arg.Sleep,
		WaitForEvent: arg.WaitForEvent, WaitResult: arg.WaitResult,
		InvokeFunction: arg.InvokeFunction, InvokeFunctionResult: arg.InvokeFunctionResult,
		Result: arg.Result,
	})
}

func (pq *pgQuerier) GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*db.History, error) {
	rows, err := pq.q.GetFunctionRunHistory(ctx, runID)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, historyFromPG), nil
}

func (pq *pgQuerier) GetHistoryItem(ctx context.Context, id ulid.ULID) (*db.History, error) {
	r, err := pq.q.GetHistoryItem(ctx, id)
	if err != nil {
		return nil, err
	}
	return historyFromPG(r), nil
}

func (pq *pgQuerier) HistoryCountRuns(ctx context.Context) (int64, error) {
	return pq.q.HistoryCountRuns(ctx)
}

// --- Queue Snapshots ---

func (pq *pgQuerier) InsertQueueSnapshotChunk(ctx context.Context, arg db.InsertQueueSnapshotChunkParams) error {
	return pq.q.InsertQueueSnapshotChunk(ctx, sqlc.InsertQueueSnapshotChunkParams{
		SnapshotID: arg.SnapshotID, ChunkID: int32(arg.ChunkID), Data: arg.Data,
	})
}

func (pq *pgQuerier) DeleteOldQueueSnapshots(ctx context.Context, limit int64) (int64, error) {
	return pq.q.DeleteOldQueueSnapshots(ctx, int32(limit))
}

func (pq *pgQuerier) GetQueueSnapshotChunks(ctx context.Context, snapshotID string) ([]*db.QueueSnapshotChunkRow, error) {
	rows, err := pq.q.GetQueueSnapshotChunks(ctx, snapshotID)
	if err != nil {
		return nil, err
	}
	out := make([]*db.QueueSnapshotChunkRow, len(rows))
	for i, r := range rows {
		out[i] = &db.QueueSnapshotChunkRow{ChunkID: int64(r.ChunkID), Data: r.Data}
	}
	return out, nil
}

func (pq *pgQuerier) GetLatestQueueSnapshotChunks(ctx context.Context) ([]*db.QueueSnapshotChunkRow, error) {
	rows, err := pq.q.GetLatestQueueSnapshotChunks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*db.QueueSnapshotChunkRow, len(rows))
	for i, r := range rows {
		out[i] = &db.QueueSnapshotChunkRow{ChunkID: int64(r.ChunkID), Data: r.Data}
	}
	return out, nil
}

// --- Spans ---

func (pq *pgQuerier) InsertSpan(ctx context.Context, arg db.InsertSpanParams) error {
	startTime := arg.StartTime.Round(0).UTC()
	endTime := arg.EndTime.Round(0).UTC()

	return pq.q.InsertSpan(ctx, sqlc.InsertSpanParams{
		SpanID: arg.SpanID, TraceID: arg.TraceID, ParentSpanID: arg.ParentSpanID,
		Name: arg.Name, StartTime: startTime, EndTime: endTime,
		RunID: arg.RunID, AccountID: arg.AccountID, AppID: arg.AppID,
		FunctionID: arg.FunctionID, EnvID: arg.EnvID,
		DynamicSpanID: arg.DynamicSpanID,
		Attributes:    bytesToNullRaw(arg.Attributes),
		Links:         bytesToNullRaw(arg.Links),
		Output:        bytesToNullRaw(arg.Output),
		Input:         bytesToNullRaw(arg.Input),
		DebugRunID:    arg.DebugRunID, DebugSessionID: arg.DebugSessionID,
		Status:   arg.Status,
		EventIds: bytesToNullRaw(arg.EventIds),
	})
}

func (pq *pgQuerier) GetSpansByRunID(ctx context.Context, runID string) ([]*db.SpanRow, error) {
	rows, err := pq.q.GetSpansByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	out := make([]*db.SpanRow, len(rows))
	for i, r := range rows {
		out[i] = &db.SpanRow{
			RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
			StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
			SpanFragments: r.SpanFragments,
		}
	}
	return out, nil
}

func (pq *pgQuerier) GetSpansByDebugRunID(ctx context.Context, debugRunID sql.NullString) ([]*db.SpanRow, error) {
	// Postgres sqlc expects string for this query.
	rows, err := pq.q.GetSpansByDebugRunID(ctx, debugRunID.String)
	if err != nil {
		return nil, err
	}
	out := make([]*db.SpanRow, len(rows))
	for i, r := range rows {
		out[i] = &db.SpanRow{
			RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
			StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
			SpanFragments: r.SpanFragments,
			DebugRunID:    debugRunID, DebugSessionID: r.DebugSessionID,
		}
	}
	return out, nil
}

func (pq *pgQuerier) GetSpansByDebugSessionID(ctx context.Context, debugSessionID sql.NullString) ([]*db.SpanRow, error) {
	// Postgres sqlc expects string for this query.
	rows, err := pq.q.GetSpansByDebugSessionID(ctx, debugSessionID.String)
	if err != nil {
		return nil, err
	}
	out := make([]*db.SpanRow, len(rows))
	for i, r := range rows {
		out[i] = &db.SpanRow{
			RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
			StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
			SpanFragments: r.SpanFragments,
			DebugRunID:    r.DebugRunID, DebugSessionID: debugSessionID,
		}
	}
	return out, nil
}

func (pq *pgQuerier) GetRunSpanByRunID(ctx context.Context, arg db.GetRunSpanByRunIDParams) (*db.SpanRow, error) {
	r, err := pq.q.GetRunSpanByRunID(ctx, sqlc.GetRunSpanByRunIDParams{RunID: arg.RunID, AccountID: arg.AccountID})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: r.SpanFragments,
	}, nil
}

func (pq *pgQuerier) GetSpanBySpanID(ctx context.Context, arg db.GetSpanBySpanIDParams) (*db.SpanRow, error) {
	r, err := pq.q.GetSpanBySpanID(ctx, sqlc.GetSpanBySpanIDParams{RunID: arg.RunID, SpanID: arg.SpanID, AccountID: arg.AccountID})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: r.SpanFragments,
	}, nil
}

func (pq *pgQuerier) GetStepSpanByStepID(ctx context.Context, arg db.GetStepSpanByStepIDParams) (*db.SpanRow, error) {
	r, err := pq.q.GetStepSpanByStepID(ctx, sqlc.GetStepSpanByStepIDParams{RunID: arg.RunID, AccountID: arg.AccountID, StepID: arg.StepID})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: r.SpanFragments,
	}, nil
}

func (pq *pgQuerier) GetSpanOutput(ctx context.Context, ids []string) ([]*db.SpanOutputRow, error) {
	rows, err := pq.q.GetSpanOutput(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]*db.SpanOutputRow, len(rows))
	for i, r := range rows {
		out[i] = &db.SpanOutputRow{
			Input:  nullRawToBytes(r.Input),
			Output: nullRawToBytes(r.Output),
		}
	}
	return out, nil
}

func (pq *pgQuerier) GetExecutionSpanByStepIDAndAttempt(ctx context.Context, arg db.GetExecutionSpanByStepIDAndAttemptParams) (*db.SpanRow, error) {
	r, err := pq.q.GetExecutionSpanByStepIDAndAttempt(ctx, sqlc.GetExecutionSpanByStepIDAndAttemptParams{
		RunID: arg.RunID, AccountID: arg.AccountID, StepID: arg.StepID, StepAttempt: arg.StepAttempt,
	})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: r.SpanFragments,
	}, nil
}

func (pq *pgQuerier) GetLatestExecutionSpanByStepID(ctx context.Context, arg db.GetLatestExecutionSpanByStepIDParams) (*db.SpanRow, error) {
	r, err := pq.q.GetLatestExecutionSpanByStepID(ctx, sqlc.GetLatestExecutionSpanByStepIDParams{
		RunID: arg.RunID, AccountID: arg.AccountID, StepID: arg.StepID,
	})
	if err != nil {
		return nil, err
	}
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: toTime(r.StartTime), EndTime: toTime(r.EndTime), ParentSpanID: r.ParentSpanID,
		SpanFragments: r.SpanFragments,
	}, nil
}

// --- Traces ---

func (pq *pgQuerier) InsertTrace(ctx context.Context, arg db.InsertTraceParams) error {
	return pq.q.InsertTrace(ctx, sqlc.InsertTraceParams{
		Timestamp: arg.Timestamp, TimestampUnixMs: arg.TimestampUnixMs,
		TraceID: arg.TraceID, SpanID: arg.SpanID, ParentSpanID: arg.ParentSpanID,
		TraceState: arg.TraceState, SpanName: arg.SpanName, SpanKind: arg.SpanKind,
		ServiceName: arg.ServiceName, ResourceAttributes: arg.ResourceAttributes,
		ScopeName: arg.ScopeName, ScopeVersion: arg.ScopeVersion,
		SpanAttributes: arg.SpanAttributes, Duration: int32(arg.Duration),
		StatusCode: arg.StatusCode, StatusMessage: arg.StatusMessage,
		Events: arg.Events, Links: arg.Links, RunID: arg.RunID.String(),
	})
}

func (pq *pgQuerier) InsertTraceRun(ctx context.Context, arg db.InsertTraceRunParams) error {
	return pq.q.InsertTraceRun(ctx, sqlc.InsertTraceRunParams{
		AccountID: arg.AccountID, WorkspaceID: arg.WorkspaceID,
		AppID: arg.AppID, FunctionID: arg.FunctionID, TraceID: arg.TraceID,
		RunID:    arg.RunID.String(),
		QueuedAt: arg.QueuedAt, StartedAt: arg.StartedAt, EndedAt: arg.EndedAt,
		Status: int32(arg.Status), SourceID: arg.SourceID, TriggerIds: arg.TriggerIds,
		Output: arg.Output, BatchID: arg.BatchID[:], IsDebounce: arg.IsDebounce,
		CronSchedule: arg.CronSchedule, HasAi: arg.HasAi,
	})
}

func (pq *pgQuerier) GetTraceRun(ctx context.Context, runID ulid.ULID) (*db.TraceRun, error) {
	// Postgres sqlc expects string for this query.
	r, err := pq.q.GetTraceRun(ctx, runID.String())
	if err != nil {
		return nil, err
	}
	return traceRunFromPG(r), nil
}

func (pq *pgQuerier) GetTraceRunsByTriggerId(ctx context.Context, eventID string) ([]*db.TraceRun, error) {
	// Postgres sqlc expects interface{} for this query.
	rows, err := pq.q.GetTraceRunsByTriggerId(ctx, eventID)
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, traceRunFromPG), nil
}

func (pq *pgQuerier) GetTraceSpans(ctx context.Context, arg db.GetTraceSpansParams) ([]*db.Trace, error) {
	rows, err := pq.q.GetTraceSpans(ctx, sqlc.GetTraceSpansParams{TraceID: arg.TraceID, RunID: arg.RunID.String()})
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, traceFromPG), nil
}

func (pq *pgQuerier) GetTraceSpanOutput(ctx context.Context, arg db.GetTraceSpanOutputParams) ([]*db.Trace, error) {
	rows, err := pq.q.GetTraceSpanOutput(ctx, sqlc.GetTraceSpanOutputParams{TraceID: arg.TraceID, SpanID: arg.SpanID})
	if err != nil {
		return nil, err
	}
	return convertSlice(rows, traceFromPG), nil
}

// --- Worker Connections ---

func (pq *pgQuerier) InsertWorkerConnection(ctx context.Context, arg db.InsertWorkerConnectionParams) error {
	return pq.q.InsertWorkerConnection(ctx, sqlc.InsertWorkerConnectionParams{
		AccountID: arg.AccountID, WorkspaceID: arg.WorkspaceID,
		AppName: arg.AppName, AppID: arg.AppID, ID: arg.ID,
		GatewayID: arg.GatewayID, InstanceID: arg.InstanceID,
		Status: int16(arg.Status), WorkerIp: arg.WorkerIp,
		MaxWorkerConcurrency: arg.MaxWorkerConcurrency,
		ConnectedAt:          arg.ConnectedAt, LastHeartbeatAt: arg.LastHeartbeatAt,
		DisconnectedAt: arg.DisconnectedAt, RecordedAt: arg.RecordedAt,
		InsertedAt: arg.InsertedAt, DisconnectReason: arg.DisconnectReason,
		GroupHash: arg.GroupHash, SdkLang: arg.SdkLang, SdkVersion: arg.SdkVersion,
		SdkPlatform: arg.SdkPlatform, SyncID: arg.SyncID, AppVersion: arg.AppVersion,
		FunctionCount: int32(arg.FunctionCount), CpuCores: int32(arg.CpuCores),
		MemBytes: arg.MemBytes, Os: arg.Os,
	})
}

func (pq *pgQuerier) GetWorkerConnection(ctx context.Context, arg db.GetWorkerConnectionParams) (*db.WorkerConnection, error) {
	r, err := pq.q.GetWorkerConnection(ctx, sqlc.GetWorkerConnectionParams{
		AccountID: arg.AccountID, WorkspaceID: arg.WorkspaceID, ConnectionID: arg.ConnectionID,
	})
	if err != nil {
		return nil, err
	}
	return workerConnectionFromPG(r), nil
}
