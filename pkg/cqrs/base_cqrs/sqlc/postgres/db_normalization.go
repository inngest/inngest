package sqlc

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	sqlc_sqlite "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/oklog/ulid/v2"
)

type NewNormalizedOpts struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxIdle     int
	ConnMaxLifetime int
}

func NewNormalized(db DBTX, o NewNormalizedOpts) sqlc_sqlite.Querier {
	if sqlDB, ok := db.(*sql.DB); ok {
		sqlDB.SetMaxIdleConns(o.MaxIdleConns)
		sqlDB.SetMaxOpenConns(o.MaxOpenConns)
		sqlDB.SetConnMaxIdleTime(time.Duration(o.ConnMaxIdle) * time.Minute)
		sqlDB.SetConnMaxLifetime(time.Duration(o.ConnMaxLifetime) * time.Minute)
	}

	return &NormalizedQueries{db: New(db)}
}

type NormalizedQueries struct {
	db *Queries
}

func (q NormalizedQueries) GetWorkerConnection(ctx context.Context, arg sqlc_sqlite.GetWorkerConnectionParams) (*sqlc_sqlite.WorkerConnection, error) {
	wc, err := q.db.GetWorkerConnection(ctx, GetWorkerConnectionParams{
		AccountID:    arg.AccountID,
		WorkspaceID:  arg.WorkspaceID,
		ConnectionID: arg.ConnectionID,
	})
	if err != nil {
		return nil, err
	}

	return wc.ToSQLite()
}

func (q NormalizedQueries) InsertWorkerConnection(ctx context.Context, arg sqlc_sqlite.InsertWorkerConnectionParams) error {
	err := q.db.InsertWorkerConnection(ctx, InsertWorkerConnectionParams{
		AccountID:            arg.AccountID,
		WorkspaceID:          arg.WorkspaceID,
		AppName:              arg.AppName,
		AppID:                arg.AppID,
		ID:                   arg.ID,
		GatewayID:            arg.GatewayID,
		InstanceID:           arg.InstanceID,
		Status:               int16(arg.Status),
		WorkerIp:             arg.WorkerIp,
		MaxWorkerConcurrency: arg.MaxWorkerConcurrency,
		ConnectedAt:          arg.ConnectedAt,
		LastHeartbeatAt:      arg.LastHeartbeatAt,
		DisconnectedAt:       arg.DisconnectedAt,
		RecordedAt:           arg.RecordedAt,
		InsertedAt:           arg.InsertedAt,
		DisconnectReason:     arg.DisconnectReason,
		GroupHash:            arg.GroupHash,
		SdkLang:              arg.SdkLang,
		SdkVersion:           arg.SdkVersion,
		SdkPlatform:          arg.SdkPlatform,
		SyncID:               arg.SyncID,
		AppVersion:           arg.AppVersion,
		FunctionCount:        int32(arg.FunctionCount),
		CpuCores:             int32(arg.CpuCores),
		MemBytes:             arg.MemBytes,
		Os:                   arg.Os,
	})
	if err != nil {
		return err
	}

	return nil
}

func (q NormalizedQueries) UpdateAppError(ctx context.Context, arg sqlc_sqlite.UpdateAppErrorParams) (*sqlc_sqlite.App, error) {
	pgParams := UpdateAppErrorParams{
		ID:    arg.ID,
		Error: arg.Error,
	}

	app, err := q.db.UpdateAppError(ctx, pgParams)
	if err != nil {
		return nil, err
	}

	return app.ToSQLite()
}

func (q NormalizedQueries) UpdateAppURL(ctx context.Context, arg sqlc_sqlite.UpdateAppURLParams) (*sqlc_sqlite.App, error) {
	pgParams := UpdateAppURLParams{
		ID:  arg.ID,
		Url: arg.Url,
	}

	app, err := q.db.UpdateAppURL(ctx, pgParams)
	if err != nil {
		return nil, err
	}

	return app.ToSQLite()
}

