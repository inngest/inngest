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

	_, err := db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, app_id, function_id, status, inputs, output)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, now, appID.String(), functionID.String(),
		"Completed", `{"event":{"name":"app/one"}}`, `{"data":"ok"}`,
	)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO inngest.run_trace_spans (account_id, env_id, run_id, run_queued_at, app_id, function_id, name, start_time, end_time, trace_id, span_id, attributes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, now, appID.String(), functionID.String(),
		"my-function", now, now, "trace-1", "span-1", `{"sys.step.name":"my-function"}`,
	)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts)
		 VALUES (?, ?, ?, ?, 'test', ?, ?, ?, '1', ?);`,
		accountID.String(), envID.String(), runID, now, runID, "app/one", `{"k":"v"}`, now,
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
	require.Contains(t, tmpl.SpanNames, "my-function")
	require.Contains(t, tmpl.EventNames, "app/one")
	require.Contains(t, tmpl.Inputs, `{"event":{"name":"app/one"}}`)
	require.Contains(t, tmpl.Attributes, `{"sys.step.name":"my-function"}`)
	require.Contains(t, tmpl.EventData, `{"k":"v"}`)
}
