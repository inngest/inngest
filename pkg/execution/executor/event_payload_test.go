package executor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
)

// benchmarkEventFanout matches the number of functions triggered by one event
// during INC-1180.
const benchmarkEventFanout = 119

var (
	benchmarkRawEvents       []json.RawMessage
	benchmarkImmutableEvents event.SerializedEvents
	benchmarkTraceInput      string
)

func TestPrepareStateAndTraceEventPayloads(t *testing.T) {
	events := benchmarkEvents(1024)

	rawEvents, _, traceInput, err := prepareStateAndTraceEventPayloads(events, event.SerializedEvents{})
	require.NoError(t, err)

	shared, err := json.Marshal(events[0].GetEvent())
	require.NoError(t, err)
	immutable, err := event.NewSerializedEvents([]json.RawMessage{shared})
	require.NoError(t, err)
	sharedRawEvents, sharedImmutableEvents, sharedTraceInput, err := prepareStateAndTraceEventPayloads(events, immutable)
	require.NoError(t, err)

	require.Nil(t, sharedRawEvents)
	require.True(t, immutable.Equal(sharedImmutableEvents))
	require.Equal(t, traceInput, sharedTraceInput)

	shared[0] = 'x'
	require.Equal(t, byte('{'), immutable.Event(0)[0], "immutable snapshot must not alias its source bytes")
	require.NotEqual(t, string(rawEvents[0]), string(shared))
}

func TestPrepareStateAndTraceEventPayloadsRejectsMismatchedSerializedEvents(t *testing.T) {
	serialized, err := event.NewSerializedEvents([]json.RawMessage{json.RawMessage(`{}`), json.RawMessage(`{}`)})
	require.NoError(t, err)
	_, _, _, err = prepareStateAndTraceEventPayloads(benchmarkEvents(1024), serialized)
	require.EqualError(t, err, "serialized event count does not match event count")
}

func TestRawEventPayloadsMaterializesSerializedEventsInOrder(t *testing.T) {
	serialized, err := event.NewSerializedEvents([]json.RawMessage{
		json.RawMessage(`{"name":"first"}`),
		json.RawMessage(`{"name":"second"}`),
	})
	require.NoError(t, err)

	require.Equal(t, []json.RawMessage{
		json.RawMessage(`{"name":"first"}`),
		json.RawMessage(`{"name":"second"}`),
	}, rawEventPayloads(nil, serialized))
}

func TestPrepareStateAndTraceEventPayloadsPreservesMultipleEventOrder(t *testing.T) {
	expectedRawEvents := []json.RawMessage{
		json.RawMessage(`{"name":"first","data":{"value":1}}`),
		json.RawMessage(`{"name":"second","data":{"value":2}}`),
	}
	serialized, err := event.NewSerializedEvents(expectedRawEvents)
	require.NoError(t, err)
	trackedEvents := []event.TrackedEvent{
		event.NewBaseTrackedEvent(event.Event{Name: "first"}, nil),
		event.NewBaseTrackedEvent(event.Event{Name: "second"}, nil),
	}

	rawEvents, immutableEvents, traceInput, err := prepareStateAndTraceEventPayloads(trackedEvents, serialized)
	require.NoError(t, err)
	require.Nil(t, rawEvents)
	require.Equal(t, expectedRawEvents, rawEventPayloads(rawEvents, immutableEvents))
	require.Equal(t, `[{"name":"first","data":{"value":1}},{"name":"second","data":{"value":2}}]`, traceInput)
}

func BenchmarkPrepareStateAndTraceEventPayloadsFanout(b *testing.B) {
	events := benchmarkEvents(10 * 1024)

	b.Run("before_per_run_serialization", func(b *testing.B) {
		for range b.N {
			for range benchmarkEventFanout {
				benchmarkRawEvents, benchmarkImmutableEvents, benchmarkTraceInput, _ = prepareStateAndTraceEventPayloads(events, event.SerializedEvents{})
			}
		}
	})

	b.Run("after_shared_serialization", func(b *testing.B) {
		for range b.N {
			shared, _ := json.Marshal(events[0].GetEvent())
			serialized, _ := event.NewSerializedEvents([]json.RawMessage{shared})
			for range benchmarkEventFanout {
				benchmarkRawEvents, benchmarkImmutableEvents, benchmarkTraceInput, _ = prepareStateAndTraceEventPayloads(events, serialized)
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
