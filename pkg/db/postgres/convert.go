package postgres

import (
	"database/sql"
	"encoding/json"
	"time"

	sqlc "github.com/inngest/inngest/pkg/db/postgres/sqlc"
	"github.com/inngest/inngest/pkg/db"
	"github.com/sqlc-dev/pqtype"
)

func appFromPG(s *sqlc.App) *db.App {
	return &db.App{
		ID: s.ID, Name: s.Name, SdkLanguage: s.SdkLanguage, SdkVersion: s.SdkVersion,
		Framework: s.Framework, Metadata: s.Metadata, Status: s.Status, Error: s.Error,
		Checksum: s.Checksum, CreatedAt: s.CreatedAt, ArchivedAt: s.ArchivedAt,
		Url: s.Url, Method: s.Method, AppVersion: s.AppVersion,
	}
}

func functionFromPG(s *sqlc.Function) *db.Function {
	return &db.Function{
		ID: s.ID, AppID: s.AppID, Name: s.Name, Slug: s.Slug,
		Config: s.Config, CreatedAt: s.CreatedAt, ArchivedAt: s.ArchivedAt,
	}
}

func eventFromPG(s *sqlc.Event) *db.Event {
	return &db.Event{
		InternalID: s.InternalID, AccountID: s.AccountID,
		WorkspaceID: s.WorkspaceID, Source: s.Source,
		SourceID: s.SourceID, ReceivedAt: s.ReceivedAt,
		EventID: s.EventID, EventName: s.EventName, EventData: s.EventData,
		EventUser: s.EventUser, EventV: s.EventV, EventTs: s.EventTs,
	}
}

func eventBatchFromPG(s *sqlc.EventBatch) *db.EventBatch {
	return &db.EventBatch{
		ID: s.ID.ULID(), AccountID: s.AccountID, WorkspaceID: s.WorkspaceID,
		AppID: s.AppID, WorkflowID: s.WorkflowID, RunID: s.RunID.ULID(),
		StartedAt: s.StartedAt, ExecutedAt: s.ExecutedAt, EventIds: s.EventIds,
	}
}

func functionFinishFromPG(s *sqlc.FunctionFinish) *db.FunctionFinish {
	// Postgres uses non-nullable fields; domain uses sql.Null* types.
	return &db.FunctionFinish{
		RunID:              s.RunID,
		Status:             sql.NullString{String: s.Status, Valid: s.Status != ""},
		Output:             sql.NullString{String: s.Output, Valid: s.Output != ""},
		CompletedStepCount: sql.NullInt64{Int64: int64(s.CompletedStepCount), Valid: true},
		CreatedAt:          sql.NullTime{Time: s.CreatedAt, Valid: !s.CreatedAt.IsZero()},
	}
}

func functionRunFromPG(s *sqlc.FunctionRun) *db.FunctionRun {
	return &db.FunctionRun{
		RunID: s.RunID, RunStartedAt: s.RunStartedAt, FunctionID: s.FunctionID,
		FunctionVersion: int64(s.FunctionVersion), TriggerType: s.TriggerType,
		EventID: s.EventID, BatchID: s.BatchID, OriginalRunID: s.OriginalRunID,
		Cron: s.Cron,
		// Postgres FunctionRun doesn't have WorkspaceID; leave zero value.
	}
}

func historyFromPG(s *sqlc.History) *db.History {
	return &db.History{
		ID: s.ID, CreatedAt: s.CreatedAt, RunStartedAt: s.RunStartedAt,
		FunctionID: s.FunctionID, FunctionVersion: int64(s.FunctionVersion),
		RunID: s.RunID, EventID: s.EventID, BatchID: s.BatchID,
		GroupID: s.GroupID, IdempotencyKey: s.IdempotencyKey, Type: s.Type,
		Attempt: int64(s.Attempt), LatencyMs: nullInt32to64(s.LatencyMs),
		StepName: s.StepName, StepID: s.StepID, StepType: s.StepType,
		Url: s.Url, CancelRequest: s.CancelRequest, Sleep: s.Sleep,
		WaitForEvent: s.WaitForEvent, WaitResult: s.WaitResult,
		InvokeFunction: s.InvokeFunction, InvokeFunctionResult: s.InvokeFunctionResult,
		Result: s.Result,
	}
}

