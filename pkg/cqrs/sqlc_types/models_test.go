package sqlc_types_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs/sqlc_types"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestDomainModelZeroValues(t *testing.T) {
	t.Run("App zero value is valid", func(t *testing.T) {
		var app sqlc_types.App
		assert.Equal(t, uuid.UUID{}, app.ID)
		assert.Equal(t, "", app.Name)
		assert.False(t, app.Framework.Valid)
		assert.False(t, app.Error.Valid)
		assert.False(t, app.ArchivedAt.Valid)
		assert.True(t, app.CreatedAt.IsZero())
	})

	t.Run("Event zero value is valid", func(t *testing.T) {
		var event sqlc_types.Event
		assert.Equal(t, ulid.ULID{}, event.InternalID)
		assert.False(t, event.AccountID.Valid)
		assert.False(t, event.WorkspaceID.Valid)
		assert.False(t, event.Source.Valid)
	})

	t.Run("FunctionRunRow embeds both structs", func(t *testing.T) {
		row := sqlc_types.FunctionRunRow{
			FunctionRun: sqlc_types.FunctionRun{
				RunID:      ulid.Make(),
				TriggerType: "event",
			},
			FunctionFinish: sqlc_types.FunctionFinish{
				Status: sql.NullString{String: "Completed", Valid: true},
			},
		}
		assert.Equal(t, "event", row.FunctionRun.TriggerType)
		assert.True(t, row.FunctionFinish.Status.Valid)
		assert.Equal(t, "Completed", row.FunctionFinish.Status.String)
	})

	t.Run("History has all nullable step fields", func(t *testing.T) {
		var h sqlc_types.History
		assert.False(t, h.StepName.Valid)
		assert.False(t, h.StepID.Valid)
		assert.False(t, h.StepType.Valid)
		assert.False(t, h.CancelRequest.Valid)
		assert.False(t, h.Sleep.Valid)
		assert.False(t, h.WaitForEvent.Valid)
		assert.False(t, h.WaitResult.Valid)
		assert.False(t, h.InvokeFunction.Valid)
		assert.False(t, h.InvokeFunctionResult.Valid)
		assert.False(t, h.Result.Valid)
	})
}

func TestDomainModelFieldAssignment(t *testing.T) {
	now := time.Now().UTC()
	appID := uuid.New()
	runID := ulid.Make()

	t.Run("App with all fields populated", func(t *testing.T) {
		app := sqlc_types.App{
			ID:          appID,
			Name:        "test-app",
			SdkLanguage: "go",
			SdkVersion:  "1.0.0",
			Framework:   sql.NullString{String: "gin", Valid: true},
			Metadata:    `{"key":"value"}`,
			Status:      "active",
			Error:       sql.NullString{},
			Checksum:    "abc123",
			CreatedAt:   now,
			ArchivedAt:  sql.NullTime{},
			Url:         "https://example.com",
			Method:      "POST",
			AppVersion:  sql.NullString{String: "2", Valid: true},
		}
		assert.Equal(t, appID, app.ID)
		assert.Equal(t, "test-app", app.Name)
		assert.True(t, app.Framework.Valid)
		assert.Equal(t, "gin", app.Framework.String)
		assert.False(t, app.Error.Valid)
		assert.False(t, app.ArchivedAt.Valid)
	})

	t.Run("FunctionRun with version as int64", func(t *testing.T) {
		run := sqlc_types.FunctionRun{
			RunID:           runID,
			RunStartedAt:    now,
			FunctionID:      appID,
			FunctionVersion: 42,
			TriggerType:     "cron",
			Cron:            sql.NullString{String: "* * * * *", Valid: true},
		}
		assert.Equal(t, int64(42), run.FunctionVersion)
		assert.True(t, run.Cron.Valid)
	})

	t.Run("WorkerConnection uses int64 for Status", func(t *testing.T) {
		wc := sqlc_types.WorkerConnection{
			Status:       3,
			FunctionCount: 10,
			CpuCores:     4,
			MemBytes:     1024 * 1024 * 1024,
		}
		assert.Equal(t, int64(3), wc.Status)
		assert.Equal(t, int64(10), wc.FunctionCount)
		assert.Equal(t, int64(4), wc.CpuCores)
	})

	t.Run("TraceRun uses int64 for Status", func(t *testing.T) {
		tr := sqlc_types.TraceRun{
			RunID:  runID,
			Status: 2,
			HasAi:  true,
		}
		assert.Equal(t, int64(2), tr.Status)
		assert.True(t, tr.HasAi)
	})

	t.Run("Trace uses int64 for Duration", func(t *testing.T) {
		trace := sqlc_types.Trace{
			Duration: 12345,
			RunID:    runID,
		}
		assert.Equal(t, int64(12345), trace.Duration)
	})

	t.Run("Span uses []byte for JSON fields", func(t *testing.T) {
		span := sqlc_types.Span{
			Attributes: []byte(`{"key":"value"}`),
			Links:      []byte(`[]`),
			Output:     []byte(`{"result":true}`),
			Input:      []byte(`{"args":[]}`),
			EventIds:   []byte(`["id1","id2"]`),
		}
		assert.JSONEq(t, `{"key":"value"}`, string(span.Attributes))
		assert.JSONEq(t, `[]`, string(span.Links))
	})

	t.Run("QueueSnapshotChunk uses string for SnapshotID", func(t *testing.T) {
		chunk := sqlc_types.QueueSnapshotChunk{
			SnapshotID: "snap-123",
			ChunkID:    0,
			Data:       []byte("data"),
		}
		assert.Equal(t, "snap-123", chunk.SnapshotID)
	})
}

