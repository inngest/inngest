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
	benchmarkPreparedSerializedEvents []string
	benchmarkEventInput               string
)

func TestPrepareEventPayloads(t *testing.T) {
	events := benchmarkEvents(1024)

	encoded, _, input, err := prepareEventPayloads(events, nil)
	require.NoError(t, err)

	shared, err := json.Marshal(events[0].GetEvent())
	require.NoError(t, err)
	immutable := string(shared)
	sharedEncoded, sharedSerialized, sharedInput, err := prepareEventPayloads(events, []string{immutable})
	require.NoError(t, err)

	require.Nil(t, sharedEncoded)
	require.Equal(t, []string{immutable}, sharedSerialized)
	require.Equal(t, input, sharedInput)

	shared[0] = 'x'
	require.Equal(t, byte('{'), immutable[0], "immutable snapshot must not alias its source bytes")
	require.NotEqual(t, string(encoded[0]), string(shared))
}

func TestPrepareEventPayloadsRejectsMismatchedSerializedEvents(t *testing.T) {
	_, _, _, err := prepareEventPayloads(benchmarkEvents(1024), []string{`{}`, `{}`})
	require.EqualError(t, err, "serialized event count does not match event count")
}

func BenchmarkPrepareEventPayloadsFanout(b *testing.B) {
	events := benchmarkEvents(10 * 1024)

	b.Run("before_per_run_serialization", func(b *testing.B) {
		for range b.N {
			for range benchmarkEventFanout {
				benchmarkPreparedEvents, benchmarkPreparedSerializedEvents, benchmarkEventInput, _ = prepareEventPayloads(events, nil)
			}
		}
	})

	b.Run("after_shared_serialization", func(b *testing.B) {
		for range b.N {
			shared, _ := json.Marshal(events[0].GetEvent())
			serialized := []string{string(shared)}
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
