package executor

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionStepIDs(t *testing.T) {
	sum := sha1.Sum([]byte("inbox:3"))
	require.Equal(t, hex.EncodeToString(sum[:]), subscriptionStepID("inbox", 3))

	require.NotEqual(t, subscriptionStepID("inbox", 1), subscriptionStepID("inbox", 2))
	require.NotEqual(t, subscriptionStepID("a", 1), subscriptionStepID("b", 1))
}

// A message is the step of its event in a subscription: the same event is the same
// message, different events never share an ID, and the stack gives the order.
func TestSubscriptionMessages(t *testing.T) {
	first := subscriptionMessageID("inbox", "evt-1")
	second := subscriptionMessageID("inbox", "evt-2")

	require.Equal(t, first, subscriptionMessageID("inbox", "evt-1"))
	require.NotEqual(t, first, second)
	require.NotEqual(t, first, subscriptionMessageID("other", "evt-1"))

	stack := []string{"step-a", second, "step-b", subscriptionMessageID("other", "evt-9"), first}
	require.Equal(t, []string{second, first}, subscriptionMessageIDs(stack, "inbox"))
	require.Equal(t, []string{subscriptionMessageID("other", "evt-9")}, subscriptionMessageIDs(stack, "other"))
	require.Empty(t, subscriptionMessageIDs(stack, "missing"))
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
			name: "open, poll and close complete in their request and stay tracked",
			ops: []*state.GeneratorOpcode{
				subscription("o", true, "open"),
				subscription("p", true, "poll"),
				subscription("c", true, "close"),
			},
			want: []string{"o", "p", "c"},
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

func TestWaitSeq(t *testing.T) {
	require.Equal(t, 3, waitSeq(&state.Pause{Metadata: map[string]any{"seq": 3}}))
	require.Equal(t, 3, waitSeq(&state.Pause{Metadata: map[string]any{"seq": int64(3)}}))
	require.Equal(t, 3, waitSeq(&state.Pause{Metadata: map[string]any{"seq": float64(3)}}))
	require.Equal(t, 0, waitSeq(&state.Pause{}))
	require.Equal(t, 0, waitSeq(&state.Pause{Metadata: map[string]any{"seq": "3"}}))
}