func TestParamStructs(t *testing.T) {
	now := time.Now().UTC()
	appID := uuid.New()

	t.Run("UpsertAppParams has all required fields", func(t *testing.T) {
		params := sqlc_types.UpsertAppParams{
			ID:          appID,
			Name:        "my-app",
			SdkLanguage: "typescript",
			SdkVersion:  "2.0.0",
			Framework:   sql.NullString{String: "next", Valid: true},
			Metadata:    "{}",
			Status:      "active",
			Error:       sql.NullString{},
			Checksum:    "checksum",
			Url:         "https://app.example.com",
			Method:      "POST",
			AppVersion:  sql.NullString{String: "1", Valid: true},
		}
		assert.Equal(t, appID, params.ID)
		assert.Equal(t, "my-app", params.Name)
	})

	t.Run("InsertHistoryParams uses int64 for numeric fields", func(t *testing.T) {
		params := sqlc_types.InsertHistoryParams{
			ID:              ulid.Make(),
			CreatedAt:       now,
			RunStartedAt:    now,
			FunctionID:      appID,
			FunctionVersion: 5,
			RunID:           ulid.Make(),
			EventID:         ulid.Make(),
			Type:            "FunctionStarted",
			Attempt:         1,
			LatencyMs:       sql.NullInt64{Int64: 150, Valid: true},
		}
		assert.Equal(t, int64(5), params.FunctionVersion)
		assert.Equal(t, int64(1), params.Attempt)
		assert.True(t, params.LatencyMs.Valid)
	})

	t.Run("InsertWorkerConnectionParams uses int64 for Status", func(t *testing.T) {
		params := sqlc_types.InsertWorkerConnectionParams{
			Status:        3,
			FunctionCount: 5,
			CpuCores:      8,
		}
		assert.Equal(t, int64(3), params.Status)
		assert.Equal(t, int64(5), params.FunctionCount)
		assert.Equal(t, int64(8), params.CpuCores)
	})

	t.Run("InsertTraceRunParams uses int64 for Status", func(t *testing.T) {
		params := sqlc_types.InsertTraceRunParams{
			Status: 1,
		}
		assert.Equal(t, int64(1), params.Status)
	})

	t.Run("InsertTraceParams uses int64 for Duration", func(t *testing.T) {
		params := sqlc_types.InsertTraceParams{
			Duration: 9999,
		}
		assert.Equal(t, int64(9999), params.Duration)
	})

	t.Run("InsertSpanParams uses []byte for JSON fields", func(t *testing.T) {
		params := sqlc_types.InsertSpanParams{
			Attributes: []byte(`{}`),
			Links:      []byte(`[]`),
			Output:     nil,
			Input:      nil,
			EventIds:   []byte(`["e1"]`),
		}
		assert.NotNil(t, params.Attributes)
		assert.Nil(t, params.Output)
	})
}
