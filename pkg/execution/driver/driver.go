package driver

import (
	"bytes"
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
				Stack:   md.Stack,
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

		// Here we trust the stack to be correct and represent all memoized
		// data for the function. We do this because we load steps separately
		// (and later) than the metadata, meaning a race condition exists
		// where more memoized step data could be added during that time.
		//
		// Therefore, we filter out any extraneous steps that aren't in the
		// stack to __pretend__ that we have loaded both the stack and state
		// atomically.
		//
		// Even though we are ignoring the most up-to-date state by using this
		// method, no steps will be re-executed, as this race condition is
		// only applicable to parallel steps, which are all planned and will
		// be filtered out with Executor-level idempotency.
		//
		// This is a workaround for the fact that we do not atomically load
		// both steps and the stack; that change should be made and this code
		// removed.
		for _, stepId := range md.Stack {
			// We also account here for the situation in which the loaded
			// step state does not contain a step ID in the stack. This is an
			// error that is impossible to recover from and is observed to
			// mean that step state has been entirely removed, such as at the
			// end of a function run.
			if _, ok := steps[stepId]; !ok {
				return nil, fmt.Errorf("state and stack mismatch: %s not found in state; the function has probably ended", stepId)
			}

			req.Actions[stepId] = steps[stepId]

			// Remove this key so we know which keys are left over at the end
			delete(steps, stepId)
		}

		// Check for altered inputs in memoized steps too - only send this if
		// the step has not yet finished and therefore is not in the stack.
		//
		// We're only checking remaining keys here so this is either inputs or
		// the small non-atomic edge case.
		for stepId, rawData := range steps {
			// Check if the raw JSON starts with `{"input"`` which indicates
			// it's a memoized step input.
			if bytes.HasPrefix(rawData, []byte(`{"input"`)) {
				req.Actions[stepId] = rawData
			}
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
