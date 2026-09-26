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
	benchmarkPreparedEvents           []json.RawMessage
	benchmarkPreparedSerializedEvents event.SerializedEvents
	benchmarkEventInput               string
)

func TestPrepareEventPayloads(t *testing.T) {
	events := benchmarkEvents(1024)

	encoded, _, input, err := prepareEventPayloads(events, event.SerializedEvents{})
	require.NoError(t, err)

	shared, err := json.Marshal(events[0].GetEvent())
	require.NoError(t, err)
	immutable, err := event.NewSerializedEvents([]json.RawMessage{shared})
	require.NoError(t, err)
	sharedEncoded, sharedSerialized, sharedInput, err := prepareEventPayloads(events, immutable)
	require.NoError(t, err)

	require.Nil(t, sharedEncoded)
	require.True(t, immutable.Equal(sharedSerialized))
	require.Equal(t, input, sharedInput)

	shared[0] = 'x'
	require.Equal(t, byte('{'), immutable.Event(0)[0], "immutable snapshot must not alias its source bytes")
	require.NotEqual(t, string(encoded[0]), string(shared))
}

func TestPrepareEventPayloadsRejectsMismatchedSerializedEvents(t *testing.T) {
	serialized, err := event.NewSerializedEvents([]json.RawMessage{json.RawMessage(`{}`), json.RawMessage(`{}`)})
	require.NoError(t, err)
	_, _, _, err = prepareEventPayloads(benchmarkEvents(1024), serialized)
	require.EqualError(t, err, "serialized event count does not match event count")
}

func BenchmarkPrepareEventPayloadsFanout(b *testing.B) {
	events := benchmarkEvents(10 * 1024)

	b.Run("before_per_run_serialization", func(b *testing.B) {
		for range b.N {
			for range benchmarkEventFanout {
				benchmarkPreparedEvents, benchmarkPreparedSerializedEvents, benchmarkEventInput, _ = prepareEventPayloads(events, event.SerializedEvents{})
			}
		}
	})

	b.Run("after_shared_serialization", func(b *testing.B) {
		for range b.N {
			shared, _ := json.Marshal(events[0].GetEvent())
			serialized, _ := event.NewSerializedEvents([]json.RawMessage{shared})
			for range benchmarkEventFanout {
				benchmarkPreparedEvents, benchmarkPreparedSerializedEvents, benchmarkEventInput, _ = prepareEventPayloads(events, serialized)
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
