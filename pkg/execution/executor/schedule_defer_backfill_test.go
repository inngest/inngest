package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackfillDeferChildRunID(t *testing.T) {
	ctx := context.Background()
	log := logger.From(ctx)

	t.Run("no deferStore is a no-op", func(t *testing.T) {
		e := &executor{log: log, deferStore: nil}
		req := execution.ScheduleRequest{
			Events: []event.TrackedEvent{
				event.NewBaseTrackedEventWithID(scheduleEvent(t, ulid.Make(), "hash-a"), ulid.Make()),
			},
		}
		e.backfillDeferChildRunID(ctx, req, ulid.Make(), log)
	})

	t.Run("one update per inngest/deferred.schedule event", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := &executor{log: log, deferStore: store}

		parentA := ulid.Make()
		parentB := ulid.Make()
		childRunID := ulid.Make()

		req := execution.ScheduleRequest{
			Events: []event.TrackedEvent{
				event.NewBaseTrackedEventWithID(scheduleEvent(t, parentA, "hash-a"), ulid.Make()),
				event.NewBaseTrackedEventWithID(scheduleEvent(t, parentB, "hash-b"), ulid.Make()),
			},
		}

		e.backfillDeferChildRunID(ctx, req, childRunID, log)

		require.Len(t, store.updates, 2)
		updatesByDeferID := map[string]deferUpdateCall{}
		for _, u := range store.updates {
			updatesByDeferID[u.DeferID] = u
		}
		assert.Equal(t, parentA, updatesByDeferID["hash-a"].ParentRunID)
		assert.Equal(t, childRunID, updatesByDeferID["hash-a"].ChildRunID)
		assert.Equal(t, parentB, updatesByDeferID["hash-b"].ParentRunID)
		assert.Equal(t, childRunID, updatesByDeferID["hash-b"].ChildRunID)
	})

	t.Run("non-deferred events are ignored", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := &executor{log: log, deferStore: store}

		req := execution.ScheduleRequest{
			Events: []event.TrackedEvent{
				event.NewBaseTrackedEventWithID(event.Event{Name: "user/event"}, ulid.Make()),
				event.NewBaseTrackedEventWithID(scheduleEvent(t, ulid.Make(), "hash-a"), ulid.Make()),
			},
		}
		e.backfillDeferChildRunID(ctx, req, ulid.Make(), log)
		require.Len(t, store.updates, 1, "only the deferred.schedule event should trigger an update")
	})

	t.Run("malformed metadata is skipped, others are processed", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := &executor{log: log, deferStore: store}

		// _inngest envelope missing → DeferredScheduleMetadata() errors.
		badEvt := event.Event{Name: consts.FnDeferScheduleName, Data: map[string]any{"x": 1}}
		req := execution.ScheduleRequest{
			Events: []event.TrackedEvent{
				event.NewBaseTrackedEventWithID(badEvt, ulid.Make()),
				event.NewBaseTrackedEventWithID(scheduleEvent(t, ulid.Make(), "hash-good"), ulid.Make()),
			},
		}
		e.backfillDeferChildRunID(ctx, req, ulid.Make(), log)
		require.Len(t, store.updates, 1)
		assert.Equal(t, "hash-good", store.updates[0].DeferID)
	})

	t.Run("invalid parent ULID is skipped, others are processed", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := &executor{log: log, deferStore: store}

		badEvt := event.Event{
			Name: consts.FnDeferScheduleName,
			Data: map[string]any{
				consts.InngestEventDataPrefix: map[string]any{
					"fn_slug":        "child",
					"parent_fn_slug": "parent",
					"parent_run_id":  "not-a-ulid",
					"defer_id":       "hash-bad",
				},
			},
		}
		req := execution.ScheduleRequest{
			Events: []event.TrackedEvent{
				event.NewBaseTrackedEventWithID(badEvt, ulid.Make()),
				event.NewBaseTrackedEventWithID(scheduleEvent(t, ulid.Make(), "hash-good"), ulid.Make()),
			},
		}
		e.backfillDeferChildRunID(ctx, req, ulid.Make(), log)
		require.Len(t, store.updates, 1)
		assert.Equal(t, "hash-good", store.updates[0].DeferID)
	})

	t.Run("event with empty defer_id is skipped", func(t *testing.T) {
		// An empty DeferID matches every row for a parent in the UPDATE
		// WHERE clause — it must never reach the store.
		store := &fakeDeferStore{}
		e := &executor{log: log, deferStore: store}

		req := execution.ScheduleRequest{
			Events: []event.TrackedEvent{
				event.NewBaseTrackedEventWithID(scheduleEvent(t, ulid.Make(), ""), ulid.Make()),
			},
		}
		e.backfillDeferChildRunID(ctx, req, ulid.Make(), log)
		assert.Empty(t, store.updates, "empty defer_id must not reach the store")
	})

	t.Run("store error does not abort the loop", func(t *testing.T) {
		store := &fakeDeferStore{updateErr: errors.New("db unavailable")}
		e := &executor{log: log, deferStore: store}

		req := execution.ScheduleRequest{
			Events: []event.TrackedEvent{
				event.NewBaseTrackedEventWithID(scheduleEvent(t, ulid.Make(), "hash-a"), ulid.Make()),
				event.NewBaseTrackedEventWithID(scheduleEvent(t, ulid.Make(), "hash-b"), ulid.Make()),
			},
		}
		e.backfillDeferChildRunID(ctx, req, ulid.Make(), log)

		seen := map[string]struct{}{}
		var attempted []string
		for _, u := range store.updates {
			if _, ok := seen[u.DeferID]; ok {
				continue
			}
			seen[u.DeferID] = struct{}{}
			attempted = append(attempted, u.DeferID)
		}
		assert.ElementsMatch(t, []string{"hash-a", "hash-b"}, attempted,
			"store errors must not short-circuit the batch")
	})
}

func scheduleEvent(t *testing.T, parentRunID ulid.ULID, deferID string) event.Event {
	t.Helper()
	return event.Event{
		Name: consts.FnDeferScheduleName,
		Data: map[string]any{
			consts.InngestEventDataPrefix: map[string]any{
				"fn_slug":        "child-fn",
				"parent_fn_slug": "parent-fn",
				"parent_run_id":  parentRunID.String(),
				"defer_id":       deferID,
			},
		},
	}
}
