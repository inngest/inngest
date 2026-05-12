package base_cqrs

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getDefersForRun(ctx context.Context, cm cqrs.Manager, runID ulid.ULID) ([]cqrs.RunDefer, error) {
	m, err := cm.GetRunDefers(ctx, []ulid.ULID{runID})
	return m[runID], err
}

func getDeferredFromForRun(ctx context.Context, cm cqrs.Manager, runID ulid.ULID) (*cqrs.RunDeferredFrom, error) {
	m, err := cm.GetRunDeferredFrom(ctx, []ulid.ULID{runID})
	return m[runID], err
}

func TestCQRSInsertRunDefer(t *testing.T) {
	ctx := context.Background()
	cm, cleanup := initCQRS(t)
	defer cleanup()

	parentRunID := ulid.Make()

	t.Run("roundtrips all fields", func(t *testing.T) {
		deferID := "hash-roundtrip"
		err := cm.InsertRunDefer(ctx, parentRunID, deferID, "user-id", "app-fn", cqrs.RunDeferStatusScheduled)
		require.NoError(t, err)

		got, err := getDefersForRun(ctx, cm, parentRunID)
		require.NoError(t, err)

		var found *cqrs.RunDefer
		for i := range got {
			if got[i].ID == deferID {
				found = &got[i]
				break
			}
		}
		require.NotNil(t, found, "expected inserted defer to be returned")
		assert.Equal(t, deferID, found.ID)
		assert.Equal(t, "user-id", found.UserDeferID)
		assert.Equal(t, "app-fn", found.FnSlug)
		assert.Equal(t, cqrs.RunDeferStatusScheduled, found.Status)
		assert.Nil(t, found.Run)
	})

	t.Run("batch insert round-trips all rows", func(t *testing.T) {
		parent := ulid.Make()
		batch := []cqrs.RunDeferInsert{
			{ParentRunID: parent, DeferID: "hash-batch-1", UserDeferID: "u1", FnSlug: "fn-1", Status: cqrs.RunDeferStatusScheduled},
			{ParentRunID: parent, DeferID: "hash-batch-2", UserDeferID: "u2", FnSlug: "fn-2", Status: cqrs.RunDeferStatusAborted},
			{ParentRunID: parent, DeferID: "hash-batch-3", UserDeferID: "u3", FnSlug: "fn-3", Status: cqrs.RunDeferStatusScheduled},
		}
		require.NoError(t, cm.InsertRunDefers(ctx, batch))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 3)
		byID := map[string]cqrs.RunDefer{}
		for _, d := range got {
			byID[d.ID] = d
		}
		for _, in := range batch {
			d, ok := byID[in.DeferID]
			require.True(t, ok, "missing %s", in.DeferID)
			assert.Equal(t, in.UserDeferID, d.UserDeferID)
			assert.Equal(t, in.FnSlug, d.FnSlug)
			assert.Equal(t, in.Status, d.Status)
		}
	})

	t.Run("batch insert with empty slice is a no-op", func(t *testing.T) {
		require.NoError(t, cm.InsertRunDefers(ctx, nil))
	})

	t.Run("upsert on (parent, defer) replaces status/slug", func(t *testing.T) {
		parent := ulid.Make()
		deferID := "hash-upsert"

		require.NoError(t, cm.InsertRunDefer(ctx, parent, deferID, "user-a", "fn-a", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.InsertRunDefer(ctx, parent, deferID, "user-b", "fn-b", cqrs.RunDeferStatusAborted))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "user-b", got[0].UserDeferID)
		assert.Equal(t, "fn-b", got[0].FnSlug)
		assert.Equal(t, cqrs.RunDeferStatusAborted, got[0].Status)
	})

	t.Run("upsert preserves child_run_id set between inserts", func(t *testing.T) {
		parent := ulid.Make()
		deferID := "hash-preserve-child"
		childRunID := ulid.Make()

		require.NoError(t, cm.InsertRunDefer(ctx, parent, deferID, "u", "fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, deferID, childRunID))
		// Re-insert (e.g. finalize retry) must NOT clear the existing linkage.
		require.NoError(t, cm.InsertRunDefer(ctx, parent, deferID, "u2", "fn2", cqrs.RunDeferStatusScheduled))

		from, err := getDeferredFromForRun(ctx, cm, childRunID)
		require.NoError(t, err)
		require.NotNil(t, from, "child linkage should still exist after upsert")
		assert.Equal(t, parent, from.ParentRunID)
	})

	t.Run("different parents don't collide", func(t *testing.T) {
		parentA := ulid.Make()
		parentB := ulid.Make()
		deferID := "hash-shared"

		require.NoError(t, cm.InsertRunDefer(ctx, parentA, deferID, "ua", "fn-a", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.InsertRunDefer(ctx, parentB, deferID, "ub", "fn-b", cqrs.RunDeferStatusScheduled))

		gotA, err := getDefersForRun(ctx, cm, parentA)
		require.NoError(t, err)
		require.Len(t, gotA, 1)
		assert.Equal(t, "ua", gotA[0].UserDeferID)

		gotB, err := getDefersForRun(ctx, cm, parentB)
		require.NoError(t, err)
		require.Len(t, gotB, 1)
		assert.Equal(t, "ub", gotB[0].UserDeferID)
	})

	t.Run("empty UserDeferID accepted", func(t *testing.T) {
		parent := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-empty-userland", "", "fn", cqrs.RunDeferStatusScheduled))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "", got[0].UserDeferID)
	})
}

func TestCQRSGetRunDefers(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	t.Run("empty parent returns empty slice, no error", func(t *testing.T) {
		got, err := getDefersForRun(ctx, cm, ulid.Make())
		require.NoError(t, err)
		assert.Len(t, got, 0)
	})

	t.Run("orders by defer_id ASC", func(t *testing.T) {
		parent := ulid.Make()
		// Insert in non-ASC order; expect results sorted.
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-c", "uc", "fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-a", "ua", "fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-b", "ub", "fn", cqrs.RunDeferStatusScheduled))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "hash-a", got[0].ID)
		assert.Equal(t, "hash-b", got[1].ID)
		assert.Equal(t, "hash-c", got[2].ID)
	})

	t.Run("joins to TraceRun when child_run_id set", func(t *testing.T) {
		parent := ulid.Make()
		childRunID := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-joined", "u", "fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, "hash-joined", childRunID))

		childFnID := uuid.New()
		require.NoError(t, cm.InsertTraceRun(ctx, &cqrs.TraceRun{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       appID,
			FunctionID:  childFnID,
			TraceID:     "trace-" + childRunID.String(),
			RunID:       childRunID.String(),
			QueuedAt:    time.Now(),
			StartedAt:   time.Now(),
			EndedAt:     time.Now(),
			Status:      enums.RunStatusCompleted,
			TriggerIDs:  []string{ulid.Make().String()},
		}))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.NotNil(t, got[0].Run, "Run should be populated when child_run_id is set and trace exists")
		assert.Equal(t, childRunID.String(), got[0].Run.RunID)
		assert.Equal(t, childFnID, got[0].Run.FunctionID)
	})

	t.Run("Run is nil when child_run_id zero", func(t *testing.T) {
		parent := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-unscheduled", "u", "fn", cqrs.RunDeferStatusScheduled))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Nil(t, got[0].Run)
	})

	t.Run("Run is nil when child trace missing (no error)", func(t *testing.T) {
		parent := ulid.Make()
		// child_run_id is set but no trace_runs row exists — simulates pruning.
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-pruned", "u", "fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, "hash-pruned", ulid.Make()))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Nil(t, got[0].Run)
	})

	t.Run("returns both SCHEDULED and ABORTED rows", func(t *testing.T) {
		parent := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-1", "u1", "fn-a", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-2", "u2", "fn-b", cqrs.RunDeferStatusAborted))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 2)

		statuses := map[string]cqrs.RunDeferStatus{}
		for _, d := range got {
			statuses[d.ID] = d.Status
		}
		assert.Equal(t, cqrs.RunDeferStatusScheduled, statuses["hash-1"])
		assert.Equal(t, cqrs.RunDeferStatusAborted, statuses["hash-2"])
	})
}

