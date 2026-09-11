package duckdbquery

import (
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	tracingv3 "github.com/inngest/inngest/pkg/tracing/v3"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// deferAttrsJSON builds a run_trace_spans.attributes JSON blob. Keys must
// come from an attr's own .Key() call, not a literal string: meta.Attr's
// serializer prepends meta.AttrKeyPrefix ("_inngest.") to every key (see
// StringAttr/withPrefix in pkg/tracing/meta/serializers.go), so a literal
// "defer.hashed_id" silently fails to match what scanDeferSpanRow actually
// looks up.
func deferAttrsJSON(t *testing.T, fields map[string]any) string {
	t.Helper()
	byt, err := json.Marshal(fields)
	require.NoError(t, err)
	return string(byt)
}

func TestGetRunDefersReturnsSingleAfterRunDefer(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	appID, functionID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	now := time.Now().UTC()

	attrs := deferAttrsJSON(t, map[string]any{
		meta.Attrs.DeferHashedID.Key():   "hash-1",
		meta.Attrs.DeferUserlandID.Key(): "user-1",
		meta.Attrs.DeferFnSlug.Key():     "deferred-fn",
		meta.Attrs.DeferStatus.Key():     enums.DeferStatusAfterRun.String(),
	})
	// The physical span_id is arbitrary — GetRunDefers groups by the
	// defer.hashed_id attribute above, not by span_id (see OnDeferAdd's own
	// doc comment), so this value only needs to be unique per row.
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, "span-1", "", meta.SpanNameDefer, now, now, attrs)

	out, err := m.GetRunDefers(ctx, []ulid.ULID{runID})
	require.NoError(t, err)
	require.Len(t, out[runID], 1)

	d := out[runID][0]
	require.Equal(t, "hash-1", d.HashedDeferID)
	require.Equal(t, "user-1", d.UserlandDeferID)
	require.Equal(t, "deferred-fn", d.FnSlug)
	require.Equal(t, enums.DeferStatusAfterRun, d.Status)
	require.Nil(t, d.RunID)
}

// TestGetRunDefersMergesAbortOverAfterRun proves a defer with both an Add
// row and a later Abort row — each with its own unique physical span_id
// (run_trace_spans requires span_id stay unique per row — see
// OnDeferAdd/OnDeferAbort's own doc comments), grouped instead by their
// shared defer.hashed_id attribute — merges field-by-field, the same way
// pkg/cqrs/manager/cqrs.go's mapSpanFromRow reconstructs a logical span from
// SQLite/Postgres's dynamic-span fragments: the Abort row's status wins
// (it's the later write), but its absent fn_slug/userland_id do NOT blank
// out the values the earlier Add row set — see mergeDeferSpanRow.
func TestGetRunDefersMergesAbortOverAfterRun(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	appID, functionID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	now := time.Now().UTC()

	addAttrs := deferAttrsJSON(t, map[string]any{
		meta.Attrs.DeferHashedID.Key():   "hash-2",
		meta.Attrs.DeferUserlandID.Key(): "user-2",
		meta.Attrs.DeferFnSlug.Key():     "deferred-fn",
		meta.Attrs.DeferStatus.Key():     enums.DeferStatusAfterRun.String(),
	})
	abortAttrs := deferAttrsJSON(t, map[string]any{
		meta.Attrs.DeferHashedID.Key(): "hash-2",
		meta.Attrs.DeferStatus.Key():   enums.DeferStatusAborted.String(),
	})
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, "span-2-add", "", meta.SpanNameDefer, now, now, addAttrs)
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, "span-2-abort", "", meta.SpanNameDefer, now.Add(time.Second), now.Add(time.Second), abortAttrs)

	out, err := m.GetRunDefers(ctx, []ulid.ULID{runID})
	require.NoError(t, err)
	require.Len(t, out[runID], 1)

	d := out[runID][0]
	require.Equal(t, "hash-2", d.HashedDeferID)
	require.Equal(t, enums.DeferStatusAborted, d.Status, "the later Abort row's status must win")
	require.Equal(t, "user-2", d.UserlandDeferID, "must be preserved from the earlier Add row")
	require.Equal(t, "deferred-fn", d.FnSlug, "must be preserved from the earlier Add row")
}

