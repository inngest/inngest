package executor

import (
	"context"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestSchedule_DeterministicRunIDForDeferredScheduleEvent unit-tests the
// findDeferredChild helper that Schedule uses to decide whether the child
// run ID should be deterministically derived from the parent. Full Schedule
// wiring is exercised by the integration suite under tests/golang/; here we
// just lock down the helper's contract.
func TestSchedule_DeterministicRunIDForDeferredScheduleEvent(t *testing.T) {
	ctx := context.Background()
	parentRunID := ulid.MustNew(ulid.Now(), nil)
	hashedID := "abc123"

	makeDeferredEvent := func(t *testing.T, md event.DeferredScheduleMetadata) event.TrackedEvent {
		t.Helper()
		id := ulid.MustNew(ulid.Now(), nil)
		return event.BaseTrackedEvent{
			ID: id,
			Event: event.Event{
				ID:   id.String(),
				Name: consts.FnDeferScheduleName,
				Data: map[string]any{
					consts.InngestEventDataPrefix: md,
				},
			},
		}
	}

	t.Run("single valid deferred.schedule event returns parsed metadata", func(t *testing.T) {
		md := event.DeferredScheduleMetadata{
			FnSlug:        "app-fn",
			ParentFnSlug:  "app-parent",
			ParentRunID:   parentRunID.String(),
			HashedDeferID: hashedID,
		}
		evts := []event.TrackedEvent{makeDeferredEvent(t, md)}

		got := findDeferredChild(logger.From(ctx), evts)
		require.NotNil(t, got)
		require.Equal(t, hashedID, got.meta.HashedDeferID)
		require.Equal(t, parentRunID, got.parentRunID)
		require.Equal(t, util.DeterministicChildRunID(parentRunID, hashedID), got.runID)
	})

	t.Run("empty events returns nil", func(t *testing.T) {
		require.Nil(t, findDeferredChild(logger.From(ctx), nil))
	})

	t.Run("non-deferred-schedule events return nil", func(t *testing.T) {
		id := ulid.MustNew(ulid.Now(), nil)
		evts := []event.TrackedEvent{event.BaseTrackedEvent{
			ID: id,
			Event: event.Event{
				ID:   id.String(),
				Name: "user/something.happened",
			},
		}}
		require.Nil(t, findDeferredChild(logger.From(ctx), evts))
	})

	t.Run("two deferred.schedule events in same batch bails out", func(t *testing.T) {
		md := event.DeferredScheduleMetadata{
			FnSlug:        "app-fn",
			ParentFnSlug:  "app-parent",
			ParentRunID:   parentRunID.String(),
			HashedDeferID: hashedID,
		}
		evts := []event.TrackedEvent{
			makeDeferredEvent(t, md),
			makeDeferredEvent(t, md),
		}
		require.Nil(t, findDeferredChild(logger.From(ctx), evts), "two deferred.schedule events must bail rather than guess")
	})

	t.Run("malformed metadata missing ParentRunID returns nil", func(t *testing.T) {
		md := event.DeferredScheduleMetadata{
			FnSlug:        "app-fn",
			ParentFnSlug:  "app-parent",
			HashedDeferID: hashedID,
			// ParentRunID intentionally omitted to fail Validate().
		}
		evts := []event.TrackedEvent{makeDeferredEvent(t, md)}
		require.Nil(t, findDeferredChild(logger.From(ctx), evts))
	})

	t.Run("bad parent run id format returns nil", func(t *testing.T) {
		md := event.DeferredScheduleMetadata{
			FnSlug:        "app-fn",
			ParentFnSlug:  "app-parent",
			ParentRunID:   "not-a-ulid",
			HashedDeferID: hashedID,
		}
		evts := []event.TrackedEvent{makeDeferredEvent(t, md)}
		require.Nil(t, findDeferredChild(logger.From(ctx), evts))
	})
}
