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
	benchmarkImmutableEvents []string
	benchmarkTraceInput      string
)

func TestPrepareStateAndTraceEventPayloads(t *testing.T) {
	events := benchmarkEvents(1024)

	rawEvents, _, traceInput, err := prepareStateAndTraceEventPayloads(events, nil)
	require.NoError(t, err)

	shared, err := json.Marshal(events[0].GetEvent())
	require.NoError(t, err)
	immutable := string(shared)
	sharedRawEvents, sharedImmutableEvents, sharedTraceInput, err := prepareStateAndTraceEventPayloads(events, []string{immutable})
	require.NoError(t, err)

	require.Nil(t, sharedRawEvents)
	require.Equal(t, []string{immutable}, sharedImmutableEvents)
	require.Equal(t, traceInput, sharedTraceInput)

	shared[0] = 'x'
	require.Equal(t, byte('{'), immutable[0], "immutable snapshot must not alias its source bytes")
	require.NotEqual(t, string(rawEvents[0]), string(shared))
}

func TestPrepareStateAndTraceEventPayloadsRejectsMismatchedSerializedEvents(t *testing.T) {
	_, _, _, err := prepareStateAndTraceEventPayloads(benchmarkEvents(1024), []string{`{}`, `{}`})
	require.EqualError(t, err, "serialized event count does not match event count")
}

func TestRawEventPayloadsMaterializesSerializedEventsInOrder(t *testing.T) {
	serialized := []string{`{"name":"first"}`, `{"name":"second"}`}

	require.Equal(t, []json.RawMessage{
		json.RawMessage(`{"name":"first"}`),
		json.RawMessage(`{"name":"second"}`),
	}, rawEventPayloads(nil, serialized))
}

func BenchmarkPrepareStateAndTraceEventPayloadsFanout(b *testing.B) {
	events := benchmarkEvents(10 * 1024)

	b.Run("before_per_run_serialization", func(b *testing.B) {
		for range b.N {
			for range benchmarkEventFanout {
				benchmarkRawEvents, benchmarkImmutableEvents, benchmarkTraceInput, _ = prepareStateAndTraceEventPayloads(events, nil)
			}
		}
	})

	b.Run("after_shared_serialization", func(b *testing.B) {
		for range b.N {
			shared, _ := json.Marshal(events[0].GetEvent())
			serialized := []string{string(shared)}
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
