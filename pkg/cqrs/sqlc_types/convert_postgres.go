package sqlc_types

import (
	sqlc_pg "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
)

// Postgres conversion functions: convert sqlc-generated Postgres types to domain types.

func AppFromPostgres(s *sqlc_pg.App) *App {
	return &App{
		ID:          s.ID,
		Name:        s.Name,
		SdkLanguage: s.SdkLanguage,
		SdkVersion:  s.SdkVersion,
		Framework:   s.Framework,
		Metadata:    s.Metadata,
		Status:      s.Status,
		Error:       s.Error,
		Checksum:    s.Checksum,
		CreatedAt:   s.CreatedAt,
		ArchivedAt:  s.ArchivedAt,
		Url:         s.Url,
		Method:      s.Method,
		AppVersion:  s.AppVersion,
	}
}

func FunctionFromPostgres(s *sqlc_pg.Function) *Function {
	return &Function{
		ID:         s.ID,
		AppID:      s.AppID,
		Name:       s.Name,
		Slug:       s.Slug,
		Config:     s.Config,
		CreatedAt:  s.CreatedAt,
		ArchivedAt: s.ArchivedAt,
	}
}

func EventFromPostgres(s *sqlc_pg.Event) *Event {
	return &Event{
		InternalID:  s.InternalID,
		AccountID:   s.AccountID,
		WorkspaceID: s.WorkspaceID,
		Source:      s.Source,
		SourceID:    s.SourceID,
		ReceivedAt:  s.ReceivedAt,
		EventID:     s.EventID,
		EventName:   s.EventName,
		EventData:   s.EventData,
		EventUser:   s.EventUser,
		EventV:      s.EventV,
		EventTs:     s.EventTs,
	}
}

func EventBatchFromPostgres(s *sqlc_pg.EventBatch) *EventBatch {
	return &EventBatch{
		ID:          s.ID,
		AccountID:   s.AccountID,
		WorkspaceID: s.WorkspaceID,
		AppID:       s.AppID,
		WorkflowID:  s.WorkflowID,
		RunID:       s.RunID,
		StartedAt:   s.StartedAt,
		ExecutedAt:  s.ExecutedAt,
		EventIds:    s.EventIds,
	}
}

func FunctionFinishFromPostgres(s *sqlc_pg.FunctionFinish) *FunctionFinish {
	return &FunctionFinish{
		RunID:              s.RunID,
		Status:             toNullStringFromString(s.Status),
		Output:             toNullStringFromString(s.Output),
		CompletedStepCount: toNullInt64FromInt32(s.CompletedStepCount),
		CreatedAt:          toNullTimeFromTime(s.CreatedAt),
	}
}

func FunctionRunFromPostgres(s *sqlc_pg.FunctionRun) *FunctionRun {
	return &FunctionRun{
		RunID:           s.RunID,
		RunStartedAt:    s.RunStartedAt,
		FunctionID:      s.FunctionID,
		FunctionVersion: int64(s.FunctionVersion),
		TriggerType:     s.TriggerType,
		EventID:         s.EventID,
		BatchID:         s.BatchID,
		OriginalRunID:   s.OriginalRunID,
		Cron:            s.Cron,
	}
}

func HistoryFromPostgres(s *sqlc_pg.History) *History {
	return &History{
		ID:                   s.ID,
		CreatedAt:            s.CreatedAt,
		RunStartedAt:         s.RunStartedAt,
		FunctionID:           s.FunctionID,
		FunctionVersion:      int64(s.FunctionVersion),
		RunID:                s.RunID,
		EventID:              s.EventID,
		BatchID:              s.BatchID,
		GroupID:              s.GroupID,
		IdempotencyKey:       s.IdempotencyKey,
		Type:                 s.Type,
		Attempt:              int64(s.Attempt),
		LatencyMs:            toNullInt64FromNullInt32(s.LatencyMs),
		StepName:             s.StepName,
		StepID:               s.StepID,
		StepType:             s.StepType,
		Url:                  s.Url,
		CancelRequest:        s.CancelRequest,
		Sleep:                s.Sleep,
		WaitForEvent:         s.WaitForEvent,
		WaitResult:           s.WaitResult,
		InvokeFunction:       s.InvokeFunction,
		InvokeFunctionResult: s.InvokeFunctionResult,
		Result:               s.Result,
	}
}

func TraceRunFromPostgres(s *sqlc_pg.TraceRun) *TraceRun {
	return &TraceRun{
		RunID:        s.RunID,
		AccountID:    s.AccountID,
		WorkspaceID:  s.WorkspaceID,
		AppID:        s.AppID,
		FunctionID:   s.FunctionID,
		TraceID:      s.TraceID,
		QueuedAt:     s.QueuedAt,
		StartedAt:    s.StartedAt,
		EndedAt:      s.EndedAt,
		Status:       int64(s.Status),
		SourceID:     s.SourceID,
		TriggerIds:   s.TriggerIds,
		Output:       s.Output,
		IsDebounce:   s.IsDebounce,
		BatchID:      s.BatchID,
		CronSchedule: s.CronSchedule,
		HasAi:        s.HasAi,
	}
}

func TraceFromPostgres(s *sqlc_pg.Trace) *Trace {
	return &Trace{
		Timestamp:          s.Timestamp,
		TimestampUnixMs:    s.TimestampUnixMs,
		TraceID:            s.TraceID,
		SpanID:             s.SpanID,
		ParentSpanID:       s.ParentSpanID,
		TraceState:         s.TraceState,
		SpanName:           s.SpanName,
		SpanKind:           s.SpanKind,
		ServiceName:        s.ServiceName,
		ResourceAttributes: s.ResourceAttributes,
		ScopeName:          s.ScopeName,
		ScopeVersion:       s.ScopeVersion,
		SpanAttributes:     s.SpanAttributes,
		Duration:           int64(s.Duration),
		StatusCode:         s.StatusCode,
		StatusMessage:      s.StatusMessage,
		Events:             s.Events,
		Links:              s.Links,
		RunID:              s.RunID,
	}
}

func WorkerConnectionFromPostgres(s *sqlc_pg.WorkerConnection) *WorkerConnection {
	return &WorkerConnection{
		AccountID:            s.AccountID,
		WorkspaceID:          s.WorkspaceID,
		AppName:              s.AppName,
		AppID:                s.AppID,
		ID:                   s.ID,
		GatewayID:            s.GatewayID,
		InstanceID:           s.InstanceID,
		Status:               int64(s.Status),
		WorkerIp:             s.WorkerIp,
		MaxWorkerConcurrency: s.MaxWorkerConcurrency,
		ConnectedAt:          s.ConnectedAt,
		LastHeartbeatAt:      s.LastHeartbeatAt,
		DisconnectedAt:       s.DisconnectedAt,
		RecordedAt:           s.RecordedAt,
		InsertedAt:           s.InsertedAt,
		DisconnectReason:     s.DisconnectReason,
		GroupHash:            s.GroupHash,
		SdkLang:              s.SdkLang,
		SdkVersion:           s.SdkVersion,
		SdkPlatform:          s.SdkPlatform,
		SyncID:               s.SyncID,
		AppVersion:           s.AppVersion,
		FunctionCount:        int64(s.FunctionCount),
		CpuCores:             int64(s.CpuCores),
		MemBytes:             s.MemBytes,
		Os:                   s.Os,
	}
}
