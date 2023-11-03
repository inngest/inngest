package driver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
)

type Driver interface {
	inngest.Runtime

	// Execute executes the given action for the given step.
	Execute(
		ctx context.Context,
		s state.State,
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
	s state.State,
	step inngest.Step,
	stackIndex int,
	env string,
	attempt int,
) ([]byte, error) {
	md := s.Metadata()

	fn := s.Function()
	events, err := mapInvocationEventNames(s.Events(), fn)
	if err != nil {
		return nil, fmt.Errorf("error mapping invocation event names: %w", err)
	}

	req := &SDKRequest{
		Events:  events,
		Event:   events[0],
		Actions: s.Actions(),
		Context: &SDKRequestContext{
			FunctionID: fn.ID,
			Env:        env,
			StepID:     step.ID,
			RunID:      s.RunID(),
			Stack: &FunctionStack{
				Stack:   s.Stack(),
				Current: stackIndex,
			},
			Attempt:                   attempt,
			DisableImmediateExecution: md.DisableImmediateExecution,
		},
		Version: md.RequestVersion,
	}

	// empty the attrs that consume the most
	if req.IsBodySizeTooLarge() {
		req.Events = []map[string]any{}
		req.Actions = map[string]any{}
		req.UseAPI = true
		req.Context.UseAPI = true
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

// mapInvocationEventNames sets the name of any invocation events to the name of the trigger event.
// This is used to ensure that SDKs receive the event they are expecting when invoked.
func mapInvocationEventNames(events []map[string]any, fn inngest.Function) ([]map[string]any, error) {
	for _, evt := range events {
		if name, ok := evt["name"].(string); !ok || name != event.InvokeFnName {
			continue
		}

		if len(fn.Triggers) != 1 {
			return nil, fmt.Errorf("invocation event found, but function %s has %d triggers; could not fill in invocation event name automatically", fn.Slug, len(fn.Triggers))
		}

		trigger := fn.Triggers[0]
		// Crons can keep the original event name
		if trigger.EventTrigger != nil && trigger.Event != "" {
			evt["name"] = trigger.Event
		}
	}
	return events, nil
}