func (q NormalizedQueries) GetAppByName(ctx context.Context, name string) (*sqlc_sqlite.App, error) {
	app, err := q.db.GetAppByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return app.ToSQLite()
}

func (q NormalizedQueries) GetLatestQueueSnapshotChunks(ctx context.Context) ([]*sqlc_sqlite.GetLatestQueueSnapshotChunksRow, error) {
	rows, err := q.db.GetLatestQueueSnapshotChunks(ctx)
	if err != nil {
		return nil, err
	}

	sqliteRows := make([]*sqlc_sqlite.GetLatestQueueSnapshotChunksRow, len(rows))
	for i, row := range rows {
		sqliteRows[i], _ = row.ToSQLite()
	}

	return sqliteRows, nil
}

func (q NormalizedQueries) GetQueueSnapshotChunks(ctx context.Context, snapshotID interface{}) ([]*sqlc_sqlite.GetQueueSnapshotChunksRow, error) {
	snapshotIDStr, ok := snapshotID.(string)
	if !ok {
		return nil, fmt.Errorf("snapshotID must be a string")
	}

	rows, err := q.db.GetQueueSnapshotChunks(ctx, snapshotIDStr)
	if err != nil {
		return nil, err
	}

	sqliteRows := make([]*sqlc_sqlite.GetQueueSnapshotChunksRow, len(rows))
	for i, row := range rows {
		sqliteRows[i], _ = row.ToSQLite()
	}

	return sqliteRows, nil

}

func (q NormalizedQueries) DeleteOldQueueSnapshots(ctx context.Context, limit int64) (int64, error) {
	return q.db.DeleteOldQueueSnapshots(ctx, int32(limit))
}

func (q NormalizedQueries) InsertQueueSnapshotChunk(ctx context.Context, params sqlc_sqlite.InsertQueueSnapshotChunkParams) error {
	snapshotIDStr, ok := params.SnapshotID.(string)
	if !ok {
		return fmt.Errorf("snapshot ID must be a string")
	}

	return q.db.InsertQueueSnapshotChunk(ctx, InsertQueueSnapshotChunkParams{
		SnapshotID: snapshotIDStr,
		ChunkID:    int32(params.ChunkID),
		Data:       params.Data,
	})
}

func (q NormalizedQueries) GetApps(ctx context.Context) ([]*sqlc_sqlite.App, error) {
	apps, err := q.db.GetApps(ctx)
	if err != nil {
		return nil, err
	}

	sqliteApps := make([]*sqlc_sqlite.App, len(apps))
	for i, app := range apps {
		sqliteApps[i], _ = app.ToSQLite()
	}

	return sqliteApps, nil
}

func (q NormalizedQueries) GetAppByChecksum(ctx context.Context, checksum string) (*sqlc_sqlite.App, error) {
	app, err := q.db.GetAppByChecksum(ctx, checksum)
	if err != nil {
		return nil, err
	}

	return app.ToSQLite()
}

func (q NormalizedQueries) GetAppByID(ctx context.Context, id uuid.UUID) (*sqlc_sqlite.App, error) {
	app, err := q.db.GetAppByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return app.ToSQLite()
}

func (q NormalizedQueries) GetAppByURL(ctx context.Context, url string) (*sqlc_sqlite.App, error) {
	app, err := q.db.GetAppByURL(ctx, url)
	if err != nil {
		return nil, err
	}

	return app.ToSQLite()
}

func (q NormalizedQueries) GetAllApps(ctx context.Context) ([]*sqlc_sqlite.App, error) {
	apps, err := q.db.GetAllApps(ctx)
	if err != nil {
		return nil, err
	}

	sqliteApps := make([]*sqlc_sqlite.App, len(apps))
	for i, app := range apps {
		sqliteApps[i], _ = app.ToSQLite()
	}

	return sqliteApps, nil
}

