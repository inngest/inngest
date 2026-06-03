package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestQuerierGetRuns(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	q := New(conn).Q()
	runID := ulid.Make()
	eventID := ulid.Make()
	batchEventID := ulid.Make()
	batchID := ulid.Make()
	appID := uuid.New()
	fnID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, insertRunSpan(ctx, q, runSpan{
		RunID:        runID,
		EventIDs:     []ulid.ULID{eventID, batchEventID},
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

	rows, err := q.GetRuns(ctx, db.GetRunsParams{EventID: eventID, Limit: 1, IncludeOutput: true})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
	require.Equal(t, eventID, rows[0].FunctionRun.EventID)
	require.Equal(t, batchID, rows[0].FunctionRun.BatchID)
	require.Equal(t, "*/5 * * * *", rows[0].FunctionRun.Cron.String)
	require.Equal(t, "Completed", rows[0].FunctionFinish.Status.String)
	require.True(t, rows[0].FunctionFinish.CreatedAt.Valid)
	require.Equal(t, "event-runs-app", rows[0].AppName)
	require.Equal(t, "event-runs-function", rows[0].FunctionSlug)
	require.Equal(t, "Event Runs Function", rows[0].FunctionName)
	require.JSONEq(t, `{"data":{"ok":true}}`, string(rows[0].Output))

	rows, err = q.GetRuns(ctx, db.GetRunsParams{EventID: batchEventID, Limit: 1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
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

type runSpan struct {
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

func insertRunSpan(ctx context.Context, q db.Querier, span runSpan) error {
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
