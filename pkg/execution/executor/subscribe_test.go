package executor

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionStepIDs(t *testing.T) {
	sum := sha1.Sum([]byte("inbox:3"))
	require.Equal(t, hex.EncodeToString(sum[:]), subscriptionStepID("inbox", 3))

	ack := sha1.Sum([]byte("inbox:ack:3"))
	require.Equal(t, hex.EncodeToString(ack[:]), subscriptionAckID("inbox", 3))

	require.NotEqual(t, subscriptionStepID("inbox", 1), subscriptionStepID("inbox", 2))
	require.NotEqual(t, subscriptionStepID("a", 1), subscriptionStepID("b", 1))
}

func TestFinalizeEventSeed(t *testing.T) {
	evt := event.Event{ID: "msg-1", Name: "agent/message", Timestamp: 1700000000000}

	first, err := finalizeEventSeed(evt).ToULID()
	require.NoError(t, err)
	second, err := finalizeEventSeed(evt).ToULID()
	require.NoError(t, err)
	require.Equal(t, first, second, "the same event ID gives the same internal ID")

	evt.ID = "msg-2"
	other, err := finalizeEventSeed(evt).ToULID()
	require.NoError(t, err)
	require.NotEqual(t, first, other)

	require.Nil(t, finalizeEventSeed(event.Event{ID: "x", Name: "inngest/function.finished", Timestamp: 1}))
	require.Nil(t, finalizeEventSeed(event.Event{Name: "agent/message", Timestamp: 1}))
}

// A subscription wait has no queue item and lasts as long as the run, so it is
// never tracked as a pending step: otherwise the wait of another stream, still
// pending after a message won, would hold back the discovery of every step the
// run completes afterwards.
func TestSubscriptionWaitsAreNotPendingSteps(t *testing.T) {
	subscription := func(id string, subscribe bool, mode string) *state.GeneratorOpcode {
		opts := map[string]any{"event": "agent/message", "timeout": ""}
		if subscribe {
			opts["subscribe"] = map[string]any{"key": "inbox:0", "mode": mode, "seq": 1}
		}
		return &state.GeneratorOpcode{ID: id, Op: enums.OpcodeWaitForEvent, Opts: opts}
	}
	wait := func(id string, subscribe bool) *state.GeneratorOpcode {
		return subscription(id, subscribe, "next")
	}

	cases := []struct {
		name string
		ops  []*state.GeneratorOpcode
		want []string
	}{
		{
			name: "two subscription waits track nothing",
			ops:  []*state.GeneratorOpcode{wait("a", true), wait("b", true)},
			want: []string{},
		},
		{
			name: "a subscription wait beside a step",
			ops: []*state.GeneratorOpcode{
				wait("a", true),
				{ID: "step", Op: enums.OpcodeStepPlanned},
			},
			want: []string{"step"},
		},
		{
			name: "open, poll, ack and close complete in their request and stay tracked",
			ops: []*state.GeneratorOpcode{
				subscription("o", true, "open"),
				subscription("p", true, "poll"),
				subscription("k", true, "ack"),
				subscription("c", true, "close"),
			},
			want: []string{"o", "p", "k", "c"},
		},
		{
			name: "a plain wait for event is still tracked",
			ops:  []*state.GeneratorOpcode{wait("plain", false), wait("a", true)},
			want: []string{"plain"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			groups := opGroups(tc.ops)
			require.Equal(t, tc.want, groups.NonLazyIDs())
		})
	}
}