func (q NormalizedQueries) UpsertApp(ctx context.Context, params sqlc_sqlite.UpsertAppParams) (*sqlc_sqlite.App, error) {
	pgParams := UpsertAppParams{
		ID:          params.ID,
		Name:        params.Name,
		SdkLanguage: params.SdkLanguage,
		SdkVersion:  params.SdkVersion,
		Framework:   params.Framework,
		Metadata:    params.Metadata,
		Status:      params.Status,
		Error:       params.Error,
		Checksum:    params.Checksum,
		Url:         params.Url,
		Method:      params.Method,
		AppVersion:  params.AppVersion,
	}

	app, err := q.db.UpsertApp(ctx, pgParams)
	if err != nil {
		return nil, err
	}

	return app.ToSQLite()
}

func (q NormalizedQueries) GetAppFunctions(ctx context.Context, appID uuid.UUID) ([]*sqlc_sqlite.Function, error) {
	functions, err := q.db.GetAppFunctions(ctx, appID)
	if err != nil {
		return nil, err
	}

	sqliteFunctions := make([]*sqlc_sqlite.Function, len(functions))
	for i, function := range functions {
		sqliteFunctions[i], _ = function.ToSQLite()
	}

	return sqliteFunctions, nil
}

func (q NormalizedQueries) DeleteApp(ctx context.Context, id uuid.UUID) error {
	return q.db.DeleteApp(ctx, id)
}

func (q NormalizedQueries) GetApp(ctx context.Context, id uuid.UUID) (*sqlc_sqlite.App, error) {
	app, err := q.db.GetApp(ctx, id)
	if err != nil {
		return nil, err
	}

	return app.ToSQLite()
}

func (q NormalizedQueries) GetFunctionBySlug(ctx context.Context, slug string) (*sqlc_sqlite.Function, error) {
	function, err := q.db.GetFunctionBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return function.ToSQLite()
}

func (q NormalizedQueries) GetFunctionByID(ctx context.Context, id uuid.UUID) (*sqlc_sqlite.Function, error) {
	function, err := q.db.GetFunctionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return function.ToSQLite()
}

func (q NormalizedQueries) GetFunctions(ctx context.Context) ([]*sqlc_sqlite.Function, error) {
	functions, err := q.db.GetFunctions(ctx)
	if err != nil {
		return nil, err
	}

	sqliteFunctions := make([]*sqlc_sqlite.Function, len(functions))
	for i, function := range functions {
		sqliteFunctions[i], _ = function.ToSQLite()
	}

	return sqliteFunctions, nil
}

func (q NormalizedQueries) GetAppFunctionsBySlug(ctx context.Context, slug string) ([]*sqlc_sqlite.Function, error) {
	functions, err := q.db.GetAppFunctionsBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	sqliteFunctions := make([]*sqlc_sqlite.Function, len(functions))
	for i, function := range functions {
		sqliteFunctions[i], _ = function.ToSQLite()
	}

	return sqliteFunctions, nil
}

func (q NormalizedQueries) InsertFunction(ctx context.Context, params sqlc_sqlite.InsertFunctionParams) (*sqlc_sqlite.Function, error) {
	pgParams := InsertFunctionParams{
		ID:        params.ID,
		AppID:     params.AppID,
		Name:      params.Name,
		Slug:      params.Slug,
		Config:    params.Config,
		CreatedAt: params.CreatedAt,
	}

	function, err := q.db.InsertFunction(ctx, pgParams)
	if err != nil {
		return nil, err
	}

	return function.ToSQLite()
}

func (q NormalizedQueries) DeleteFunctionsByAppID(ctx context.Context, appID uuid.UUID) error {
	return q.db.DeleteFunctionsByAppID(ctx, appID)
}

func (q NormalizedQueries) DeleteFunctionsByIDs(ctx context.Context, ids []uuid.UUID) error {
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.String()
	}
	return q.db.DeleteFunctionsByIDs(ctx, strIDs)
}

