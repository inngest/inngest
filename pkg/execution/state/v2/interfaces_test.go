package state

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
)

func TestCreateStateMaterializeEvents(t *testing.T) {
	rawEvents := []json.RawMessage{
		json.RawMessage(`{"name":"first"}`),
		json.RawMessage(`{"name":"second"}`),
	}
	serialized, err := event.NewSerializedEvents(rawEvents)
	require.NoError(t, err)
	state := CreateState{SerializedEvents: serialized}

	require.Equal(t, rawEvents, state.MaterializeEvents())
	require.Empty(t, state.SerializedEvents)
	require.Equal(t, state.Events, state.MaterializeEvents(), "materialized events should be reused")
}
