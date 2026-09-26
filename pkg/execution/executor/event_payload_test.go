package executor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
)

const benchmarkEventFanout = 119

var (
	benchmarkPreparedEvents []json.RawMessage
	benchmarkEventInput     string
)

func TestPrepareEventPayloads(t *testing.T) {
	events := benchmarkEvents(1024)

	encoded, input, err := prepareEventPayloads(events, nil)
	require.NoError(t, err)

	shared, err := json.Marshal(events[0].GetEvent())
	require.NoError(t, err)
	sharedEncoded, sharedInput, err := prepareEventPayloads(events, []json.RawMessage{shared})
	require.NoError(t, err)

	require.Equal(t, encoded, sharedEncoded)
	require.Equal(t, input, sharedInput)
	require.Same(t, &shared[0], &sharedEncoded[0][0])
}

func TestPrepareEventPayloadsRejectsMismatchedSerializedEvents(t *testing.T) {
	_, _, err := prepareEventPayloads(benchmarkEvents(1024), []json.RawMessage{json.RawMessage(`{}`), json.RawMessage(`{}`)})
	require.EqualError(t, err, "serialized event count does not match event count")
}

func BenchmarkPrepareEventPayloadsFanout(b *testing.B) {
	events := benchmarkEvents(10 * 1024)

	b.Run("before_per_run_serialization", func(b *testing.B) {
		for range b.N {
			for range benchmarkEventFanout {
				benchmarkPreparedEvents, benchmarkEventInput, _ = prepareEventPayloads(events, nil)
			}
		}
	})

	b.Run("after_shared_serialization", func(b *testing.B) {
		for range b.N {
			shared, _ := json.Marshal(events[0].GetEvent())
			serialized := []json.RawMessage{shared}
			for range benchmarkEventFanout {
				benchmarkPreparedEvents, benchmarkEventInput, _ = prepareEventPayloads(events, serialized)
			}
		}
	})
}

func benchmarkEvents(payloadSize int) []event.TrackedEvent {
	return []event.TrackedEvent{event.NewBaseTrackedEvent(event.Event{
		Name: "app/large.event",
		Data: map[string]any{"payload": strings.Repeat("x", payloadSize)},
	}, nil)}
}
