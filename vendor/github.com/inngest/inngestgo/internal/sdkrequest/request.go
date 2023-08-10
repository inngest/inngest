package sdkrequest

import "encoding/json"

// Request represents an incoming invoke request used to call functions from Inngest.
type Request struct {
	Event   json.RawMessage            `json:"event"`
	Events  []json.RawMessage          `json:"events"`
	Steps   map[string]json.RawMessage `json:"steps"`
	CallCtx CallCtx                    `json:"ctx"`
	UseAPI  bool                       `json:"use_api"`
}

// CallCtx represents context for individual function calls.  This logs the function ID, the
// specific run ID, and sep information.
type CallCtx struct {
	Env        string    `json:"env"`
	FunctionID string    `json:"fn_id"`
	RunID      string    `json:"run_id"`
	StepID     string    `json:"step_id"`
	Stack      CallStack `json:"stack"`
}

type CallStack struct {
	Current uint     `json:"current"`
	Stack   []string `json:"stack"`
}
