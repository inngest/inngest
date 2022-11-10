package state

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/dateutil"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/xhit/go-str2duration/v2"
)

type Retryable interface {
	Retryable() bool
}

type GeneratorOpcode struct {
	// Op represents the type of operation invoked in the function.
	Op enums.Opcode `json:"op"`
	// ID represents a hashed unique ID for the operation.  This acts
	// as the generated step ID for the state store.
	ID string `json:"id"`
	// Name represents the name of the step, or the sleep duration for
	// sleeps.
	Name string `json:"name"`
	// Opts indicate options for the operation, eg. matching expressions
	// when setting up async event listeners via `waitForEvent`, or retry
	// policies for steps.
	Opts any `json:"opts"`
	// Data is the resulting data from the operation, eg. the step
	// output.
	Data json.RawMessage `json:"data"`
}

func (g GeneratorOpcode) WaitForEventOpts() (*WaitForEventOpts, error) {
	opts := WaitForEventOpts{
		Event: g.Name,
	}
	if opts.Event == "" {
		return nil, fmt.Errorf("An event name must be provided when waiting for an event")
	}

	if err := json.Unmarshal(g.Data, &opts); err != nil {
		return nil, err
	}
	return &opts, nil
}

func (g GeneratorOpcode) SleepDuration() (time.Duration, error) {
	if g.Op != enums.OpcodeSleep {
		return 0, fmt.Errorf("unable to return sleep duration for opcode %s", g.Op.String())
	}

	// Quick heuristic to check if this is likely a date layout
	if len(g.Name) >= 10 {
		if parsed, err := dateutil.Parse(g.Name); err == nil {
			return time.Until(parsed).Round(time.Second), nil
		}
	}

	return str2duration.ParseDuration(g.Name)
}

type WaitForEventOpts struct {
	Timeout string  `json:"timeout"`
	If      *string `json:"if"`
	// Event is taken from GeneratorOpcode.ID
	Event string `json:"-"`
}

func (w WaitForEventOpts) Expires() (time.Time, error) {
	dur, err := str2duration.ParseDuration(w.Timeout)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(dur), nil
}

// DriverResponse is returned after a driver executes an action.  This represents any
// output from running the step, including the output (as a JSON map), the error, and
// whether the driver's response is "scheduled", eg. the driver is running the job
// asynchronously.
//
// In asynchronous cases, we expect that the driver informs us of the response via
// an event in the future.
type DriverResponse struct {
	// Step represents the step that this response is for.
	Step inngest.Step `json:"step"`

	// Generator indicates that this response is a partial repsonse from a
	// SDK-based step (generator) function.  These functions are invoked
	// multiple times with function state, and return a 206 Partial Content
	// with an opcode indicating the next action (eg. wait for event, run step,
	// sleep, etc.)
	//
	// The flow for an SDK-based step/generator function is:
	//
	//    1. Function runs.
	//    2. It hits a step.  The step immediately runs, and we return an
	//       opcode [consts.RanStep, "step name/data", { output }]
	//    3. We store this in the state, then continue to invoke the function
	//       with mutated state.  Each tool inside the function (step/wait)
	//       returns a new opcode which we store in step state.
	Generator *GeneratorOpcode `json:"generator,omitempty"`

	// Scheduled, if set to true, represents that the action has been
	// scheduled and will run asynchronously.  The output is not available.
	//
	// Managing messaging and monitoring of asynchronous jobs is outside of
	// the scope of this executor.  It's possible to store your own queues
	// and state for managing asynchronous jobs in another manager.
	Scheduled bool `json:"scheduled"`

	// Output is the output from an action, as a JSON-marshalled value.
	Output any `json:"output"`

	// Err represents the error from the action, if the action errored.
	// If the action terminated successfully this must be nil.
	Err error `json:"err"`

	// ActionVersion returns the version of the action executed, as some workflows
	// may have ranges.  This must be included in a driver.Response as this is the
	// return result from an executor.
	ActionVersion *inngest.VersionInfo `json:"actionVersion"`

	// final indicates whether the error has been marked as final.  This occurs
	// when the response errors and the executor detects that this is the final
	// retry of the step.
	//
	// When final is true, Retryable() always returns false.
	final bool
}

// SetFinal indicates that this error is final, regardless of the status code
// returned.  This is used to prevent retries when the max limit is reached.
func (r *DriverResponse) SetFinal() {
	r.final = true
}

// Retryable returns whether the response indicates that the action is
// retryable.
//
// This is based of the action's output.  If the output contains a "status"
// field, we retry on any 5xx status; 4xx statuses are _not_ retried.  If the
// output contains no "status" field, we will always assume that we can retry
// the action.
//
// Note that responses where Err is nil are not retryable, and if Final() is
// set to true this response is also not retryable.
func (r DriverResponse) Retryable() bool {
	if r.Err == nil || r.final {
		return false
	}

	// Convert output into a map to check whether this responds with our
	// suggested JSON response
	mapped, ok := r.Output.(map[string]any)
	if !ok {
		// This doesn't contain the response, so default to retrying.
		return true
	}

	status, ok := mapped["status"]
	if !ok {
		// Fall back to statusCode for AWS Lambda compatibility in
		// an attempt to use this field.
		status, ok = mapped["statusCode"]
		if !ok {
			// If actions don't return a status, we assume that they're
			// always retryable.  We prefer that actions respond with a
			// { "status": xxx, "body": ... } format to disable retries.
			return true
		}
	}

	switch v := status.(type) {
	case float64:
		if int(v) > 499 {
			return true
		}
	case int64:
		if int(v) > 499 {
			return true
		}
	case int:
		if int(v) > 499 {
			return true
		}
	}
	return false
}

// Final returns whether this response is final and the backing state store can
// record this step as finalized when recording the response.
//
// Only non-retryable errors should be marked as final;  successful responses will
// have their child edges evaluated and should be recorded as final once the next
// steps are enqueued.  This ensures that the number of scheduled and finalized steps
// in state only matches once the function ends.
func (r *DriverResponse) Final() bool {
	if r.final {
		return true
	}

	// If there's an error, return true if the error is not retryable.
	if r.Err != nil && !r.Retryable() {
		return true
	}

	return false
}

// Error allows Response to fulfil the Error interface.
func (r DriverResponse) Error() string {
	if r.Err == nil {
		return ""
	}
	return r.Err.Error()
}

func (r DriverResponse) Unwrap() error {
	return r.Err
}