func traceFromPG(s *sqlc.Trace) *db.Trace {
	return &db.Trace{
		Timestamp: s.Timestamp, TimestampUnixMs: s.TimestampUnixMs,
		TraceID: s.TraceID, SpanID: s.SpanID, ParentSpanID: s.ParentSpanID,
		TraceState: s.TraceState, SpanName: s.SpanName, SpanKind: s.SpanKind,
		ServiceName: s.ServiceName, ResourceAttributes: s.ResourceAttributes,
		ScopeName: s.ScopeName, ScopeVersion: s.ScopeVersion,
		SpanAttributes: s.SpanAttributes, Duration: int64(s.Duration),
		StatusCode: s.StatusCode, StatusMessage: s.StatusMessage,
		Events: s.Events, Links: s.Links, RunID: s.RunID,
	}
}

func traceRunFromPG(s *sqlc.TraceRun) *db.TraceRun {
	return &db.TraceRun{
		RunID: s.RunID, AccountID: s.AccountID, WorkspaceID: s.WorkspaceID,
		AppID: s.AppID, FunctionID: s.FunctionID, TraceID: s.TraceID,
		QueuedAt: s.QueuedAt, StartedAt: s.StartedAt, EndedAt: s.EndedAt,
		Status: int64(s.Status), SourceID: s.SourceID, TriggerIds: s.TriggerIds,
		Output: s.Output, IsDebounce: s.IsDebounce, BatchID: s.BatchID,
		CronSchedule: s.CronSchedule, HasAi: s.HasAi,
	}
}

func workerConnectionFromPG(s *sqlc.WorkerConnection) *db.WorkerConnection {
	return &db.WorkerConnection{
		AccountID: s.AccountID, WorkspaceID: s.WorkspaceID, AppName: s.AppName,
		AppID: s.AppID, ID: s.ID, GatewayID: s.GatewayID, InstanceID: s.InstanceID,
		Status: int64(s.Status), WorkerIp: s.WorkerIp,
		MaxWorkerConcurrency: s.MaxWorkerConcurrency,
		ConnectedAt:          s.ConnectedAt, LastHeartbeatAt: s.LastHeartbeatAt,
		DisconnectedAt: s.DisconnectedAt, RecordedAt: s.RecordedAt,
		InsertedAt: s.InsertedAt, DisconnectReason: s.DisconnectReason,
		GroupHash: s.GroupHash, SdkLang: s.SdkLang, SdkVersion: s.SdkVersion,
		SdkPlatform: s.SdkPlatform, SyncID: s.SyncID, AppVersion: s.AppVersion,
		FunctionCount: int64(s.FunctionCount), CpuCores: int64(s.CpuCores),
		MemBytes: s.MemBytes, Os: s.Os,
	}
}

func functionRunRowFromPG(run *sqlc.FunctionRun, finish *sqlc.FunctionFinish) *db.FunctionRunRow {
	return &db.FunctionRunRow{
		FunctionRun:    *functionRunFromPG(run),
		FunctionFinish: *functionFinishFromPG(finish),
	}
}

// toTime extracts a time.Time from the interface{} returned by sqlc for
// aggregated timestamp columns (e.g. MIN(start_time)).
func toTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	if t, ok := v.(time.Time); ok {
		return t
	}
	return time.Time{}
}

// --- helpers ---

func nullInt32to64(n sql.NullInt32) sql.NullInt64 {
	if !n.Valid {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(n.Int32), Valid: true}
}

func nullRawToBytes(n pqtype.NullRawMessage) []byte {
	if !n.Valid {
		return nil
	}
	return []byte(n.RawMessage)
}

func bytesToNullRaw(b []byte) pqtype.NullRawMessage {
	if b == nil {
		return pqtype.NullRawMessage{}
	}
	return pqtype.NullRawMessage{RawMessage: json.RawMessage(b), Valid: true}
}

func convertSlice[S any, D any](src []*S, fn func(*S) *D) []*D {
	out := make([]*D, len(src))
	for i, s := range src {
		out[i] = fn(s)
	}
	return out
}
