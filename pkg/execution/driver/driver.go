package driver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
)

type Driver interface {
	inngest.Runtime

	// Execute executes the given action for the given step.
	Execute(
		ctx context.Context,
		sl sv2.StateLoader,
		md sv2.Metadata,
		item queue.Item,
		edge inngest.Edge,
		step inngest.Step,
		stackIndex int,
		attempt int,
	) (*state.DriverResponse, error)
}

// MarshalV1 marshals state as an input to driver runtimes.
func MarshalV1(
	ctx context.Context,
	sl sv2.StateLoader,
	md sv2.Metadata,
	step inngest.Step,
	stackIndex int,
	env string,
	attempt int,
) ([]byte, error) {

	req := &SDKRequest{
		Event:   map[string]any{},
		Events:  []map[string]any{},
		Actions: map[string]any{},
		Context: &SDKRequestContext{
			UseAPI:     true,
			FunctionID: md.ID.FunctionID,
			Env:        env,
			StepID:     step.ID,
			RunID:      md.ID.RunID,
			Stack: &FunctionStack{
				Stack:   md.Stack,
				Current: stackIndex,
			},
			Attempt:                   attempt,
			DisableImmediateExecution: md.Config.ForceStepPlan,
		},
		Version: md.Config.RequestVersion,
		UseAPI:  true,
	}

	if md.Metrics.StateSize <= consts.MaxBodySize {
		// Load the actual function state here.
		state, err := sl.LoadState(ctx, md.ID)
		if err != nil {
			return nil, fmt.Errorf("error loading state in driver marshaller: %w", err)
		}

		// Unmarshal events.
		evts := make([]map[string]any, len(state.Events))
		for n, i := range state.Events {
			evts[n] = map[string]any{}
			if err := json.Unmarshal(i, &evts[n]); err != nil {
				return nil, fmt.Errorf("error unmarshalling event in driver marshaller: %w", err)
			}
		}
		req.Event = evts[0]
		req.Events = evts

		// We do not need to unmarshal state, as it's already marshalled.
		for k, v := range state.Steps {
			req.Actions[k] = v
		}
	}

	j, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request to JSON: %w", err)
	}

	b, err := jcs.Transform(j)
	if err != nil {
		return nil, fmt.Errorf("error transforming request with JCS: %w", err)
	}

	return b, nil
}
