package db_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db"
	dbpostgres "github.com/inngest/inngest/pkg/db/postgres"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/tests/testutil"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const envTestDatabase = "TEST_DATABASE"

// newTestAdapter creates a db.Adapter backed by either SQLite (default) or
// Postgres (when TEST_DATABASE=postgres). The returned cleanup function closes
// the database and, for Postgres, terminates the test container.
func newTestAdapter(t *testing.T) (db.Adapter, func()) {
	t.Helper()

	if os.Getenv(envTestDatabase) == "postgres" {
		pc, err := testutil.StartPostgres(t)
		require.NoError(t, err)

		conn, err := dbpostgres.Open(t.Context(), dbpostgres.Options{
			URI:     pc.URI,
			ForTest: true,
		})
		require.NoError(t, err)

		adapter := dbpostgres.New(conn)
		return adapter, func() {
			conn.Close()
			_ = pc.Terminate(t.Context())
		}
	}

	conn, err := dbsqlite.Open(t.Context(), dbsqlite.Options{
		Persist: false,
		ForTest: true,
	})
	require.NoError(t, err)

	adapter := dbsqlite.New(conn)
	return adapter, func() { conn.Close() }
}

// ---------------------------------------------------------------------------
// Adapter contract tests
// ---------------------------------------------------------------------------

func TestAdapterDialect(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	d := adapter.Dialect()
	switch os.Getenv(envTestDatabase) {
	case "postgres":
		assert.Equal(t, db.DialectPostgres, d)
	default:
		assert.Equal(t, db.DialectSQLite, d)
	}
}

func TestAdapterConn(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	conn := adapter.Conn()
	require.NotNil(t, conn)
	require.NoError(t, conn.Ping())
}

func TestAdapterWithTx(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()

	txAdapter, err := adapter.WithTx(ctx)
	require.NoError(t, err)
	require.NotNil(t, txAdapter)

	// TxAdapter should also report the same dialect
	assert.Equal(t, adapter.Dialect(), txAdapter.Dialect())

	require.NoError(t, txAdapter.Rollback(ctx))
}

// ---------------------------------------------------------------------------
// App CRUD round-trip
// ---------------------------------------------------------------------------

func TestUpsertAppRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()
	appID := uuid.New()

	// Insert
	app, err := q.UpsertApp(ctx, db.UpsertAppParams{
		ID:          appID,
		Name:        "test-app",
		SdkLanguage: "go",
		SdkVersion:  "1.0.0",
		Framework:   sql.NullString{String: "gin", Valid: true},
		Metadata:    `{"env":"test"}`,
		Status:      "active",
		Checksum:    "checksum-1",
		Url:         "https://example.com/api/inngest",
		Method:      "POST",
	})
	require.NoError(t, err)
	require.NotNil(t, app)
	assert.Equal(t, appID, app.ID)
	assert.Equal(t, "test-app", app.Name)
	assert.False(t, app.CreatedAt.IsZero(), "insert must populate created_at")

	// Read back
	got, err := q.GetAppByID(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, "test-app", got.Name)
	assert.Equal(t, "go", got.SdkLanguage)
	assert.Equal(t, app.CreatedAt.UTC(), got.CreatedAt.UTC(), "created_at must round-trip unchanged")

	// Update via upsert
	updated, err := q.UpsertApp(ctx, db.UpsertAppParams{
		ID:          appID,
		Name:        "test-app-v2",
		SdkLanguage: "go",
		SdkVersion:  "2.0.0",
		Framework:   sql.NullString{String: "gin", Valid: true},
		Metadata:    `{}`,
		Status:      "active",
		Checksum:    "checksum-2",
		Url:         "https://example.com/api/inngest",
		Method:      "POST",
	})
	require.NoError(t, err)
	assert.Equal(t, "test-app-v2", updated.Name)
	assert.Equal(t, app.CreatedAt.UTC(), updated.CreatedAt.UTC(), "created_at must be preserved on update")

	// Delete (soft-delete: sets archived_at)
	err = q.DeleteApp(ctx, appID)
	require.NoError(t, err)

	archived, err := q.GetAppByID(ctx, appID)
	require.NoError(t, err)
	assert.True(t, archived.ArchivedAt.Valid, "deleted app should have archived_at set")
}

// ---------------------------------------------------------------------------
// Function CRUD round-trip
// ---------------------------------------------------------------------------

func TestInsertFunctionRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()

	appID := uuid.New()
	_, err := q.UpsertApp(ctx, db.UpsertAppParams{
		ID: appID, Name: "fn-test-app", SdkLanguage: "go", SdkVersion: "1.0.0",
		Metadata: "{}", Status: "active", Checksum: "c", Url: "http://x", Method: "POST",
	})
	require.NoError(t, err)

	fnID := uuid.New()
	fn, err := q.UpsertFunction(ctx, db.UpsertFunctionParams{
		ID:        fnID,
		AppID:     appID,
		Name:      "my-function",
		Slug:      "my-function-slug",
		Config:    `{"retries":{"attempts":3}}`,
		CreatedAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	assert.Equal(t, fnID, fn.ID)
	assert.Equal(t, "my-function", fn.Name)
	assert.False(t, fn.ArchivedAt.Valid)

	got, err := q.GetFunctionByID(ctx, fnID)
	require.NoError(t, err)
	assert.Equal(t, "my-function", got.Name)
	assert.JSONEq(t, `{"retries":{"attempts":3}}`, string(got.Config))
	assert.False(t, got.ArchivedAt.Valid)

	fns, err := q.GetAppFunctions(ctx, appID)
	require.NoError(t, err)
	assert.Len(t, fns, 1)

	allFns, err := q.GetFunctions(ctx)
	require.NoError(t, err)
	assert.Len(t, allFns, 1)
}

func TestGetRuns(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()

	appID := uuid.New()
	fnID := uuid.New()
	eventID := ulid.Make()
	firstBatchEventID := ulid.Make()
	thirdBatchEventID := ulid.Make()
	batchID := ulid.Make()
	runID := ulid.Make()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, insertRunListSpan(ctx, q, runListSpan{
		RunID:        runID,
		EventIDs:     []ulid.ULID{firstBatchEventID, eventID, thirdBatchEventID},
		BatchID:      batchID,
		AppID:        appID,
		FunctionID:   fnID,
		AppName:      "event-runs-app",
		FunctionSlug: "event-runs-function",
		FunctionName: "Event Runs Function",
		Output:       []byte(`{"data":{"ok":true}}`),
		Cron:         "*/5 * * * *",
		StartedAt:    startedAt,
		EndedAt:      startedAt.Add(time.Second),
		Status:       enums.StepStatusCompleted.String(),
	}))

	rows, err := q.GetRuns(ctx, db.GetRunsParams{
		EventID:       eventID,
		Limit:         10,
		IncludeOutput: true,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, runID, rows[0].FunctionRun.RunID)
	assert.Equal(t, batchID, rows[0].FunctionRun.BatchID)
	assert.Equal(t, "*/5 * * * *", rows[0].FunctionRun.Cron.String)
	assert.Equal(t, "event-runs-app", rows[0].AppName)
	assert.Equal(t, "event-runs-function", rows[0].FunctionSlug)
	assert.Equal(t, "Event Runs Function", rows[0].FunctionName)
	assert.Equal(t, "Completed", rows[0].FunctionFinish.Status.String)
	assert.JSONEq(t, `{"data":{"ok":true}}`, string(rows[0].Output))

	for _, batchEventID := range []ulid.ULID{firstBatchEventID, eventID, thirdBatchEventID} {
		rows, err := q.GetRuns(ctx, db.GetRunsParams{
			EventID: batchEventID,
			Limit:   10,
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, runID, rows[0].FunctionRun.RunID)
	}
}

type runListSpan struct {
	RunID        ulid.ULID
	EventIDs     []ulid.ULID
	BatchID      ulid.ULID
	AppID        uuid.UUID
	FunctionID   uuid.UUID
	AppName      string
	FunctionSlug string
	FunctionName string
	Output       []byte
	Cron         string
	StartedAt    time.Time
	EndedAt      time.Time
	Status       string
}

func insertRunListSpan(ctx context.Context, q db.Querier, span runListSpan) error {
	attrs := map[string]any{
		meta.Attrs.AppName.Key():      span.AppName,
		meta.Attrs.FunctionSlug.Key(): span.FunctionSlug,
		meta.Attrs.FunctionName.Key(): span.FunctionName,
	}
	if !span.BatchID.IsZero() {
		attrs[meta.Attrs.BatchID.Key()] = span.BatchID.String()
	}
	if span.Cron != "" {
		attrs[meta.Attrs.CronSchedule.Key()] = span.Cron
	}
	attrBytes, _ := json.Marshal(attrs)

	eventIDs := make([]string, len(span.EventIDs))
	for i, id := range span.EventIDs {
		eventIDs[i] = id.String()
	}
	eventIDBytes, _ := json.Marshal(eventIDs)

	return q.InsertSpan(ctx, db.InsertSpanParams{
		SpanID:     ulid.Make().String(),
		TraceID:    "trace-" + span.RunID.String(),
		Name:       meta.SpanNameRun,
		StartTime:  span.StartedAt,
		EndTime:    span.EndedAt,
		RunID:      span.RunID.String(),
		AccountID:  uuid.NewString(),
		AppID:      span.AppID.String(),
		FunctionID: span.FunctionID.String(),
		EnvID:      uuid.NewString(),
		Attributes: attrBytes,
		Links:      []byte(`[]`),
		Output:     span.Output,
		Status:     sql.NullString{String: span.Status, Valid: span.Status != ""},
		EventIds:   eventIDBytes,
	})
}

// ---------------------------------------------------------------------------
// Event insert and read
// ---------------------------------------------------------------------------

func TestInsertEventRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()
	now := time.Now().UTC()

	eventID := ulid.Make()
	err := q.InsertEvent(ctx, db.InsertEventParams{
		InternalID: eventID,
		EventID:    "test-event-id",
		EventName:  "test/event",
		EventData:  `{"key":"value"}`,
		EventUser:  `{}`,
		EventTs:    now,
		ReceivedAt: now,
	})
	require.NoError(t, err)

	got, err := q.GetEventByInternalID(ctx, eventID)
	require.NoError(t, err)
	assert.Equal(t, "test/event", got.EventName)
	assert.Equal(t, "test-event-id", got.EventID)
	assert.JSONEq(t, `{"key":"value"}`, got.EventData)
	assert.JSONEq(t, `{}`, got.EventUser)
	assert.False(t, got.AccountID.Valid)
	assert.False(t, got.WorkspaceID.Valid)
	assert.WithinDuration(t, now, got.ReceivedAt, time.Second)

	scopedEventID := ulid.Make()
	accountID := uuid.New()
	workspaceID := uuid.New()
	insertEventWithScopes(t, ctx, adapter, scopedEventID, accountID, workspaceID)

	scoped, err := q.GetEventByInternalID(ctx, scopedEventID)
	require.NoError(t, err)
	assert.Equal(t, accountID.String(), scoped.AccountID.String)
	assert.Equal(t, workspaceID.String(), scoped.WorkspaceID.String)

	batch, err := q.GetEventsByInternalIDs(ctx, []ulid.ULID{eventID, scopedEventID})
	require.NoError(t, err)
	assert.Len(t, batch, 2)
}

// ---------------------------------------------------------------------------
// Span insert + query (BLOB-in-JSON regression)
// ---------------------------------------------------------------------------

func TestInsertSpanRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()

	spanID := ulid.Make().String()
	traceID := ulid.Make().String()
	runID := ulid.Make().String()
	accountID := uuid.New().String()

	err := q.InsertSpan(ctx, db.InsertSpanParams{
		SpanID:     spanID,
		TraceID:    traceID,
		Name:       "executor.run",
		StartTime:  time.Now().UTC(),
		EndTime:    time.Now().UTC().Add(100 * time.Millisecond),
		RunID:      runID,
		AccountID:  accountID,
		AppID:      uuid.New().String(),
		FunctionID: uuid.New().String(),
		EnvID:      uuid.New().String(),
		Attributes: []byte(`{"sdk.language":"go"}`),
		Links:      []byte(`[]`),
		Output:     []byte(`{"data":{"num":42}}`),
		Input:      []byte(`{"events":[{}]}`),
		Status:     sql.NullString{String: "completed", Valid: true},
		EventIds:   []byte(`["event-1"]`),
		DynamicSpanID: sql.NullString{
			String: "dyn-1",
			Valid:  true,
		},
	})
	require.NoError(t, err)

	// GetSpansByRunID uses json_group_array(json_object('attributes', attributes, ...))
	// which fails with "JSON cannot hold BLOB values" if []byte is stored as BLOB.
	spans, err := q.GetSpansByRunID(ctx, runID)
	require.NoError(t, err, "query must not fail with 'JSON cannot hold BLOB values'")
	require.Len(t, spans, 1)

	var fragments []map[string]any
	require.NoError(t, json.Unmarshal(spans[0].SpanFragments, &fragments))
	require.Len(t, fragments, 1)
	assert.Equal(t, spanID, fragments[0]["span_id"])

	// Verify output is readable (not double-encoded)
	outputs, err := q.GetSpanOutput(ctx, []string{spanID})
	require.NoError(t, err)
	require.Len(t, outputs, 1)

	var parsed map[string]any
	err = json.Unmarshal(outputs[0].Output, &parsed)
	require.NoError(t, err, "output must be valid JSON, not double-encoded")
	assert.Contains(t, parsed, "data")

	runSpan, err := q.GetRunSpanByRunID(ctx, db.GetRunSpanByRunIDParams{
		RunID:     runID,
		AccountID: accountID,
	})
	require.NoError(t, err)
	assert.Equal(t, traceID, runSpan.TraceID)

	spanRow, err := q.GetSpanBySpanID(ctx, db.GetSpanBySpanIDParams{
		RunID:     runID,
		SpanID:    spanID,
		AccountID: accountID,
	})
	require.NoError(t, err)
	assert.Equal(t, traceID, spanRow.TraceID)

	status, eventIDs := readStoredSpanState(t, ctx, adapter, traceID, spanID)
	assert.Equal(t, "completed", status)
	assert.Contains(t, eventIDs, "event-1")
}

