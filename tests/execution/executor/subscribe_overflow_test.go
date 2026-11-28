package executor

import (
	"fmt"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/require"
)

// overflowJobID is the id of the job an arrival at the step or state size
// bound queues, as the queue stores it (hashed).
func (h *subscribeHarness) overflowJobID() string {
	return queue.HashID(h.infra.ctx, fmt.Sprintf("%s-inbox-overflow", h.run.ID.RunID))
}

// finish runs the queued overflow job the way the queue would.
func (h *subscribeHarness) finishOverflowRun() {
	h.t.Helper()

	jobs, err := h.infra.rq.RunJobs(h.infra.ctx, h.infra.queueShard.Name(), queue.Scope{
		AccountID:  h.run.ID.Tenant.AccountID,
		EnvID:      h.run.ID.Tenant.EnvID,
		FunctionID: h.run.ID.FunctionID,
	}, h.run.ID.RunID, 1000, 0)
	require.NoError(h.t, err)

	for _, job := range jobs {
		if job.JobID != h.overflowJobID() {
			continue
		}

		item, ok := job.Raw.(*queue.QueueItem)
		require.True(h.t, ok)

		edge, ok := item.Data.Payload.(queue.PayloadEdge)
		require.True(h.t, ok)

		_, err := h.exec.Execute(queue.WithJobID(h.infra.ctx, job.JobID), statev2.V1FromMetadata(*h.run), item.Data, edge.Edge)
		require.NoError(h.t, err)

		return
	}

	h.t.Fatal("no overflow job was queued")
}

func (h *subscribeHarness) runIsGone() bool {
	_, err := h.infra.smv2.LoadMetadata(h.infra.ctx, h.run.ID)

	return err != nil
}

func (h *subscribeHarness) stepCount() int {
	md, err := h.infra.smv2.LoadMetadata(h.infra.ctx, h.run.ID)
	require.NoError(h.t, err)

	return md.Metrics.StepCount
}

// An arrival at the step bound saves nothing and queues one call, which ends
// the run with the step overflow error. A second arrival queues nothing more.
func TestSubscribeArrivalAtStepBound(t *testing.T) {
	bound := 3
	h := newSubscribeHarnessWith(t,
		[]executor.ExecutorOpt{executor.WithStepLimits(func(statev2.ID) int { return bound })},
		subscribeOp("inbox", "test/msg"),
	)

	for h.stepCount() < bound {
		require.NoError(t, h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	}
	saved := h.steps()
	require.Zero(t, timesIn(h.jobIDs(), h.overflowJobID()), "below the bound nothing is queued")

	over := newEventID()
	require.NoError(t, h.deliverID(over, event.Event{Name: "test/msg", Data: map[string]any{"n": 2}}))
	require.Equal(t, saved, h.steps(), "the message at the bound is not saved")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.overflowJobID()))

	require.NoError(t, h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 3}}))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.overflowJobID()), "one call however many arrive")

	h.finishOverflowRun()
	require.True(t, h.runIsGone(), "the queued call ended the run")
}

// An arrival that would take state past the state size bound saves nothing and
// queues one call, which ends the run with the state size error.
func TestSubscribeArrivalAtStateBound(t *testing.T) {
	h := newSubscribeHarnessWith(t,
		[]executor.ExecutorOpt{executor.WithStateSizeLimits(func(statev2.ID) int { return 1500 })},
		subscribeOp("inbox", "test/msg"),
	)

	require.NoError(t, h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	saved := h.steps()
	require.Zero(t, timesIn(h.jobIDs(), h.overflowJobID()))

	big := string(make([]byte, 2000))
	require.NoError(t, h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"big": big}}))
	require.Equal(t, saved, h.steps(), "the message past the bound is not saved")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.overflowJobID()))

	h.finishOverflowRun()
	require.True(t, h.runIsGone(), "the queued call ended the run")
}
