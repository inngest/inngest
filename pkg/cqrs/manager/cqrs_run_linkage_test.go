package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mirrors the attrs an executor.defer span carries in production; keep aligned with the typed Attrs serializers.
func deferSpanAttrs(t *testing.T, hashedID, userlandID, fnSlug string, status enums.DeferStatus) []byte {
	t.Helper()
	statusText, err := status.MarshalText()
	require.NoError(t, err)
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.DeferHashedID.Key():   hashedID,
		meta.Attrs.DeferUserlandID.Key(): userlandID,
		meta.Attrs.DeferFnSlug.Key():     fnSlug,
		meta.Attrs.DeferStatus.Key():     string(statusText),
	})
	require.NoError(t, err)
	return byt
}

// Mirrors the child-run-id executor.defer span the executor emits on the PARENT
// run when a deferred child is scheduled: hashed ID + child run ID, no status.
func childRunIDDeferSpanAttrs(t *testing.T, hashedID string, childRunID ulid.ULID) []byte {
	t.Helper()
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.DeferHashedID.Key():   hashedID,
		meta.Attrs.DeferChildRunID.Key(): childRunID.String(),
	})
	require.NoError(t, err)
	return byt
}

func insertChildTraceRun(t *testing.T, cm cqrs.Manager, runID ulid.ULID, accountID, workspaceID, appID, fnID uuid.UUID) {
	t.Helper()
	now := time.Now()
	err := cm.InsertTraceRun(context.Background(), &cqrs.TraceRun{
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		AppID:       appID,
		FunctionID:  fnID,
		TraceID:     "trace-" + runID.String(),
		RunID:       runID.String(),
		QueuedAt:    now,
		StartedAt:   now,
		EndedAt:     now,
		Status:      enums.RunStatusCompleted,
	})
	require.NoError(t, err)
}

// GetRunDefers must surface every defer on the parent, even when the child run hasn't
// materialized yet. If this fails, the UI silently drops pending defers based on
// child-run progress.
func TestGetRunDefers_ReadsExecutorDeferSpans(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	fnID := uuid.New()

	parentRunID := ulid.MustNew(ulid.Now(), nil)

	// Two defers; only the first will get a child TraceRow.
	defers := []struct {
		hashedID   string
		userlandID string
		fnSlug     string
		status     enums.DeferStatus
	}{
		{"hash-aaa", "user-aaa", "app-fn-aaa", enums.DeferStatusAfterRun},
		{"hash-bbb", "user-bbb", "app-fn-bbb", enums.DeferStatusAfterRun},
	}

	for i, d := range defers {
		insertTestSpan(t, cm, testSpanFields{
			RunID:         parentRunID.String(),
			DynamicSpanID: fmt.Sprintf("dyn-defer-%d", i),
			Name:          meta.SpanNameDefer,
			AccountID:     accountID.String(),
			AppID:         appID.String(),
			FunctionID:    fnID.String(),
			EnvID:         workspaceID.String(),
			Attributes:    deferSpanAttrs(t, d.hashedID, d.userlandID, d.fnSlug, d.status),
		})
	}

	// Link only the first defer to a scheduled child run. Linkage requires
	// both a child-run-id span on the parent and an existing TraceRun row.
	linkedChildRunID := ulid.MustNew(ulid.Now(), nil)
	insertChildTraceRun(t, cm, linkedChildRunID, accountID, workspaceID, appID, fnID)
	insertTestSpan(t, cm, testSpanFields{
		RunID:         parentRunID.String(),
		DynamicSpanID: "dyn-defer-child-0",
		Name:          meta.SpanNameDefer,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    fnID.String(),
		EnvID:         workspaceID.String(),
		Attributes:    childRunIDDeferSpanAttrs(t, defers[0].hashedID, linkedChildRunID),
	})

	got, err := cm.GetRunDefers(ctx, []ulid.ULID{parentRunID})
	require.NoError(t, err)

	parentDefers, ok := got[parentRunID]
	require.True(t, ok, "expected entry for parent run %s", parentRunID)
	require.Len(t, parentDefers, 2)

	// GetRunDefers promises sort by hashed defer ID for stable output.
	require.True(t, sort.SliceIsSorted(parentDefers, func(i, j int) bool {
		return parentDefers[i].HashedDeferID < parentDefers[j].HashedDeferID
	}))

	// hash-aaa is the linked one.
	first := parentDefers[0]
	assert.Equal(t, "hash-aaa", first.HashedDeferID)
	assert.Equal(t, "user-aaa", first.UserlandDeferID)
	assert.Equal(t, "app-fn-aaa", first.FnSlug)
	assert.Equal(t, enums.DeferStatusAfterRun, first.Status)
	require.NotNil(t, first.RunID, "AfterRun defer with a scheduled child must carry the child run ID")
	assert.Equal(t, linkedChildRunID, *first.RunID)

	// hash-bbb has no child trace row.
	second := parentDefers[1]
	assert.Equal(t, "hash-bbb", second.HashedDeferID)
	assert.Equal(t, enums.DeferStatusAfterRun, second.Status)
	assert.Nil(t, second.RunID, "unlinked defer must surface with RunID == nil")
}