// ---------------------------------------------------------------------------
// Transaction commit + rollback
// ---------------------------------------------------------------------------

func TestQuerierTransaction(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("commit persists data", func(t *testing.T) {
		commitAppID := uuid.New()

		txAdapter, err := adapter.WithTx(ctx)
		require.NoError(t, err)

		_, err = txAdapter.Q().UpsertApp(ctx, db.UpsertAppParams{
			ID: commitAppID, Name: "tx-app", SdkLanguage: "go", SdkVersion: "1.0.0",
			Metadata: "{}", Status: "active", Checksum: "c", Url: "http://x", Method: "POST",
		})
		require.NoError(t, err)

		require.NoError(t, txAdapter.Commit(ctx))

		got, err := adapter.Q().GetAppByID(ctx, commitAppID)
		require.NoError(t, err)
		assert.Equal(t, "tx-app", got.Name)
	})

	t.Run("rollback discards data", func(t *testing.T) {
		rollbackAppID := uuid.New()

		txAdapter, err := adapter.WithTx(ctx)
		require.NoError(t, err)

		_, err = txAdapter.Q().UpsertApp(ctx, db.UpsertAppParams{
			ID: rollbackAppID, Name: "rolled-back-app", SdkLanguage: "go", SdkVersion: "1.0.0",
			Metadata: "{}", Status: "active", Checksum: "c", Url: "http://x", Method: "POST",
		})
		require.NoError(t, err)

		require.NoError(t, txAdapter.Rollback(ctx))

		_, err = adapter.Q().GetAppByID(ctx, rollbackAppID)
		require.Error(t, err, "rolled-back app should not exist")
	})
}