func (q NormalizedQueries) UpdateFunctionConfig(ctx context.Context, arg sqlc_sqlite.UpdateFunctionConfigParams) (*sqlc_sqlite.Function, error) {
	pgParams := UpdateFunctionConfigParams{
		Config: arg.Config,
		ID:     arg.ID,
	}

	function, err := q.db.UpdateFunctionConfig(ctx, pgParams)
	if err != nil {
		return nil, err
	}

	return function.ToSQLite()
}

func (q NormalizedQueries) InsertEvent(ctx context.Context, e sqlc_sqlite.InsertEventParams) error {
	pgParams := InsertEventParams{
		InternalID: e.InternalID,
		ReceivedAt: e.ReceivedAt,
		EventID:    e.EventID,
		EventName:  e.EventName,
		EventData:  e.EventData,
		EventUser:  e.EventUser,
		EventV:     e.EventV,
		EventTs:    e.EventTs,
	}

	return q.db.InsertEvent(ctx, pgParams)
}

func (q NormalizedQueries) InsertEventBatch(ctx context.Context, eb sqlc_sqlite.InsertEventBatchParams) error {
	pgParams := InsertEventBatchParams{
		ID:          eb.ID,
		AccountID:   eb.AccountID,
		WorkspaceID: eb.WorkspaceID,
		AppID:       eb.AppID,
		WorkflowID:  eb.WorkflowID,
		RunID:       eb.RunID,
		StartedAt:   eb.StartedAt,
		ExecutedAt:  eb.ExecutedAt,
		EventIds:    eb.EventIds,
	}

	return q.db.InsertEventBatch(ctx, pgParams)
}

func (q NormalizedQueries) GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*sqlc_sqlite.Event, error) {
	event, err := q.db.GetEventByInternalID(ctx, internalID)
	if err != nil {
		return nil, err
	}

	return event.ToSQLite()
}

func (q NormalizedQueries) GetEventBatchesByEventID(ctx context.Context, eventID string) ([]*sqlc_sqlite.EventBatch, error) {
	batches, err := q.db.GetEventBatchesByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	sqliteBatches := make([]*sqlc_sqlite.EventBatch, len(batches))
	for i, batch := range batches {
		sqliteBatches[i], _ = batch.ToSQLite()
	}

	return sqliteBatches, nil
}

func (q NormalizedQueries) GetEventBatchByRunID(ctx context.Context, runID ulid.ULID) (*sqlc_sqlite.EventBatch, error) {
	batch, err := q.db.GetEventBatchByRunID(ctx, runID.String())
	if err != nil {
		return nil, err
	}

	return batch.ToSQLite()
}

func (q NormalizedQueries) GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*sqlc_sqlite.Event, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	bytIDs := make([][]byte, len(ids))
	for i, id := range ids {
		bytIDs[i] = id.Bytes()
	}

	events, err := q.db.GetEventsByInternalIDs(ctx, bytIDs)
	if err != nil {
		return nil, err
	}

	sqliteEvents := make([]*sqlc_sqlite.Event, len(events))
	for i, event := range events {
		sqliteEvents[i], _ = event.ToSQLite()
	}

	return sqliteEvents, nil
}

func (q NormalizedQueries) GetEventsIDbound(ctx context.Context, params sqlc_sqlite.GetEventsIDboundParams) ([]*sqlc_sqlite.Event, error) {
	pgParams := GetEventsIDboundParams{
		InternalID:   params.After,
		InternalID_2: params.Before,
		EventName:    params.IncludeInternal,
		Limit:        int32(params.Limit),
	}

	events, err := q.db.GetEventsIDbound(ctx, pgParams)
	if err != nil {
		return nil, err
	}

	sqliteEvents := make([]*sqlc_sqlite.Event, len(events))
	for i, event := range events {
		sqliteEvents[i], _ = event.ToSQLite()
	}

	return sqliteEvents, nil
}

