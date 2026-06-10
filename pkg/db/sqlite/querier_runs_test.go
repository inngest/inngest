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
	endedAt := startedAt.Add(time.Second)
	upsertTestApp(ctx, t, q, appID)

	root := runSpan{
		RunID:        runID,
		EventIDs:     []ulid.ULID{eventID, batchEventID},
		BatchID:      batchID,
		AppID:        appID,
		FunctionID:   fnID,
		FunctionSlug: "event-runs-function",
		FunctionName: "Event Runs Function",
		Output:       []byte(`{"data":{"ok":true}}`),
		Cron:         "*/5 * * * *",
		StartedAt:    startedAt,
		EndedAt:      startedAt,
		Status:       enums.StepStatusQueued.String(),
	}
	require.NoError(t, insertRunSpan(ctx, q, root))
	require.NoError(t, extendRunSpan(ctx, q, root, endedAt, enums.StepStatusCompleted.String()))

	rows, err := q.GetRuns(ctx, db.GetRunsParams{EventID: eventID, Limit: 1, IncludeOutput: true})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
	require.Equal(t, eventID, rows[0].FunctionRun.EventID)
	require.Equal(t, batchID, rows[0].FunctionRun.BatchID)
	require.Equal(t, "cron", rows[0].FunctionRun.TriggerType)
	require.Equal(t, "*/5 * * * *", rows[0].FunctionRun.Cron.String)
	require.Equal(t, startedAt, rows[0].FunctionRun.RunStartedAt)
	require.Equal(t, "Completed", rows[0].FunctionFinish.Status.String)
	require.True(t, rows[0].FunctionFinish.CreatedAt.Valid)
	require.Equal(t, endedAt, rows[0].FunctionFinish.CreatedAt.Time)
	require.Equal(t, "event-runs-app", rows[0].AppName)
	require.Equal(t, "event-runs-function", rows[0].FunctionSlug)
	require.Equal(t, "Event Runs Function", rows[0].FunctionName)
	require.JSONEq(t, `{"data":{"ok":true}}`, string(rows[0].Output))

	rows, err = q.GetRuns(ctx, db.GetRunsParams{EventID: batchEventID, Limit: 1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "cron", rows[0].FunctionRun.TriggerType)
	require.Equal(t, runID, rows[0].FunctionRun.RunID)
	require.Empty(t, rows[0].Output)
}

func TestQuerierGetRunsPrefersFunctionOutputSpan(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	q := New(conn).Q()
	runID := ulid.Make()
	eventID := ulid.Make()
	appID := uuid.New()
	fnID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	childStartedAt := startedAt.Add(50 * time.Millisecond)
	endedAt := startedAt.Add(time.Second)
	upsertTestApp(ctx, t, q, appID)

	root := runSpan{
		RunID:        runID,
		EventIDs:     []ulid.ULID{eventID},
		AppID:        appID,
		FunctionID:   fnID,
		FunctionSlug: "event-runs-function",
		FunctionName: "Event Runs Function",
		StartedAt:    startedAt,
		EndedAt:      startedAt,
		Status:       enums.StepStatusQueued.String(),
	}
	require.NoError(t, insertRunSpan(ctx, q, root))
	require.NoError(t, extendRunSpan(ctx, q, root, endedAt, enums.StepStatusCompleted.String()))

	attrs, err := json.Marshal(map[string]any{
		meta.Attrs.IsFunctionOutput.Key(): true,
	})
	require.NoError(t, err)
	require.NoError(t, q.InsertSpan(ctx, db.InsertSpanParams{
		SpanID:       ulid.Make().String(),
		TraceID:      "trace-" + runID.String(),
		Name:         meta.SpanNameExecution,
		ParentSpanID: sql.NullString{String: "run-span", Valid: true},
		StartTime:    childStartedAt,
		EndTime:      endedAt.Add(-100 * time.Millisecond),
		RunID:        runID.String(),
		AccountID:    uuid.NewString(),
		AppID:        appID.String(),
		FunctionID:   fnID.String(),
		EnvID:        uuid.NewString(),
		Attributes:   attrs,
		Links:        []byte(`[]`),
		Output:       []byte(`{"data":{"body":"Hello, World!"}}`),
		Status:       sql.NullString{String: enums.StepStatusCompleted.String(), Valid: true},
		EventIds:     []byte(`[]`),
	}))

	//
	// a later step output without the function-output marker must not win
	require.NoError(t, q.InsertSpan(ctx, db.InsertSpanParams{
		SpanID:       ulid.Make().String(),
		TraceID:      "trace-" + runID.String(),
		Name:         meta.SpanNameExecution,
		ParentSpanID: sql.NullString{String: "run-span", Valid: true},
		StartTime:    childStartedAt,
		EndTime:      endedAt.Add(-50 * time.Millisecond),
		RunID:        runID.String(),
		AccountID:    uuid.NewString(),
		AppID:        appID.String(),
		FunctionID:   fnID.String(),
		EnvID:        uuid.NewString(),
		Attributes:   []byte(`{}`),
		Links:        []byte(`[]`),
		Output:       []byte(`{"data":{"step":"intermediate"}}`),
		Status:       sql.NullString{String: enums.StepStatusCompleted.String(), Valid: true},
		EventIds:     []byte(`[]`),
	}))

	rows, err := q.GetRuns(ctx, db.GetRunsParams{EventID: eventID, Limit: 1, IncludeOutput: true})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, startedAt, rows[0].FunctionRun.RunStartedAt)
	require.Equal(t, "Completed", rows[0].FunctionFinish.Status.String)
	require.True(t, rows[0].FunctionFinish.CreatedAt.Valid)
	require.Equal(t, endedAt, rows[0].FunctionFinish.CreatedAt.Time)
	require.JSONEq(t, `{"data":{"body":"Hello, World!"}}`, string(rows[0].Output))
}

