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
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mirrors the attrs an executor.defer span carries in production; keep aligned with the typed Attrs serializers.
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

// Mirrors the parent-linkage attrs a deferred child run's executor.run span carries.
func runSpanAttrsWithParent(t *testing.T, parentRunID ulid.ULID, hashedID string) []byte {
	t.Helper()
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.DeferParentRunID.Key(): parentRunID.String(),
		meta.Attrs.DeferHashedID.Key():    hashedID,
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
// materialized yet. If this fails, the UI silently drops pending/aborted defers based
// on child-run progress.
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

	// Link only the first defer's child.
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

// Mirrors the executor.step fragment of an invoke; the invoked run ID lives on the EXTEND fragment.
func invokeStepSpanAttrs(t *testing.T, stepName string) []byte {
	t.Helper()
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.StepName.Key(): stepName,
	})
	require.NoError(t, err)
	return byt
}

// Mirrors the EXTEND fragment carrying the invoked run ID; shares dynamic_span_id with the executor.step fragment.
func invokeExtendSpanAttrs(t *testing.T, invokedRunID ulid.ULID) []byte {
	t.Helper()
	byt, err := json.Marshal(map[string]any{
		meta.Attrs.StepInvokeRunID.Key(): invokedRunID.String(),
	})
	require.NoError(t, err)
	return byt
}

// Invoke linkage spans two fragments sharing a dynamic_span_id: executor.step carries the
// step name, EXTEND carries the invoked run ID. GetRunInvokedFrom must merge both — losing
// either one blanks the child's "invoked from" panel.
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

	// Both fragments must be present for mapSpanFromRow's read-time merge.
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

// The child's own executor.run span is the authoritative parent link for deferred runs (the
// parent's defer span alone isn't sufficient). If this fails, a deferred child can't render
// its parent breadcrumb.
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

	// Both TraceRuns must exist for GetRunDeferredFrom to return the parent pointer.
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
