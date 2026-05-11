package base_cqrs

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
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deferSpanAttrs returns a JSON blob shaped like a real executor.defer span's
// attributes column: the typed Attrs serializers use the "_inngest." prefix,
// so we mirror that here so that ExtractTypedValues in mapSpanFromRow reads
// the values back into ExtractedValues correctly.
func deferSpanAttrs(t *testing.T, hashedID, userID, fnSlug string, status enums.DeferStatus) []byte {
	t.Helper()
	statusText, err := status.MarshalText()
	require.NoError(t, err)
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.DeferHashedID.Key(): hashedID,
		meta.Attrs.DeferUserID.Key():   userID,
		meta.Attrs.DeferFnSlug.Key():   fnSlug,
		meta.Attrs.DeferStatus.Key():   string(statusText),
	})
	require.NoError(t, err)
	return byt
}

// runSpanAttrsWithParent mirrors the attributes a child run's executor.run
// span carries when it was scheduled via deferred.schedule: just the parent
// linkage attrs that the GetRunDeferredFrom CQRS method reads back.
func runSpanAttrsWithParent(t *testing.T, parentRunID ulid.ULID, hashedID string) []byte {
	t.Helper()
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.DeferParentRunID.Key(): parentRunID.String(),
		meta.Attrs.DeferHashedID.Key():    hashedID,
	})
	require.NoError(t, err)
	return byt
}

// insertChildTraceRun inserts a minimal TraceRun for the given run ID so that
// GetTraceRunsByRunIDs returns it and GetRunDefers can stitch it onto the
// parent's RunDefer entry.
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

// TestGetRunDefers_ReadsExecutorDeferSpans seeds two executor.defer spans on
// a single parent, links one of the deterministic child run IDs to a
// TraceRun, and asserts GetRunDefers returns both defers, sorted by hashed
// ID, with Run populated only for the linked one.
func TestGetRunDefers_ReadsExecutorDeferSpans(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	fnID := uuid.New()

	parentRunID := ulid.MustNew(ulid.Now(), nil)

	// Two defers on the same parent. The first will have a child trace row,
	// the second will not (Run must be nil for it).
	defers := []struct {
		hashedID string
		userID   string
		fnSlug   string
		status   enums.DeferStatus
	}{
		{"hash-aaa", "user-aaa", "app-fn-aaa", enums.DeferStatusAfterRun},
		{"hash-bbb", "user-bbb", "app-fn-bbb", enums.DeferStatusAborted},
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
			Attributes:    deferSpanAttrs(t, d.hashedID, d.userID, d.fnSlug, d.status),
		})
	}

	// Insert a trace run for the deterministic child run ID of the first
	// defer only — the second defer's child should remain unlinked (Run nil).
	linkedChildRunID := util.DeterministicChildRunID(parentRunID, defers[0].hashedID)
	insertChildTraceRun(t, cm, linkedChildRunID, accountID, workspaceID, appID, fnID)

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
	assert.Equal(t, "user-aaa", first.UserDeferID)
	assert.Equal(t, "app-fn-aaa", first.FnSlug)
	assert.Equal(t, enums.DeferStatusAfterRun, first.Status)
	require.NotNil(t, first.Run, "AfterRun defer with a present child trace run must be stitched in")
	assert.Equal(t, linkedChildRunID.String(), first.Run.RunID)

	// hash-bbb has no child trace row.
	second := parentDefers[1]
	assert.Equal(t, "hash-bbb", second.HashedDeferID)
	assert.Equal(t, enums.DeferStatusAborted, second.Status)
	assert.Nil(t, second.Run, "Aborted/unlinked defer must surface with Run == nil")
}

// invokeStepSpanAttrs mirrors the attributes on the parent's executor.step
// fragment of an invoke step span: just the step display name. The invoked
// run ID lives on a separate EXTEND fragment (see invokeExtendSpanAttrs).
func invokeStepSpanAttrs(t *testing.T, stepName string) []byte {
	t.Helper()
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.StepName.Key(): stepName,
	})
	require.NoError(t, err)
	return byt
}

// invokeExtendSpanAttrs mirrors the attributes on the EXTEND fragment that
// the executor emits when the invoked run ID becomes known. The fragment
// shares dynamic_span_id with the original executor.step row and shares
// the parent's run_id; only the invoked-run-id attribute distinguishes it.
func invokeExtendSpanAttrs(t *testing.T, invokedRunID ulid.ULID) []byte {
	t.Helper()
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.StepInvokeRunID.Key(): invokedRunID.String(),
	})
	require.NoError(t, err)
	return byt
}

