package checkpoint

import (
	"encoding/json"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
)

// checkpointRunContext implements execution.RunContext for use in checkpoint API calls
type checkpointRunContext struct {
	md         state.Metadata
	httpClient exechttp.RequestExecutor
	events     []json.RawMessage

	// Data from queue.Item that we actually need
	groupID         string
	attemptCount    int
	maxAttempts     int
	priorityFactor  *int64
	concurrencyKeys []state.CustomConcurrency
	parallelMode    enums.ParallelMode
}

func (c *checkpointRunContext) Metadata() *state.Metadata {
	return &c.md
}

func (c *checkpointRunContext) Events() []json.RawMessage {
	return c.events
}

func (c *checkpointRunContext) HTTPClient() exechttp.RequestExecutor {
	return c.httpClient
}

func (c *checkpointRunContext) GroupID() string {
	return c.groupID
}

func (c *checkpointRunContext) AttemptCount() int {
	return c.attemptCount
}

func (c *checkpointRunContext) MaxAttempts() *int {
	return &c.maxAttempts
}

func (c *checkpointRunContext) ShouldRetry() bool {
	return c.attemptCount < (c.maxAttempts - 1)
}

func (c *checkpointRunContext) IncrementAttempt() {
	c.attemptCount++
}

func (c *checkpointRunContext) PriorityFactor() *int64 {
	return c.priorityFactor
}

func (c *checkpointRunContext) ConcurrencyKeys() []state.CustomConcurrency {
	return c.concurrencyKeys
}

func (c *checkpointRunContext) ParallelMode() enums.ParallelMode {
	return c.parallelMode
}

func (c *checkpointRunContext) LifecycleItem() queue.Item {
	// For checkpoint context, we create a minimal queue.Item for lifecycle events
	// This is the one place we still need to construct a queue.Item, but it's much simpler
	return queue.Item{
		Identifier: sv1.Identifier{
			WorkspaceID: c.md.ID.Tenant.EnvID,
			AppID:       c.md.ID.Tenant.AppID,
			WorkflowID:  c.md.ID.FunctionID,
			RunID:       c.md.ID.RunID,
		},
		WorkspaceID:           c.md.ID.Tenant.EnvID,
		GroupID:               c.groupID,
		Attempt:               c.attemptCount,
		PriorityFactor:        c.priorityFactor,
		CustomConcurrencyKeys: c.concurrencyKeys,
		ParallelMode:          c.parallelMode,
		Payload:               queue.PayloadEdge{
			// intentionally blank.
		},
	}
}

func (c *checkpointRunContext) SetStatusCode(code int) {
	// this is a noop.
}

func (c *checkpointRunContext) UpdateOpcodeError(op *state.GeneratorOpcode, err state.UserError) {
	// this is a noop.
}

func (c *checkpointRunContext) UpdateOpcodeOutput(op *state.GeneratorOpcode, output json.RawMessage) {
	// this is a noop.
}

func (c *checkpointRunContext) SetError(err error) {
	// this is a noop.
}

func (c *checkpointRunContext) ExecutionSpan() *meta.SpanReference {
	// this is currently a noop.  we may need to implement
	// this in the future.
	return nil
}

func (c *checkpointRunContext) ParentSpan() *meta.SpanReference {
	return tracing.RunSpanRefFromMetadata(&c.md)
}