func TestQuerierGetRunsIgnoresChildSpanStatus(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	q := New(conn).Q()
	runID := ulid.Make()
	eventID := ulid.Make()
	appID := uuid.New()
	fnID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	childStartedAt := startedAt.Add(50 * time.Millisecond)
	childEndedAt := startedAt.Add(time.Second)
	upsertTestApp(ctx, t, q, appID)

	root := runSpan{
		RunID:        runID,
		EventIDs:     []ulid.ULID{eventID},
		AppID:        appID,
		FunctionID:   fnID,
		FunctionSlug: "event-runs-function",
		FunctionName: "Event Runs Function",
		StartedAt:    startedAt,
		EndedAt:      startedAt,
		Status:       enums.StepStatusQueued.String(),
	}
	require.NoError(t, insertRunSpan(ctx, q, root))
	require.NoError(t, extendRunSpan(ctx, q, root, startedAt.Add(25*time.Millisecond), enums.StepStatusRunning.String()))

	//
	// a completed step span must not mark the whole run completed
	require.NoError(t, q.InsertSpan(ctx, db.InsertSpanParams{
		SpanID:       ulid.Make().String(),
		TraceID:      "trace-" + runID.String(),
		Name:         meta.SpanNameExecution,
		ParentSpanID: sql.NullString{String: "run-span", Valid: true},
		StartTime:    childStartedAt,
		EndTime:      childEndedAt,
		RunID:        runID.String(),
		AccountID:    uuid.NewString(),
		AppID:        appID.String(),
		FunctionID:   fnID.String(),
		EnvID:        uuid.NewString(),
		Attributes:   []byte(`{}`),
		Links:        []byte(`[]`),
		Output:       []byte(`{"data":{"step":"done"}}`),
		Status:       sql.NullString{String: enums.StepStatusCompleted.String(), Valid: true},
		EventIds:     []byte(`[]`),
	}))

	rows, err := q.GetRuns(ctx, db.GetRunsParams{EventID: eventID, Limit: 1, IncludeOutput: true})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "Running", rows[0].FunctionFinish.Status.String)
	require.False(t, rows[0].FunctionFinish.CreatedAt.Valid)
	require.JSONEq(t, `{"data":{"step":"done"}}`, string(rows[0].Output))
}

