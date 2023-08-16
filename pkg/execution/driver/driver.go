package driver

import (
	"context"
	"encoding/json"

	"github.com/gowebpki/jcs"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
)

type Driver interface {
	inngest.Runtime

	// Execute executes the given action for the given step.
	Execute(
		ctx context.Context,
		s state.State,
		edge inngest.Edge,
		step inngest.Step,
		stackIndex int,
		attempt int,
	) (*state.DriverResponse, error)
}

// MarshalV1 marshals state as an input to driver runtimes.
func MarshalV1(
	ctx context.Context,
	s state.State,
	edge inngest.Edge,
	step inngest.Step,
	stackIndex int,
	env string,
	attempt int,
) ([]byte, error) {
	req := &SDKRequest{
		Events:  s.Events(),
		Event:   s.Event(),
		Actions: s.Actions(),
		Context: &SDKRequestContext{
			FunctionID: s.Function().ID,
			Env:        env,
			StepID:     step.ID,
			RunID:      s.RunID(),
			Stack: &FunctionStack{
				Stack:   s.Stack(),
				Current: stackIndex,
			},
			Attempt: attempt,
		},
	}

	if edge.DisableImmediateExecution {
		req.DisableImmediateExecution = true
	}

	// empty the attrs that consume the most
	if req.IsBodySizeTooLarge() {
		req.Events = []map[string]any{}
		req.Actions = map[string]any{}
		req.UseAPI = true
	}

	j, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return jcs.Transform(j)
}
