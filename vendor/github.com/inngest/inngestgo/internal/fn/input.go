package fn

import (
	"encoding/json"

	"github.com/inngest/inngestgo/internal/event"
)

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
	RequestID  string `json:"request_id"`
	JobID      string `json:"job_id"`
}

// LoopInputKind says what started a loop run.
type LoopInputKind string

const (
	// LoopInputEvent is a run started by a triggering event.
	LoopInputEvent LoopInputKind = "event"
	// LoopInputSubscription is a run started by an event the instance
	// subscribed to.  LoopCtx.ID is the subscription id.
	LoopInputSubscription LoopInputKind = "subscription"
	// LoopInputWake is a run started by a wake the instance asked for.
	// LoopCtx.ID is the wake id.
	LoopInputWake LoopInputKind = "wake"
	// LoopInputTick is a run started by the tick interval in the loop
	// config.
	LoopInputTick LoopInputKind = "tick"
)

// LoopInput is the read-only input to a loop handler.  Event, Events, and
// InputCtx match Input so the handler reads both through the same
// reflection path.  the state is the handler's third argument.
type LoopInput[EVENT any] struct {
	Event    event.GenericEvent[EVENT]   `json:"event"`
	Events   []event.GenericEvent[EVENT] `json:"events"`
	InputCtx InputCtx                    `json:"ctx"`
	Loop     LoopCtx
}

// LoopCtx says what kind of input started this run.  ID is the wake id or
// the subscription id for those kinds and empty otherwise.  Version is the
// state version the run loaded.
type LoopCtx struct {
	Kind LoopInputKind
	ID   string
	// Key is the value of the loop's key expression for this instance.
	Key     string
	Version int64
}

// StatefulFunction is a function with a state record per key.
// InitialState returns the state for an instance that has no commit yet.
type StatefulFunction interface {
	InitialState() json.RawMessage
}
