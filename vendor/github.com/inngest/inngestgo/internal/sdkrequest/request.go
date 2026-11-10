package sdkrequest

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Request represents an incoming invoke request used to call functions from Inngest.
type Request struct {
	// Event represents the input event.  If the input is a batch of events, this
	// represents the first event in the batch (for backwards compatibility).
	Event json.RawMessage `json:"event"`
	// Events represents the array of input events, if the function run is for
	// a batch of events.
	Events []json.RawMessage `json:"events"`
	// Steps indicates the current step state for the function run.
	Steps map[string]json.RawMessage `json:"steps"`
	// CallCtx represents call context - metadata around the current function run.
	CallCtx CallCtx `json:"ctx"`
	// UseAPI indicates whether the input request is too large (> 4MB) to be pushed
	// to each function run, and should instead be fetched from the API on run.
	//
	// Deprecated: this is moved into CallCtx for V3+/non-TypeScript SDKs.
	UseAPI bool `json:"use_api"`
	// Loop is set for loop functions only.  a loop request without it
	// fails the run with no retry.
	Loop *LoopRequest `json:"loop,omitempty"`
}

// LoopRequest carries the committed loop state and the input that woke
// the run.
type LoopRequest struct {
	// State is null until the first commit.  the handler then gets the
	// function's initial state.
	State   json.RawMessage `json:"state"`
	Version int64           `json:"version"`
	Key     string          `json:"key"`
	Input   LoopInput       `json:"input"`
}

// LoopInput says why the run was invoked.  Kind is one of "event",
// "subscription", "wake", and "tick".  ID is the wake id or the
// subscription id for those kinds and empty otherwise.
type LoopInput struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// UsesAPI reports whether the request data should be fetched from the API.
// It supports both the current context field and the deprecated top-level field.
func (r Request) UsesAPI() bool {
	return r.CallCtx.UseAPI || r.UseAPI
}

// CallCtx represents context for individual function calls.  This logs the function ID, the
// specific run ID, and sep information.
type CallCtx struct {
	DisableImmediateExecution bool      `json:"disable_immediate_execution"`
	UseAPI                    bool      `json:"use_api"`
	Env                       string    `json:"env"`
	FunctionID                uuid.UUID `json:"fn_id"`
	RunID                     string    `json:"run_id"`
	StepID                    string    `json:"step_id"`
	Stack                     CallStack `json:"stack"`
	Attempt                   int       `json:"attempt"`
	QueueItemRef              string    `json:"qi_id"`
	RequestID                 string    `json:"-"`
	JobID                     string    `json:"-"`
	// InstanceID is the connect worker's instance ID.  it is empty for
	// requests that do not come through connect.  checkpoint requests send it
	// in the x-inngest-instance-id header.
	InstanceID  string `json:"-"`
	MaxAttempts *int   `json:"max_attempts"`
}

type CallStack struct {
	Current uint     `json:"current"`
	Stack   []string `json:"stack"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
