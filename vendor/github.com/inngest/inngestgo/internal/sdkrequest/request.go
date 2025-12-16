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
	UseAPI bool `json:"use_api"`
}

// CallCtx represents context for individual function calls.  This logs the function ID, the
// specific run ID, and sep information.
type CallCtx struct {
	DisableImmediateExecution bool      `json:"disable_immediate_execution"`
	Env                       string    `json:"env"`
	FunctionID                uuid.UUID `json:"fn_id"`
	RunID                     string    `json:"run_id"`
	StepID                    string    `json:"step_id"`
	Stack                     CallStack `json:"stack"`
	Attempt                   int       `json:"attempt"`
	QueueItemRef              string    `json:"qi_id"`
	MaxAttempts               *int      `json:"max_attempts"`
}

type CallStack struct {
	Current uint     `json:"current"`
	Stack   []string `json:"stack"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}
