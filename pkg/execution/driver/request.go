package driver

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/oklog/ulid/v2"
)

type SDKRequest struct {
	Event   map[string]any     `json:"event,omitempty"`
	Events  []map[string]any   `json:"events,omitempty"`
	Actions map[string]any     `json:"steps"`
	Context *SDKRequestContext `json:"ctx"`

	// UseAPI tells the SDK to retrieve `Events` and `Actions` data
	// from the API instead of expecting it to be in the request body.
	// This is a way to get around serverless provider's request body
	// size limits.
	UseAPI bool `json:"use_api"`
}

func (req *SDKRequest) IsBodySizeTooLarge() bool {
	byt, err := json.Marshal(req)
	if err != nil {
		// log error
		return false
	}
	return len(byt) >= consts.MaxBodySize
}

type SDKRequestContext struct {
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
