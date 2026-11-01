package executor

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2/server"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/require"
)

// redisCommands records the redis commands run against the run state,
// including the ones a Lua script makes.
type redisCommands struct {
	mu   sync.Mutex
	seen []string
}

func (h *subscribeHarness) watchRedis() *redisCommands {
	rc := &redisCommands{}

	h.infra.runState.Server().SetPreHook(func(_ *server.Peer, cmd string, args ...string) bool {
		rc.mu.Lock()
		defer rc.mu.Unlock()

		rc.seen = append(rc.seen, strings.ToUpper(cmd))

		return false
	})

	return rc
}

func (rc *redisCommands) times(cmd string) int {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	n := 0
	for _, seen := range rc.seen {
		if seen == cmd {
			n++
		}
	}

	return n
}

func (rc *redisCommands) reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.seen = nil
}

// saveStep saves a step the way a step.run result is saved.
func (h *subscribeHarness) saveStep(id string, data string) {
	h.t.Helper()

	_, err := h.infra.smv2.SaveStep(h.infra.ctx, h.run.ID, id, []byte(data))
	require.NoError(h.t, err)
}

// A wait recorded after many unrelated steps reads only the steps saved since
// the SDK loaded state: the gap, here one message.
func TestSubscribeWaitScanReadsOnlyTheGap(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	for i := 0; i < 500; i++ {
		h.saveStep(fmt.Sprintf("work-%d", i), `{"data":1}`)
	}

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	msg := messageStepID("inbox", id)

	rc := h.watchRedis()
	wake, err := h.infra.smv2.RecordInboxWait(h.infra.ctx, h.run.ID, statev2.InboxWait{
		ID:       "next-1",
		Inbox:    "inbox",
		Seen:     "work-499",
		Incoming: "step",
	})
	require.NoError(t, err)
	require.NotNil(t, wake)
	require.Equal(t, msg, wake.MessageStep)

	require.Zero(t, rc.times("LRANGE"), "the stack is not read whole")
	require.Zero(t, rc.times("HGETALL"))
	require.Equal(t, 1, rc.times("HGET"), "one step is in the gap, so one value is read")
}

// A step.run that returns the shape of a message is a normal step: it does not
// wake a read.
func TestSubscribeStepShapedLikeAMessageIsNotAMessage(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	h.saveStep("lookalike", `{"data":{"name":"test/msg"},"inbox":"inbox","src":[]}`)

	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	require.Zero(t, timesIn(h.jobIDs(), h.wakeJobID("next-1", "lookalike")), "the read was not woken by a step.run result")
}

// An event that arrives again changes nothing: the same step with the same
// value, and the read is woken once.
func TestSubscribeRedeliveredEventChangesNothing(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	id := newEventID()
	evt := event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}
	msg := messageStepID("inbox", id)

	require.NoError(t, h.deliverID(id, evt))
	saved := h.steps()
	jobs := h.jobIDs()

	md, err := h.infra.smv2.LoadMetadata(h.infra.ctx, h.run.ID)
	require.NoError(t, err)

	require.NoError(t, h.deliverID(id, evt))

	require.Equal(t, saved, h.steps(), "same steps, same values")
	require.Equal(t, len(jobs), len(h.jobIDs()))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)))

	after, err := h.infra.smv2.LoadMetadata(h.infra.ctx, h.run.ID)
	require.NoError(t, err)
	require.Equal(t, md.Metrics.StepCount, after.Metrics.StepCount, "the redelivery cost no step")
}

// One event that matches two sources of one inbox is one message that lists
// both sources.
func TestSubscribeEventMatchingTwoSourcesIsOneMessage(t *testing.T) {
	both := &state.GeneratorOpcode{
		Op:   enums.OpcodeSubscribe,
		ID:   "inbox",
		Name: "inbox",
		Opts: map[string]any{"sources": []map[string]any{
			{"event": "test/msg"},
			{"event": "test/msg", "if": "async.data.n == 1"},
		}},
	}
	h := newSubscribeHarness(t, both)

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))

	messages := 0
	var value struct {
		Inbox string   `json:"inngest/inbox"`
		Src   []string `json:"src"`
	}
	for stepID, data := range h.steps() {
		if stepID == messageStepID("inbox", id) {
			messages++
			require.NoError(t, json.Unmarshal([]byte(data), &value))
		}
	}

	require.Equal(t, 1, messages, "saved once")
	require.Equal(t, "inbox", value.Inbox)
	require.Len(t, value.Src, 2, "both sources are listed")
	require.NotEqual(t, value.Src[0], value.Src[1])

}