// A defer span carrying a status the GraphQL converter can't surface (e.g.
// Scheduled) must be skipped, not error out the whole query. A single odd span
// must never blank out every defer on the run.
func TestGetRunDefers_SkipsUnsurfaceableStatus(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	fnID := uuid.New()

	parentRunID := ulid.MustNew(ulid.Now(), nil)

	// A good after_run defer plus a span with an unsurfaceable status.
	insertTestSpan(t, cm, testSpanFields{
		RunID:         parentRunID.String(),
		DynamicSpanID: "dyn-defer-good",
		Name:          meta.SpanNameDefer,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    fnID.String(),
		EnvID:         workspaceID.String(),
		Attributes:    deferSpanAttrs(t, "hash-good", "user-good", "fn-good", enums.DeferStatusAfterRun),
	})
	insertTestSpan(t, cm, testSpanFields{
		RunID:         parentRunID.String(),
		DynamicSpanID: "dyn-defer-weird",
		Name:          meta.SpanNameDefer,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    fnID.String(),
		EnvID:         workspaceID.String(),
		Attributes:    deferSpanAttrs(t, "hash-weird", "user-weird", "fn-weird", enums.DeferStatusScheduled),
	})

	got, err := cm.GetRunDefers(ctx, []ulid.ULID{parentRunID})
	require.NoError(t, err, "an unsurfaceable status must be skipped, not fail the query")

	parentDefers := got[parentRunID]
	require.Len(t, parentDefers, 1, "only the surfaceable defer should remain")
	assert.Equal(t, "hash-good", parentDefers[0].HashedDeferID)
}

// Rejected is terminal — a defer that fails validation persists a Rejected
// sentinel and emits an executor.defer span carrying that status. GetRunDefers
// must surface it (with RunID == nil) so the UI can show that the defer never
// scheduled.
func TestGetRunDefers_SurfacesRejected(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	fnID := uuid.New()

	parentRunID := ulid.MustNew(ulid.Now(), nil)
	insertTestSpan(t, cm, testSpanFields{
		RunID:         parentRunID.String(),
		DynamicSpanID: "dyn-defer-rejected",
		Name:          meta.SpanNameDefer,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    fnID.String(),
		EnvID:         workspaceID.String(),
		Attributes:    deferSpanAttrs(t, "hash-rejected", "user-rejected", "fn-rejected", enums.DeferStatusRejected),
	})

	got, err := cm.GetRunDefers(ctx, []ulid.ULID{parentRunID})
	require.NoError(t, err)

	parentDefers := got[parentRunID]
	require.Len(t, parentDefers, 1)
	d := parentDefers[0]
	assert.Equal(t, "hash-rejected", d.HashedDeferID)
	assert.Equal(t, enums.DeferStatusRejected, d.Status)
	assert.Nil(t, d.RunID, "Rejected defer never scheduled a child run")
}

