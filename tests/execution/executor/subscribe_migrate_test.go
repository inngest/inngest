package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/require"
)

// migrate moves a run's state the way the cross-cluster migration does: it
// snapshots everything the state service can load, removes the run, and writes
// the snapshot back.
func (h *subscribeHarness) migrate() {
	h.t.Helper()

	ctx := context.Background()
	svc := h.infra.smv2
	id := h.run.ID

	md, err := svc.LoadMetadata(ctx, id)
	require.NoError(h.t, err)
	events, err := svc.LoadEvents(ctx, id)
	require.NoError(h.t, err)
	steps, err := svc.LoadSteps(ctx, id)
	require.NoError(h.t, err)
	inputs, err := svc.LoadStepInputs(ctx, id)
	require.NoError(h.t, err)
	pending, err := svc.LoadPending(ctx, id)
	require.NoError(h.t, err)
	waits, err := svc.LoadInboxWaits(ctx, id)
	require.NoError(h.t, err)

	pureSteps := map[string]json.RawMessage{}
	for stepID, data := range steps {
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(data, &probe); err == nil && len(probe) == 1 {
			if _, wrapped := probe["input"]; wrapped {
				continue
			}
		}
		pureSteps[stepID] = data
	}

	require.NoError(h.t, svc.Delete(ctx, id, state.WithIsMigration()))
	require.NoError(h.t, svc.Migrate(ctx, statev2.MigrateState{
		Metadata:     md,
		Events:       events,
		Steps:        pureSteps,
		StepInputs:   inputs,
		Stack:        md.Stack,
		PendingSteps: pending,
		InboxWaits:   waits,
	}))
}

// A run migrated while a read waits is woken by the next message of its inbox.
func TestSubscribeMigratedWaitingReadWakesOnTheNextMessage(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))
	require.Zero(t, timesIn(h.jobIDs(), h.wakeJobID("next-1", "")), "the read is waiting")

	h.migrate()

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))

	msg := messageStepID("inbox", id)
	require.Contains(t, h.steps(), msg, "the message is saved on the destination")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)), "the migrated read was woken")

}
