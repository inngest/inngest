package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/stretchr/testify/require"
)

func testSerializedEvents(t *testing.T, events ...event.TrackedEvent) event.SerializedEvents {
	t.Helper()
	serialized, err := event.NewSerializedEventsFromTrackedEvents(events)
	require.NoError(t, err)
	return serialized
}

func TestScheduleRejectsSerializedEventsWithoutTrackedMetadata(t *testing.T) {
	serialized, err := event.NewSerializedEvents([]json.RawMessage{json.RawMessage(`{"name":"test"}`)})
	require.NoError(t, err)

	_, _, err = (&executor{}).Schedule(context.Background(), execution.ScheduleRequest{Events: serialized})
	require.EqualError(t, err, "schedule request events do not contain tracked metadata")
}
