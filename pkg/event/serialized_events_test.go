package event

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerializedEventsOwnsImmutableEventAndInputSnapshots(t *testing.T) {
	source := json.RawMessage(`{"name":"app/event"}`)
	serialized, err := NewSerializedEvents([]json.RawMessage{source})
	require.NoError(t, err)

	source[0] = 'x'
	require.Equal(t, `{"name":"app/event"}`, serialized.Event(0))
	require.Equal(t, `[{"name":"app/event"}]`, serialized.Input())

	raw := serialized.RawMessages()
	raw[0][0] = 'x'
	require.Equal(t, byte('{'), serialized.Event(0)[0])
}

func TestSerializedEventsRejectsInvalidJSON(t *testing.T) {
	_, err := NewSerializedEvents([]json.RawMessage{json.RawMessage(`{"name":`)})
	require.EqualError(t, err, "serialized event 0 is invalid JSON")
}
