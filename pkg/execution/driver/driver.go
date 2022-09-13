package driver

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function/env"
)

type Driver interface {
	inngest.Runtime

	// Execute executes the given action for the given step.
	Execute(ctx context.Context, s state.State, av inngest.ActionVersion, step inngest.Step) (*state.DriverResponse, error)
}

// EnvManager is a driver which reads and utilizes environment variables when
// executing actions.  For example, the Docker driver utilizes an EnvReader to
// read specific env variables for each exectuion.
type EnvManager interface {
	SetEnvReader(r env.EnvReader)
}

// MarshalV1 marshals state as an input to driver runtimes.
func MarshalV1(ctx context.Context, s state.State, step inngest.Step) ([]byte, error) {
	data := map[string]interface{}{
		"event": s.Event(),
		"steps": s.Actions(),
		"ctx": map[string]interface{}{
			// fn_id is used within entrypoints to SDK-based functions in
			// order to specify the ID of the function to run via RPC.
			"fn_id": s.Workflow().ID,
			// step_id is used within entrypoints to SDK-based functions in
			// order to specify the step of the function to run via RPC.
			"step_id": step.ID,
			// XXX: Pass in opentracing context within ctx.
			"run_id": s.RunID(),
		},
	}
	return json.Marshal(data)
}