func (q NormalizedQueries) InsertFunctionRun(ctx context.Context, e sqlc_sqlite.InsertFunctionRunParams) error {
	pgParams := InsertFunctionRunParams{
		RunID:           e.RunID,
		RunStartedAt:    e.RunStartedAt,
		FunctionID:      e.FunctionID,
		FunctionVersion: int32(e.FunctionVersion),
		TriggerType:     e.TriggerType,
		EventID:         e.EventID,
		BatchID:         e.BatchID,
		OriginalRunID:   e.OriginalRunID,
		Cron:            e.Cron,
	}

	return q.db.InsertFunctionRun(ctx, pgParams)
}

func (q NormalizedQueries) GetFunctionRunsFromEvents(ctx context.Context, eventIDs []ulid.ULID) ([]*sqlc_sqlite.GetFunctionRunsFromEventsRow, error) {
	bytEventIDs := make([][]byte, len(eventIDs))
	for i, id := range eventIDs {
		bytEventIDs[i] = id.Bytes()
	}

	rows, err := q.db.GetFunctionRunsFromEvents(ctx, bytEventIDs)
	if err != nil {
		return nil, err
	}

	sqliteRows := make([]*sqlc_sqlite.GetFunctionRunsFromEventsRow, len(rows))
	for i, row := range rows {
		sqliteRows[i], _ = row.ToSQLite()
	}

	return sqliteRows, nil
}

func (q NormalizedQueries) GetFunctionRun(ctx context.Context, id ulid.ULID) (*sqlc_sqlite.GetFunctionRunRow, error) {
	row, err := q.db.GetFunctionRun(ctx, id)
	if err != nil {
		return nil, err
	}

	return row.ToSQLite()
}

func (q NormalizedQueries) GetFunctionRunsTimebound(ctx context.Context, params sqlc_sqlite.GetFunctionRunsTimeboundParams) ([]*sqlc_sqlite.GetFunctionRunsTimeboundRow, error) {
	pgParams := GetFunctionRunsTimeboundParams{}

	rows, err := q.db.GetFunctionRunsTimebound(ctx, pgParams)
	if err != nil {
		return nil, err
	}

	sqliteRows := make([]*sqlc_sqlite.GetFunctionRunsTimeboundRow, len(rows))
	for i, row := range rows {
		sqliteRows[i], _ = row.ToSQLite()
	}

	return sqliteRows, nil
}

func (q NormalizedQueries) GetFunctionRunFinishesByRunIDs(ctx context.Context, runIDs []ulid.ULID) ([]*sqlc_sqlite.FunctionFinish, error) {
	finishes, err := q.db.GetFunctionRunFinishesByRunIDs(ctx, runIDs)
	if err != nil {
		return nil, err
	}

	sqliteFinishes := make([]*sqlc_sqlite.FunctionFinish, len(finishes))
	for i, finish := range finishes {
		sqliteFinishes[i], _ = finish.ToSQLite()
	}

	return sqliteFinishes, nil
}

func (q NormalizedQueries) InsertHistory(ctx context.Context, h sqlc_sqlite.InsertHistoryParams) error {
	latencyMs := sql.NullInt32{}
	if h.LatencyMs.Valid {
		latencyMs.Int32 = int32(h.LatencyMs.Int64)
		latencyMs.Valid = true
	}

	pgParams := InsertHistoryParams{
		ID:                   h.ID,
		CreatedAt:            h.CreatedAt,
		RunStartedAt:         h.RunStartedAt,
		FunctionID:           h.FunctionID,
		FunctionVersion:      int32(h.FunctionVersion),
		RunID:                h.RunID,
		EventID:              h.EventID,
		BatchID:              h.BatchID,
		GroupID:              h.GroupID,
		IdempotencyKey:       h.IdempotencyKey,
		Type:                 h.Type,
		Attempt:              int32(h.Attempt),
		LatencyMs:            latencyMs,
		StepName:             h.StepName,
		StepID:               h.StepID,
		StepType:             h.StepType,
		Url:                  h.Url,
		CancelRequest:        h.CancelRequest,
		Sleep:                h.Sleep,
		WaitForEvent:         h.WaitForEvent,
		WaitResult:           h.WaitResult,
		InvokeFunction:       h.InvokeFunction,
		InvokeFunctionResult: h.InvokeFunctionResult,
		Result:               h.Result,
	}

	return q.db.InsertHistory(ctx, pgParams)
}