func TestCQRSGetRunDeferredFrom(t *testing.T) {
	ctx := context.Background()
	appID := uuid.New()
	cm, cleanup := initCQRS(t, withInitCQRSOptApp(appID))
	defer cleanup()

	t.Run("no row returns (nil, nil)", func(t *testing.T) {
		got, err := getDeferredFromForRun(ctx, cm, ulid.Make())
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("returns parent linkage", func(t *testing.T) {
		parent := ulid.Make()
		child := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-linkage", "u", "parent-fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, "hash-linkage", child))

		got, err := getDeferredFromForRun(ctx, cm, child)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, parent, got.ParentRunID)
	})

	t.Run("joins to parent TraceRun", func(t *testing.T) {
		parent := ulid.Make()
		child := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-joined-parent", "u", "parent-fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, "hash-joined-parent", child))

		parentFnID := uuid.New()
		require.NoError(t, cm.InsertTraceRun(ctx, &cqrs.TraceRun{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       appID,
			FunctionID:  parentFnID,
			TraceID:     "trace-" + parent.String(),
			RunID:       parent.String(),
			QueuedAt:    time.Now(),
			StartedAt:   time.Now(),
			EndedAt:     time.Now(),
			Status:      enums.RunStatusCompleted,
			TriggerIDs:  []string{ulid.Make().String()},
		}))

		got, err := getDeferredFromForRun(ctx, cm, child)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotNil(t, got.ParentRun)
		assert.Equal(t, parent.String(), got.ParentRun.RunID)
		assert.Equal(t, parentFnID, got.ParentRun.FunctionID)
	})

	t.Run("ParentRun is nil when parent trace pruned", func(t *testing.T) {
		parent := ulid.Make()
		child := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-orphan", "u", "parent-fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, "hash-orphan", child))

		got, err := getDeferredFromForRun(ctx, cm, child)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, parent, got.ParentRunID)
		assert.Nil(t, got.ParentRun)
	})
}

