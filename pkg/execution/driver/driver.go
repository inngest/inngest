package driver

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/gowebpki/jcs"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
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
	req := &SDKInvokeRequest{
		Events:  s.Events(),
		Event:   s.Event(),
		Actions: s.Actions(),
		Context: &SDKInvokeRequestContext{
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
	// NOTE: Should this also be based on SDK versions?
	if req.IsBatchSizeTooLarge() {
		req.Events = nil
		req.BatchID = s.Identifier().BatchID
	}

	j, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return jcs.Transform(j)
}

type SDKInvokeRequest struct {
	Event   map[string]any           `json:"event,omitempty"`
	Events  []map[string]any         `json:"events,omitempty"`
	Actions map[string]any           `json:"steps"`
	Context *SDKInvokeRequestContext `json:"ctx"`
	BatchID *ulid.ULID               `json:"batch_id,omitempty"`
}

func (req *SDKInvokeRequest) IsBatchSizeTooLarge() bool {
	byt, err := json.Marshal(req.Events)
	if err != nil {
		// log error
		return false
	}
	return len(byt) > consts.MaxBodySize
}

type SDKInvokeRequestContext struct {
	// FunctionID is used within entrypoints to SDK-based functions in
	// order to specify the ID of the function to run via RPC.
	FunctionID uuid.UUID `json:"fn_id"`

	// Env is the name of the environment that the function is running in.
	// though this is self-discoverable most of the time, for static envs
	// the SDK has no knowledge of the name as it only has a signing key.
	Env string `json:"env"`

	// StepID is used within entrypoints to SDK-based functions in
	// order to specify the step of the function to run via RPC.
	StepID string `json:"step_id"`

	// XXX: Pass in opentracing context within ctx.
	RunID ulid.ULID `json:"run_id"`

	Stack *FunctionStack `json:"stack"`
}
