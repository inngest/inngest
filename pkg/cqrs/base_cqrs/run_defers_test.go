package base_cqrs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestDecodeDeferOpcode(t *testing.T) {
	op := deferAddOpcode("hash1", "foo", "slug-a", `{}`)
	md := sv2.Metadata{Config: *sv2.InitConfig(&sv2.Config{})}
	h, ok, err := history.BuildDeferHistory(md, queue.Item{}, op)
	require.NoError(t, err)
	require.True(t, ok)
	rawResult, err := json.Marshal(h.Result)
	require.NoError(t, err)
	row := &dbpkg.RunDeferOpcode{Result: sql.NullString{String: string(rawResult), Valid: true}}

	t.Run("decodes the opcode the lifecycle writes", func(t *testing.T) {
		got, err := decodeDeferOpcode(row)
		require.NoError(t, err)
		require.Equal(t, enums.OpcodeDeferAdd, got.Op)
		require.Equal(t, "hash1", got.ID)
		require.NotNil(t, got.Userland)
		require.Equal(t, "foo", got.Userland.ID)
	})

	t.Run("rejects rows with non-string RawOutput", func(t *testing.T) {
		bad, err := json.Marshal(history.Result{RawOutput: map[string]any{"foo": "bar"}})
		require.NoError(t, err)
		_, err = decodeDeferOpcode(&dbpkg.RunDeferOpcode{Result: sql.NullString{String: string(bad), Valid: true}})
		require.Error(t, err)
	})

	t.Run("rejects missing Result", func(t *testing.T) {
		_, err := decodeDeferOpcode(&dbpkg.RunDeferOpcode{Result: sql.NullString{Valid: false}})
		require.Error(t, err)
	})
}

// TestGetRunDefers covers the read path end-to-end: history rows produced by
// BuildDeferHistory (the same builder OnDefer uses) → fold into a defer
// projection joined to child runs by deterministic schedule event id.
func TestGetRunDefers(t *testing.T) {
	ctx := context.Background()
	cm, cleanup := initCQRS(t)
	defer cleanup()

	parentRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	parentMD := sv2.Metadata{
		ID:     sv2.ID{RunID: parentRunID},
		Config: *sv2.InitConfig(&sv2.Config{}),
	}

	scheduledHashed := "scheduled-hash"
	cancelledHashed := "cancelled-hash"

	scheduledEventID, err := event.DeferredScheduleEventID(parentRunID, scheduledHashed)
	require.NoError(t, err)

	childRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	childInternalEventID := ulid.MustNew(ulid.Now(), rand.Reader)

	require.NoError(t, cm.InsertEvent(ctx, cqrs.Event{
		ID:         childInternalEventID,
		EventID:    scheduledEventID.String(),
		EventName:  consts.FnDeferScheduleName,
		EventData:  map[string]any{},
		ReceivedAt: time.Now(),
	}))
	require.NoError(t, cm.InsertFunctionRun(ctx, cqrs.FunctionRun{
		RunID:        childRunID,
		RunStartedAt: time.Now(),
		FunctionID:   uuid.New(),
		EventID:      childInternalEventID,
	}))

	writeDefer(t, cm, parentMD, deferAddOpcode(scheduledHashed, "scheduled-userland", "child-fn-a", `{"foo":"bar"}`))
	writeDefer(t, cm, parentMD, deferAddOpcode(cancelledHashed, "cancelled-userland", "child-fn-b", `{}`))
	writeDefer(t, cm, parentMD, state.GeneratorOpcode{
		Op:   enums.OpcodeDeferCancel,
		ID:   "cancel-op-hash",
		Opts: json.RawMessage(`{"target_hashed_id":"` + cancelledHashed + `"}`),
	})

	defers, err := cm.GetRunDefers(ctx, parentRunID)
	require.NoError(t, err)
	require.Len(t, defers, 2)

	byUserID := map[string]cqrs.RunDefer{}
	for _, d := range defers {
		byUserID[d.ID] = d
	}

	scheduled, ok := byUserID["scheduled-userland"]
	require.True(t, ok)
	require.Equal(t, cqrs.RunDeferStatusScheduled, scheduled.Status)
	require.Equal(t, "child-fn-a", scheduled.FnSlug)
	require.JSONEq(t, `{"foo":"bar"}`, string(scheduled.Input))
	require.NotNil(t, scheduled.Run)
	require.Equal(t, childRunID, scheduled.Run.RunID)

	cancelled, ok := byUserID["cancelled-userland"]
	require.True(t, ok)
	require.Equal(t, cqrs.RunDeferStatusAborted, cancelled.Status)
	require.Nil(t, cancelled.Run, "aborted defers should not surface a child run")
}

// TestGetRunDeferredFrom covers the inverse linkage: a child run triggered by
// inngest/deferred.schedule resolves back to its parent via the event envelope.
func TestGetRunDeferredFrom(t *testing.T) {
	ctx := context.Background()
	cm, cleanup := initCQRS(t)
	defer cleanup()

	parentRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	childRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	childInternalEventID := ulid.MustNew(ulid.Now(), rand.Reader)

	scheduledEventID, err := event.DeferredScheduleEventID(parentRunID, "hash-x")
	require.NoError(t, err)

	require.NoError(t, cm.InsertEvent(ctx, cqrs.Event{
		ID:        childInternalEventID,
		EventID:   scheduledEventID.String(),
		EventName: consts.FnDeferScheduleName,
		EventData: map[string]any{
			consts.InngestEventDataPrefix: event.DeferredScheduleMetadata{
				FnSlug:       "child-fn",
				ParentFnSlug: "parent-fn",
				ParentRunID:  parentRunID.String(),
			},
		},
		ReceivedAt: time.Now(),
	}))
	require.NoError(t, cm.InsertFunctionRun(ctx, cqrs.FunctionRun{
		RunID:        childRunID,
		RunStartedAt: time.Now(),
		FunctionID:   uuid.New(),
		EventID:      childInternalEventID,
	}))

	df, err := cm.GetRunDeferredFrom(ctx, childRunID)
	require.NoError(t, err)
	require.NotNil(t, df)
	require.Equal(t, parentRunID, df.ParentRunID)
	require.Equal(t, "parent-fn", df.ParentFnSlug)
}

func deferAddOpcode(hashedID, userlandID, fnSlug, inputJSON string) state.GeneratorOpcode {
	op := state.GeneratorOpcode{
		Op:   enums.OpcodeDeferAdd,
		ID:   hashedID,
		Opts: json.RawMessage(`{"fn_slug":"` + fnSlug + `","input":` + inputJSON + `}`),
	}
	op.Userland = &struct {
		ID    string `json:"id"`
		Index int    `json:"index,omitempty"`
	}{ID: userlandID}
	return op
}

// writeDefer routes through the same builder OnDefer uses, so the test's write
// shape and the production write shape can't drift.
func writeDefer(t *testing.T, cm cqrs.Manager, md sv2.Metadata, op state.GeneratorOpcode) {
	t.Helper()

	h, ok, err := history.BuildDeferHistory(md, queue.Item{}, op)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, cm.InsertHistory(t.Context(), h))
}
