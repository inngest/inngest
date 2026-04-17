package db_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	gqlloader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	gqlmodels "github.com/inngest/inngest/pkg/coreapi/graph/models"
	gqlresolvers "github.com/inngest/inngest/pkg/coreapi/graph/resolvers"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	"github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/db/driverhelp"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type schemaColumn struct {
	Name    string
	Type    string
	NotNull bool
	Default string
}

type logicalColumn struct {
	Name string
	Type string
}

type adapterWithHelpers interface {
	db.Adapter
	Helpers() driverhelp.DialectHelpers
}

func TestSchemaColumnsMatchSqlc(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	actual := readRuntimeSchema(t, adapter.Conn(), adapter.Dialect())
	expected := readExpectedSchema(t, adapter.Dialect())
	applyLegacyRuntimeCompatibilityOverrides(expected, adapter.Dialect())

	require.Equal(t, expected, actual)
}

func TestCrossDialectSchemaParity(t *testing.T) {
	sqliteSchema := toLogicalSchema(readExpectedSchema(t, db.DialectSQLite))
	postgresSchema := toLogicalSchema(readExpectedSchema(t, db.DialectPostgres))

	require.Equal(t, postgresSchema, sqliteSchema)
}

func TestDefaultValues(t *testing.T) {
	adapter, cleanup := newSQLiteTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	conn := adapter.Conn()

	t.Run("apps", func(t *testing.T) {
		appID := uuid.New().String()

		_, err := conn.ExecContext(ctx, `
			INSERT INTO apps (id, name, sdk_language, sdk_version, status, checksum, url)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, appID, "defaults-app", "go", "1.0.0", "active", "defaults-app-checksum", "https://example.com/defaults")
		require.NoError(t, err)

		var metadata, method, createdAt string
		require.NoError(t, conn.QueryRowContext(ctx, `
			SELECT metadata, method, CAST(created_at AS TEXT)
			FROM apps
			WHERE id = ?
		`, appID).Scan(&metadata, &method, &createdAt))

		assert.JSONEq(t, `{}`, metadata)
		assert.Equal(t, "serve", method)
		assert.NotEmpty(t, createdAt)
	})

	t.Run("events", func(t *testing.T) {
		eventID := ulid.Make().String()

		_, err := conn.ExecContext(ctx, `
			INSERT INTO events (internal_id, event_id, event_name, event_ts)
			VALUES (?, ?, ?, ?)
		`, eventID, "evt-defaults", "app/defaults", time.Now().UTC())
		require.NoError(t, err)

		var receivedAt, eventData, eventUser string
		require.NoError(t, conn.QueryRowContext(ctx, `
			SELECT CAST(received_at AS TEXT), event_data, event_user
			FROM events
			WHERE internal_id = ?
		`, eventID).Scan(&receivedAt, &eventData, &eventUser))

		assert.NotEmpty(t, receivedAt)
		assert.JSONEq(t, `{}`, eventData)
		assert.JSONEq(t, `{}`, eventUser)
	})

	t.Run("function_runs", func(t *testing.T) {
		runID := ulid.Make().String()
		eventID := ulid.Make().String()

		_, err := conn.ExecContext(ctx, `
			INSERT INTO function_runs (run_id, function_version, event_id)
			VALUES (?, ?, ?)
		`, runID, 1, eventID)
		require.NoError(t, err)

		var runStartedAt, triggerType string
		require.NoError(t, conn.QueryRowContext(ctx, `
			SELECT CAST(run_started_at AS TEXT), trigger_type
			FROM function_runs
			WHERE run_id = ?
		`, runID).Scan(&runStartedAt, &triggerType))

		assert.NotEmpty(t, runStartedAt)
		assert.Equal(t, "event", triggerType)
	})

	t.Run("function_finishes", func(t *testing.T) {
		runID := ulid.Make()

		_, err := conn.ExecContext(ctx, `
			INSERT INTO function_finishes (run_id, status)
			VALUES (?, ?)
		`, runID[:], "completed")
		require.NoError(t, err)

		var output, createdAt string
		var completedStepCount int64
		require.NoError(t, conn.QueryRowContext(ctx, `
			SELECT output, completed_step_count, CAST(created_at AS TEXT)
			FROM function_finishes
			WHERE run_id = ?
		`, runID[:]).Scan(&output, &completedStepCount, &createdAt))

		assert.JSONEq(t, `{}`, output)
		assert.Equal(t, int64(1), completedStepCount)
		assert.NotEmpty(t, createdAt)
	})

	t.Run("history", func(t *testing.T) {
		historyID := ulid.Make()
		runID := ulid.Make()
		eventID := ulid.Make()

		_, err := conn.ExecContext(ctx, `
			INSERT INTO history (id, function_version, run_id, event_id, idempotency_key, type, attempt)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, historyID[:], 1, runID[:], eventID[:], "history-defaults", "FunctionStarted", 0)
		require.NoError(t, err)

		var createdAt, runStartedAt string
		require.NoError(t, conn.QueryRowContext(ctx, `
			SELECT CAST(created_at AS TEXT), CAST(run_started_at AS TEXT)
			FROM history
			WHERE id = ?
		`, historyID[:]).Scan(&createdAt, &runStartedAt))

		assert.NotEmpty(t, createdAt)
		assert.NotEmpty(t, runStartedAt)
	})

	t.Run("event_batches", func(t *testing.T) {
		batchID := ulid.Make().String()
		runID := ulid.Make().String()

		_, err := conn.ExecContext(ctx, `
			INSERT INTO event_batches (id, run_id, started_at, event_ids)
			VALUES (?, ?, ?, ?)
		`, batchID, runID, time.Now().UTC(), []byte(`["evt-defaults"]`))
		require.NoError(t, err)

		var executedAt string
		require.NoError(t, conn.QueryRowContext(ctx, `
			SELECT CAST(executed_at AS TEXT)
			FROM event_batches
			WHERE id = ?
		`, batchID).Scan(&executedAt))

		assert.NotEmpty(t, executedAt)
	})

	t.Run("trace_runs", func(t *testing.T) {
		runID := ulid.Make().String()

		_, err := conn.ExecContext(ctx, `
			INSERT INTO trace_runs (
				run_id, account_id, workspace_id, app_id, function_id, trace_id,
				queued_at, started_at, ended_at, status, source_id, trigger_ids, is_debounce
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			runID,
			uuid.New().String(),
			uuid.New().String(),
			uuid.New().String(),
			uuid.New().String(),
			[]byte("trace-defaults"),
			1, 2, 3, 1,
			"source-defaults",
			[]byte(`["evt-defaults"]`),
			false,
		)
		require.NoError(t, err)

		var hasAI int64
		require.NoError(t, conn.QueryRowContext(ctx, `
			SELECT CAST(has_ai AS INTEGER)
			FROM trace_runs
			WHERE run_id = ?
		`, runID).Scan(&hasAI))

		assert.Zero(t, hasAI)
	})
}

func TestNotNullConstraints(t *testing.T) {
	adapter, cleanup := newSQLiteTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	conn := adapter.Conn()
	for tableName, requiredColumns := range sqliteRequiredColumnsWithoutDefaults() {
		spec, ok := sqliteInsertSpecs()[tableName]
		require.Truef(t, ok, "missing sqlite insert spec for %s", tableName)

		for _, columnName := range requiredColumns {
			t.Run(tableName+"."+columnName, func(t *testing.T) {
				query, args := buildSQLiteInsert(tableName, spec, columnName)
				_, err := conn.ExecContext(ctx, query, args...)
				require.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), "not null")
			})
		}
	}
}

func TestForeignKeyAndPrimaryKey(t *testing.T) {
	adapter, cleanup := newSQLiteTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	conn := adapter.Conn()

	for _, tc := range sqlitePrimaryKeyDuplicateCases() {
		t.Run(tc.name, func(t *testing.T) {
			query, args := buildSQLiteInsert(tc.tableName, tc.values, "")
			_, err := conn.ExecContext(ctx, query, args...)
			require.NoError(t, err)

			_, err = conn.ExecContext(ctx, query, args...)
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "unique")
		})
	}
}

func TestFunctionRunRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()
	runID := ulid.Make()
	eventID := ulid.Make()
	now := time.Now().UTC()
	finishedAt := now.Add(2 * time.Second)

	require.NoError(t, q.InsertFunctionRun(ctx, db.InsertFunctionRunParams{
		RunID:           runID,
		RunStartedAt:    now,
		FunctionID:      uuid.New(),
		FunctionVersion: 3,
		TriggerType:     "event",
		EventID:         eventID,
		Cron:            sql.NullString{},
		WorkspaceID:     uuid.New(),
	}))

	require.NoError(t, q.InsertFunctionFinish(ctx, db.InsertFunctionFinishParams{
		RunID:              runID,
		Status:             sql.NullString{String: "completed", Valid: true},
		Output:             sql.NullString{String: `{"ok":true}`, Valid: true},
		CompletedStepCount: sql.NullInt64{Int64: 4, Valid: true},
		CreatedAt:          sql.NullTime{Time: finishedAt, Valid: true},
	}))

	got, err := q.GetFunctionRun(ctx, runID)
	require.NoError(t, err)

	assert.Equal(t, runID, got.FunctionRun.RunID)
	assert.Equal(t, eventID, got.FunctionRun.EventID)
	assert.Equal(t, int64(3), got.FunctionRun.FunctionVersion)
	assert.Equal(t, "completed", got.FunctionFinish.Status.String)
	assert.JSONEq(t, `{"ok":true}`, got.FunctionFinish.Output.String)
	assert.Equal(t, int64(4), got.FunctionFinish.CompletedStepCount.Int64)
	assert.WithinDuration(t, finishedAt, got.FunctionFinish.CreatedAt.Time, time.Second)
}

func TestEventBatchRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()
	batchID := ulid.Make()
	runID := ulid.Make()
	now := time.Now().UTC()
	executedAt := now.Add(2 * time.Second)
	eventIDs := []byte(`["evt-1","evt-2"]`)

	require.NoError(t, q.InsertEventBatch(ctx, db.InsertEventBatchParams{
		ID:          batchID,
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       uuid.New(),
		WorkflowID:  uuid.New(),
		RunID:       runID,
		StartedAt:   now,
		ExecutedAt:  executedAt,
		EventIds:    eventIDs,
	}))

	got, err := q.GetEventBatchByRunID(ctx, runID)
	require.NoError(t, err)

	assert.Equal(t, batchID, got.ID)
	assert.Equal(t, runID, got.RunID)
	assert.Equal(t, eventIDs, got.EventIds)
	assert.WithinDuration(t, now, got.StartedAt, time.Second)
	assert.WithinDuration(t, executedAt, got.ExecutedAt, time.Second)

	found, err := q.GetEventBatchesByEventID(ctx, "evt-2")
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, batchID, found[0].ID)
}

func TestTraceAndTraceRunRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()
	runID := ulid.Make()
	traceID := "trace-regression"
	now := time.Now().UTC()

	require.NoError(t, q.InsertTraceRun(ctx, db.InsertTraceRunParams{
		RunID:        runID,
		AccountID:    uuid.New(),
		WorkspaceID:  uuid.New(),
		AppID:        uuid.New(),
		FunctionID:   uuid.New(),
		TraceID:      []byte(traceID),
		QueuedAt:     now.UnixMilli(),
		StartedAt:    now.Add(time.Second).UnixMilli(),
		EndedAt:      now.Add(2 * time.Second).UnixMilli(),
		Status:       2,
		SourceID:     "source-regression",
		TriggerIds:   []byte(`["evt-trace"]`),
		Output:       []byte(`{"ok":true}`),
		BatchID:      ulid.Make(),
		IsDebounce:   true,
		CronSchedule: sql.NullString{String: "*/5 * * * *", Valid: true},
		HasAi:        false,
	}))

	require.NoError(t, q.InsertTrace(ctx, db.InsertTraceParams{
		Timestamp:          now,
		TimestampUnixMs:    now.UnixMilli(),
		TraceID:            traceID,
		SpanID:             "span-regression",
		ParentSpanID:       sql.NullString{},
		TraceState:         sql.NullString{},
		SpanName:           "executor.run",
		SpanKind:           "internal",
		ServiceName:        "inngest",
		ResourceAttributes: []byte(`{"service.name":"inngest"}`),
		ScopeName:          "regression",
		ScopeVersion:       "1.0.0",
		SpanAttributes:     []byte(`{"component":"executor"}`),
		Duration:           250,
		StatusCode:         "ok",
		StatusMessage:      sql.NullString{},
		Events:             []byte(`[]`),
		Links:              []byte(`[]`),
		RunID:              runID,
	}))

	gotRun, err := q.GetTraceRun(ctx, runID)
	require.NoError(t, err)
	assert.Equal(t, runID, gotRun.RunID)
	assert.Equal(t, traceID, string(gotRun.TraceID))
	assert.Equal(t, int64(2), gotRun.Status)
	assert.True(t, gotRun.IsDebounce)
	assert.Equal(t, "*/5 * * * *", gotRun.CronSchedule.String)

	runsByTrigger, err := q.GetTraceRunsByTriggerId(ctx, "evt-trace")
	require.NoError(t, err)
	require.Len(t, runsByTrigger, 1)
	assert.Equal(t, runID, runsByTrigger[0].RunID)

	spans, err := q.GetTraceSpans(ctx, db.GetTraceSpansParams{TraceID: traceID, RunID: runID})
	require.NoError(t, err)
	require.Len(t, spans, 1)
	assert.Equal(t, "span-regression", spans[0].SpanID)
	assert.Equal(t, "executor.run", spans[0].SpanName)

	outputs, err := q.GetTraceSpanOutput(ctx, db.GetTraceSpanOutputParams{TraceID: traceID, SpanID: "span-regression"})
	require.NoError(t, err)
	require.Len(t, outputs, 1)
	assert.Equal(t, "span-regression", outputs[0].SpanID)
}

func TestWorkerConnectionRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()
	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	syncID := uuid.New()
	connectionID := ulid.Make()

	require.NoError(t, q.InsertWorkerConnection(ctx, db.InsertWorkerConnectionParams{
		AccountID:            accountID,
		WorkspaceID:          workspaceID,
		AppName:              "worker-app",
		AppID:                &appID,
		ID:                   connectionID,
		GatewayID:            ulid.Make(),
		InstanceID:           "instance-1",
		Status:               2,
		WorkerIp:             "127.0.0.1",
		MaxWorkerConcurrency: 10,
		ConnectedAt:          100,
		LastHeartbeatAt:      sql.NullInt64{Int64: 110, Valid: true},
		DisconnectedAt:       sql.NullInt64{},
		RecordedAt:           120,
		InsertedAt:           130,
		DisconnectReason:     sql.NullString{},
		GroupHash:            []byte("group-hash"),
		SdkLang:              "go",
		SdkVersion:           "1.0.0",
		SdkPlatform:          "linux",
		SyncID:               &syncID,
		AppVersion:           sql.NullString{String: "2026.04", Valid: true},
		FunctionCount:        5,
		CpuCores:             4,
		MemBytes:             2048,
		Os:                   "darwin",
	}))

	got, err := q.GetWorkerConnection(ctx, db.GetWorkerConnectionParams{
		AccountID:    accountID,
		WorkspaceID:  workspaceID,
		ConnectionID: connectionID,
	})
	require.NoError(t, err)

	assert.Equal(t, connectionID, got.ID)
	assert.Equal(t, "worker-app", got.AppName)
	require.NotNil(t, got.AppID)
	assert.Equal(t, appID, *got.AppID)
	require.NotNil(t, got.SyncID)
	assert.Equal(t, syncID, *got.SyncID)
	assert.Equal(t, int64(10), got.MaxWorkerConcurrency)
	assert.Equal(t, "2026.04", got.AppVersion.String)
}

func TestGetEventByID(t *testing.T) {
	manager, gqlCtx, cleanup := newTestManagerAndResolverCtx(t)
	defer cleanup()

	resolver := &gqlresolvers.Resolver{Data: manager}
	query := resolver.Query()
	now := time.Now().UTC()
	eventID := ulid.Make()

	require.NoError(t, manager.InsertEvent(gqlCtx, cqrs.Event{
		ID:         eventID,
		EventID:    "evt-gql",
		EventName:  "app/gql.event",
		EventData:  map[string]any{"msg": "hi"},
		EventUser:  map[string]any{"id": "user_123"},
		EventTS:    now.UnixMilli(),
		ReceivedAt: now,
	}))

	got, err := query.EventV2(gqlCtx, eventID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, eventID, got.ID)
	assert.Equal(t, "app/gql.event", got.Name)
	assert.False(t, got.ReceivedAt.IsZero())
	assert.False(t, got.OccurredAt.IsZero())

	raw, err := resolver.EventV2().Raw(gqlCtx, got)
	require.NoError(t, err)
	assert.Contains(t, raw, `"name":"app/gql.event"`)
	assert.Contains(t, raw, `"id":"evt-gql"`)
	assert.Contains(t, raw, `"msg":"hi"`)
}

func TestFunctionRunStatusLifecycle(t *testing.T) {
	manager, gqlCtx, cleanup := newTestManagerAndResolverCtx(t)
	defer cleanup()

	resolver := &gqlresolvers.Resolver{Data: manager}
	query := resolver.Query()
	runID := ulid.Make()
	now := time.Now().UTC()

	insertRun := func(status enums.RunStatus, startedAt, endedAt time.Time) *gqlmodels.FunctionRunV2 {
		require.NoError(t, manager.InsertTraceRun(gqlCtx, &cqrs.TraceRun{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			FunctionID:  uuid.New(),
			TraceID:     "trace-" + runID.String(),
			RunID:       runID.String(),
			QueuedAt:    now,
			StartedAt:   startedAt,
			EndedAt:     endedAt,
			SourceID:    "source-" + status.String(),
			TriggerIDs:  []string{ulid.Make().String()},
			Status:      status,
		}))

		run, err := query.Run(gqlCtx, runID.String())
		require.NoError(t, err)
		require.NotNil(t, run)
		return run
	}

	t.Run("queued", func(t *testing.T) {
		run := insertRun(enums.RunStatusScheduled, time.Time{}, time.Time{})
		assert.Equal(t, gqlmodels.FunctionRunStatusQueued, run.Status)
		assert.Nil(t, run.StartedAt)
		assert.Nil(t, run.EndedAt)
	})

	t.Run("running", func(t *testing.T) {
		startedAt := now.Add(5 * time.Second)
		run := insertRun(enums.RunStatusRunning, startedAt, time.Time{})
		assert.Equal(t, gqlmodels.FunctionRunStatusRunning, run.Status)
		require.NotNil(t, run.StartedAt)
		assert.WithinDuration(t, startedAt, *run.StartedAt, time.Second)
		assert.Nil(t, run.EndedAt)
	})

	for _, tc := range []struct {
		name     string
		status   enums.RunStatus
		expected gqlmodels.FunctionRunStatus
	}{
		{name: "completed", status: enums.RunStatusCompleted, expected: gqlmodels.FunctionRunStatusCompleted},
		{name: "failed", status: enums.RunStatusFailed, expected: gqlmodels.FunctionRunStatusFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			startedAt := now.Add(10 * time.Second)
			endedAt := now.Add(20 * time.Second)
			run := insertRun(tc.status, startedAt, endedAt)
			assert.Equal(t, tc.expected, run.Status)
			require.NotNil(t, run.StartedAt)
			assert.WithinDuration(t, startedAt, *run.StartedAt, time.Second)
			require.NotNil(t, run.EndedAt)
			assert.WithinDuration(t, endedAt, *run.EndedAt, time.Second)
		})
	}
}

func TestEventListFiltering(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()
	externalID := ulid.MustParse("00000000000000000000000001")
	internalNoRunID := ulid.MustParse("00000000000000000000000002")
	internalWithRunID := ulid.MustParse("00000000000000000000000003")

	for _, evt := range []struct {
		id   ulid.ULID
		name string
	}{
		{id: externalID, name: "app/order.created"},
		{id: internalNoRunID, name: "inngest/function.finished"},
		{id: internalWithRunID, name: "inngest/function.failed"},
	} {
		require.NoError(t, q.InsertEvent(ctx, db.InsertEventParams{
			InternalID: evt.id,
			EventID:    evt.id.String(),
			EventName:  evt.name,
			EventData:  `{"ok":true}`,
			EventUser:  `{}`,
			EventTs:    time.Now().UTC(),
			ReceivedAt: time.Now().UTC(),
		}))
	}

	require.NoError(t, q.InsertFunctionRun(ctx, db.InsertFunctionRunParams{
		RunID:           ulid.Make(),
		RunStartedAt:    time.Now().UTC(),
		FunctionID:      uuid.New(),
		FunctionVersion: 1,
		TriggerType:     "event",
		EventID:         externalID,
		WorkspaceID:     uuid.New(),
	}))

	require.NoError(t, q.InsertFunctionRun(ctx, db.InsertFunctionRunParams{
		RunID:           ulid.Make(),
		RunStartedAt:    time.Now().UTC(),
		FunctionID:      uuid.New(),
		FunctionVersion: 1,
		TriggerType:     "event",
		EventID:         internalWithRunID,
		WorkspaceID:     uuid.New(),
	}))

	withoutInternal, err := q.GetEventsIDbound(ctx, db.GetEventsIDboundParams{
		After:           ulid.MustParse("00000000000000000000000000"),
		Before:          ulid.MustParse("7ZZZZZZZZZZZZZZZZZZZZZZZZZ"),
		IncludeInternal: "false",
		Limit:           10,
	})
	require.NoError(t, err)
	require.Len(t, withoutInternal, 2)
	assert.Equal(t, internalWithRunID, withoutInternal[0].InternalID)
	assert.Equal(t, externalID, withoutInternal[1].InternalID)

	withInternal, err := q.GetEventsIDbound(ctx, db.GetEventsIDboundParams{
		After:           ulid.MustParse("00000000000000000000000000"),
		Before:          ulid.MustParse("7ZZZZZZZZZZZZZZZZZZZZZZZZZ"),
		IncludeInternal: "true",
		Limit:           10,
	})
	require.NoError(t, err)
	require.Len(t, withInternal, 3)
	assert.Equal(t, internalWithRunID, withInternal[0].InternalID)
	assert.Equal(t, internalNoRunID, withInternal[1].InternalID)
	assert.Equal(t, externalID, withInternal[2].InternalID)
}

func newSQLiteTestAdapter(t *testing.T) (db.Adapter, func()) {
	t.Helper()

	conn, err := base_cqrs.New(t.Context(), base_cqrs.BaseCQRSOptions{
		Persist: false,
		ForTest: true,
	})
	require.NoError(t, err)

	adapter := dbsqlite.New(conn)
	return adapter, func() { conn.Close() }
}

func newTestManagerAndResolverCtx(t *testing.T) (cqrs.Manager, context.Context, func()) {
	t.Helper()

	adapter, cleanup := newTestAdapter(t)
	helperAdapter, ok := adapter.(adapterWithHelpers)
	require.True(t, ok, "test adapter must expose dialect helpers")

	manager := base_cqrs.NewCQRS(helperAdapter)
	ctx := gqlloader.ToCtx(context.Background(), gqlloader.NewLoaders(gqlloader.LoaderParams{DB: manager}))
	return manager, ctx, cleanup
}

func sqliteRequiredColumnsWithoutDefaults() map[string][]string {
	return map[string][]string{
		"apps":                  {"name", "sdk_language", "sdk_version", "status", "checksum", "url"},
		"events":                {"event_id", "event_name", "event_ts"},
		"functions":             {"name", "slug", "config"},
		"function_runs":         {"function_version", "event_id"},
		"history":               {"function_version", "run_id", "event_id", "idempotency_key", "type", "attempt"},
		"event_batches":         {"run_id", "started_at", "event_ids"},
		"traces":                {"timestamp", "timestamp_unix_ms", "trace_id", "span_id", "span_name", "span_kind", "service_name", "resource_attributes", "scope_name", "scope_version", "span_attributes", "duration", "status_code", "events", "links"},
		"trace_runs":            {"account_id", "workspace_id", "app_id", "function_id", "trace_id", "queued_at", "started_at", "ended_at", "status", "source_id", "trigger_ids", "is_debounce"},
		"queue_snapshot_chunks": {"snapshot_id", "chunk_id"},
		"worker_connections":    {"account_id", "workspace_id", "app_name", "id", "gateway_id", "instance_id", "status", "worker_ip", "connected_at", "recorded_at", "inserted_at", "group_hash", "sdk_lang", "sdk_version", "sdk_platform", "function_count", "cpu_cores", "mem_bytes", "os"},
		"spans":                 {"span_id", "trace_id", "name", "start_time", "end_time", "account_id", "app_id", "function_id", "run_id", "env_id"},
	}
}

func sqlitePrimaryKeyDuplicateCases() []struct {
	name      string
	tableName string
	values    map[string]any
} {
	specs := sqliteInsertSpecs()
	return []struct {
		name      string
		tableName string
		values    map[string]any
	}{
		{name: "apps.id", tableName: "apps", values: specs["apps"]},
		{name: "event_batches.id", tableName: "event_batches", values: specs["event_batches"]},
		{name: "trace_runs.run_id", tableName: "trace_runs", values: specs["trace_runs"]},
		{name: "queue_snapshot_chunks.snapshot_id_chunk_id", tableName: "queue_snapshot_chunks", values: specs["queue_snapshot_chunks"]},
		{name: "worker_connections.id_app_name", tableName: "worker_connections", values: specs["worker_connections"]},
		{name: "spans.trace_id_span_id", tableName: "spans", values: specs["spans"]},
	}
}

func sqliteInsertSpecs() map[string]map[string]any {
	now := time.Now().UTC()
	historyID := ulid.Make()
	historyRunID := ulid.Make()
	historyEventID := ulid.Make()
	traceRunID := ulid.Make()

	return map[string]map[string]any{
		"apps": {
			"id":           uuid.New().String(),
			"name":         "sqlite-app",
			"sdk_language": "go",
			"sdk_version":  "1.0.0",
			"status":       "active",
			"checksum":     "checksum-app",
			"url":          "https://example.com/inngest",
		},
		"events": {
			"internal_id": ulid.Make().String(),
			"event_id":    "evt-required",
			"event_name":  "app/required",
			"event_ts":    now,
		},
		"functions": {
			"id":     uuid.New().String(),
			"name":   "sqlite-function",
			"slug":   "sqlite-function",
			"config": `{"retries":{"attempts":3}}`,
		},
		"function_runs": {
			"run_id":           ulid.Make().String(),
			"function_version": 1,
			"event_id":         ulid.Make().String(),
		},
		"history": {
			"id":               historyID[:],
			"function_version": 1,
			"run_id":           historyRunID[:],
			"event_id":         historyEventID[:],
			"idempotency_key":  "history-key",
			"type":             "FunctionStarted",
			"attempt":          0,
		},
		"event_batches": {
			"id":         ulid.Make().String(),
			"run_id":     ulid.Make().String(),
			"started_at": now,
			"event_ids":  []byte(`["evt-batch"]`),
		},
		"traces": {
			"timestamp":           now,
			"timestamp_unix_ms":   now.UnixMilli(),
			"trace_id":            "trace-required",
			"span_id":             "span-required",
			"span_name":           "executor.required",
			"span_kind":           "internal",
			"service_name":        "inngest",
			"resource_attributes": []byte(`{"service.name":"inngest"}`),
			"scope_name":          "regression",
			"scope_version":       "1.0.0",
			"span_attributes":     []byte(`{"component":"executor"}`),
			"duration":            100,
			"status_code":         "ok",
			"events":              []byte(`[]`),
			"links":               []byte(`[]`),
		},
		"trace_runs": {
			"run_id":       traceRunID.String(),
			"account_id":   uuid.New().String(),
			"workspace_id": uuid.New().String(),
			"app_id":       uuid.New().String(),
			"function_id":  uuid.New().String(),
			"trace_id":     []byte("trace-run-required"),
			"queued_at":    1,
			"started_at":   2,
			"ended_at":     3,
			"status":       200,
			"source_id":    "source-required",
			"trigger_ids":  []byte(`["evt-trace"]`),
			"is_debounce":  false,
		},
		"queue_snapshot_chunks": {
			"snapshot_id": ulid.Make().String(),
			"chunk_id":    1,
		},
		"worker_connections": {
			"account_id":             uuid.New().String(),
			"workspace_id":           uuid.New().String(),
			"app_name":               "worker-app",
			"id":                     ulid.Make().String(),
			"gateway_id":             ulid.Make().String(),
			"instance_id":            "instance-1",
			"status":                 1,
			"worker_ip":              "127.0.0.1",
			"max_worker_concurrency": 5,
			"connected_at":           100,
			"recorded_at":            110,
			"inserted_at":            120,
			"group_hash":             []byte("group-hash"),
			"sdk_lang":               "go",
			"sdk_version":            "1.0.0",
			"sdk_platform":           "darwin",
			"function_count":         2,
			"cpu_cores":              4,
			"mem_bytes":              2048,
			"os":                     "darwin",
		},
		"spans": {
			"span_id":     "span-primary",
			"trace_id":    "trace-primary",
			"name":        "executor.run",
			"start_time":  now,
			"end_time":    now.Add(time.Second),
			"account_id":  uuid.New().String(),
			"app_id":      uuid.New().String(),
			"function_id": uuid.New().String(),
			"run_id":      ulid.Make().String(),
			"env_id":      uuid.New().String(),
		},
	}
}

func buildSQLiteInsert(tableName string, values map[string]any, omitColumn string) (string, []any) {
	columns := make([]string, 0, len(values))
	for columnName := range values {
		if columnName == omitColumn {
			continue
		}
		columns = append(columns, columnName)
	}
	sort.Strings(columns)

	placeholders := make([]string, len(columns))
	args := make([]any, len(columns))
	for i, columnName := range columns {
		placeholders[i] = "?"
		args[i] = values[columnName]
		columns[i] = fmt.Sprintf(`"%s"`, columnName)
	}

	return fmt.Sprintf(
		`INSERT INTO "%s" (%s) VALUES (%s)`,
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	), args
}

func insertEventWithScopes(t *testing.T, ctx context.Context, adapter db.Adapter, internalID ulid.ULID, accountID, workspaceID uuid.UUID) {
	t.Helper()

	var (
		query string
		args  []any
	)

	switch adapter.Dialect() {
	case db.DialectPostgres:
		query = `
			INSERT INTO events (
				internal_id, account_id, workspace_id, received_at, event_id, event_name, event_data, event_user, event_ts
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		args = []any{
			internalID[:],
			accountID.String(),
			workspaceID.String(),
			time.Now().UTC(),
			"evt-scoped",
			"app/scoped",
			`{"scoped":true}`,
			`{"user_id":"u_123"}`,
			time.Now().UTC(),
		}
	default:
		query = `
			INSERT INTO events (
				internal_id, account_id, workspace_id, received_at, event_id, event_name, event_data, event_user, event_ts
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		args = []any{
			internalID,
			accountID.String(),
			workspaceID.String(),
			time.Now().UTC(),
			"evt-scoped",
			"app/scoped",
			`{"scoped":true}`,
			`{"user_id":"u_123"}`,
			time.Now().UTC(),
		}
	}

	_, err := adapter.Conn().ExecContext(ctx, query, args...)
	require.NoError(t, err)
}

func readStoredSpanState(t *testing.T, ctx context.Context, adapter db.Adapter, traceID, spanID string) (string, string) {
	t.Helper()

	var (
		status   string
		eventIDs string
		err      error
	)

	switch adapter.Dialect() {
	case db.DialectPostgres:
		err = adapter.Conn().QueryRowContext(ctx, `
			SELECT status, COALESCE(event_ids::text, '')
			FROM spans
			WHERE trace_id = $1 AND span_id = $2
		`, traceID, spanID).Scan(&status, &eventIDs)
	default:
		err = adapter.Conn().QueryRowContext(ctx, `
			SELECT status, COALESCE(CAST(event_ids AS TEXT), '')
			FROM spans
			WHERE trace_id = ? AND span_id = ?
		`, traceID, spanID).Scan(&status, &eventIDs)
	}
	require.NoError(t, err)
	return status, eventIDs
}

func readExpectedSchema(t *testing.T, dialect db.Dialect) map[string][]schemaColumn {
	t.Helper()

	path := filepath.Join("..", "cqrs", "base_cqrs", "sqlc", string(dialect), "schema.sql")
	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	return parseSchemaColumns(t, string(contents))
}

func applyLegacyRuntimeCompatibilityOverrides(schema map[string][]schemaColumn, dialect db.Dialect) {
	if dialect != db.DialectSQLite {
		return
	}

	// The legacy SQLite runtime schema intentionally did not enforce
	// uniqueness on events.internal_id, and the goose baseline preserves that
	// behavior for compatibility during the migration transition.
	for i := range schema["events"] {
		if schema["events"][i].Name == "internal_id" {
			schema["events"][i].NotNull = false
			return
		}
	}
}

func parseSchemaColumns(t *testing.T, contents string) map[string][]schemaColumn {
	t.Helper()

	result := map[string][]schemaColumn{}
	for _, statement := range splitSQLStatements(contents) {
		statement = strings.TrimSpace(statement)
		if !strings.HasPrefix(strings.ToUpper(statement), "CREATE TABLE ") {
			continue
		}

		tableName, definitions := parseCreateTableStatement(t, statement)
		for _, definition := range splitTopLevel(definitions, ',') {
			column, ok := parseSchemaColumnLine(definition)
			if !ok {
				continue
			}
			result[tableName] = append(result[tableName], column)
		}
	}

	return result
}

func parseCreateTableStatement(t *testing.T, statement string) (string, string) {
	t.Helper()

	remainder := strings.TrimSpace(statement[len("CREATE TABLE "):])
	if strings.HasPrefix(strings.ToUpper(remainder), "IF NOT EXISTS ") {
		remainder = strings.TrimSpace(remainder[len("IF NOT EXISTS "):])
	}

	openIdx := strings.Index(remainder, "(")
	require.NotEqual(t, -1, openIdx, "invalid CREATE TABLE statement: %q", statement)

	tableName := normalizeIdentifier(remainder[:openIdx])
	body := strings.TrimSpace(remainder[openIdx+1:])
	if strings.HasSuffix(body, ")") {
		body = strings.TrimSpace(body[:len(body)-1])
	}

	return tableName, body
}

func parseSchemaColumnLine(line string) (schemaColumn, bool) {
	line = strings.TrimSpace(strings.TrimSuffix(line, ","))
	upper := strings.ToUpper(line)
	if strings.HasPrefix(upper, "PRIMARY KEY") || strings.HasPrefix(upper, "UNIQUE") || strings.HasPrefix(upper, "CONSTRAINT") {
		return schemaColumn{}, false
	}

	nameEnd := strings.IndexAny(line, " \t")
	if nameEnd == -1 {
		return schemaColumn{}, false
	}

	name := strings.Trim(line[:nameEnd], `"`)
	remainder := strings.TrimSpace(line[nameEnd+1:])
	typeEnd := len(remainder)
	for _, marker := range []string{" DEFAULT ", " NOT NULL", " PRIMARY KEY", " UNIQUE", " CHECK", " REFERENCES", " CONSTRAINT"} {
		if idx := strings.Index(strings.ToUpper(remainder), marker); idx >= 0 && idx < typeEnd {
			typeEnd = idx
		}
	}

	defaultExpr := ""
	if idx := strings.Index(strings.ToUpper(remainder), " DEFAULT "); idx >= 0 {
		defaultExpr = strings.TrimSpace(remainder[idx+len(" DEFAULT "):])
		for _, marker := range []string{" NOT NULL", " PRIMARY KEY", " UNIQUE", " CHECK", " REFERENCES", " CONSTRAINT"} {
			if end := strings.Index(strings.ToUpper(defaultExpr), marker); end >= 0 {
				defaultExpr = strings.TrimSpace(defaultExpr[:end])
				break
			}
		}
	}

	return schemaColumn{
		Name:    name,
		Type:    normalizeType(strings.TrimSpace(remainder[:typeEnd])),
		NotNull: strings.Contains(strings.ToUpper(remainder), " NOT NULL") || strings.Contains(strings.ToUpper(remainder), " PRIMARY KEY"),
		Default: normalizeDefault(defaultExpr),
	}, true
}

func splitSQLStatements(schema string) []string {
	return splitTopLevel(stripLineComments(schema), ';')
}

func stripLineComments(schema string) string {
	lines := strings.Split(schema, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func splitTopLevel(input string, separator rune) []string {
	var (
		result   []string
		start    int
		depth    int
		inString bool
		prevRune rune
	)

	for idx, r := range input {
		switch r {
		case '\'':
			if prevRune != '\\' {
				inString = !inString
			}
		case '(':
			if !inString {
				depth++
			}
		case ')':
			if !inString && depth > 0 {
				depth--
			}
		}

		if r == separator && !inString && depth == 0 {
			part := strings.TrimSpace(input[start:idx])
			if part != "" {
				result = append(result, part)
			}
			start = idx + 1
		}

		prevRune = r
	}

	if tail := strings.TrimSpace(input[start:]); tail != "" {
		result = append(result, tail)
	}

	return result
}

func normalizeIdentifier(name string) string {
	name = strings.TrimSpace(strings.Trim(name, `"`))
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	return strings.Trim(name, `"`)
}

func readRuntimeSchema(t *testing.T, conn *sql.DB, dialect db.Dialect) map[string][]schemaColumn {
	t.Helper()

	tableNames := readRuntimeTableNames(t, conn, dialect)
	result := make(map[string][]schemaColumn, len(tableNames))

	for _, tableName := range tableNames {
		result[tableName] = readRuntimeColumns(t, conn, dialect, tableName)
	}

	return result
}

func readRuntimeTableNames(t *testing.T, conn *sql.DB, dialect db.Dialect) []string {
	t.Helper()

	var (
		rows *sql.Rows
		err  error
	)

	switch dialect {
	case db.DialectPostgres:
		rows, err = conn.Query(`
			SELECT table_name
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_type = 'BASE TABLE'
			  AND table_name <> 'goose_db_version'
			ORDER BY table_name
		`)
	default:
		rows, err = conn.Query(`
			SELECT name
			FROM sqlite_master
			WHERE type = 'table'
			  AND name NOT LIKE 'sqlite_%'
			  AND name <> 'goose_db_version'
			ORDER BY name
		`)
	}
	require.NoError(t, err)
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tableNames = append(tableNames, name)
	}
	require.NoError(t, rows.Err())

	return tableNames
}

func readRuntimeColumns(t *testing.T, conn *sql.DB, dialect db.Dialect, tableName string) []schemaColumn {
	t.Helper()

	switch dialect {
	case db.DialectPostgres:
		rows, err := conn.Query(`
			SELECT column_name, data_type, is_nullable, column_default, character_maximum_length
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = $1
			ORDER BY ordinal_position
		`, tableName)
		require.NoError(t, err)
		defer rows.Close()

		var columns []schemaColumn
		for rows.Next() {
			var (
				name          string
				dataType      string
				isNullable    string
				defaultValue  sql.NullString
				maxCharLength sql.NullInt64
			)
			require.NoError(t, rows.Scan(&name, &dataType, &isNullable, &defaultValue, &maxCharLength))
			columns = append(columns, schemaColumn{
				Name:    name,
				Type:    normalizeType(postgresColumnType(dataType, maxCharLength)),
				NotNull: isNullable == "NO",
				Default: normalizeDefault(defaultValue.String),
			})
		}
		require.NoError(t, rows.Err())
		return columns
	default:
		rows, err := conn.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, tableName))
		require.NoError(t, err)
		defer rows.Close()

		var columns []schemaColumn
		for rows.Next() {
			var (
				cid          int
				name         string
				dataType     string
				notNull      int
				defaultValue sql.NullString
				primaryKey   int
			)
			require.NoError(t, rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey))
			columns = append(columns, schemaColumn{
				Name:    name,
				Type:    normalizeType(dataType),
				NotNull: notNull == 1 || primaryKey > 0,
				Default: normalizeDefault(defaultValue.String),
			})
		}
		require.NoError(t, rows.Err())
		return columns
	}
}

func postgresColumnType(dataType string, maxCharLength sql.NullInt64) string {
	switch dataType {
	case "character varying":
		if maxCharLength.Valid {
			return fmt.Sprintf("varchar(%d)", maxCharLength.Int64)
		}
		return "varchar"
	case "character":
		if maxCharLength.Valid {
			return fmt.Sprintf("char(%d)", maxCharLength.Int64)
		}
		return "char"
	case "timestamp without time zone":
		return "timestamp"
	case "timestamp with time zone":
		return "timestamptz"
	default:
		return dataType
	}
}

func normalizeType(dataType string) string {
	dataType = strings.ToLower(strings.TrimSpace(dataType))
	dataType = strings.Join(strings.Fields(dataType), " ")
	dataType = strings.ReplaceAll(dataType, "character varying", "varchar")
	dataType = strings.ReplaceAll(dataType, "character(", "char(")
	dataType = strings.ReplaceAll(dataType, "character", "char")
	dataType = strings.ReplaceAll(dataType, "integer", "int")
	dataType = strings.ReplaceAll(dataType, "timestamp without time zone", "timestamp")
	dataType = strings.ReplaceAll(dataType, "timestamp with time zone", "timestamptz")
	return dataType
}

func normalizeDefault(defaultExpr string) string {
	defaultExpr = strings.TrimSpace(defaultExpr)
	for strings.HasPrefix(defaultExpr, "(") && strings.HasSuffix(defaultExpr, ")") {
		defaultExpr = strings.TrimSpace(defaultExpr[1 : len(defaultExpr)-1])
	}
	if idx := strings.Index(defaultExpr, "::"); idx >= 0 {
		defaultExpr = defaultExpr[:idx]
	}
	defaultExpr = strings.TrimSpace(defaultExpr)
	if defaultExpr == "" {
		return ""
	}
	if strings.HasPrefix(defaultExpr, "'") {
		return defaultExpr
	}
	return strings.ToLower(defaultExpr)
}

func toLogicalSchema(schema map[string][]schemaColumn) map[string][]logicalColumn {
	result := make(map[string][]logicalColumn, len(schema))

	for tableName, columns := range schema {
		for _, column := range columns {
			if tableName == "function_runs" && column.Name == "workspace_id" {
				continue
			}
			result[tableName] = append(result[tableName], logicalColumn{
				Name: column.Name,
				Type: logicalType(column.Name, column.Type),
			})
		}
		sort.Slice(result[tableName], func(i, j int) bool {
			return result[tableName][i].Name < result[tableName][j].Name
		})
	}

	return result
}

func logicalType(columnName, dataType string) string {
	switch {
	case isULIDColumn(columnName) && (dataType == "char(26)" || dataType == "blob" || dataType == "bytea"):
		return "ulid"
	case isUUIDLikeTextColumn(columnName) && (dataType == "char(36)" || dataType == "text"):
		return "uuid"
	}

	switch {
	case dataType == "uuid" || dataType == "char(36)":
		return "uuid"
	case dataType == "blob" || dataType == "bytea":
		return "bytes"
	case dataType == "json" || dataType == "jsonb":
		return "json"
	case dataType == "bool" || dataType == "boolean":
		return "bool"
	case dataType == "datetime" || dataType == "timestamp" || dataType == "timestamptz":
		return "timestamp"
	case strings.HasPrefix(dataType, "varchar") || dataType == "text":
		return "text"
	case dataType == "int" || dataType == "integer" || dataType == "bigint" || dataType == "smallint" || dataType == "uint64":
		return "int"
	default:
		return dataType
	}
}

func isULIDColumn(columnName string) bool {
	switch columnName {
	case "internal_id", "run_id", "event_id", "batch_id", "original_run_id", "id", "gateway_id":
		return true
	default:
		return false
	}
}

func isUUIDLikeTextColumn(columnName string) bool {
	switch columnName {
	case "debug_run_id", "debug_session_id":
		return true
	default:
		return false
	}
}
