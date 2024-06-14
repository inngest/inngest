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
	rawEvts, err := sl.LoadEvents(ctx, md.ID)
	if err != nil {
		return nil, fmt.Errorf("error loading events in driver marshaller: %w", err)
	}

	evts := make([]map[string]any, len(rawEvts))
	for n, i := range rawEvts {
		evts[n] = map[string]any{}
		if err := json.Unmarshal(i, &evts[n]); err != nil {
			return nil, fmt.Errorf("error unmarshalling event in driver marshaller: %w", err)
		}
		// ensure the user object is always an array.
		if evts[n]["user"] == nil {
			evts[n]["user"] = map[string]any{}
		}
	}

	stack := md.Stack
	if stack == nil {
		stack = make([]string, 0)
	}

	req := &SDKRequest{
		// For backcompat, we always send `Event`, but `Events` could be made
		// empty if the overall request size is too large.
		Event:   evts[0],
		Events:  evts,
		Actions: map[string]any{},
		Context: &SDKRequestContext{
			UseAPI:     true,
			FunctionID: md.ID.FunctionID,
			Env:        env,
			StepID:     step.ID,
			RunID:      md.ID.RunID,
			Stack: &FunctionStack{
				Stack:   stack,
				Current: stackIndex,
			},
			Attempt:                   attempt,
			DisableImmediateExecution: md.Config.ForceStepPlan,
		},
		Version: md.Config.RequestVersion,
		UseAPI:  true,
	}

	// Ensure that we're not sending data that's too large to the SDK.
	if md.Metrics.StateSize <= (consts.MaxBodySize - 1024) {
		// Load the actual function state here.
		steps, err := sl.LoadSteps(ctx, md.ID)
		if err != nil {
			return nil, fmt.Errorf("error loading state in driver marshaller: %w", err)
		}

		// We do not need to unmarshal state, as it's already marshalled.
		for k, v := range steps {
			req.Actions[k] = v
		}

		req.UseAPI = false
		req.Context.UseAPI = false
	}

	j, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request to JSON: %w", err)
	}

	// And here, to double check, ensure that the length isn't excessive once again.
	// This is because, as Jack points out, for backcompat we send both events and the
	// first event.  We also may have incorrect state sizes for runs before this is tracked.
	if len(j) > consts.MaxBodySize {
		req.Events = []map[string]any{}
		req.Actions = map[string]any{}
		req.UseAPI = true
		req.Context.UseAPI = true
		if j, err = json.Marshal(req); err != nil {
			return nil, fmt.Errorf("error marshalling request to JSON: %w", err)
		}

	}

	b, err := jcs.Transform(j)
	if err != nil {
		return nil, fmt.Errorf("error transforming request with JCS: %w", err)
	}

	return b, nil
}