func TestCQRSUpdateRunDeferChildRunID(t *testing.T) {
	ctx := context.Background()
	cm, cleanup := initCQRS(t)
	defer cleanup()

	t.Run("no matching row is a no-op", func(t *testing.T) {
		err := cm.UpdateRunDeferChildRunID(ctx, ulid.Make(), "missing", ulid.Make())
		require.NoError(t, err)
	})

	t.Run("sets child_run_id", func(t *testing.T) {
		parent := ulid.Make()
		child := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-set", "u", "fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, "hash-set", child))

		from, err := getDeferredFromForRun(ctx, cm, child)
		require.NoError(t, err)
		require.NotNil(t, from)
		assert.Equal(t, parent, from.ParentRunID)
	})

	t.Run("re-point overwrites the previous child (last-write-wins)", func(t *testing.T) {
		// Replays of a deferred.schedule event normally dedupe on a deterministic
		// event ID, but if the link is ever re-pointed (e.g. manual replay), the
		// latest child wins and the previous linkage is dropped — there is no
		// history.
		parent := ulid.Make()
		first := ulid.Make()
		second := ulid.Make()

		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-repoint", "u", "fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, "hash-repoint", first))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, "hash-repoint", second))

		fromFirst, err := getDeferredFromForRun(ctx, cm, first)
		require.NoError(t, err)
		assert.Nil(t, fromFirst, "previous linkage must be dropped")

		fromSecond, err := getDeferredFromForRun(ctx, cm, second)
		require.NoError(t, err)
		require.NotNil(t, fromSecond)
		assert.Equal(t, parent, fromSecond.ParentRunID)
	})

	t.Run("does not touch sibling defers", func(t *testing.T) {
		parent := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-target", "u", "fn", cqrs.RunDeferStatusScheduled))
		require.NoError(t, cm.InsertRunDefer(ctx, parent, "hash-sibling", "u", "fn", cqrs.RunDeferStatusScheduled))

		targetChild := ulid.Make()
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parent, "hash-target", targetChild))

		// The target row now resolves to its child linkage.
		fromTarget, err := getDeferredFromForRun(ctx, cm, targetChild)
		require.NoError(t, err)
		require.NotNil(t, fromTarget)
		assert.Equal(t, parent, fromTarget.ParentRunID)

		// The sibling row still has no child linkage attached.
		defers, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, defers, 2)
		var sibling *cqrs.RunDefer
		for i := range defers {
			if defers[i].ID == "hash-sibling" {
				sibling = &defers[i]
			}
		}
		require.NotNil(t, sibling)
		assert.Nil(t, sibling.Run, "sibling defer should still have no Run")
	})
}
