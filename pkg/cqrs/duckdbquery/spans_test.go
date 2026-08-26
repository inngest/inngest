package duckdbquery

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func seedSpanRow(t *testing.T, ctx context.Context, m *Manager, appID, functionID uuid.UUID, runIDULID ulid.ULID, spanID, parentSpanID, name string, start, end time.Time, attrs string) {
	t.Helper()
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO inngest.run_trace_spans
		 (account_id, env_id, run_id, run_queued_at, app_id, function_id, name, start_time, end_time, trace_id, span_id, parent_span_id, attributes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		uuid.New().String(), uuid.New().String(), runIDULID.String(), start, appID.String(), functionID.String(),
		name, start, end, "trace-1", spanID, parentSpanID, attrs,
	)
	require.NoError(t, err)
}

func TestGetSpansByRunIDBuildsTreeFromFlatRows(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	appID, functionID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil)
	now := time.Now().UTC()

	// Root span: empty parent_span_id.
	seedSpanRow(t, ctx, m, appID, functionID, runID, "root-span", "", "executor.run", now, now.Add(time.Second), "{}")
	// Child span: parent is the root.
	seedSpanRow(t, ctx, m, appID, functionID, runID, "child-span", "root-span", "executor.step", now.Add(time.Millisecond), now.Add(500*time.Millisecond), "{}")

	root, err := m.GetSpansByRunID(ctx, runID)
	require.NoError(t, err)
	require.Equal(t, "root-span", root.SpanID)
	require.True(t, root.GetIsRoot())
	require.Len(t, root.Children, 1)
	require.Equal(t, "child-span", root.Children[0].SpanID)
	require.Equal(t, appID, root.Children[0].AppID)
	require.Equal(t, functionID, root.Children[0].FunctionID)
}

// TestGetSpansByRunIDSetsOutputIDWhenSpanHasOutputOrInput proves spanFromRow
// mirrors pkg/cqrs/manager/cqrs.go's own rule ("if this span has finished,
// set a preliminary output ID") for the flat-span model, where a span's own
// row already carries its output/input directly — no separate output-span
// indirection to resolve. A span with neither column set (e.g. still
// running) must get no OutputID at all, and the OutputID a finished span
// does get must actually round-trip through GetSpanOutput.
func TestGetSpansByRunIDSetsOutputIDWhenSpanHasOutputOrInput(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	appID, functionID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil)
	now := time.Now().UTC()

	// Root span: still running, no output/input recorded yet.
	seedSpanRow(t, ctx, m, appID, functionID, runID, "root-span", "", "executor.run", now, now.Add(time.Second), "{}")
	// Child span: finished, has output.
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO inngest.run_trace_spans
		 (account_id, env_id, run_id, run_queued_at, app_id, function_id, name, start_time, end_time, trace_id, span_id, parent_span_id, attributes, output)
		 VALUES (?, ?, ?, ?, ?, ?, 'executor.step', ?, ?, 'trace-1', 'output-span', 'root-span', '{}', ?);`,
		uuid.New().String(), uuid.New().String(), runID.String(), now, appID.String(), functionID.String(),
		now.Add(time.Millisecond), now.Add(500*time.Millisecond), `{"data":"done"}`,
	)
	require.NoError(t, err)

	root, err := m.GetSpansByRunID(ctx, runID)
	require.NoError(t, err)
	require.Nil(t, root.GetOutputID(), "a span with no output/input yet must not get an OutputID")
	require.Len(t, root.Children, 1)

	finished := root.Children[0]
	outputID := finished.GetOutputID()
	require.NotNil(t, outputID, "a finished span with output must get an OutputID")

	id := &cqrs.SpanIdentifier{}
	require.NoError(t, id.Decode(*outputID))
	require.Equal(t, runID.String(), id.RunID)
	require.Equal(t, "output-span", id.SpanID)
	require.NotNil(t, id.Preview)
	require.True(t, *id.Preview)

	out, err := m.GetSpanOutput(ctx, *id)
	require.NoError(t, err)
	require.Equal(t, `"done"`, string(out.Data))
}

func TestGetSpansByRunIDErrorsWhenNoSpansFound(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	m := Wrap(nil, db).(*Manager)
	_, err := m.GetSpansByRunID(t.Context(), ulid.MustNew(ulid.Now(), nil))
	require.Error(t, err)
}

func TestGetSpanOutputUnwrapsDataEnvelope(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	runID := ulid.MustNew(ulid.Now(), nil)
	now := time.Now().UTC()
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO inngest.run_trace_spans
		 (account_id, env_id, run_id, run_queued_at, app_id, function_id, name, start_time, end_time, trace_id, span_id, attributes, output, input)
		 VALUES (?, ?, ?, ?, ?, ?, 'executor.step', ?, ?, 'trace-1', 'span-1', '{}', ?, ?);`,
		uuid.New().String(), uuid.New().String(), runID.String(), now, uuid.New().String(), uuid.New().String(),
		now, now, `{"data":"my-result"}`, `{"input":true}`,
	)
	require.NoError(t, err)

	out, err := m.GetSpanOutput(ctx, cqrs.SpanIdentifier{RunID: runID.String(), SpanID: "span-1"})
	require.NoError(t, err)
	require.False(t, out.IsError)
	require.Equal(t, `"my-result"`, string(out.Data))
	require.JSONEq(t, `{"input":true}`, string(out.Input))
}

func TestGetSpanOutputRequiresRunIDAndSpanID(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	m := Wrap(nil, db).(*Manager)
	_, err := m.GetSpanOutput(t.Context(), cqrs.SpanIdentifier{})
	require.Error(t, err)
}
