package state

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/inngest/inngest/pkg/dateutil"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/xhit/go-str2duration/v2"
	"golang.org/x/exp/slog"
)

const DefaultErrorMessage = "Unknown error running SDK"

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
	opts := &WaitForEventOpts{}
	if err := opts.UnmarshalAny(g.Opts); err != nil {
		return nil, err
	}
	if opts.Event == "" {
		// use the step name as a fallback, for v1/2 of the TS SDK.
		opts.Event = g.Name
	}
	if opts.Event == "" {
		return nil, fmt.Errorf("An event name must be provided when waiting for an event")
	}
	return opts, nil
}

func (g GeneratorOpcode) SleepDuration() (time.Duration, error) {
	if g.Op != enums.OpcodeSleep {
		return 0, fmt.Errorf("unable to return sleep duration for opcode %s", g.Op.String())
	}

	opts := &SleepOpts{}
	if err := opts.UnmarshalAny(g.Opts); err != nil {
		return 0, err
	}

	if opts.Duration == "" {
		// use step name as a fallback for v1/2 of the TS SDK
		opts.Duration = g.Name
	}
	if len(opts.Duration) == 0 {
		return 0, nil
	}

	// Quick heuristic to check if this is likely a date layout
	if len(opts.Duration) >= 10 {
		if parsed, err := dateutil.Parse(opts.Duration); err == nil {
			at := time.Until(parsed).Round(time.Second)
			if at < 0 {
				return time.Duration(0), nil
			}
			return at, nil
		}
	}

	return str2duration.ParseDuration(opts.Duration)
}

type SleepOpts struct {
	Duration string `json:"duration"`
}

func (s *SleepOpts) UnmarshalAny(a any) error {
	opts := SleepOpts{}
	var mappedByt []byte
	switch typ := a.(type) {
	case []byte:
		mappedByt = typ
	default:
		byt, err := json.Marshal(a)
		if err != nil {
			return err
		}
		mappedByt = byt
	}
	if err := json.Unmarshal(mappedByt, &opts); err != nil {
		return err
	}
	*s = opts
	return nil
}

type WaitForEventOpts struct {
	Timeout string  `json:"timeout"`
	If      *string `json:"if"`
	// Event is taken from GeneratorOpcode.Name if this is empty.
	Event string `json:"event"`
}

func (w *WaitForEventOpts) UnmarshalAny(a any) error {
	opts := WaitForEventOpts{}
	var mappedByt []byte
	switch typ := a.(type) {
	case []byte:
		mappedByt = typ
	default:
		byt, err := json.Marshal(a)
		if err != nil {
			return err
		}
		mappedByt = byt
	}
	if err := json.Unmarshal(mappedByt, &opts); err != nil {
		return err
	}
	*w = opts
	return nil
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

	// Duration is how long the step took to run, from the driver itsef.
	Duration time.Duration `json:"dur"`

	// RequestVersion represents the hashing version used within the current SDK request.
	//
	// This allows us to store the hash version for each function run to check backcompat.
	RequestVersion int `json:"request_version"`

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
	Generator []*GeneratorOpcode `json:"generator,omitempty"`

	// Scheduled, if set to true, represents that the action has been
	// scheduled and will run asynchronously.  The output is not available.
	//
	// Managing messaging and monitoring of asynchronous jobs is outside of
	// the scope of this executor.  It's possible to store your own queues
	// and state for managing asynchronous jobs in another manager.
	Scheduled bool `json:"scheduled"`

	// Output is the output from an action, as a JSON-marshalled value.
	Output any `json:"output"`

	// OutputSize is the size of the response payload, verbatim, in bytes.
	OutputSize int `json:"size"`

	// Err represents the error from the action, if the action errored.
	// If the action terminated successfully this must be nil.
	Err *string `json:"err"`

	// RetryAt is an optional retry at field, specifying when we should retry
	// the step if the step errored.
	RetryAt *time.Time `json:"retryAt,omitempty"`

	// Noretry, if true, indicates that we should never retry this step.
	NoRetry bool `json:"noRetry,omitempty"`

	// final indicates whether the error has been marked as final.  This occurs
	// when the response errors and the executor detects that this is the final
	// retry of the step.
	//
	// When final is true, Retryable() always returns false.
	final bool

	StatusCode int `json:"statusCode,omitempty"`
}

// SetFinal indicates that this error is final, regardless of the status code
// returned.  This is used to prevent retries when the max limit is reached.
func (r *DriverResponse) SetFinal() {
	r.NoRetry = true
	r.final = true
}

