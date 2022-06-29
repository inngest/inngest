package driver

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/state"
)

type Driver interface {
	inngest.Runtime

	// Execute executes the given action for the given step.
	Execute(ctx context.Context, s state.State, av inngest.ActionVersion, step inngest.Step) (*state.DriverResponse, error)
}

// MarshalV1 marshals state as an input to driver runtimes.
func MarshalV1(s state.State) ([]byte, error) {
	data := map[string]interface{}{
		"event": s.Event(),
		"steps": s.Actions(),
		"ctx": map[string]interface{}{
			"workflow_id": s.WorkflowID(),
		},
	}
	return json.Marshal(data)
}
