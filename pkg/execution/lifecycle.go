package execution

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
)

// LifecycleListener listens to lifecycle events on the executor.
type LifecycleListener interface {
	// Close closes the listener and flushes any pending writes.
	Close() error

	OnStepStarted(
		context.Context,
		state.Identifier,
		queue.Item,
		inngest.Edge,
		inngest.Step,
		state.State,
	)

	// OnStepFinished is called when a step finishes.  This may be
	// a success, a temporary error, or a failure.
	OnStepFinished(
		context.Context,
		state.Identifier,
		queue.Item,
		inngest.Edge,
		inngest.Step,
		state.DriverResponse,
	)

	OnWaitForEvent(
		context.Context,
		state.Identifier,
		queue.Item,
		state.GeneratorOpcode,
	)

	// OnStepFailed(
	// 	context.Context,
	// 	state.Identifier,
	// 	queue.Item,
	// 	error,
	// )
}

type NoopLifecyceListener struct{}

func (NoopLifecyceListener) OnStepStarted(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	edge inngest.Edge,
	step inngest.Step,
	state state.State,
) {
}

func (NoopLifecyceListener) OnStepFinished(
	context.Context,
	state.Identifier,
	queue.Item,
	inngest.Step,
	*state.DriverResponse,
	error,
) {
}

func (NoopLifecyceListener) OnWaitForEvent(
	context.Context,
	state.Identifier,
	queue.Item,
	state.GeneratorOpcode,
) {
}
