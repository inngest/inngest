package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestFinalizeEventsNoLifecycleRecursion ensures runs triggered by internal
// lifecycle events never emit lifecycle events of their own. Otherwise a
// cancelled function.cancelled handler emits another function.cancelled, which
// triggers every subscribed handler again (INC-1389).
func TestFinalizeEventsNoLifecycleRecursion(t *testing.T) {
	tests := []struct {
		name       string
		trigger    string
		cancel     bool
		wantEvents []string
	}{
		{
			name:       "user event, cancelled",
			trigger:    "app/user.created",
			cancel:     true,
			wantEvents: []string{event.FnFinishedName, event.FnCancelledName},
		},
		{
			name:       "user event, completed",
			trigger:    "app/user.created",
			wantEvents: []string{event.FnFinishedName},
		},
		{
			name:    "function.cancelled trigger, cancelled",
			trigger: event.FnCancelledName,
			cancel:  true,
		},
		{
			name:    "function.cancelled trigger, completed",
			trigger: event.FnCancelledName,
		},
		{
			name:    "function.failed trigger, cancelled",
			trigger: event.FnFailedName,
			cancel:  true,
		},
		{
			name:    "function.finished trigger, cancelled",
			trigger: event.FnFinishedName,
			cancel:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var published []event.Event
			called := false
			e := &executor{
				log: logger.VoidLogger(),
				finishHandler: func(_ context.Context, _ sv2.ID, evts []event.Event) error {
					called = true
					published = append(published, evts...)
					return nil
				},
			}

			trigger, err := json.Marshal(event.Event{
				ID:   ulid.Make().String(),
				Name: tc.trigger,
				Data: map[string]any{"run_id": ulid.Make().String()},
			})
			require.NoError(t, err)

			opts := execution.FinalizeOpts{
				Metadata: sv2.Metadata{ID: sv2.ID{
					RunID:      ulid.Make(),
					FunctionID: uuid.New(),
					Tenant:     sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New()},
				}},
				Response: execution.FinalizeResponse{Type: execution.FinalizeResponseRunComplete},
				Optional: execution.FinalizeOptional{
					FnSlug:      "app-handler",
					InputEvents: []json.RawMessage{trigger},
					Cancel:      tc.cancel,
				},
			}

			require.NoError(t, e.finalizeEvents(context.Background(), opts, nil))
			require.True(t, called, "finish handler must always be called")

			names := make([]string, 0, len(published))
			for _, evt := range published {
				names = append(names, evt.Name)
			}
			require.ElementsMatch(t, tc.wantEvents, names)
		})
	}
}
