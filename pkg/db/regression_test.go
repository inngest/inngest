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
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	"github.com/inngest/inngest/pkg/db"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
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

func TestSchemaColumnsMatchSqlc(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	actual := readRuntimeSchema(t, adapter.Conn(), adapter.Dialect())
	expected := readExpectedSchema(t, adapter.Dialect())

	require.Equal(t, expected, actual)
}

func TestCrossDialectSchemaParity(t *testing.T) {
	sqliteSchema := toLogicalSchema(readExpectedSchema(t, db.DialectSQLite))
	postgresSchema := toLogicalSchema(readExpectedSchema(t, db.DialectPostgres))

	require.Equal(t, postgresSchema, sqliteSchema)
}

func TestSQLiteDefaultValues(t *testing.T) {
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

		_, err := conn.ExecContext(ctx, `INSERT INTO function_finishes (run_id) VALUES (?)`, runID[:])
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

func TestSQLiteNotNullConstraints(t *testing.T) {
	adapter, cleanup := newSQLiteTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	conn := adapter.Conn()

	testCases := []struct {
		name string
		sql  string
		args []any
	}{
		{
			name: "apps.name",
			sql: `
				INSERT INTO apps (id, sdk_language, sdk_version, status, checksum, url)
				VALUES (?, ?, ?, ?, ?, ?)
			`,
			args: []any{uuid.New().String(), "go", "1.0.0", "active", "missing-name", "https://example.com"},
		},
		{
			name: "events.event_id",
			sql: `
				INSERT INTO events (internal_id, event_name, event_ts)
				VALUES (?, ?, ?)
			`,
			args: []any{ulid.Make().String(), "app/missing-event-id", time.Now().UTC()},
		},
		{
			name: "function_runs.event_id",
			sql: `
				INSERT INTO function_runs (run_id, function_version)
				VALUES (?, ?)
			`,
			args: []any{ulid.Make().String(), 1},
		},
		{
			name: "trace_runs.account_id",
			sql: `
				INSERT INTO trace_runs (
					run_id, workspace_id, app_id, function_id, trace_id,
					queued_at, started_at, ended_at, status, source_id, trigger_ids, is_debounce
				)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
			args: []any{
				ulid.Make().String(),
				uuid.New().String(),
				uuid.New().String(),
				uuid.New().String(),
				[]byte("trace-not-null"),
				1, 2, 3, 1,
				"source-not-null",
				[]byte(`["evt-not-null"]`),
				false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := conn.ExecContext(ctx, tc.sql, tc.args...)
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "not null")
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

func readExpectedSchema(t *testing.T, dialect db.Dialect) map[string][]schemaColumn {
	t.Helper()

	path := filepath.Join("..", "cqrs", "base_cqrs", "sqlc", string(dialect), "schema.sql")
	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	return parseSchemaColumns(t, string(contents))
}

func parseSchemaColumns(t *testing.T, contents string) map[string][]schemaColumn {
	t.Helper()

	result := map[string][]schemaColumn{}
	currentTable := ""

	for _, rawLine := range strings.Split(contents, "\n") {
		line := stripLineComment(rawLine)
		if line == "" {
			continue
		}

		if currentTable == "" {
			if !strings.HasPrefix(strings.ToUpper(line), "CREATE TABLE ") {
				continue
			}

			currentTable = parseCreateTableName(t, line)
			result[currentTable] = nil
			continue
		}

		if line == ");" {
			currentTable = ""
			continue
		}

		column, ok := parseSchemaColumnLine(line)
		if !ok {
			continue
		}
		result[currentTable] = append(result[currentTable], column)
	}

	return result
}

func parseCreateTableName(t *testing.T, line string) string {
	t.Helper()

	remainder := strings.TrimSpace(line[len("CREATE TABLE "):])
	if strings.HasPrefix(strings.ToUpper(remainder), "IF NOT EXISTS ") {
		remainder = strings.TrimSpace(remainder[len("IF NOT EXISTS "):])
	}

	idx := strings.Index(remainder, "(")
	require.NotEqual(t, -1, idx, "invalid CREATE TABLE statement: %q", line)

	return strings.TrimSpace(strings.Trim(remainder[:idx], `"`))
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

func stripLineComment(line string) string {
	if idx := strings.Index(line, "--"); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(line)
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
	}

	switch {
	case dataType == "uuid" || dataType == "char(36)":
		return "uuid"
	case dataType == "blob" || dataType == "bytea":
		return "bytes"
	case dataType == "json" || dataType == "jsonb":
		return "json"
	case dataType == "datetime" || dataType == "timestamp" || dataType == "timestamptz":
		return "timestamp"
	case strings.HasPrefix(dataType, "varchar") || dataType == "text":
		return "text"
	case dataType == "int" || dataType == "integer" || dataType == "bigint" || dataType == "smallint":
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
