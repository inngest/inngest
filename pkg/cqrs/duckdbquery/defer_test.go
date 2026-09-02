package duckdbquery

import (
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
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
	spanID := tracing.DeferSpanRef(runID, "hash-1").DynamicSpanID

	attrs := deferAttrsJSON(t, map[string]any{
		meta.Attrs.DeferHashedID.Key():   "hash-1",
		meta.Attrs.DeferUserlandID.Key(): "user-1",
		meta.Attrs.DeferFnSlug.Key():     "deferred-fn",
		meta.Attrs.DeferStatus.Key():     enums.DeferStatusAfterRun.String(),
	})
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, spanID, "", meta.SpanNameDefer, now, now, attrs)

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
// row and a later Abort row (same deterministic span_id, run_trace_spans is
// append-only) merges field-by-field, the same way
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
	spanID := tracing.DeferSpanRef(runID, "hash-2").DynamicSpanID

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
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, spanID, "", meta.SpanNameDefer, now, now, addAttrs)
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, spanID, "", meta.SpanNameDefer, now.Add(time.Second), now.Add(time.Second), abortAttrs)

	out, err := m.GetRunDefers(ctx, []ulid.ULID{runID})
	require.NoError(t, err)
	require.Len(t, out[runID], 1)

	d := out[runID][0]
	require.Equal(t, "hash-2", d.HashedDeferID)
	require.Equal(t, enums.DeferStatusAborted, d.Status, "the later Abort row's status must win")
	require.Equal(t, "user-2", d.UserlandDeferID, "must be preserved from the earlier Add row")
	require.Equal(t, "deferred-fn", d.FnSlug, "must be preserved from the earlier Add row")
}

// TestGetRunDefersMergesScheduleLinkOverAfterRun mirrors the abort case for
// the other terminal transition: a schedule-link row (written by
// OnFunctionScheduled once the child run is created) stamps the child's run
// ID, and userland_id — which the schedule-link write never carries either
// (see updateParentDeferSpans' own doc comment) — is still preserved from
// the earlier Add row via the same field-level merge.
func TestGetRunDefersMergesScheduleLinkOverAfterRun(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	m := Wrap(nil, db).(*Manager)

	appID, functionID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	childRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	now := time.Now().UTC()
	spanID := tracing.DeferSpanRef(runID, "hash-3").DynamicSpanID

	addAttrs := deferAttrsJSON(t, map[string]any{
		meta.Attrs.DeferHashedID.Key():   "hash-3",
		meta.Attrs.DeferUserlandID.Key(): "user-3",
		meta.Attrs.DeferFnSlug.Key():     "deferred-fn",
		meta.Attrs.DeferStatus.Key():     enums.DeferStatusAfterRun.String(),
	})
	linkAttrs := deferAttrsJSON(t, map[string]any{
		meta.Attrs.DeferHashedID.Key():   "hash-3",
		meta.Attrs.DeferFnSlug.Key():     "deferred-fn",
		meta.Attrs.DeferStatus.Key():     enums.DeferStatusAfterRun.String(),
		meta.Attrs.DeferChildRunID.Key(): childRunID.String(),
	})
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, spanID, "", meta.SpanNameDefer, now, now, addAttrs)
	seedSpanRow(t, ctx, m, uuid.New(), uuid.New(), appID, functionID, runID, spanID, "", meta.SpanNameDefer, now.Add(time.Second), now.Add(time.Second), linkAttrs)

	out, err := m.GetRunDefers(ctx, []ulid.ULID{runID})
	require.NoError(t, err)
	require.Len(t, out[runID], 1)

	d := out[runID][0]
	require.Equal(t, "hash-3", d.HashedDeferID)
	require.Equal(t, enums.DeferStatusAfterRun, d.Status)
	require.NotNil(t, d.RunID)
	require.Equal(t, childRunID, *d.RunID)
	require.Equal(t, "user-3", d.UserlandDeferID, "must be preserved from the earlier Add row")
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
