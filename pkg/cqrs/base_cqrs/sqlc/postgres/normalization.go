package sqlc

import (
	"database/sql"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
)

func (a *App) ToSQLite() (*sqlc.App, error) {
	return &sqlc.App{
		ID:             a.ID,
		Name:           a.Name,
		SdkLanguage:    a.SdkLanguage,
		SdkVersion:     a.SdkVersion,
		Framework:      a.Framework,
		Metadata:       a.Metadata,
		Status:         a.Status,
		Error:          a.Error,
		Checksum:       a.Checksum,
		CreatedAt:      a.CreatedAt,
		ArchivedAt:     a.ArchivedAt,
		Url:            a.Url,
		ConnectionType: a.ConnectionType,
	}, nil
}

func (f *Function) ToSQLite() (*sqlc.Function, error) {
	return &sqlc.Function{
		ID:         f.ID,
		AppID:      f.AppID,
		Name:       f.Name,
		Slug:       f.Slug,
		Config:     f.Config,
		CreatedAt:  f.CreatedAt,
		ArchivedAt: f.ArchivedAt,
	}, nil
}

func (e *Event) ToSQLite() (*sqlc.Event, error) {
	return &sqlc.Event{
		InternalID:  e.InternalID,
		AccountID:   e.AccountID,
		WorkspaceID: e.WorkspaceID,
		Source:      e.Source,
		SourceID:    e.SourceID,
		ReceivedAt:  e.ReceivedAt,
		EventID:     e.EventID,
		EventName:   e.EventName,
		EventData:   e.EventData,
		EventUser:   e.EventUser,
		EventV:      e.EventV,
		EventTs:     e.EventTs,
	}, nil
}

func (b *EventBatch) ToSQLite() (*sqlc.EventBatch, error) {
	return &sqlc.EventBatch{
		ID:          b.ID,
		AccountID:   b.AccountID,
		WorkspaceID: b.WorkspaceID,
		AppID:       b.AppID,
		WorkflowID:  b.WorkflowID,
		RunID:       b.RunID,
		StartedAt:   b.StartedAt,
		ExecutedAt:  b.ExecutedAt,
		EventIds:    b.EventIds,
	}, nil
}

func (r *FunctionRun) ToSQLite() (*sqlc.FunctionRun, error) {
	return &sqlc.FunctionRun{
		RunID:           r.RunID,
		RunStartedAt:    r.RunStartedAt,
		FunctionID:      r.FunctionID,
		FunctionVersion: int64(r.FunctionVersion),
		TriggerType:     r.TriggerType,
		EventID:         r.EventID,
		BatchID:         r.BatchID,
		OriginalRunID:   r.OriginalRunID,
		Cron:            r.Cron,
	}, nil
}

func (f *FunctionFinish) ToSQLite() (*sqlc.FunctionFinish, error) {
	status := sql.NullString{}
	if f.Status != "" {
		status.String = f.Status
		status.Valid = true
	}

	output := sql.NullString{}
	if f.Output != "" {
		output.String = f.Output
		output.Valid = true
	}

	completedStepCount := sql.NullInt64{}
	if f.CompletedStepCount != 0 {
		completedStepCount.Int64 = int64(f.CompletedStepCount)
		completedStepCount.Valid = true
	}

	createdAt := sql.NullTime{}
	if !f.CreatedAt.IsZero() {
		createdAt.Time = f.CreatedAt
		createdAt.Valid = true
	}

	return &sqlc.FunctionFinish{
		RunID:              f.RunID,
		Status:             status,
		Output:             output,
		CompletedStepCount: completedStepCount,
		CreatedAt:          createdAt,
	}, nil
}

func (h *History) ToSQLite() (*sqlc.History, error) {
	latencyMs := sql.NullInt64{}
	if h.LatencyMs.Valid {
		latencyMs.Int64 = int64(h.LatencyMs.Int32)
		latencyMs.Valid = true
	}

	return &sqlc.History{
		ID:                   h.ID,
		CreatedAt:            h.CreatedAt,
		RunStartedAt:         h.RunStartedAt,
		FunctionID:           h.FunctionID,
		FunctionVersion:      int64(h.FunctionVersion),
		RunID:                h.RunID,
		EventID:              h.EventID,
		BatchID:              h.BatchID,
		GroupID:              h.GroupID,
		IdempotencyKey:       h.IdempotencyKey,
		Type:                 h.Type,
		Attempt:              int64(h.Attempt),
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
	}, nil
}

func (t *Trace) ToSQLite() (*sqlc.Trace, error) {
	return &sqlc.Trace{
		Timestamp:          t.Timestamp,
		TimestampUnixMs:    int64(t.TimestampUnixMs),
		TraceID:            t.TraceID,
		SpanID:             t.SpanID,
		ParentSpanID:       t.ParentSpanID,
		TraceState:         t.TraceState,
		SpanName:           t.SpanName,
		SpanKind:           t.SpanKind,
		ServiceName:        t.ServiceName,
		ResourceAttributes: t.ResourceAttributes,
		ScopeName:          t.ScopeName,
		ScopeVersion:       t.ScopeVersion,
		SpanAttributes:     t.SpanAttributes,
		Duration:           int64(t.Duration),
		StatusCode:         t.StatusCode,
		StatusMessage:      t.StatusMessage,
		Events:             t.Events,
		Links:              t.Links,
		RunID:              t.RunID,
	}, nil
}

