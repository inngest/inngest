package sqlc_types

import (
	"database/sql"
	"fmt"

	sqlc_sqlite "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
)

// SQLite conversion functions: convert sqlc-generated SQLite types to domain types.

func AppFromSQLite(s *sqlc_sqlite.App) *App {
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

func FunctionFromSQLite(s *sqlc_sqlite.Function) *Function {
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

func EventFromSQLite(s *sqlc_sqlite.Event) *Event {
	return &Event{
		InternalID:  s.InternalID,
		AccountID:   toNullString(s.AccountID),
		WorkspaceID: toNullString(s.WorkspaceID),
		Source:      s.Source,
		SourceID:    toNullString(s.SourceID),
		ReceivedAt:  s.ReceivedAt,
		EventID:     s.EventID,
		EventName:   s.EventName,
		EventData:   s.EventData,
		EventUser:   s.EventUser,
		EventV:      s.EventV,
		EventTs:     s.EventTs,
	}
}

func EventBatchFromSQLite(s *sqlc_sqlite.EventBatch) *EventBatch {
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

func FunctionFinishFromSQLite(s *sqlc_sqlite.FunctionFinish) *FunctionFinish {
	return &FunctionFinish{
		RunID:              s.RunID,
		Status:             s.Status,
		Output:             s.Output,
		CompletedStepCount: s.CompletedStepCount,
		CreatedAt:          s.CreatedAt,
	}
}

func FunctionRunFromSQLite(s *sqlc_sqlite.FunctionRun) *FunctionRun {
	return &FunctionRun{
		RunID:           s.RunID,
		RunStartedAt:    s.RunStartedAt,
		FunctionID:      s.FunctionID,
		FunctionVersion: s.FunctionVersion,
		TriggerType:     s.TriggerType,
		EventID:         s.EventID,
		BatchID:         s.BatchID,
		OriginalRunID:   s.OriginalRunID,
		Cron:            s.Cron,
		WorkspaceID:     s.WorkspaceID,
	}
}

func HistoryFromSQLite(s *sqlc_sqlite.History) *History {
	return &History{
		ID:                   s.ID,
		CreatedAt:            s.CreatedAt,
		RunStartedAt:         s.RunStartedAt,
		FunctionID:           s.FunctionID,
		FunctionVersion:      s.FunctionVersion,
		RunID:                s.RunID,
		EventID:              s.EventID,
		BatchID:              s.BatchID,
		GroupID:              s.GroupID,
		IdempotencyKey:       s.IdempotencyKey,
		Type:                 s.Type,
		Attempt:              s.Attempt,
		LatencyMs:            s.LatencyMs,
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

func TraceRunFromSQLite(s *sqlc_sqlite.TraceRun) *TraceRun {
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
		Status:       s.Status,
		SourceID:     s.SourceID,
		TriggerIds:   s.TriggerIds,
		Output:       s.Output,
		IsDebounce:   s.IsDebounce,
		BatchID:      s.BatchID,
		CronSchedule: s.CronSchedule,
		HasAi:        s.HasAi,
	}
}

func TraceFromSQLite(s *sqlc_sqlite.Trace) *Trace {
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
		Duration:           s.Duration,
		StatusCode:         s.StatusCode,
		StatusMessage:      s.StatusMessage,
		Events:             s.Events,
		Links:              s.Links,
		RunID:              s.RunID,
	}
}

func WorkerConnectionFromSQLite(s *sqlc_sqlite.WorkerConnection) *WorkerConnection {
	return &WorkerConnection{
		AccountID:            s.AccountID,
		WorkspaceID:          s.WorkspaceID,
		AppName:              s.AppName,
		AppID:                s.AppID,
		ID:                   s.ID,
		GatewayID:            s.GatewayID,
		InstanceID:           s.InstanceID,
		Status:               s.Status,
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
		FunctionCount:        s.FunctionCount,
		CpuCores:             s.CpuCores,
		MemBytes:             s.MemBytes,
		Os:                   s.Os,
	}
}

// toNullString converts an interface{} (used by SQLite sqlc for untyped columns) to sql.NullString.
func toNullString(v interface{}) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	s := fmt.Sprintf("%v", v)
	return sql.NullString{String: s, Valid: true}
}
