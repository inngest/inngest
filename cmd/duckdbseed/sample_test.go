package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBuildTemplatesFallsBackToDefaultsWhenSourceIsEmpty(t *testing.T) {
	_, db := newTestDB(t, 1)

	tmpl, sampled, err := BuildTemplates(t.Context(), db, 200)
	require.NoError(t, err)
	require.False(t, sampled, "an empty source must report that it fell back to defaults")
	require.NotEmpty(t, tmpl.Tenants, "an empty source must still yield at least one usable tenant")
	require.NotEmpty(t, tmpl.Statuses)
}

func TestSampleTemplatesReadsRealRowsAsTemplates(t *testing.T) {
	_, db := newTestDB(t, 1)
	ctx := t.Context()

	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	runID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	now := time.Now().UTC()

	eventInternalID := "01ARZ3NDEKTSV4RRFFQ69G5FAX"

	startedAt := now.Add(100 * time.Millisecond)
	endedAt := now.Add(time.Second)
	receivedAt := now.Add(-200 * time.Millisecond)

	_, err := db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, started_at, ended_at, app_id, function_id, status, inputs, output, event_ids)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, now, startedAt, endedAt, appID.String(), functionID.String(),
		"Completed", `{"event":{"name":"app/one"}}`, `{"data":"ok"}`, []string{eventInternalID},
	)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO inngest.run_trace_spans (account_id, env_id, run_id, run_queued_at, app_id, function_id, name, start_time, end_time, trace_id, span_id, attributes, output, input)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, now, appID.String(), functionID.String(),
		"my-function", startedAt, endedAt, "trace-1", "span-1", `{"sys.step.name":"my-function"}`, `{"data":"ok"}`, `{"event":{"name":"app/one"}}`,
	)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts)
		 VALUES (?, ?, ?, ?, 'test', ?, ?, ?, '1', ?);`,
		accountID.String(), envID.String(), eventInternalID, receivedAt, eventInternalID, "app/one", `{"k":"v"}`, receivedAt,
	)
	require.NoError(t, err)

	tmpl, err := SampleTemplates(ctx, db, 200)
	require.NoError(t, err)

	require.Len(t, tmpl.Tenants, 1)
	require.Equal(t, accountID, tmpl.Tenants[0].AccountID)
	require.Equal(t, envID, tmpl.Tenants[0].EnvID)
	require.Equal(t, appID, tmpl.Tenants[0].AppID)
	require.Equal(t, functionID, tmpl.Tenants[0].FunctionID)

	require.Contains(t, tmpl.Statuses, "Completed")
	require.Contains(t, tmpl.Inputs, `{"event":{"name":"app/one"}}`)

	require.Len(t, tmpl.Traces, 1)
	trace := tmpl.Traces[0]
	require.Equal(t, "trace-1", trace.TraceID)
	require.Len(t, trace.Spans, 1)
	root := trace.Spans[0]
	require.Equal(t, "span-1", root.SpanID)
	require.Nil(t, root.ParentSpanID)
	require.Equal(t, "my-function", root.Name)
	require.Equal(t, `{"sys.step.name":"my-function"}`, root.Attributes)
	require.Equal(t, `{"data":"ok"}`, root.Output)
	require.Equal(t, startedAt.Sub(now), root.StartOffset, "span offsets are relative to the run's own queued_at")
	require.Equal(t, endedAt.Sub(now), root.EndOffset)
	require.NotNil(t, trace.StartedOffset)
	require.Equal(t, startedAt.Sub(now), *trace.StartedOffset)
	require.NotNil(t, trace.EndedOffset)
	require.Equal(t, endedAt.Sub(now), *trace.EndedOffset)

	require.Len(t, tmpl.EventProfiles, 1)
	require.Equal(t, []EventTemplate{{Name: "app/one", Data: `{"k":"v"}`, Offset: receivedAt.Sub(now)}}, tmpl.EventProfiles[0])
}

// TestSampleTemplatesPreservesRealSpanIDsAndParentsVerbatim proves the fix
// for a real bug affecting every run sampled from an actual `inngest dev`
// database: sampleTraces used to try to detect which span was "the root"
// by checking parent_span_id IS NULL -- but a real root span
// ("executor.run") is created with no OTel parent, and OTel's zero SpanID
// stringifies to the literal "0000000000000000", never SQL NULL or "", so
// that check never matched any real run's root at all, silently dropping
// every sampled trace (falling back to the 2-span defaults). The fix
// removes root detection entirely: SpanTemplate now carries each
// span_id/parent_span_id completely unchanged, so the exact same
// relationship (child's parent_span_id text matching root's span_id text)
// still holds after replay without this package ever needing to resolve
// it itself. This seeds a run shaped exactly like real dualwrite output:
// the root's own parent_span_id is the literal OTel zero-SpanID sentinel,
// and every other span's parent_span_id is the deterministic value that
// equals the root's own span_id (mirroring pkg/tracing/util.go's
// RunSpanRefFromMetadata).
func TestSampleTemplatesPreservesRealSpanIDsAndParentsVerbatim(t *testing.T) {
	_, db := newTestDB(t, 1)
	ctx := t.Context()

	accountID, envID := uuid.New(), uuid.New()
	appID, functionID := uuid.New(), uuid.New()
	runID := "01ARZ3NDEKTSV4RRFFQ69G5FBB"
	now := time.Now().UTC()

	_, err := db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, app_id, function_id, status, inputs, output)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, now, appID.String(), functionID.String(),
		"Completed", `{}`, `{}`,
	)
	require.NoError(t, err)

	const rootSpanID = "deadbeefdeadbeef" // stands in for the real deterministic run-derived span ID
	insertSpan := func(spanID, parentSpanID string, start time.Time) {
		_, err := db.ExecContext(ctx,
			`INSERT INTO inngest.run_trace_spans (account_id, env_id, run_id, run_queued_at, app_id, function_id, name, start_time, end_time, trace_id, span_id, parent_span_id, attributes)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
			accountID.String(), envID.String(), runID, now, appID.String(), functionID.String(),
			spanID, start, start.Add(time.Second), "trace-1", spanID, parentSpanID, `{}`,
		)
		require.NoError(t, err)
	}
	// The real root: its own parent_span_id is OTel's zero-SpanID sentinel,
	// which never resolves to any span_id in the table.
	insertSpan(rootSpanID, "0000000000000000", now)
	// Real children: parented to the deterministic run reference, which
	// happens to equal the root's own span_id above.
	insertSpan("step-0", rootSpanID, now.Add(time.Second))
	insertSpan("step-1", rootSpanID, now.Add(2*time.Second))

	tmpl, err := SampleTemplates(ctx, db, 200)
	require.NoError(t, err)

	require.Len(t, tmpl.Traces, 1, "the real root's span must not be silently dropped")
	trace := tmpl.Traces[0]
	require.Len(t, trace.Spans, 3)

	bySpanID := map[string]SpanTemplate{}
	for _, s := range trace.Spans {
		bySpanID[s.SpanID] = s
	}
	require.Contains(t, bySpanID, rootSpanID)
	require.NotNil(t, bySpanID[rootSpanID].ParentSpanID)
	require.Equal(t, "0000000000000000", *bySpanID[rootSpanID].ParentSpanID, "the sentinel is carried through unchanged, not interpreted")
	require.NotNil(t, bySpanID["step-0"].ParentSpanID)
	require.Equal(t, rootSpanID, *bySpanID["step-0"].ParentSpanID)
	require.NotNil(t, bySpanID["step-1"].ParentSpanID)
	require.Equal(t, rootSpanID, *bySpanID["step-1"].ParentSpanID)

	// Replaying the sampled trace must preserve the exact same
	// relationship, purely because SpanID/ParentSpanID are copied
	// unchanged -- no root resolution happens on replay either.
	run := RunRow{RunID: "new-run", QueuedAt: now}
	spans := generateSpanTree(trace, run)
	byNewSpanID := map[string]SpanRow{}
	for _, s := range spans {
		byNewSpanID[s.SpanID] = s
	}
	require.Equal(t, "0000000000000000", *byNewSpanID[rootSpanID].ParentSpanID)
	require.Equal(t, rootSpanID, *byNewSpanID["step-1"].ParentSpanID)
}