// SetError sets the Err field to the string of the error specified.
func (r *DriverResponse) SetError(err error) {
	if err == nil {
		return
	}
	str := err.Error()
	r.Err = &str
}

// NextRetryAt fulfils the queue.RetryAtSpecifier interface
func (r DriverResponse) NextRetryAt() *time.Time {
	return r.RetryAt
}

func (r DriverResponse) Error() string {
	if r.Err == nil {
		return ""
	}
	return *r.Err
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

	if r.NoRetry {
		// If there's a no retry flag set this is never retryable.
		return false
	}

	status := r.StatusCode
	if status == 0 {
		if mapped, ok := r.Output.(map[string]any); ok {
			// Fall back to statusCode for AWS Lambda compatibility in
			// an attempt to use this field.
			v, ok := mapped["statusCode"]
			if !ok {
				// If actions don't return a status, we assume that they're
				// always retryable.
				return true
			}

			switch val := v.(type) {
			case float64:
				status = int(val)
			case int64:
				status = int(val)
			case int:
				status = val
			default:
				slog.Default().Error(
					"unexpected status code type",
					"type", fmt.Sprintf("%T", v),
				)
			}
		}
	}

	if status == 0 {
		slog.Default().Error("missing status code")
		return true
	}

	if status > 499 {
		return true
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
//
// Final MUST exist as state stores need to push step IDs to the stack when the response
// is final.  We must do this prior to calling state.Finalized(), as the stack must be
// mutated prior to enqueuing steps.
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

// UserError returns the error that the user reported for this response. Can be
// used to safely fetch the error from the response.
//
// Will return nil if there is no error.
//
// An ideal error is in the type:
//
//	type UserError struct {
//	        Name    string      `json:"name"`
//	        Message string      `json:"message"`
//	        Stack   string      `json:"stack"`
//	        Cause   string      `json:"cause,omitempty"`
//	        Status  json.Number `json:"status,omitempty"`
//	}
//
// However, no types are defined, and we use any error we can get our hands on!
//
// NOTE: There are several required fields:  "name", "message".
func (r DriverResponse) UserError() map[string]any {
	if r.Output == nil && r.Err != nil {
		return map[string]any{
			"error":   *r.Err,
			"name":    "Error",
			"message": *r.Err,
		}
	}

	if mapped, ok := r.Output.(map[string]any); ok {
		if processed, err := processErrorFields(mapped); err == nil {
			// Ensure that all fields are added.
			if _, ok := processed["name"]; !ok {
				processed["name"] = "Error"
			}
			if _, ok := processed["message"]; !ok {
				processed["message"] = DefaultErrorMessage
			}
			return processed
		}
	}

	err := DefaultErrorMessage
	if r.Err != nil {
		err = *r.Err
	}

	output := any(DefaultErrorMessage)
	switch v := r.Output.(type) {
	case json.RawMessage:
		if len(v) > 0 {
			output = v
		}
	case []byte:
		if len(v) > 0 {
			output = v
		}
	case string:
		if len(v) > 0 {
			output = v
		}
	case interface{}:
		if v != nil {
			output = v
		}
	case nil:
		// ignore.
	default:
		output = v
	}

	return map[string]any{
		"error":   err,
		"name":    "Error",
		"message": output,
	}
}

// processErrorFields looks for an error field then a body field to handle
// error messages from step responses.
func processErrorFields(input map[string]any) (map[string]any, error) {
	fields := []string{"error", "body"}
	for _, f := range fields {
		// Attempt to fetch the JS/SDK error from the body.
		switch v := input[f].(type) {
		case map[string]any:
			return v, nil
		case json.RawMessage:
			if mapped, err := processErrorString(string(v)); err == nil {
				return mapped, nil
			}
		case []byte:
			if mapped, err := processErrorString(string(v)); err == nil {
				return mapped, nil
			}
		case string:
			if mapped, err := processErrorString(v); err == nil {
				return mapped, nil
			}
		}
	}
	return input, nil
}

// processErrorString attempts to unquote and unmarshal a JSON-encoded string
func processErrorString(s string) (map[string]any, error) {
	// Bound inner error fields to 32kb
	if len(s) > 32*1024 {
		return nil, fmt.Errorf("error field too large")
	}

	if unquote, err := strconv.Unquote(s); err == nil {
		s = unquote
	}
	mapped := map[string]any{}
	err := json.Unmarshal([]byte(s), &mapped)
	return mapped, err
}
