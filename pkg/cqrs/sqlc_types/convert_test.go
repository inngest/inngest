package sqlc_types_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	sqlc_pg "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	sqlc_sqlite "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/cqrs/sqlc_types"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testNow   = time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	testAppID = uuid.MustParse("12345678-1234-1234-1234-123456789012")
	testRunID = ulid.MustParse("01HQ3J9XQJZK0A1B2C3D4E5F6G")
)

// TestSQLiteAppConversion verifies that SQLite App converts correctly to domain App.
func TestSQLiteAppConversion(t *testing.T) {
	src := &sqlc_sqlite.App{
		ID:          testAppID,
		Name:        "my-app",
		SdkLanguage: "go",
		SdkVersion:  "1.0.0",
		Framework:   sql.NullString{String: "fiber", Valid: true},
		Metadata:    `{"env":"prod"}`,
		Status:      "active",
		Error:       sql.NullString{},
		Checksum:    "sha256:abc",
		CreatedAt:   testNow,
		ArchivedAt:  sql.NullTime{},
		Url:         "https://example.com/api/inngest",
		Method:      "POST",
		AppVersion:  sql.NullString{String: "3", Valid: true},
	}

	result := sqlc_types.AppFromSQLite(src)

	assert.Equal(t, testAppID, result.ID)
	assert.Equal(t, "my-app", result.Name)
	assert.Equal(t, "go", result.SdkLanguage)
	assert.Equal(t, "1.0.0", result.SdkVersion)
	assert.True(t, result.Framework.Valid)
	assert.Equal(t, "fiber", result.Framework.String)
	assert.Equal(t, `{"env":"prod"}`, result.Metadata)
	assert.Equal(t, "active", result.Status)
	assert.False(t, result.Error.Valid)
	assert.Equal(t, "sha256:abc", result.Checksum)
	assert.Equal(t, testNow, result.CreatedAt)
	assert.False(t, result.ArchivedAt.Valid)
	assert.Equal(t, "https://example.com/api/inngest", result.Url)
	assert.Equal(t, "POST", result.Method)
	assert.True(t, result.AppVersion.Valid)
	assert.Equal(t, "3", result.AppVersion.String)
}

// TestPostgresAppConversion verifies that Postgres App converts correctly to domain App.
func TestPostgresAppConversion(t *testing.T) {
	src := &sqlc_pg.App{
		ID:          testAppID,
		Name:        "pg-app",
		SdkLanguage: "typescript",
		SdkVersion:  "2.0.0",
		Framework:   sql.NullString{},
		Metadata:    "{}",
		Status:      "active",
		Error:       sql.NullString{String: "some error", Valid: true},
		Checksum:    "sha256:def",
		CreatedAt:   testNow,
		ArchivedAt:  sql.NullTime{Time: testNow.Add(24 * time.Hour), Valid: true},
		Url:         "https://pg.example.com",
		Method:      "PUT",
		AppVersion:  sql.NullString{},
	}

	result := sqlc_types.AppFromPostgres(src)

	assert.Equal(t, testAppID, result.ID)
	assert.Equal(t, "pg-app", result.Name)
	assert.Equal(t, "typescript", result.SdkLanguage)
	assert.False(t, result.Framework.Valid)
	assert.True(t, result.Error.Valid)
	assert.Equal(t, "some error", result.Error.String)
	assert.True(t, result.ArchivedAt.Valid)
	assert.False(t, result.AppVersion.Valid)
}

// TestBothDialectsProduceSameApp verifies that identical data from both dialects
// produces identical domain types.
func TestBothDialectsProduceSameApp(t *testing.T) {
	sqliteApp := &sqlc_sqlite.App{
		ID:          testAppID,
		Name:        "common-app",
		SdkLanguage: "python",
		SdkVersion:  "3.0.0",
		Framework:   sql.NullString{},
		Metadata:    "{}",
		Status:      "active",
		Error:       sql.NullString{},
		Checksum:    "checksum",
		CreatedAt:   testNow,
		ArchivedAt:  sql.NullTime{},
		Url:         "https://app.test",
		Method:      "POST",
		AppVersion:  sql.NullString{},
	}

	pgApp := &sqlc_pg.App{
		ID:          testAppID,
		Name:        "common-app",
		SdkLanguage: "python",
		SdkVersion:  "3.0.0",
		Framework:   sql.NullString{},
		Metadata:    "{}",
		Status:      "active",
		Error:       sql.NullString{},
		Checksum:    "checksum",
		CreatedAt:   testNow,
		ArchivedAt:  sql.NullTime{},
		Url:         "https://app.test",
		Method:      "POST",
		AppVersion:  sql.NullString{},
	}

	fromSQLite := sqlc_types.AppFromSQLite(sqliteApp)
	fromPG := sqlc_types.AppFromPostgres(pgApp)

	assert.Equal(t, fromSQLite, fromPG, "same data from both dialects should produce identical domain types")
}