// TestGetRunDefersResolvesChildRunIDViaEventID proves the child run ID comes
// from resolveDeferChildRunIDs joining the defer's own defer.event_id
// (present from the parent's very first Add row, since it's a pure function
// of (run ID, hashedID) — see event.DeferEventID) against inngest.runs.
// event_ids, not from any value written back onto the parent's defer span
// once the child schedules — nothing here touches the parent's span a
// second time, unlike the old schedule-link write.
func TestGetRunDefersResolvesChildRunIDViaEventID(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	childRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	childFunctionID := uuid.New()
	now := time.Now().UTC()
	hashedID := "hash-3"
	dynamicSpanID := tracing.DeferSpanRef(runID, hashedID).DynamicSpanID

	eventID, err := event.DeferEventID(runID, hashedID)
	require.NoError(t, err)

	addAttrs := deferAttrsJSON(t, map[string]any{
		meta.Attrs.DynamicSpanID.Key():   dynamicSpanID,
		meta.Attrs.DeferHashedID.Key():   hashedID,
		meta.Attrs.DeferUserlandID.Key(): "user-3",
		meta.Attrs.DeferFnSlug.Key():     "deferred-fn",
		meta.Attrs.DeferStatus.Key():     enums.DeferStatusAfterRun.String(),
		meta.Attrs.DeferEventID.Key():    eventID.String(),
	})
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, "span-3", "", meta.SpanNameDefer, now, now, addAttrs)

	// Mirrors what materializeRuns would produce once the child's own
	// run.queued span batch flushes: a row in inngest.runs whose event_ids
	// includes the same deterministic event ID.
	_, err = m.db.ExecContext(ctx,
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, scheduled_at, app_id, app_name, function_id, function_slug, inputs, attributes, status, event_ids)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '[]', '{}', ?, ?);`,
		accountID.String(), envID.String(), childRunID.String(), now, now, appID.String(), "test-app", childFunctionID.String(), "deferred-fn",
		enums.StepStatusQueued.String(), []string{eventID.String()},
	)
	require.NoError(t, err)

	out, err := m.GetRunDefers(ctx, []ulid.ULID{runID})
	require.NoError(t, err)
	require.Len(t, out[runID], 1)

	d := out[runID][0]
	require.Equal(t, hashedID, d.HashedDeferID)
	require.Equal(t, enums.DeferStatusAfterRun, d.Status)
	require.NotNil(t, d.RunID)
	require.Equal(t, childRunID, *d.RunID)
	require.Equal(t, "user-3", d.UserlandDeferID)
}

func TestGetRunDefersReturnsEmptyForNoDefers(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	out, err := m.GetRunDefers(ctx, []ulid.ULID{ulid.MustNew(ulid.Now(), rand.Reader)})
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestGetRunDeferredFromReadsParentLinkage(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	appID, functionID := uuid.New(), uuid.New()
	childRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	parentRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	now := time.Now().UTC()

	attrs := deferAttrsJSON(t, map[string]any{
		meta.Attrs.DeferParentRunIDs.Key(): []string{parentRunID.String()},
		meta.Attrs.DeferParentFnSlug.Key(): "parent-fn",
	})
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, childRunID, "child-queued-span", "", tracingv3.SpanNameRunQueued, now, now, attrs)

	out, err := m.GetRunDeferredFrom(ctx, []ulid.ULID{childRunID})
	require.NoError(t, err)
	require.Len(t, out[childRunID], 1)
	require.Equal(t, parentRunID, out[childRunID][0].RunID)
	require.Equal(t, "parent-fn", out[childRunID][0].FnSlug)
}

func TestGetRunDeferredFromReturnsEmptyForNormalRun(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	appID, functionID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	now := time.Now().UTC()

	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, "queued-span", "", tracingv3.SpanNameRunQueued, now, now, "{}")

	out, err := m.GetRunDeferredFrom(ctx, []ulid.ULID{runID})
	require.NoError(t, err)
	require.Empty(t, out)
}
