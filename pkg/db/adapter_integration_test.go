package db_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	"github.com/inngest/inngest/pkg/db"
	dbpostgres "github.com/inngest/inngest/pkg/db/postgres"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
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

		conn, err := base_cqrs.New(t.Context(), base_cqrs.BaseCQRSOptions{
			PostgresURI: pc.URI,
			ForTest:     true,
		})
		require.NoError(t, err)

		adapter := dbpostgres.New(conn)
		return adapter, func() {
			conn.Close()
			_ = pc.Terminate(t.Context())
		}
	}

	conn, err := base_cqrs.New(t.Context(), base_cqrs.BaseCQRSOptions{
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

func TestQuerierAppRoundTrip(t *testing.T) {
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

	// Read back
	got, err := q.GetAppByID(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, "test-app", got.Name)
	assert.Equal(t, "go", got.SdkLanguage)

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

func TestQuerierFunctionRoundTrip(t *testing.T) {
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
	fn, err := q.InsertFunction(ctx, db.InsertFunctionParams{
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

	got, err := q.GetFunctionByID(ctx, fnID)
	require.NoError(t, err)
	assert.Equal(t, "my-function", got.Name)
	assert.JSONEq(t, `{"retries":{"attempts":3}}`, string(got.Config))

	fns, err := q.GetAppFunctions(ctx, appID)
	require.NoError(t, err)
	assert.Len(t, fns, 1)
}

// ---------------------------------------------------------------------------
// Event insert and read
// ---------------------------------------------------------------------------

func TestQuerierEventRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()

	eventID := ulid.Make()
	err := q.InsertEvent(ctx, db.InsertEventParams{
		InternalID: eventID,
		EventID:    "test-event-id",
		EventName:  "test/event",
		EventData:  `{"key":"value"}`,
		EventUser:  `{}`,
		EventTs:    time.Now().UTC(),
		ReceivedAt: time.Now().UTC(),
	})
	require.NoError(t, err)

	got, err := q.GetEventByInternalID(ctx, eventID)
	require.NoError(t, err)
	assert.Equal(t, "test/event", got.EventName)
	assert.Equal(t, "test-event-id", got.EventID)
}

// ---------------------------------------------------------------------------
// Span insert + query (BLOB-in-JSON regression)
// ---------------------------------------------------------------------------

func TestQuerierSpanRoundTrip(t *testing.T) {
	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	ctx := context.Background()
	q := adapter.Q()

	spanID := ulid.Make().String()
	traceID := ulid.Make().String()
	runID := ulid.Make().String()

	err := q.InsertSpan(ctx, db.InsertSpanParams{
		SpanID:     spanID,
		TraceID:    traceID,
		Name:       "executor.run",
		StartTime:  time.Now().UTC(),
		EndTime:    time.Now().UTC().Add(100 * time.Millisecond),
		RunID:      runID,
		AccountID:  uuid.New().String(),
		AppID:      uuid.New().String(),
		FunctionID: uuid.New().String(),
		EnvID:      uuid.New().String(),
		Attributes: []byte(`{"sdk.language":"go"}`),
		Links:      []byte(`[]`),
		Output:     []byte(`{"data":{"num":42}}`),
		Input:      []byte(`{"events":[{}]}`),
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

	// Verify output is readable (not double-encoded)
	outputs, err := q.GetSpanOutput(ctx, runID, []string{spanID})
	require.NoError(t, err)
	require.Len(t, outputs, 1)

	var parsed map[string]any
	err = json.Unmarshal(outputs[0].Output, &parsed)
	require.NoError(t, err, "output must be valid JSON, not double-encoded")
	assert.Contains(t, parsed, "data")
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

func TestQuerierHistoryRoundTrip(t *testing.T) {
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
		Type:            "FunctionStarted",
		Attempt:         0,
		IdempotencyKey:  ulid.Make().String(),
	})
	require.NoError(t, err)

	got, err := q.GetHistoryItem(ctx, historyID)
	require.NoError(t, err)
	assert.Equal(t, "FunctionStarted", got.Type)
	assert.Equal(t, runID, got.RunID)

	items, err := q.GetFunctionRunHistory(ctx, runID)
	require.NoError(t, err)
	assert.Len(t, items, 1)
}