func TestFunctionConversion(t *testing.T) {
	funcID := uuid.New()

	t.Run("SQLite", func(t *testing.T) {
		src := &sqlc_sqlite.Function{
			ID:     funcID,
			AppID:  testAppID,
			Name:   "my-function",
			Slug:   "my-app-my-function",
			Config: `{"retries":3}`,
			CreatedAt: testNow,
		}
		result := sqlc_types.FunctionFromSQLite(src)
		assert.Equal(t, funcID, result.ID)
		assert.Equal(t, testAppID, result.AppID)
		assert.Equal(t, "my-function", result.Name)
		assert.Equal(t, `{"retries":3}`, result.Config)
	})

	t.Run("Postgres", func(t *testing.T) {
		src := &sqlc_pg.Function{
			ID:     funcID,
			AppID:  testAppID,
			Name:   "my-function",
			Slug:   "my-app-my-function",
			Config: `{"retries":3}`,
			CreatedAt: testNow,
		}
		result := sqlc_types.FunctionFromPostgres(src)
		assert.Equal(t, funcID, result.ID)
		assert.Equal(t, testAppID, result.AppID)
	})
}

func TestFunctionRunConversion(t *testing.T) {
	t.Run("SQLite preserves int64 FunctionVersion", func(t *testing.T) {
		src := &sqlc_sqlite.FunctionRun{
			RunID:           testRunID,
			RunStartedAt:    testNow,
			FunctionID:      testAppID,
			FunctionVersion: 42,
			TriggerType:     "event",
			WorkspaceID:     uuid.New(),
		}
		result := sqlc_types.FunctionRunFromSQLite(src)
		assert.Equal(t, int64(42), result.FunctionVersion)
		assert.NotEqual(t, uuid.UUID{}, result.WorkspaceID)
	})

	t.Run("Postgres widens int32 to int64", func(t *testing.T) {
		src := &sqlc_pg.FunctionRun{
			RunID:           testRunID,
			RunStartedAt:    testNow,
			FunctionID:      testAppID,
			FunctionVersion: 42,
			TriggerType:     "event",
		}
		result := sqlc_types.FunctionRunFromPostgres(src)
		assert.Equal(t, int64(42), result.FunctionVersion)
	})
}

func TestFunctionFinishConversion(t *testing.T) {
	t.Run("SQLite nullable fields pass through", func(t *testing.T) {
		src := &sqlc_sqlite.FunctionFinish{
			RunID:              testRunID,
			Status:             sql.NullString{String: "Completed", Valid: true},
			Output:             sql.NullString{String: `{"data":1}`, Valid: true},
			CompletedStepCount: sql.NullInt64{Int64: 5, Valid: true},
			CreatedAt:          sql.NullTime{Time: testNow, Valid: true},
		}
		result := sqlc_types.FunctionFinishFromSQLite(src)
		assert.True(t, result.Status.Valid)
		assert.Equal(t, "Completed", result.Status.String)
		assert.True(t, result.CompletedStepCount.Valid)
		assert.Equal(t, int64(5), result.CompletedStepCount.Int64)
	})

	t.Run("Postgres non-nullable fields become nullable", func(t *testing.T) {
		src := &sqlc_pg.FunctionFinish{
			RunID:              testRunID,
			Status:             "Completed",
			Output:             `{"data":1}`,
			CompletedStepCount: 5,
			CreatedAt:          testNow,
		}
		result := sqlc_types.FunctionFinishFromPostgres(src)
		assert.True(t, result.Status.Valid)
		assert.Equal(t, "Completed", result.Status.String)
		assert.True(t, result.CompletedStepCount.Valid)
		assert.Equal(t, int64(5), result.CompletedStepCount.Int64)
		assert.True(t, result.CreatedAt.Valid)
	})

	t.Run("Postgres empty string status becomes invalid NullString", func(t *testing.T) {
		src := &sqlc_pg.FunctionFinish{
			RunID:  testRunID,
			Status: "",
			Output: "",
		}
		result := sqlc_types.FunctionFinishFromPostgres(src)
		assert.False(t, result.Status.Valid)
		assert.False(t, result.Output.Valid)
	})
}

func TestHistoryConversion(t *testing.T) {
	t.Run("Postgres widens int32 fields to int64", func(t *testing.T) {
		src := &sqlc_pg.History{
			ID:              testRunID,
			CreatedAt:       testNow,
			RunStartedAt:    testNow,
			FunctionID:      testAppID,
			FunctionVersion: 10,
			RunID:           testRunID,
			Type:            "StepCompleted",
			Attempt:         3,
			LatencyMs:       sql.NullInt32{Int32: 250, Valid: true},
			StepName:        sql.NullString{String: "send-email", Valid: true},
		}
		result := sqlc_types.HistoryFromPostgres(src)
		assert.Equal(t, int64(10), result.FunctionVersion)
		assert.Equal(t, int64(3), result.Attempt)
		assert.True(t, result.LatencyMs.Valid)
		assert.Equal(t, int64(250), result.LatencyMs.Int64)
		assert.True(t, result.StepName.Valid)
		assert.Equal(t, "send-email", result.StepName.String)
	})
}

