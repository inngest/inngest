package executor

import (
	"encoding/json"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing/meta"
)

type runInstance struct {
	md         sv2.Metadata
	f          inngest.Function
	events     []json.RawMessage
	item       queue.Item
	edge       inngest.Edge
	resp       *state.DriverResponse
	httpClient exechttp.RequestExecutor
	stackIndex int

	// If specified, this is the span reference that represents this execution:
	// the current request outgoing to the user's SDK.
	// This span will be updated once the SDK responds.
	execSpan *meta.SpanReference

	// If specified, this is the span reference for the parent discovery call.
	//
	// This is necessary to properly tie the parent span to queue items for eg.
	// step.sleep, which require a completion span in some other future thread.
	parentSpan *meta.SpanReference
}

// RunContext interface implementation for runInstance
func (r *runInstance) Metadata() *sv2.Metadata {
	return &r.md
}

func (r *runInstance) Events() []json.RawMessage {
	return r.events
}

func (r *runInstance) HTTPClient() exechttp.RequestExecutor {
	return r.httpClient
}

func (r *runInstance) GroupID() string {
	return r.item.GroupID
}

func (r *runInstance) AttemptCount() int {
	return r.item.Attempt
}

func (r *runInstance) MaxAttempts() *int {
	max := r.item.GetMaxAttempts()
	return &max
}

func (r *runInstance) ShouldRetry() bool {
	if r.resp.NoRetry {
		return false
	}
	if r.resp.UserError != nil && r.resp.UserError.NoRetry {
		return false
	}
	return queue.ShouldRetry(nil, r.item.Attempt, r.item.GetMaxAttempts())
}

func (r *runInstance) IncrementAttempt() {
	r.item.Attempt++
}

func (r *runInstance) PriorityFactor() *int64 {
	return r.item.PriorityFactor
}

func (r *runInstance) ConcurrencyKeys() []state.CustomConcurrency {
	return r.item.CustomConcurrencyKeys
}

func (r *runInstance) ParallelMode() enums.ParallelMode {
	return r.item.ParallelMode
}

func (r *runInstance) LifecycleItem() queue.Item {
	return r.item
}

func (r *runInstance) SetStatusCode(code int) {
	r.resp.StatusCode = code
}

func (r *runInstance) UpdateOpcodeError(op *state.GeneratorOpcode, err state.UserError) {
	r.resp.UpdateOpcodeError(op, err)
}

func (r *runInstance) UpdateOpcodeOutput(op *state.GeneratorOpcode, output json.RawMessage) {
	r.resp.UpdateOpcodeOutput(op, output)
}

func (r *runInstance) SetError(err error) {
	r.resp.SetError(err)
}

func (r *runInstance) ExecutionSpan() *meta.SpanReference {
	return r.execSpan
}

func (r *runInstance) ParentSpan() *meta.SpanReference {
	return r.parentSpan
}