func (tr *TraceRun) ToSQLite() (*sqlc.TraceRun, error) {
	return &sqlc.TraceRun{
		RunID:        tr.RunID,
		AccountID:    tr.AccountID,
		WorkspaceID:  tr.WorkspaceID,
		AppID:        tr.AppID,
		FunctionID:   tr.FunctionID,
		TraceID:      tr.TraceID,
		QueuedAt:     int64(tr.QueuedAt),
		StartedAt:    int64(tr.StartedAt),
		EndedAt:      int64(tr.EndedAt),
		Status:       int64(tr.Status),
		SourceID:     tr.SourceID,
		TriggerIds:   tr.TriggerIds,
		Output:       tr.Output,
		IsDebounce:   tr.IsDebounce,
		BatchID:      tr.BatchID,
		CronSchedule: tr.CronSchedule,
		HasAi:        tr.HasAi,
	}, nil
}

func (r *GetLatestQueueSnapshotChunksRow) ToSQLite() (*sqlc.GetLatestQueueSnapshotChunksRow, error) {
	return &sqlc.GetLatestQueueSnapshotChunksRow{
		ChunkID: int64(r.ChunkID),
		Data:    r.Data,
	}, nil
}

func (r *GetQueueSnapshotChunksRow) ToSQLite() (*sqlc.GetQueueSnapshotChunksRow, error) {
	return &sqlc.GetQueueSnapshotChunksRow{
		ChunkID: int64(r.ChunkID),
		Data:    r.Data,
	}, nil
}

func (r *GetFunctionRunsFromEventsRow) ToSQLite() (*sqlc.GetFunctionRunsFromEventsRow, error) {
	run, err := r.FunctionRun.ToSQLite()
	if err != nil {
		return nil, err
	}

	pgFinish := FunctionFinish{
		RunID:              r.FunctionRun.RunID,
		Status:             r.FinishStatus,
		Output:             r.FinishOutput,
		CompletedStepCount: r.FinishCompletedStepCount,
		CreatedAt:          r.FinishCreatedAt,
	}
	finish, err := pgFinish.ToSQLite()
	if err != nil {
		return nil, err
	}

	return &sqlc.GetFunctionRunsFromEventsRow{
		FunctionRun:    *run,
		FunctionFinish: *finish,
	}, nil
}

func (r *GetFunctionRunRow) ToSQLite() (*sqlc.GetFunctionRunRow, error) {
	run, err := r.FunctionRun.ToSQLite()
	if err != nil {
		return nil, err
	}

	finish, err := r.FunctionFinish.ToSQLite()
	if err != nil {
		return nil, err
	}

	return &sqlc.GetFunctionRunRow{
		FunctionRun:    *run,
		FunctionFinish: *finish,
	}, nil
}

func (r *GetFunctionRunsTimeboundRow) ToSQLite() (*sqlc.GetFunctionRunsTimeboundRow, error) {
	run, err := r.FunctionRun.ToSQLite()
	if err != nil {
		return nil, err
	}

	finish, err := r.FunctionFinish.ToSQLite()
	if err != nil {
		return nil, err
	}

	return &sqlc.GetFunctionRunsTimeboundRow{
		FunctionRun:    *run,
		FunctionFinish: *finish,
	}, nil
}

func (r *GetFunctionRunsRow) ToSQLite() (*sqlc.GetFunctionRunsRow, error) {
	run, err := r.FunctionRun.ToSQLite()
	if err != nil {
		return nil, err
	}

	finish, err := r.FunctionFinish.ToSQLite()
	if err != nil {
		return nil, err
	}

	return &sqlc.GetFunctionRunsRow{
		FunctionRun:    *run,
		FunctionFinish: *finish,
	}, nil
}

func (wc *WorkerConnection) ToSQLite() (*sqlc.WorkerConnection, error) {
	var lastHeartbeatAt, disconnectedAt sql.NullInt64
	if wc.LastHeartbeatAt.Valid {
		lastHeartbeatAt.Int64 = wc.LastHeartbeatAt.Int64
		lastHeartbeatAt.Valid = true
	}
	if wc.DisconnectedAt.Valid {
		disconnectedAt.Int64 = wc.DisconnectedAt.Int64
		disconnectedAt.Valid = true
	}

	return &sqlc.WorkerConnection{
		AccountID:        wc.AccountID,
		WorkspaceID:      wc.WorkspaceID,
		AppID:            wc.AppID,
		ID:               wc.ID,
		GatewayID:        wc.GatewayID,
		InstanceID:       wc.InstanceID,
		Status:           int64(wc.Status),
		WorkerIp:         wc.WorkerIp,
		ConnectedAt:      wc.ConnectedAt,
		LastHeartbeatAt:  lastHeartbeatAt,
		DisconnectedAt:   disconnectedAt,
		RecordedAt:       wc.RecordedAt,
		InsertedAt:       wc.InsertedAt,
		DisconnectReason: wc.DisconnectReason,
		GroupHash:        wc.GroupHash,
		SdkLang:          wc.SdkLang,
		SdkVersion:       wc.SdkVersion,
		SdkPlatform:      wc.SdkPlatform,
		SyncID:           wc.SyncID,
		BuildID:          wc.BuildID,
		FunctionCount:    int64(wc.FunctionCount),
		CpuCores:         int64(wc.CpuCores),
		MemBytes:         wc.MemBytes,
		Os:               wc.Os,
	}, nil
}
