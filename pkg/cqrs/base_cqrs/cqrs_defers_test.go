package base_cqrs

import (
	"context"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parentID returns an sv2.ID wrapping runID with zero-valued tenant fields.
// The Postgres dev-server DeferStore implementation ignores tenancy (it is
// single-tenant), so uuid.Nil for AccountID/EnvID is sufficient for these
// tests. Cloud tenancy is exercised by the ClickHouse implementation in the
// downstream monorepo.
func parentID(runID ulid.ULID) sv2.ID {
	return sv2.ID{RunID: runID}
}

func insert(hashedDeferID, userDeferID, fnSlug string, status enums.DeferStatus) cqrs.RunDeferInsert {
	return cqrs.RunDeferInsert{
		HashedDeferID: hashedDeferID,
		UserDeferID:   userDeferID,
		FnSlug:        fnSlug,
		Status:        status,
	}
}

func childUpdate(hashedDeferID string, childRunID ulid.ULID) cqrs.RunDeferUpdate {
	return cqrs.RunDeferUpdate{HashedDeferID: hashedDeferID, ChildRunID: childRunID}
}

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
		hashedDeferID := "hash-roundtrip"
		err := cm.InsertRunDefer(ctx, parentID(parentRunID), insert(hashedDeferID, "user-id", "app-fn", enums.DeferStatusAfterRun))
		require.NoError(t, err)

		got, err := getDefersForRun(ctx, cm, parentRunID)
		require.NoError(t, err)

		var found *cqrs.RunDefer
		for i := range got {
			if got[i].HashedDeferID == hashedDeferID {
				found = &got[i]
				break
			}
		}
		require.NotNil(t, found, "expected inserted defer to be returned")
		assert.Equal(t, hashedDeferID, found.HashedDeferID)
		spew.Dump("\n\nhashedDeferID is ", hashedDeferID, "\n\n")
		assert.Equal(t, "user-id", found.UserDeferID)
		assert.Equal(t, "app-fn", found.FnSlug)
		assert.Equal(t, enums.DeferStatusAfterRun, found.Status)
		assert.Nil(t, found.Run)
	})

	t.Run("batch insert round-trips all rows", func(t *testing.T) {
		parent := ulid.Make()
		batch := []cqrs.RunDeferInsert{
			insert("hash-batch-1", "u1", "fn-1", enums.DeferStatusAfterRun),
			insert("hash-batch-2", "u2", "fn-2", enums.DeferStatusAborted),
			insert("hash-batch-3", "u3", "fn-3", enums.DeferStatusAfterRun),
		}
		require.NoError(t, cm.InsertRunDefers(ctx, parentID(parent), batch))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 3)
		byID := map[string]cqrs.RunDefer{}
		for _, d := range got {
			byID[d.HashedDeferID] = d
		}
		for _, in := range batch {
			d, ok := byID[in.HashedDeferID]
			require.True(t, ok, "missing %s", in.HashedDeferID)
			assert.Equal(t, in.UserDeferID, d.UserDeferID)
			assert.Equal(t, in.FnSlug, d.FnSlug)
			assert.Equal(t, in.Status, d.Status)
		}
	})

	t.Run("batch insert with empty slice is a no-op", func(t *testing.T) {
		require.NoError(t, cm.InsertRunDefers(ctx, parentID(ulid.Make()), nil))
	})

	t.Run("upsert on (parent, defer) replaces status/slug", func(t *testing.T) {
		parent := ulid.Make()
		hashedDeferID := "hash-upsert"

		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert(hashedDeferID, "user-a", "fn-a", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert(hashedDeferID, "user-b", "fn-b", enums.DeferStatusAborted)))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "user-b", got[0].UserDeferID)
		assert.Equal(t, "fn-b", got[0].FnSlug)
		assert.Equal(t, enums.DeferStatusAborted, got[0].Status)
	})

	t.Run("upsert preserves child_run_id set between inserts", func(t *testing.T) {
		parent := ulid.Make()
		hashedDeferID := "hash-preserve-child"
		childRunID := ulid.Make()

		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert(hashedDeferID, "u", "fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate(hashedDeferID, childRunID)))
		// Re-insert (e.g. finalize retry) must NOT clear the existing linkage.
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert(hashedDeferID, "u2", "fn2", enums.DeferStatusAfterRun)))

		from, err := getDeferredFromForRun(ctx, cm, childRunID)
		require.NoError(t, err)
		require.NotNil(t, from, "child linkage should still exist after upsert")
		assert.Equal(t, parent, from.ParentRunID)
	})

	t.Run("different parents don't collide", func(t *testing.T) {
		parentA := ulid.Make()
		parentB := ulid.Make()
		hashedDeferID := "hash-shared"

		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parentA), insert(hashedDeferID, "ua", "fn-a", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parentB), insert(hashedDeferID, "ub", "fn-b", enums.DeferStatusAfterRun)))

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
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-empty-userland", "", "fn", enums.DeferStatusAfterRun)))

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
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-c", "uc", "fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-a", "ua", "fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-b", "ub", "fn", enums.DeferStatusAfterRun)))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "hash-a", got[0].HashedDeferID)
		assert.Equal(t, "hash-b", got[1].HashedDeferID)
		assert.Equal(t, "hash-c", got[2].HashedDeferID)
	})

	t.Run("joins to TraceRun when child_run_id set", func(t *testing.T) {
		parent := ulid.Make()
		childRunID := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-joined", "u", "fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate("hash-joined", childRunID)))

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
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-unscheduled", "u", "fn", enums.DeferStatusAfterRun)))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Nil(t, got[0].Run)
	})

	t.Run("Run is nil when child trace missing (no error)", func(t *testing.T) {
		parent := ulid.Make()
		// child_run_id is set but no trace_runs row exists — simulates pruning.
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-pruned", "u", "fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate("hash-pruned", ulid.Make())))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Nil(t, got[0].Run)
	})

	t.Run("returns both SCHEDULED and ABORTED rows", func(t *testing.T) {
		parent := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-1", "u1", "fn-a", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-2", "u2", "fn-b", enums.DeferStatusAborted)))

		got, err := getDefersForRun(ctx, cm, parent)
		require.NoError(t, err)
		require.Len(t, got, 2)

		statuses := map[string]enums.DeferStatus{}
		for _, d := range got {
			statuses[d.HashedDeferID] = d.Status
		}
		assert.Equal(t, enums.DeferStatusAfterRun, statuses["hash-1"])
		assert.Equal(t, enums.DeferStatusAborted, statuses["hash-2"])
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
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-linkage", "u", "parent-fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate("hash-linkage", child)))

		got, err := getDeferredFromForRun(ctx, cm, child)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, parent, got.ParentRunID)
	})

	t.Run("joins to parent TraceRun", func(t *testing.T) {
		parent := ulid.Make()
		child := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-joined-parent", "u", "parent-fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate("hash-joined-parent", child)))

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
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-orphan", "u", "parent-fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate("hash-orphan", child)))

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
		err := cm.UpdateRunDeferChildRunID(ctx, parentID(ulid.Make()), childUpdate("missing", ulid.Make()))
		require.NoError(t, err)
	})

	t.Run("sets child_run_id", func(t *testing.T) {
		parent := ulid.Make()
		child := ulid.Make()
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-set", "u", "fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate("hash-set", child)))

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

		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-repoint", "u", "fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate("hash-repoint", first)))
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate("hash-repoint", second)))

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
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-target", "u", "fn", enums.DeferStatusAfterRun)))
		require.NoError(t, cm.InsertRunDefer(ctx, parentID(parent), insert("hash-sibling", "u", "fn", enums.DeferStatusAfterRun)))

		targetChild := ulid.Make()
		require.NoError(t, cm.UpdateRunDeferChildRunID(ctx, parentID(parent), childUpdate("hash-target", targetChild)))

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
			if defers[i].HashedDeferID == "hash-sibling" {
				sibling = &defers[i]
			}
		}
		require.NotNil(t, sibling)
		assert.Nil(t, sibling.Run, "sibling defer should still have no Run")
	})
}