func TestWorkerConnectionConversion(t *testing.T) {
	wcID := ulid.Make()
	gwID := ulid.Make()

	t.Run("Postgres widens int16 Status and int32 fields", func(t *testing.T) {
		appID := uuid.New()
		src := &sqlc_pg.WorkerConnection{
			AccountID:     testAppID,
			WorkspaceID:   uuid.New(),
			AppName:       "worker-1",
			AppID:         &appID,
			ID:            wcID,
			GatewayID:     gwID,
			InstanceID:    "inst-1",
			Status:        3,
			FunctionCount: 15,
			CpuCores:      8,
			MemBytes:      1024 * 1024 * 512,
		}
		result := sqlc_types.WorkerConnectionFromPostgres(src)
		assert.Equal(t, int64(3), result.Status)
		assert.Equal(t, int64(15), result.FunctionCount)
		assert.Equal(t, int64(8), result.CpuCores)
		require.NotNil(t, result.AppID)
		assert.Equal(t, appID, *result.AppID)
	})
}

func TestTraceConversion(t *testing.T) {
	t.Run("Postgres widens int32 Duration to int64", func(t *testing.T) {
		src := &sqlc_pg.Trace{
			Timestamp:       testNow,
			TimestampUnixMs: testNow.UnixMilli(),
			TraceID:         "trace-1",
			SpanID:          "span-1",
			SpanName:        "test-span",
			Duration:        500,
			StatusCode:      "OK",
			RunID:           testRunID,
		}
		result := sqlc_types.TraceFromPostgres(src)
		assert.Equal(t, int64(500), result.Duration)
	})
}

func TestTraceRunConversion(t *testing.T) {
	t.Run("Postgres widens int32 Status to int64", func(t *testing.T) {
		src := &sqlc_pg.TraceRun{
			RunID:      testRunID,
			AccountID:  testAppID,
			Status:     2,
			IsDebounce: true,
			HasAi:      false,
		}
		result := sqlc_types.TraceRunFromPostgres(src)
		assert.Equal(t, int64(2), result.Status)
		assert.True(t, result.IsDebounce)
		assert.False(t, result.HasAi)
	})
}

func TestEventConversion(t *testing.T) {
	t.Run("SQLite interface{} fields become NullString", func(t *testing.T) {
		src := &sqlc_sqlite.Event{
			InternalID: testRunID,
			AccountID:  "acc-123",
			WorkspaceID: "ws-456",
			SourceID:    nil,
			ReceivedAt:  testNow,
			EventID:     "evt-1",
			EventName:   "user.created",
			EventData:   `{"name":"test"}`,
			EventUser:   `{}`,
			EventTs:     testNow,
		}
		result := sqlc_types.EventFromSQLite(src)
		assert.True(t, result.AccountID.Valid)
		assert.Equal(t, "acc-123", result.AccountID.String)
		assert.True(t, result.WorkspaceID.Valid)
		assert.Equal(t, "ws-456", result.WorkspaceID.String)
		assert.False(t, result.SourceID.Valid, "nil interface should become invalid NullString")
	})

	t.Run("Postgres NullString fields pass through", func(t *testing.T) {
		src := &sqlc_pg.Event{
			InternalID:  testRunID,
			AccountID:   sql.NullString{String: "acc-123", Valid: true},
			WorkspaceID: sql.NullString{String: "ws-456", Valid: true},
			SourceID:    sql.NullString{},
			ReceivedAt:  testNow,
			EventID:     "evt-1",
			EventName:   "user.created",
			EventData:   `{"name":"test"}`,
			EventUser:   `{}`,
			EventTs:     testNow,
		}
		result := sqlc_types.EventFromPostgres(src)
		assert.True(t, result.AccountID.Valid)
		assert.Equal(t, "acc-123", result.AccountID.String)
		assert.False(t, result.SourceID.Valid)
	})
}

// TestQuerierInterfaceCompliance is a compile-time check placeholder.
// In Phase 2, concrete adapter types will implement sqlc_types.Querier
// and we'll add: var _ sqlc_types.Querier = (*adapter)(nil)
func TestQuerierInterfaceCompliance(t *testing.T) {
	// This test exists to document that Querier is defined and importable.
	// Actual compile-time checks happen when adapters implement it in Phase 2.
	var q sqlc_types.Querier
	assert.Nil(t, q)
}
