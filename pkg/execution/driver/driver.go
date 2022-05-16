package driver

import (
	"context"
	"encoding/json"

	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/execution/state"
)

type Driver interface {
	inngest.Runtime
	Execute(context.Context, state.State, inngest.ActionVersion, inngest.Step) (*Response, error)
}

type Response struct {
	// Scheduled, if set to true, represents that the action has been
	// scheduled and will run asynchronously.  The output is not available.
	//
	// Managing messaging and monitoring of asynchronous jobs is outside of
	// the scope of this executor.  It's possible to store your own queues
	// and state for managing asynchronous jobs in another manager.
	Scheduled bool

	// Output is the output from an action, as a JSON map.
	Output map[string]interface{}

	// Err represents the error from the action, if the action errored.
	// If the action terminated successfully this must be nil.
	Err error
}

// Retryable returns whether the response indicates that the action is
// retryable.
//
// This is based of the action's output.  If the output contains a "status"
// field, we retry on any 5xx status; 4xx statuses are _not_ retried.  If the
// output contains no "status" field, we will always assume that we can retry
// the action.
//
// Note that responses where Err is nil are not retryable.
func (r Response) Retryable() bool {
	if r.Err == nil {
		return false
	}

	status, ok := r.Output["status"]
	if !ok {
		// If actions don't return a status, we assume that they're
		// always retryable.  We prefer that actions respond with a
		// { "status": xxx, "body": ... } format to disable retries.
		return true
	}
	switch v := status.(type) {
	case float64:
		if int(v) >= 499 {
			return true
		}
	case int64:
		if int(v) >= 499 {
			return true
		}
	case int:
		if int(v) >= 499 {
			return true
		}
	}
	return false
}

// Error allows Response to fulfil the Error interface.
func (r Response) Error() string {
	if r.Err == nil {
		return ""
	}
	return r.Err.Error()
}

func (r Response) Unwrap() error {
	return r.Err
}

// MarshalV1 marshals state as an input to driver runtimes.
func MarshalV1(s state.State) ([]byte, error) {
	data := map[string]interface{}{
		"event": s.Event(),
		"steps": s.Actions(),
		"ctx": map[string]interface{}{
			"workflow_id": s.WorkflowID(),
		},
	}
	return json.Marshal(data)
}
