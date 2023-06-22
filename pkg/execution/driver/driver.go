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
	) (*state.DriverResponse, error)
}

type FunctionStack struct {
	Stack   []string `json:"stack"`
	Current int      `json:"current"`
}

// MarshalV1 marshals state as an input to driver runtimes.
func MarshalV1(ctx context.Context, s state.State, step inngest.Step, stackIndex int, env string) ([]byte, error) {
	req := &SDKRequest{
		// Events:  s.Events(),
		Events:  []map[string]any{},
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
		},
	}

	// if req.IsBodySizeTooLarge() {
	// 	req.Events = []map[string]any{}
	// 	req.Actions = map[string]any{}
	// 	req.UseAPI = true
	// }

	j, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return jcs.Transform(j)
}
