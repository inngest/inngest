package driver

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type SDKRequest struct {
	Event   map[string]any     `json:"event"`
	Events  []map[string]any   `json:"events"`
	Actions map[string]any     `json:"steps"`
	Context *SDKRequestContext `json:"ctx"`
	// Version indicates the version used to manage the SDK request context.
	//
	// A value of -1 means that the function is starting and has no version.
	Version int `json:"version"`

	// DEPRECATED: NOTE: This is moved into SDKRequestContext for V3+/Non-TS SDKs
	UseAPI bool `json:"use_api"`
}

type SDKRequestContext struct {
	// FunctionID is used within entrypoints to SDK-based functions in
	// order to specify the ID of the function to run via RPC.
	FunctionID uuid.UUID `json:"fn_id"`

	// RunID  is the ID of the current
	RunID ulid.ULID `json:"run_id"`

	// Env is the name of the environment that the function is running in.
	// though this is self-discoverable most of the time, for static envs
	// the SDK has no knowledge of the name as it only has a signing key.
	Env string `json:"env"`

	// StepID is used within entrypoints to SDK-based functions in
	// order to specify the step of the function to run via RPC.
	StepID string `json:"step_id"`

	// Attempt is the zero-index attempt number.
	Attempt int `json:"attempt"`

	// Stack represents the function stack at the time of the step invocation.
	Stack *FunctionStack `json:"stack"`

	// DisableImmediateExecution is used to tell the SDK whether it should
	// disallow immediate execution of steps as they are found.
	DisableImmediateExecution bool `json:"disable_immediate_execution"`

	// UseAPI tells the SDK to retrieve `Events` and `Actions` data
	// from the API instead of expecting it to be in the request body.
	// This is a way to get around serverless provider's request body
	// size limits.
	UseAPI bool `json:"use_api"`

	// XXX: Pass in opentracing context within ctx.
}

type FunctionStack struct {
	Stack   []string `json:"stack"`
	Current int      `json:"current"`
}

func (m FunctionStack) MarshalJSON() ([]byte, error) {
	if m.Stack == nil {
		m.Stack = make([]string, 0)
	}

	type alias FunctionStack // Avoid infinite recursion
	return json.Marshal(alias(m))
}
