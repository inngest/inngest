package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestQuerierGetRuns(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	q := New(conn).Q()
	appID := uuid.New()
	_, err = q.UpsertApp(ctx, db.UpsertAppParams{
		ID: appID, Name: "event-runs-app", SdkLanguage: "go", SdkVersion: "1.0.0",
		Metadata: "{}", Status: "active", Checksum: "checksum", Url: "http://example.com", Method: "POST",
	})
	require.NoError(t, err)

	fnID := uuid.New()
	_, err = q.UpsertFunction(ctx, db.UpsertFunctionParams{
		ID:        fnID,
		AppID:     appID,
		Name:      "Event Runs Function",
		Slug:      "event-runs-app-event-runs-function",
		Config:    `{"name":"Event Runs Function","slug":"event-runs-function"}`,
		CreatedAt: time.Now().UTC(),
	})
	require.NoError(t, err)

	eventID := ulid.Make()
	runID := ulid.Make()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, q.InsertFunctionRun(ctx, db.InsertFunctionRunParams{
		RunID:           runID,
		RunStartedAt:    startedAt,
		FunctionID:      fnID,
		FunctionVersion: 1,
		TriggerType:     "event",
		EventID:         eventID,
		WorkspaceID:     uuid.New(),
	}))
	require.NoError(t, q.InsertFunctionFinish(ctx, db.InsertFunctionFinishParams{
		RunID:              runID,
		Status:             sql.NullString{String: "completed", Valid: true},
		Output:             sql.NullString{String: "", Valid: true},
		CompletedStepCount: sql.NullInt64{Int64: 1, Valid: true},
		CreatedAt:          sql.NullTime{Time: startedAt.Add(time.Second), Valid: true},
	}))
	require.NoError(t, q.InsertTraceRun(ctx, db.InsertTraceRunParams{
		RunID:       runID,
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       appID,
		FunctionID:  fnID,
		TraceID:     []byte("trace-id"),
		QueuedAt:    startedAt.UnixMilli(),
		StartedAt:   startedAt.UnixMilli(),
		EndedAt:     startedAt.Add(time.Second).UnixMilli(),
		Status:      300,
		SourceID:    "test",
		TriggerIds:  []byte(eventID.String()),
		Output:      []byte(`{"data":{"ok":true}}`),
	}))

	rows, err := q.GetRuns(ctx, db.GetRunsParams{EventID: eventID, Limit: 1, IncludeOutput: true})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
	require.Equal(t, "completed", rows[0].FunctionFinish.Status.String)
	require.Equal(t, "event-runs-app", rows[0].AppName)
	require.JSONEq(t, `{"data":{"ok":true}}`, string(rows[0].Output))

	rows, err = q.GetRuns(ctx, db.GetRunsParams{EventID: eventID, Limit: 1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Empty(t, rows[0].Output)
}

func TestQuerierGetRunsError(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{ForTest: true})
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	_, err = New(conn).Q().GetRuns(ctx, db.GetRunsParams{
		EventID: ulid.Make(),
		Limit:   1,
	})
	require.Error(t, err)
}