// The parent's child-run-id executor.defer span is the authoritative parent
// link for deferred runs: it lives on the PARENT and records defer.child_run_id.
// GetRunDeferredFrom queries those spans by child run ID. If this fails, a
// deferred child can't render its parent breadcrumb.
// childRunDeferParentsAttrs mirrors the executor.run-span attributes a deferred
// child carries: defer.parent_run_ids names every parent that scheduled it.
func childRunDeferParentsAttrs(t *testing.T, parentRunIDs ...ulid.ULID) []byte {
	t.Helper()
	parents := make([]string, len(parentRunIDs))
	for i, id := range parentRunIDs {
		parents[i] = id.String()
	}
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.DeferParentRunIDs.Key(): parents,
	})
	require.NoError(t, err)
	return byt
}

func TestGetRunDeferredFrom_ReadsChildRunIDDeferSpan(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	fnID := uuid.New()

	parentRunID := ulid.MustNew(ulid.Now(), nil)
	childRunID := ulid.MustNew(ulid.Now(), nil)

	// Both TraceRuns must exist for GetRunDeferredFrom to return the parent pointer.
	insertChildTraceRun(t, cm, parentRunID, accountID, workspaceID, appID, fnID)
	insertChildTraceRun(t, cm, childRunID, accountID, workspaceID, appID, fnID)

	// The breadcrumb lives on the CHILD's own executor.run span via
	// defer.parent_run_ids — no parent-side span is needed.
	insertTestSpan(t, cm, testSpanFields{
		RunID:         childRunID.String(),
		DynamicSpanID: "dyn-child-run",
		Name:          meta.SpanNameRun,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    fnID.String(),
		EnvID:         workspaceID.String(),
		Attributes:    childRunDeferParentsAttrs(t, parentRunID),
	})

	got, err := cm.GetRunDeferredFrom(ctx, []ulid.ULID{childRunID})
	require.NoError(t, err)

	rdfs, ok := got[childRunID]
	require.True(t, ok, "expected entry for child run %s", childRunID)
	require.Len(t, rdfs, 1)
	assert.Equal(t, parentRunID, rdfs[0].RunID)
}

// A batched child run can descend from defers on several parents (N events ->
// 1 run). GetRunDeferredFrom must return every parent, sorted for stable output
// — collapsing to a single parent would drop linkage the UI needs to show.
func TestGetRunDeferredFrom_MultipleParents(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	fnID := uuid.New()

	// ulid.Make() is monotonic, so these three IDs are guaranteed distinct even
	// when created back-to-back (unlike MustNew(Now(), nil), which uses zero
	// entropy and collides within a millisecond).
	childRunID := ulid.Make()
	parentA := ulid.Make()
	parentB := ulid.Make()

	insertChildTraceRun(t, cm, childRunID, accountID, workspaceID, appID, fnID)
	insertChildTraceRun(t, cm, parentA, accountID, workspaceID, appID, fnID)
	insertChildTraceRun(t, cm, parentB, accountID, workspaceID, appID, fnID)

	// The child's executor.run span lists every scheduling parent in one
	// defer.parent_run_ids attribute.
	insertTestSpan(t, cm, testSpanFields{
		RunID:         childRunID.String(),
		DynamicSpanID: "dyn-child-run",
		Name:          meta.SpanNameRun,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    fnID.String(),
		EnvID:         workspaceID.String(),
		Attributes:    childRunDeferParentsAttrs(t, parentA, parentB),
	})

	got, err := cm.GetRunDeferredFrom(ctx, []ulid.ULID{childRunID})
	require.NoError(t, err)

	rdfs := got[childRunID]
	require.Len(t, rdfs, 2, "a batched child must surface every parent it descends from")

	gotParents := []ulid.ULID{rdfs[0].RunID, rdfs[1].RunID}
	wantParents := []ulid.ULID{parentA, parentB}
	sort.Slice(wantParents, func(i, j int) bool { return wantParents[i].Compare(wantParents[j]) < 0 })
	assert.Equal(t, wantParents, gotParents, "parents must be returned sorted for stable output")
}