func (q NormalizedQueries) GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*sqlc_sqlite.History, error) {
	history, err := q.db.GetFunctionRunHistory(ctx, runID)
	if err != nil {
		return nil, err
	}

	sqliteHistory := make([]*sqlc_sqlite.History, len(history))
	for i, h := range history {
		sqliteHistory[i], _ = h.ToSQLite()
	}

	return sqliteHistory, nil
}

func (q NormalizedQueries) InsertTrace(ctx context.Context, span sqlc_sqlite.InsertTraceParams) error {
	pgSpan := InsertTraceParams{
		Timestamp:          span.Timestamp,
		TimestampUnixMs:    span.TimestampUnixMs,
		TraceID:            span.TraceID,
		SpanID:             span.SpanID,
		ParentSpanID:       span.ParentSpanID,
		TraceState:         span.TraceState,
		SpanName:           span.SpanName,
		SpanKind:           span.SpanKind,
		ServiceName:        span.ServiceName,
		ResourceAttributes: span.ResourceAttributes,
		ScopeName:          span.ScopeName,
		ScopeVersion:       span.ScopeVersion,
		SpanAttributes:     span.SpanAttributes,
		Duration:           int32(span.Duration),
		StatusCode:         span.StatusCode,
		StatusMessage:      span.StatusMessage,
		Events:             span.Events,
		Links:              span.Links,
		RunID:              span.RunID.String(),
	}

	return q.db.InsertTrace(ctx, pgSpan)
}

func (q NormalizedQueries) InsertTraceRun(ctx context.Context, span sqlc_sqlite.InsertTraceRunParams) error {
	pgSpan := InsertTraceRunParams{
		AccountID:    span.AccountID,
		WorkspaceID:  span.WorkspaceID,
		AppID:        span.AppID,
		FunctionID:   span.FunctionID,
		TraceID:      span.TraceID,
		RunID:        span.RunID.String(),
		QueuedAt:     span.QueuedAt,
		StartedAt:    span.StartedAt,
		EndedAt:      span.EndedAt,
		Status:       int32(span.Status),
		SourceID:     span.SourceID,
		TriggerIds:   span.TriggerIds,
		Output:       span.Output,
		BatchID:      span.BatchID.Bytes(),
		IsDebounce:   span.IsDebounce,
		CronSchedule: span.CronSchedule,
		HasAi:        span.HasAi,
	}

	return q.db.InsertTraceRun(ctx, pgSpan)
}

func (q NormalizedQueries) GetTraceSpans(ctx context.Context, arg sqlc_sqlite.GetTraceSpansParams) ([]*sqlc_sqlite.Trace, error) {
	pgArg := GetTraceSpansParams{
		TraceID: arg.TraceID,
		RunID:   arg.RunID.String(),
	}

	traces, err := q.db.GetTraceSpans(ctx, pgArg)
	if err != nil {
		return nil, err
	}

	sqliteTraces := make([]*sqlc_sqlite.Trace, len(traces))
	for i, trace := range traces {
		sqliteTraces[i], _ = trace.ToSQLite()
	}

	return sqliteTraces, nil
}

func (q NormalizedQueries) GetTraceRun(ctx context.Context, runID ulid.ULID) (*sqlc_sqlite.TraceRun, error) {
	traceRun, err := q.db.GetTraceRun(ctx, runID.String())
	if err != nil {
		return nil, err
	}

	return traceRun.ToSQLite()
}

func (q NormalizedQueries) GetTraceSpanOutput(ctx context.Context, arg sqlc_sqlite.GetTraceSpanOutputParams) ([]*sqlc_sqlite.Trace, error) {
	pgArg := GetTraceSpanOutputParams{
		TraceID: arg.TraceID,
		SpanID:  arg.SpanID,
	}

	traces, err := q.db.GetTraceSpanOutput(ctx, pgArg)
	if err != nil {
		return nil, err
	}

	sqliteTraces := make([]*sqlc_sqlite.Trace, len(traces))
	for i, trace := range traces {
		sqliteTraces[i], _ = trace.ToSQLite()
	}

	return sqliteTraces, nil
}

