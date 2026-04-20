package fn

import "github.com/inngest/inngestgo/internal/event"

// Input is the input data passed to your function.  It is comprised of the triggering event
// and call context.
type Input[DATA any] struct {
	Event    event.GenericEvent[DATA]   `json:"event"`
	Events   []event.GenericEvent[DATA] `json:"events"`
	InputCtx InputCtx                   `json:"ctx"`
}

type InputCtx struct {
	Env        string `json:"env"`
	FunctionID string `json:"fn_id"`
	RunID      string `json:"run_id"`
	StepID     string `json:"step_id"`
	Attempt    int    `json:"attempt"`
}
