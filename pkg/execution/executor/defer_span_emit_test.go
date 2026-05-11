package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestBuildDeferEvents_EmitsExecutorDeferSpan asserts that buildDeferEvents
// emits an executor.defer span for AfterRun and Aborted defers (and only
// those), populates the seed and attributes per the contract documented on
// util.DeterministicChildRunID, and produces a deferred.schedule event only
// for AfterRun.
func TestBuildDeferEvents_EmitsExecutorDeferSpan(t *testing.T) {
	rec := newRecordingTracerProvider()
	e := &executor{
		log:            logger.From(context.Background()),
		tracerProvider: rec,
	}

	parentRunID := ulid.MustNew(ulid.Now(), nil)
	fnSlug := "app-parent-fn"

	opts := execution.FinalizeOpts{
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID:      parentRunID,
				FunctionID: uuid.New(),
				Tenant: sv2.Tenant{
					AccountID: uuid.New(),
					EnvID:     uuid.New(),
					AppID:     uuid.New(),
				},
			},
		},
		Optional: execution.FinalizeOptional{
			FnSlug: fnSlug,
		},
	}

	afterRun := sv2.Defer{
		FnSlug:         "app-defer-after",
		HashedID:       "hash-after",
		UserlandID:     "user-after",
		ScheduleStatus: enums.DeferStatusAfterRun,
		Input:          json.RawMessage(`{"k":"v"}`),
	}
	aborted := sv2.Defer{
		FnSlug:         "app-defer-aborted",
		HashedID:       "hash-aborted",
		UserlandID:     "user-aborted",
		ScheduleStatus: enums.DeferStatusAborted,
	}
	rejected := sv2.Defer{
		FnSlug:         "app-defer-rejected",
		HashedID:       "hash-rejected",
		UserlandID:     "user-rejected",
		ScheduleStatus: enums.DeferStatusRejected,
	}

	defers := map[string]sv2.Defer{
		afterRun.HashedID: afterRun,
		aborted.HashedID:  aborted,
		rejected.HashedID: rejected,
	}

	events, err := e.buildDeferEvents(context.Background(), opts, defers)
	require.NoError(t, err)

	// Two CreateSpan calls: AfterRun + Aborted. Rejected is out of contract.
	require.Len(t, rec.createCalls, 2, "executor.defer span must emit for AfterRun and Aborted only")

	byHashed := map[string]recordedCreateCall{}
	for _, c := range rec.createCalls {
		require.Equal(t, meta.SpanNameDefer, c.Name)
		require.NotNil(t, c.Opts)
		require.NotNil(t, c.Opts.Attributes)

		hashedPtr, ok := c.Opts.Attributes.Get(meta.Attrs.DeferHashedID.Key()).(*string)
		require.True(t, ok, "DeferHashedID must be a *string on the captured span")
		require.NotNil(t, hashedPtr)
		byHashed[*hashedPtr] = c
	}

	require.Contains(t, byHashed, afterRun.HashedID)
	require.Contains(t, byHashed, aborted.HashedID)
	require.NotContains(t, byHashed, rejected.HashedID, "rejected defers must not emit a span")

	for _, d := range []sv2.Defer{afterRun, aborted} {
		c := byHashed[d.HashedID]
		// Seed must follow the "s"-tag convention from
		// util.DeterministicChildRunID: parent || hashed || "s".
		expectedSeed := []byte(parentRunID.String() + d.HashedID + "s")
		require.Equal(t, expectedSeed, c.Opts.Seed,
			"executor.defer seed must be parent || hashed || 's' for defer %q", d.HashedID)

		userPtr, ok := c.Opts.Attributes.Get(meta.Attrs.DeferUserID.Key()).(*string)
		require.True(t, ok, "DeferUserID must be a *string")
		require.NotNil(t, userPtr)
		require.Equal(t, d.UserlandID, *userPtr)

		slugPtr, ok := c.Opts.Attributes.Get(meta.Attrs.DeferFnSlug.Key()).(*string)
		require.True(t, ok, "DeferFnSlug must be a *string")
		require.NotNil(t, slugPtr)
		require.Equal(t, d.FnSlug, *slugPtr)

		statusPtr, ok := c.Opts.Attributes.Get(meta.Attrs.DeferStatus.Key()).(*enums.DeferStatus)
		require.True(t, ok, "DeferStatus must be a *enums.DeferStatus")
		require.NotNil(t, statusPtr)
		require.Equal(t, d.ScheduleStatus, *statusPtr)
	}

	// Only AfterRun produces a deferred.schedule event.
	require.Len(t, events, 1)
	require.Equal(t, consts.FnDeferScheduleName, events[0].Name)
}

// TestSchedule_DeterministicRunIDForDeferredScheduleEvent unit-tests the
// deferredScheduleFromEvents helper that Schedule uses to decide whether the
// child run ID should be deterministically derived from the parent. Full
// Schedule wiring is exercised by the integration suite under tests/golang/;
// here we just lock down the helper's contract.
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

		gotMeta, gotParent, ok := deferredScheduleFromEvents(logger.From(ctx), evts)
		require.True(t, ok)
		require.NotNil(t, gotMeta)
		require.Equal(t, hashedID, gotMeta.HashedDeferID)
		require.Equal(t, parentRunID, gotParent)
	})

	t.Run("empty events returns false", func(t *testing.T) {
		gotMeta, gotParent, ok := deferredScheduleFromEvents(logger.From(ctx), nil)
		require.False(t, ok)
		require.Nil(t, gotMeta)
		require.Equal(t, ulid.ULID{}, gotParent)
	})

	t.Run("non-deferred-schedule events return false", func(t *testing.T) {
		id := ulid.MustNew(ulid.Now(), nil)
		evts := []event.TrackedEvent{event.BaseTrackedEvent{
			ID: id,
			Event: event.Event{
				ID:   id.String(),
				Name: "user/something.happened",
			},
		}}
		gotMeta, gotParent, ok := deferredScheduleFromEvents(logger.From(ctx), evts)
		require.False(t, ok)
		require.Nil(t, gotMeta)
		require.Equal(t, ulid.ULID{}, gotParent)
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
		gotMeta, gotParent, ok := deferredScheduleFromEvents(logger.From(ctx), evts)
		require.False(t, ok, "two deferred.schedule events must bail rather than guess")
		require.Nil(t, gotMeta)
		require.Equal(t, ulid.ULID{}, gotParent)
	})

	t.Run("malformed metadata missing ParentRunID returns false", func(t *testing.T) {
		md := event.DeferredScheduleMetadata{
			FnSlug:        "app-fn",
			ParentFnSlug:  "app-parent",
			HashedDeferID: hashedID,
			// ParentRunID intentionally omitted to fail Validate().
		}
		evts := []event.TrackedEvent{makeDeferredEvent(t, md)}
		gotMeta, gotParent, ok := deferredScheduleFromEvents(logger.From(ctx), evts)
		require.False(t, ok)
		require.Nil(t, gotMeta)
		require.Equal(t, ulid.ULID{}, gotParent)
	})

	t.Run("bad parent run id format returns false", func(t *testing.T) {
		md := event.DeferredScheduleMetadata{
			FnSlug:        "app-fn",
			ParentFnSlug:  "app-parent",
			ParentRunID:   "not-a-ulid",
			HashedDeferID: hashedID,
		}
		evts := []event.TrackedEvent{makeDeferredEvent(t, md)}
		gotMeta, gotParent, ok := deferredScheduleFromEvents(logger.From(ctx), evts)
		require.False(t, ok)
		require.Nil(t, gotMeta)
		require.Equal(t, ulid.ULID{}, gotParent)
	})
}