func (q NormalizedQueries) GetTraceRunsByTriggerId(ctx context.Context, eventID string) ([]*sqlc_sqlite.TraceRun, error) {
	traces, err := q.db.GetTraceRunsByTriggerId(ctx, eventID)
	if err != nil {
		return nil, err
	}

	sqliteTraces := make([]*sqlc_sqlite.TraceRun, len(traces))
	for i, trace := range traces {
		sqliteTraces[i], _ = trace.ToSQLite()
	}

	return sqliteTraces, nil
}

func (q NormalizedQueries) InsertFunctionFinish(ctx context.Context, arg sqlc_sqlite.InsertFunctionFinishParams) error {
	var completedStepCount int32
	if arg.CompletedStepCount.Valid {
		completedStepCount = int32(arg.CompletedStepCount.Int64)
	}

	pgArg := InsertFunctionFinishParams{
		RunID:              arg.RunID,
		Status:             arg.Status.String,
		Output:             arg.Output.String,
		CompletedStepCount: completedStepCount,
		CreatedAt:          arg.CreatedAt.Time,
	}

	return q.db.InsertFunctionFinish(ctx, pgArg)
}

func (q NormalizedQueries) HistoryCountRuns(ctx context.Context) (int64, error) {
	return q.db.HistoryCountRuns(ctx)
}

func (q NormalizedQueries) GetHistoryItem(ctx context.Context, id ulid.ULID) (*sqlc_sqlite.History, error) {
	history, err := q.db.GetHistoryItem(ctx, id)
	if err != nil {
		return nil, err
	}

	return history.ToSQLite()
}

func (q NormalizedQueries) GetFunctionRuns(ctx context.Context) ([]*sqlc_sqlite.GetFunctionRunsRow, error) {
	rows, err := q.db.GetFunctionRuns(ctx)
	if err != nil {
		return nil, err
	}

	sqliteRows := make([]*sqlc_sqlite.GetFunctionRunsRow, len(rows))
	for i, row := range rows {
		sqliteRows[i], _ = row.ToSQLite()
	}

	return sqliteRows, nil
}

func (q NormalizedQueries) GetSpansByRunID(ctx context.Context, runID string) ([]*sqlc_sqlite.GetSpansByRunIDRow, error) {
	rows, err := q.db.GetSpansByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}

	sqliteRows := make([]*sqlc_sqlite.GetSpansByRunIDRow, len(rows))
	for i, row := range rows {
		sqliteRows[i], _ = row.ToSQLite()
	}

	return sqliteRows, nil
}

func (q NormalizedQueries) GetSpansByDebugRunID(ctx context.Context, debugRunID sql.NullString) ([]*sqlc_sqlite.GetSpansByDebugRunIDRow, error) {
	rows, err := q.db.GetSpansByDebugRunID(ctx, debugRunID.String)
	if err != nil {
		return nil, err
	}

	sqliteRows := make([]*sqlc_sqlite.GetSpansByDebugRunIDRow, len(rows))
	for i, row := range rows {
		sqliteRows[i], _ = row.ToSQLite()
	}

	return sqliteRows, nil
}

func (q NormalizedQueries) GetSpansByDebugSessionID(ctx context.Context, debugSessionID sql.NullString) ([]*sqlc_sqlite.GetSpansByDebugSessionIDRow, error) {
	rows, err := q.db.GetSpansByDebugSessionID(ctx, debugSessionID.String)
	if err != nil {
		return nil, err
	}

	sqliteRows := make([]*sqlc_sqlite.GetSpansByDebugSessionIDRow, len(rows))
	for i, row := range rows {
		sqliteRows[i], _ = row.ToSQLite()
	}

	return sqliteRows, nil
}

