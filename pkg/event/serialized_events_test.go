package event

import (
	"encoding/json"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestSerializedEventsOwnsImmutableEventAndInputSnapshots(t *testing.T) {
	tests := []struct {
		name   string
		events []json.RawMessage
		input  string
	}{
		{name: "empty", input: `[]`},
		{name: "single", events: []json.RawMessage{json.RawMessage(`{"name":"app/event"}`)}, input: `[{"name":"app/event"}]`},
		{name: "multiple", events: []json.RawMessage{
			json.RawMessage(`{"name":"first","data":{"value":1}}`),
			json.RawMessage(`{"name":"second"}`),
			json.RawMessage(`{"name":"third","data":[1,2,3]}`),
		}, input: `[{"name":"first","data":{"value":1}},{"name":"second"},{"name":"third","data":[1,2,3]}]`},
		{name: "multibyte UTF-8", events: []json.RawMessage{
			json.RawMessage(`{"name":"café","data":{"city":"아산"}}`),
			json.RawMessage(`{"name":"아산","data":{"drink":"café"}}`),
		}, input: `[{"name":"café","data":{"city":"아산"}},{"name":"아산","data":{"drink":"café"}}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedEvents := make([]string, len(tt.events))
			for i := range tt.events {
				expectedEvents[i] = string(tt.events[i])
			}

			serialized, err := NewSerializedEvents(tt.events)
			require.NoError(t, err)
			for i := range tt.events {
				tt.events[i][0] = 'x'
				require.Equal(t, expectedEvents[i], serialized.Event(i))
			}
			require.Equal(t, tt.input, serialized.Input())

			raw := serialized.RawMessages()
			for i := range raw {
				raw[i][0] = 'x'
				require.Equal(t, byte('{'), serialized.Event(i)[0])
			}
		})
	}
}

func TestSerializedEventsRejectsInvalidJSON(t *testing.T) {
	_, err := NewSerializedEvents([]json.RawMessage{json.RawMessage(`{"name":`)})
	require.EqualError(t, err, "serialized event 0 is invalid JSON")
}

func TestSerializedEventsRetainsTrackedMetadataInEventOrder(t *testing.T) {
	firstID := ulid.Make()
	secondID := ulid.Make()
	tracked := []TrackedEvent{
		NewBaseTrackedEventWithID(Event{Name: "café", Data: map[string]any{"city": "아산"}}, firstID),
		NewBaseTrackedEventWithID(Event{Name: "아산", Data: map[string]any{"drink": "café"}}, secondID),
	}

	serialized, err := NewSerializedEventsFromTrackedEvents(tracked)
	require.NoError(t, err)
	require.True(t, serialized.HasTrackedEvents())
	require.Equal(t, firstID, serialized.TrackedEvent(0).GetInternalID())
	require.Equal(t, secondID, serialized.TrackedEvent(1).GetInternalID())
	require.Equal(t, "café", serialized.TrackedEvent(0).GetEvent().Name)
	require.Equal(t, "아산", serialized.TrackedEvent(1).GetEvent().Name)
	require.Equal(t, `[{"name":"café","data":{"city":"아산"}},{"name":"아산","data":{"drink":"café"}}]`, serialized.Input())
	require.Equal(t, serialized.Event(0), string(serialized.RawMessages()[0]))
}