func TestQuerierGetRunsUsesLatestRootExtensionStatus(t *testing.T) {
	ctx := context.Background()
	conn, err := Open(ctx, Options{ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	q := New(conn).Q()
	runID := ulid.Make()
	eventID := ulid.Make()
	appID := uuid.New()
	fnID := uuid.New()
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	cancelledAt := startedAt.Add(time.Second)
	upsertTestApp(ctx, t, q, appID)

	root := runSpan{
		RunID:        runID,
		EventIDs:     []ulid.ULID{eventID},
		AppID:        appID,
		FunctionID:   fnID,
		FunctionSlug: "event-runs-function",
		FunctionName: "Event Runs Function",
		StartedAt:    startedAt,
		EndedAt:      startedAt,
		Status:       enums.StepStatusQueued.String(),
	}
	require.NoError(t, insertRunSpan(ctx, q, root))
	require.NoError(t, extendRunSpan(ctx, q, root, startedAt.Add(10*time.Millisecond), enums.StepStatusRunning.String()))
	require.NoError(t, extendRunSpan(ctx, q, root, cancelledAt, enums.StepStatusCancelled.String()))

	rows, err := q.GetRuns(ctx, db.GetRunsParams{EventID: eventID, Limit: 1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "Cancelled", rows[0].FunctionFinish.Status.String)
	require.True(t, rows[0].FunctionFinish.CreatedAt.Valid)
	require.Equal(t, cancelledAt, rows[0].FunctionFinish.CreatedAt.Time)
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

func upsertTestApp(ctx context.Context, t *testing.T, q db.Querier, appID uuid.UUID) {
	t.Helper()
	_, err := q.UpsertApp(ctx, db.UpsertAppParams{
		ID:          appID,
		Name:        "event-runs-app",
		SdkLanguage: "go",
		SdkVersion:  "1.0.0",
		Metadata:    "{}",
		Status:      "active",
		Checksum:    "checksum",
		Url:         "https://example.com/inngest",
		Method:      "POST",
	})
	require.NoError(t, err)
}

type runSpan struct {
	RunID        ulid.ULID
	EventIDs     []ulid.ULID
	BatchID      ulid.ULID
	AppID        uuid.UUID
	FunctionID   uuid.UUID
	FunctionSlug string
	FunctionName string
	Output       []byte
	Cron         string
	StartedAt    time.Time
	EndedAt      time.Time
	Status       string
}

// the exporter stores the run root as an executor.run row whose
// dynamic_span_id doubles as the grouping key for later EXTEND rows
func insertRunSpan(ctx context.Context, q db.Querier, span runSpan) error {
	attrs := map[string]any{
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
		SpanID:        runRootSpanID(span.RunID),
		TraceID:       "trace-" + span.RunID.String(),
		Name:          meta.SpanNameRun,
		StartTime:     span.StartedAt,
		EndTime:       span.EndedAt,
		RunID:         span.RunID.String(),
		AccountID:     uuid.NewString(),
		AppID:         span.AppID.String(),
		FunctionID:    span.FunctionID.String(),
		EnvID:         uuid.NewString(),
		DynamicSpanID: sql.NullString{String: runRootSpanID(span.RunID), Valid: true},
		Attributes:    attrBytes,
		Links:         []byte(`[]`),
		Output:        span.Output,
		Status:        sql.NullString{String: span.Status, Valid: span.Status != ""},
		EventIds:      eventIDBytes,
	})
}

// status transitions arrive as EXTEND rows sharing the root's dynamic_span_id
func extendRunSpan(ctx context.Context, q db.Querier, span runSpan, at time.Time, status string) error {
	return q.InsertSpan(ctx, db.InsertSpanParams{
		SpanID:        ulid.Make().String(),
		TraceID:       "trace-" + span.RunID.String(),
		Name:          meta.SpanNameDynamicExtension,
		StartTime:     at,
		EndTime:       at,
		RunID:         span.RunID.String(),
		AccountID:     uuid.NewString(),
		AppID:         span.AppID.String(),
		FunctionID:    span.FunctionID.String(),
		EnvID:         uuid.NewString(),
		DynamicSpanID: sql.NullString{String: runRootSpanID(span.RunID), Valid: true},
		Attributes:    []byte(`{}`),
		Links:         []byte(`[]`),
		Status:        sql.NullString{String: status, Valid: status != ""},
		EventIds:      []byte(`[]`),
	})
}

func runRootSpanID(runID ulid.ULID) string {
	return "run-span-" + runID.String()
}