func (q NormalizedQueries) GetRunSpanByRunID(ctx context.Context, args sqlc_sqlite.GetRunSpanByRunIDParams) (*sqlc_sqlite.GetRunSpanByRunIDRow, error) {
	row, err := q.db.GetRunSpanByRunID(ctx, GetRunSpanByRunIDParams(args))
	if err != nil {
		return nil, err
	}

	return row.ToSQLite()
}

func (q NormalizedQueries) GetStepSpanByStepID(ctx context.Context, args sqlc_sqlite.GetStepSpanByStepIDParams) (*sqlc_sqlite.GetStepSpanByStepIDRow, error) {
	row, err := q.db.GetStepSpanByStepID(ctx, GetStepSpanByStepIDParams(args))
	if err != nil {
		return nil, err
	}

	return row.ToSQLite()
}

func (q NormalizedQueries) GetExecutionSpanByStepIDAndAttempt(ctx context.Context, args sqlc_sqlite.GetExecutionSpanByStepIDAndAttemptParams) (*sqlc_sqlite.GetExecutionSpanByStepIDAndAttemptRow, error) {
	row, err := q.db.GetExecutionSpanByStepIDAndAttempt(ctx, GetExecutionSpanByStepIDAndAttemptParams(args))
	if err != nil {
		return nil, err
	}

	return row.ToSQLite()
}

func (q NormalizedQueries) GetLatestExecutionSpanByStepID(ctx context.Context, args sqlc_sqlite.GetLatestExecutionSpanByStepIDParams) (*sqlc_sqlite.GetLatestExecutionSpanByStepIDRow, error) {
	row, err := q.db.GetLatestExecutionSpanByStepID(ctx, GetLatestExecutionSpanByStepIDParams(args))
	if err != nil {
		return nil, err
	}

	return row.ToSQLite()
}

func (q NormalizedQueries) GetSpanBySpanID(ctx context.Context, args sqlc_sqlite.GetSpanBySpanIDParams) (*sqlc_sqlite.GetSpanBySpanIDRow, error) {
	row, err := q.db.GetSpanBySpanID(ctx, GetSpanBySpanIDParams(args))
	if err != nil {
		return nil, err
	}

	return row.ToSQLite()
}

func (q NormalizedQueries) GetSpanOutput(ctx context.Context, arg sqlc_sqlite.GetSpanOutputParams) ([]*sqlc_sqlite.GetSpanOutputRow, error) {
	rows, err := q.db.GetSpanOutput(ctx, arg.Ids)
	if err != nil {
		return nil, err
	}

	sqliteRows := make([]*sqlc_sqlite.GetSpanOutputRow, len(rows))
	for i, row := range rows {
		sqliteRows[i], _ = row.ToSQLite()
	}

	return sqliteRows, nil
}

func (q NormalizedQueries) InsertSpan(ctx context.Context, arg sqlc_sqlite.InsertSpanParams) error {
	pgArg := InsertSpanParams{
		AccountID:      arg.AccountID,
		AppID:          arg.AppID,
		Attributes:     toNullRawMessage(arg.Attributes),
		DynamicSpanID:  arg.DynamicSpanID,
		EndTime:        arg.EndTime,
		EnvID:          arg.EnvID,
		FunctionID:     arg.FunctionID,
		Links:          toNullRawMessage(arg.Links),
		Name:           arg.Name,
		Output:         toNullRawMessage(arg.Output),
		ParentSpanID:   arg.ParentSpanID,
		RunID:          arg.RunID,
		SpanID:         arg.SpanID,
		StartTime:      arg.StartTime,
		TraceID:        arg.TraceID,
		Input:          toNullRawMessage(arg.Input),
		DebugRunID:     arg.DebugRunID,
		DebugSessionID: arg.DebugSessionID,
		Status:         arg.Status,
		EventIds:       toNullRawMessage(arg.EventIds),
	}

	return q.db.InsertSpan(ctx, pgArg)
}
