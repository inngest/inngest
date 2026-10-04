package state

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateStateMaterializeEvents(t *testing.T) {
	rawEvents := []json.RawMessage{
		json.RawMessage(`{"name":"first"}`),
		json.RawMessage(`{"name":"second"}`),
	}
	serialized := []string{string(rawEvents[0]), string(rawEvents[1])}
	state := CreateState{SerializedEvents: serialized}

	require.Equal(t, rawEvents, state.MaterializeEvents())
	require.Empty(t, state.SerializedEvents)
	require.Equal(t, state.Events, state.MaterializeEvents(), "materialized events should be reused")
}