// TestGetRunInvokedFrom_ReadsParentInvokeStepSpan seeds a parent run with
// an executor.step + EXTEND fragment pair sharing a dynamic_span_id and a
// child run that the invoke produced. GetRunInvokedFrom must locate the
// parent linkage by reverse-walking from the child run ID through the
// EXTEND fragment's StepInvokeRunID attribute, then pick up StepName from
// the sibling executor.step fragment.
func TestGetRunInvokedFrom_ReadsParentInvokeStepSpan(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	fnID := uuid.New()

	parentRunID := ulid.MustNew(ulid.Now(), nil)
	childRunID := ulid.MustNew(ulid.Now(), nil)

	insertChildTraceRun(t, cm, parentRunID, accountID, workspaceID, appID, fnID)
	insertChildTraceRun(t, cm, childRunID, accountID, workspaceID, appID, fnID)

	// Two fragments share the dynamic_span_id and trace_id; the first
	// carries the step display name, the second is the EXTEND row with
	// the invoked run ID. Both fields are required for the read-time
	// merge in mapSpanFromRow (the query GROUPs by both).
	dynamicSpanID := "dyn-invoke-step"
	traceID := "trace-invoke-step"
	insertTestSpan(t, cm, testSpanFields{
		RunID:         parentRunID.String(),
		TraceID:       traceID,
		DynamicSpanID: dynamicSpanID,
		Name:          meta.SpanNameStep,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    fnID.String(),
		EnvID:         workspaceID.String(),
		Attributes:    invokeStepSpanAttrs(t, "invoke-target-step"),
	})
	insertTestSpan(t, cm, testSpanFields{
		RunID:         parentRunID.String(),
		TraceID:       traceID,
		DynamicSpanID: dynamicSpanID,
		Name:          meta.SpanNameDynamicExtension,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    fnID.String(),
		EnvID:         workspaceID.String(),
		Attributes:    invokeExtendSpanAttrs(t, childRunID),
	})

	got, err := cm.GetRunInvokedFrom(ctx, []ulid.ULID{childRunID})
	require.NoError(t, err)

	rif, ok := got[childRunID]
	require.True(t, ok, "expected entry for child run %s", childRunID)
	require.NotNil(t, rif)
	assert.Equal(t, parentRunID, rif.ParentRunID)
	require.NotNil(t, rif.StepName)
	assert.Equal(t, "invoke-target-step", *rif.StepName)
	require.NotNil(t, rif.ParentRun, "ParentRun must be stitched in when the parent's TraceRun exists")
	assert.Equal(t, parentRunID.String(), rif.ParentRun.RunID)
}

// TestGetRunDeferredFrom_ReadsChildExecutorRunSpan seeds a parent run, a
// child run, and an executor.run span on the child whose attributes carry
// the parent linkage. GetRunDeferredFrom must return the parent run pointer
// stitched onto the RunDeferredFrom entry.
func TestGetRunDeferredFrom_ReadsChildExecutorRunSpan(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()

	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	accountID := uuid.New()
	workspaceID := uuid.New()
	fnID := uuid.New()

	parentRunID := ulid.MustNew(ulid.Now(), nil)
	childRunID := ulid.MustNew(ulid.Now(), nil)

	// Both runs exist as TraceRuns so GetRunDeferredFrom can hand back the
	// parent pointer.
	insertChildTraceRun(t, cm, parentRunID, accountID, workspaceID, appID, fnID)
	insertChildTraceRun(t, cm, childRunID, accountID, workspaceID, appID, fnID)

	insertTestSpan(t, cm, testSpanFields{
		RunID:         childRunID.String(),
		DynamicSpanID: "dyn-child-run",
		Name:          meta.SpanNameRun,
		AccountID:     accountID.String(),
		AppID:         appID.String(),
		FunctionID:    fnID.String(),
		EnvID:         workspaceID.String(),
		Attributes:    runSpanAttrsWithParent(t, parentRunID, "hash-aaa"),
	})

	got, err := cm.GetRunDeferredFrom(ctx, []ulid.ULID{childRunID})
	require.NoError(t, err)

	rdf, ok := got[childRunID]
	require.True(t, ok, "expected entry for child run %s", childRunID)
	require.NotNil(t, rdf)
	assert.Equal(t, parentRunID, rdf.ParentRunID)
	require.NotNil(t, rdf.ParentRun, "ParentRun must be stitched in when the parent's TraceRun exists")
	assert.Equal(t, parentRunID.String(), rdf.ParentRun.RunID)
}
