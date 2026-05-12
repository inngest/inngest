package executor

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDeferStore struct {
	inserts []deferInsertCall
	updates []deferUpdateCall

	insertErr error
	updateErr error
}

type deferInsertCall struct {
	ParentRunID ulid.ULID
	DeferID     string
	UserDeferID string
	FnSlug      string
	Status      cqrs.RunDeferStatus
}

type deferUpdateCall struct {
	ParentRunID ulid.ULID
	DeferID     string
	ChildRunID  ulid.ULID
}

func (f *fakeDeferStore) InsertRunDefer(_ context.Context, parentRunID ulid.ULID, deferID, userDeferID, fnSlug string, status cqrs.RunDeferStatus) error {
	f.inserts = append(f.inserts, deferInsertCall{
		ParentRunID: parentRunID,
		DeferID:     deferID,
		UserDeferID: userDeferID,
		FnSlug:      fnSlug,
		Status:      status,
	})
	return f.insertErr
}

func (f *fakeDeferStore) InsertRunDefers(ctx context.Context, defers []cqrs.RunDeferInsert) error {
	for _, d := range defers {
		if err := f.InsertRunDefer(ctx, d.ParentRunID, d.DeferID, d.UserDeferID, d.FnSlug, d.Status); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeDeferStore) UpdateRunDeferChildRunID(_ context.Context, parentRunID ulid.ULID, deferID string, childRunID ulid.ULID) error {
	f.updates = append(f.updates, deferUpdateCall{
		ParentRunID: parentRunID,
		DeferID:     deferID,
		ChildRunID:  childRunID,
	})
	return f.updateErr
}

var _ cqrs.DeferStore = (*fakeDeferStore)(nil)

func newFinalizeTestExecutor(store cqrs.DeferStore) *executor {
	return &executor{
		log:            logger.From(context.Background()),
		tracerProvider: tracing.NewNoopTracerProvider(),
		deferStore:     store,
	}
}

// finalizeTestOpts initializes Config's internal mutex via InitConfig so
// FunctionSlug reads inside buildDeferEvents do not panic.
func finalizeTestOpts(runID ulid.ULID, fnSlug string) execution.FinalizeOpts {
	cfg := sv2.Config{}
	sv2.InitConfig(&cfg)
	if fnSlug != "" {
		cfg.SetFunctionSlug(fnSlug)
	}
	return execution.FinalizeOpts{
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID:      runID,
				FunctionID: uuid.New(),
				Tenant: sv2.Tenant{
					AccountID: uuid.New(),
					EnvID:     uuid.New(),
					AppID:     uuid.New(),
				},
			},
			Config: cfg,
		},
	}
}