// ---------------------------------------------------------------------------
// History insert + read
// ---------------------------------------------------------------------------

func TestInsertHistoryRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()

	runID := ulid.Make()
	historyID := ulid.Make()
	now := time.Now().UTC()

	err := q.InsertHistory(ctx, db.InsertHistoryParams{
		ID:              historyID,
		CreatedAt:       now,
		RunStartedAt:    now.Add(-time.Second),
		FunctionID:      uuid.New(),
		FunctionVersion: 1,
		RunID:           runID,
		EventID:         ulid.Make(),
		GroupID:         sql.NullString{String: "group-a", Valid: true},
		Type:            "FunctionStarted",
		Attempt:         0,
		IdempotencyKey:  ulid.Make().String(),
		StepName:        sql.NullString{String: "fetch", Valid: true},
		StepID:          sql.NullString{String: "step-1", Valid: true},
		StepType:        sql.NullString{String: "step", Valid: true},
		Url:             sql.NullString{String: "https://example.com/step", Valid: true},
		Result:          sql.NullString{String: `{"ok":true}`, Valid: true},
	})
	require.NoError(t, err)

	got, err := q.GetHistoryItem(ctx, historyID)
	require.NoError(t, err)
	assert.Equal(t, "FunctionStarted", got.Type)
	assert.Equal(t, runID, got.RunID)
	assert.Equal(t, "group-a", got.GroupID.String)
	assert.Equal(t, "fetch", got.StepName.String)
	assert.Equal(t, "step", got.StepType.String)
	assert.Equal(t, "https://example.com/step", got.Url.String)
	assert.JSONEq(t, `{"ok":true}`, got.Result.String)

	items, err := q.GetFunctionRunHistory(ctx, runID)
	require.NoError(t, err)
	assert.Len(t, items, 1)
}
