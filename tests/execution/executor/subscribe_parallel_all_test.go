package executor

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

// stepRunOp is the op an SDK reports for a step.run that ran and returned data.
func stepRunOp(id string, data string) *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeStepRun,
		ID:   id,
		Name: id,
		Data: json.RawMessage(data),
	}
}

// `Promise.all([slow(), inbox.next()])` with the message sent after the step's
// body ended (the e2e flow inbox-parallel "parallel-all-late-step"). Three
// writes can meet: the message save (M), the step's result save (S), and the
// response of the step's request, which reports the still-pending read again
// (R). The executor handles S and R at the same time and M comes from the event
// stream, so every order of the three is possible. Whichever order happens, the
// run must end up with a job that continues it: the last of the three to run
// finds nothing else pending and wakes the run.
func TestSubscribeParallelAllLateStepAlwaysWakes(t *testing.T) {
	points := map[string]string{
		"M": "SaveInboxMessage",
		"R": "RecordInboxWait",
		"S": "SaveStep",
	}

	orders := [][3]string{
		{"M", "R", "S"}, {"M", "S", "R"},
		{"R", "M", "S"}, {"R", "S", "M"},
		{"S", "M", "R"}, {"S", "R", "M"},
	}

	for _, order := range orders {
		t.Run(order[0]+order[1]+order[2], func(t *testing.T) {
			h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
			h.report(plannedOp("work"), subscribeWaitOp("next-1", "inbox", "inbox"))
			before := h.jobIDs()

			h.crash.hold("before:"+points[order[1]], "after:"+points[order[0]])
			h.crash.hold("before:"+points[order[2]], "after:"+points[order[1]])

			delivered := make(chan error, 1)
			go func() {
				delivered <- h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 1}})
			}()

			h.report(stepRunOp("work", `"done"`), subscribeWaitOp("next-1", "inbox", "inbox"))
			require.NoError(t, <-delivered)

			require.Contains(t, h.steps(), "work")
			require.NotZero(t, h.discoveries(before), "nothing continues the run: it stays running")
		})
	}
}