func TestBuildDeferEvents(t *testing.T) {
	ctx := context.Background()

	t.Run("empty defers returns nil events and no store calls", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := newFinalizeTestExecutor(store)
		opts := finalizeTestOpts(ulid.Make(), "parent-fn")

		events, err := e.buildDeferEvents(ctx, opts, nil)
		require.NoError(t, err)
		assert.Nil(t, events)
		assert.Empty(t, store.inserts)
	})

	t.Run("missing fnSlug returns an error", func(t *testing.T) {
		// FnSlug intentionally empty in both Optional and Config.
		e := newFinalizeTestExecutor(&fakeDeferStore{})
		opts := finalizeTestOpts(ulid.Make(), "")

		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-fn",
				HashedID:       "hash-a",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "function slug")
		assert.Nil(t, events)
	})

	t.Run("AfterRun writes SCHEDULED row and emits an event with correct metadata", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := newFinalizeTestExecutor(store)
		runID := ulid.Make()
		opts := finalizeTestOpts(runID, "parent-fn")

		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-fn",
				HashedID:       "hash-a",
				UserlandID:     "user-a",
				ScheduleStatus: enums.DeferStatusAfterRun,
				Input:          json.RawMessage(`{"x":1}`),
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, events, 1)

		require.Len(t, store.inserts, 1)
		assert.Equal(t, runID, store.inserts[0].ParentRunID)
		assert.Equal(t, "hash-a", store.inserts[0].DeferID)
		assert.Equal(t, "user-a", store.inserts[0].UserDeferID)
		assert.Equal(t, "child-fn", store.inserts[0].FnSlug)
		assert.Equal(t, cqrs.RunDeferStatusScheduled, store.inserts[0].Status)

		evt := events[0]
		assert.Equal(t, consts.FnDeferScheduleName, evt.Name)
		assert.EqualValues(t, 1, evt.Data["x"])
		meta, ok := evt.Data[consts.InngestEventDataPrefix].(event.DeferredScheduleMetadata)
		require.True(t, ok)
		assert.Equal(t, "child-fn", meta.FnSlug)
		assert.Equal(t, "parent-fn", meta.ParentFnSlug)
		assert.Equal(t, runID.String(), meta.ParentRunID)
		assert.Equal(t, "hash-a", meta.DeferID)
	})

	t.Run("AfterRun event ID is deterministic across calls", func(t *testing.T) {
		runID := ulid.Make()
		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-fn",
				HashedID:       "hash-a",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
		}
		e := newFinalizeTestExecutor(&fakeDeferStore{})
		opts := finalizeTestOpts(runID, "parent-fn")

		first, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, first, 1)

		// Same (runID, hashedID) → same event ID so the runner dedupes any
		// duplicate publish.
		second, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, second, 1)
		assert.Equal(t, first[0].ID, second[0].ID)
	})

	t.Run("Aborted writes ABORTED row but emits no event", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := newFinalizeTestExecutor(store)
		runID := ulid.Make()
		opts := finalizeTestOpts(runID, "parent-fn")

		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-fn",
				HashedID:       "hash-aborted",
				UserlandID:     "user-aborted",
				ScheduleStatus: enums.DeferStatusAborted,
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		assert.Empty(t, events)

		require.Len(t, store.inserts, 1)
		assert.Equal(t, cqrs.RunDeferStatusAborted, store.inserts[0].Status)
	})

	t.Run("Rejected and Scheduled (unknown-to-row) are skipped (no row, no event)", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := newFinalizeTestExecutor(store)
		opts := finalizeTestOpts(ulid.Make(), "parent-fn")

		defers := map[string]sv2.Defer{
			"r": {
				FnSlug:         "child-fn",
				HashedID:       "hash-rejected",
				ScheduleStatus: enums.DeferStatusRejected,
			},
			"s": {
				FnSlug:         "child-fn",
				HashedID:       "hash-pre-scheduled",
				ScheduleStatus: enums.DeferStatusScheduled,
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		assert.Empty(t, events)
		assert.Empty(t, store.inserts, "Rejected and (eagerly-)Scheduled rows must not be persisted")
	})

	t.Run("invalid defer (Validate fails) is skipped, batch continues", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := newFinalizeTestExecutor(store)
		runID := ulid.Make()
		opts := finalizeTestOpts(runID, "parent-fn")

		defers := map[string]sv2.Defer{
			"bad": {
				// Missing FnSlug → Validate fails.
				HashedID:       "hash-bad",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
			"good": {
				FnSlug:         "child-fn",
				HashedID:       "hash-good",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)

		require.Len(t, events, 1)
		meta, ok := events[0].Data[consts.InngestEventDataPrefix].(event.DeferredScheduleMetadata)
		require.True(t, ok)
		assert.Equal(t, "hash-good", meta.DeferID)

		require.Len(t, store.inserts, 1)
		assert.Equal(t, "hash-good", store.inserts[0].DeferID)
	})

	t.Run("non-object JSON input is skipped, batch continues", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := newFinalizeTestExecutor(store)
		opts := finalizeTestOpts(ulid.Make(), "parent-fn")

		defers := map[string]sv2.Defer{
			"bad": {
				FnSlug:         "child-fn",
				HashedID:       "hash-array",
				ScheduleStatus: enums.DeferStatusAfterRun,
				Input:          json.RawMessage(`[1,2,3]`),
			},
			"good": {
				FnSlug:         "child-fn",
				HashedID:       "hash-ok",
				ScheduleStatus: enums.DeferStatusAfterRun,
				Input:          json.RawMessage(`{"ok":true}`),
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, events, 1)
		meta, ok := events[0].Data[consts.InngestEventDataPrefix].(event.DeferredScheduleMetadata)
		require.True(t, ok)
		assert.Equal(t, "hash-ok", meta.DeferID)
	})

	t.Run("null input yields an event with only the _inngest envelope", func(t *testing.T) {
		e := newFinalizeTestExecutor(&fakeDeferStore{})
		opts := finalizeTestOpts(ulid.Make(), "parent-fn")

		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-fn",
				HashedID:       "hash-null",
				ScheduleStatus: enums.DeferStatusAfterRun,
				Input:          json.RawMessage(`null`),
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.Len(t, events[0].Data, 1)
		_, ok := events[0].Data[consts.InngestEventDataPrefix]
		assert.True(t, ok)
	})

	t.Run("user input cannot overwrite the _inngest envelope", func(t *testing.T) {
		runID := ulid.Make()
		e := newFinalizeTestExecutor(&fakeDeferStore{})
		opts := finalizeTestOpts(runID, "parent-fn")

		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-fn",
				HashedID:       "hash-overwrite",
				ScheduleStatus: enums.DeferStatusAfterRun,
				// User input tries to clobber the `_inngest` envelope.
				Input: json.RawMessage(`{"_inngest":{"hacked":true},"x":1}`),
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, events, 1)

		meta, ok := events[0].Data[consts.InngestEventDataPrefix].(event.DeferredScheduleMetadata)
		require.True(t, ok, "_inngest envelope must be the typed server value, not user-supplied data")
		assert.Equal(t, "hash-overwrite", meta.DeferID)
		assert.Equal(t, runID.String(), meta.ParentRunID)
		assert.EqualValues(t, 1, events[0].Data["x"])
	})

	t.Run("InsertRunDefer failure still emits the event", func(t *testing.T) {
		// Persistence is best-effort: a store failure must not suppress the
		// scheduled event, otherwise the child run would never be triggered.
		store := &fakeDeferStore{insertErr: errors.New("db unavailable")}
		e := newFinalizeTestExecutor(store)
		opts := finalizeTestOpts(ulid.Make(), "parent-fn")

		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-fn",
				HashedID:       "hash-a",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, events, 1, "event must publish even if the row failed to persist")
		require.Len(t, store.inserts, 1, "insert was still attempted")
	})

	t.Run("nil deferStore still emits events", func(t *testing.T) {
		e := newFinalizeTestExecutor(nil)
		opts := finalizeTestOpts(ulid.Make(), "parent-fn")

		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-fn",
				HashedID:       "hash-a",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, events, 1)
	})

	t.Run("multiple defers fan out (compared as a set)", func(t *testing.T) {
		store := &fakeDeferStore{}
		e := newFinalizeTestExecutor(store)
		opts := finalizeTestOpts(ulid.Make(), "parent-fn")

		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-a",
				HashedID:       "hash-a",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
			"b": {
				FnSlug:         "child-b",
				HashedID:       "hash-b",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
			"c": {
				FnSlug:         "child-c",
				HashedID:       "hash-c",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, events, 3)

		gotIDs := map[string]bool{}
		for _, evt := range events {
			meta, ok := evt.Data[consts.InngestEventDataPrefix].(event.DeferredScheduleMetadata)
			require.True(t, ok)
			gotIDs[meta.DeferID] = true
		}
		assert.Equal(t, map[string]bool{"hash-a": true, "hash-b": true, "hash-c": true}, gotIDs)

		gotInserts := map[string]bool{}
		for _, ins := range store.inserts {
			gotInserts[ins.DeferID] = true
		}
		assert.Equal(t, map[string]bool{"hash-a": true, "hash-b": true, "hash-c": true}, gotInserts)
	})

	t.Run("Optional.FnSlug overrides metadata slug", func(t *testing.T) {
		runID := ulid.Make()
		e := newFinalizeTestExecutor(&fakeDeferStore{})
		opts := finalizeTestOpts(runID, "from-metadata")
		opts.Optional.FnSlug = "from-optional"

		defers := map[string]sv2.Defer{
			"a": {
				FnSlug:         "child-fn",
				HashedID:       "hash-a",
				ScheduleStatus: enums.DeferStatusAfterRun,
			},
		}
		events, err := e.buildDeferEvents(ctx, opts, defers)
		require.NoError(t, err)
		require.Len(t, events, 1)
		meta, ok := events[0].Data[consts.InngestEventDataPrefix].(event.DeferredScheduleMetadata)
		require.True(t, ok)
		assert.Equal(t, "from-optional", meta.ParentFnSlug)
	})
}

