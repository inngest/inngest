package sqlite

import (
	"database/sql"
	"fmt"

	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/db"
)

func appFromSQLite(s *sqlc.App) *db.App {
	return &db.App{
		ID: s.ID, Name: s.Name, SdkLanguage: s.SdkLanguage, SdkVersion: s.SdkVersion,
		Framework: s.Framework, Metadata: s.Metadata, Status: s.Status, Error: s.Error,
		Checksum: s.Checksum, CreatedAt: s.CreatedAt, ArchivedAt: s.ArchivedAt,
		Url: s.Url, Method: s.Method, AppVersion: s.AppVersion,
	}
}

func functionFromSQLite(s *sqlc.Function) *db.Function {
	return &db.Function{
		ID: s.ID, AppID: s.AppID, Name: s.Name, Slug: s.Slug,
		Config: s.Config, CreatedAt: s.CreatedAt, ArchivedAt: s.ArchivedAt,
	}
}

func eventFromSQLite(s *sqlc.Event) *db.Event {
	return &db.Event{
		InternalID: s.InternalID, AccountID: toNullString(s.AccountID),
		WorkspaceID: toNullString(s.WorkspaceID), Source: s.Source,
		SourceID: toNullString(s.SourceID), ReceivedAt: s.ReceivedAt,
		EventID: s.EventID, EventName: s.EventName, EventData: s.EventData,
		EventUser: s.EventUser, EventV: s.EventV, EventTs: s.EventTs,
	}
}

func eventBatchFromSQLite(s *sqlc.EventBatch) *db.EventBatch {
	return &db.EventBatch{
		ID: s.ID, AccountID: s.AccountID, WorkspaceID: s.WorkspaceID,
		AppID: s.AppID, WorkflowID: s.WorkflowID, RunID: s.RunID,
		StartedAt: s.StartedAt, ExecutedAt: s.ExecutedAt, EventIds: s.EventIds,
	}
}

func functionFinishFromSQLite(s *sqlc.FunctionFinish) *db.FunctionFinish {
	return &db.FunctionFinish{
		RunID: s.RunID, Status: s.Status, Output: s.Output,
		CompletedStepCount: s.CompletedStepCount, CreatedAt: s.CreatedAt,
	}
}

func functionRunFromSQLite(s *sqlc.FunctionRun) *db.FunctionRun {
	return &db.FunctionRun{
		RunID: s.RunID, RunStartedAt: s.RunStartedAt, FunctionID: s.FunctionID,
		FunctionVersion: s.FunctionVersion, TriggerType: s.TriggerType,
		EventID: s.EventID, BatchID: s.BatchID, OriginalRunID: s.OriginalRunID,
		Cron: s.Cron, WorkspaceID: s.WorkspaceID,
	}
}

func historyFromSQLite(s *sqlc.History) *db.History {
	return &db.History{
		ID: s.ID, CreatedAt: s.CreatedAt, RunStartedAt: s.RunStartedAt,
		FunctionID: s.FunctionID, FunctionVersion: s.FunctionVersion,
		RunID: s.RunID, EventID: s.EventID, BatchID: s.BatchID,
		GroupID: s.GroupID, IdempotencyKey: s.IdempotencyKey, Type: s.Type,
		Attempt: s.Attempt, LatencyMs: s.LatencyMs, StepName: s.StepName,
		StepID: s.StepID, StepType: s.StepType, Url: s.Url,
		CancelRequest: s.CancelRequest, Sleep: s.Sleep,
		WaitForEvent: s.WaitForEvent, WaitResult: s.WaitResult,
		InvokeFunction: s.InvokeFunction, InvokeFunctionResult: s.InvokeFunctionResult,
		Result: s.Result,
	}
}

func traceFromSQLite(s *sqlc.Trace) *db.Trace {
	return &db.Trace{
		Timestamp: s.Timestamp, TimestampUnixMs: s.TimestampUnixMs,
		TraceID: s.TraceID, SpanID: s.SpanID, ParentSpanID: s.ParentSpanID,
		TraceState: s.TraceState, SpanName: s.SpanName, SpanKind: s.SpanKind,
		ServiceName: s.ServiceName, ResourceAttributes: s.ResourceAttributes,
		ScopeName: s.ScopeName, ScopeVersion: s.ScopeVersion,
		SpanAttributes: s.SpanAttributes, Duration: s.Duration,
		StatusCode: s.StatusCode, StatusMessage: s.StatusMessage,
		Events: s.Events, Links: s.Links, RunID: s.RunID,
	}
}

func traceRunFromSQLite(s *sqlc.TraceRun) *db.TraceRun {
	return &db.TraceRun{
		RunID: s.RunID, AccountID: s.AccountID, WorkspaceID: s.WorkspaceID,
		AppID: s.AppID, FunctionID: s.FunctionID, TraceID: s.TraceID,
		QueuedAt: s.QueuedAt, StartedAt: s.StartedAt, EndedAt: s.EndedAt,
		Status: s.Status, SourceID: s.SourceID, TriggerIds: s.TriggerIds,
		Output: s.Output, IsDebounce: s.IsDebounce, BatchID: s.BatchID,
		CronSchedule: s.CronSchedule, HasAi: s.HasAi,
	}
}

func workerConnectionFromSQLite(s *sqlc.WorkerConnection) *db.WorkerConnection {
	return &db.WorkerConnection{
		AccountID: s.AccountID, WorkspaceID: s.WorkspaceID, AppName: s.AppName,
		AppID: s.AppID, ID: s.ID, GatewayID: s.GatewayID, InstanceID: s.InstanceID,
		Status: s.Status, WorkerIp: s.WorkerIp,
		MaxWorkerConcurrency: s.MaxWorkerConcurrency,
		ConnectedAt: s.ConnectedAt, LastHeartbeatAt: s.LastHeartbeatAt,
		DisconnectedAt: s.DisconnectedAt, RecordedAt: s.RecordedAt,
		InsertedAt: s.InsertedAt, DisconnectReason: s.DisconnectReason,
		GroupHash: s.GroupHash, SdkLang: s.SdkLang, SdkVersion: s.SdkVersion,
		SdkPlatform: s.SdkPlatform, SyncID: s.SyncID, AppVersion: s.AppVersion,
		FunctionCount: s.FunctionCount, CpuCores: s.CpuCores,
		MemBytes: s.MemBytes, Os: s.Os,
	}
}

func functionRunRowFromSQLite(run *sqlc.FunctionRun, finish *sqlc.FunctionFinish) *db.FunctionRunRow {
	return &db.FunctionRunRow{
		FunctionRun:    *functionRunFromSQLite(run),
		FunctionFinish: *functionFinishFromSQLite(finish),
	}
}

func spanRowFromSQLiteRunID(r *sqlc.GetSpansByRunIDRow) *db.SpanRow {
	return &db.SpanRow{
		RunID: r.RunID, TraceID: r.TraceID, DynamicSpanID: r.DynamicSpanID,
		StartTime: r.StartTime, EndTime: r.EndTime, ParentSpanID: r.ParentSpanID,
		SpanFragments: toBytes(r.SpanFragments),
	}
}

func toNullString(v interface{}) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	s := fmt.Sprintf("%v", v)
	return sql.NullString{String: s, Valid: true}
}
