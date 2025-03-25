package middleware

import (
	"context"

	"github.com/inngest/inngestgo/internal/event"
	"github.com/inngest/inngestgo/internal/fn"
)

// CallContext is used in middleware, and represents information around the current
// execution request.
type CallContext struct {
	FunctionOpts fn.FunctionOpts
	// Env represents the environment that this run is executing within.
	Env string
	// RunID is the ULID run ID, as a string, for this run.
	RunID string
	// Attempt is the 0-indexed attempt for this request.
	Attempt int
}

// Middleware is the middleware interface that each implementation must fulfil.
// To avoid implementing each method as a noop, embed the BaseMiddleware struct
// in your struct implementation.
//
// The order of middleware execution is as follows:
//
//   - TransformInput
//   - BeforeExecution
//   - AfterExecution
//   - TransformOutput
//
// Note that if your handlers have many middleware, TransformInput and BeforeExecution
// are executed in the order that middleware is added.  AfterExecution and TransformOutput
// are executed in reverse order.
type Middleware interface {
	// TransformInput is called before entering the Inngest function. It gives
	// an opportunity to modify the input before it is sent to the function.
	TransformInput(
		ctx context.Context,
		call CallContext,
		input *TransformableInput,
	)

	// BeforeExecution is called before executing "new code".
	BeforeExecution(ctx context.Context, call CallContext)

	// AfterExecution is called after executing "new code".  It is called with
	// the result of any step (or the return value of the functon), plus any
	// error from the step or function.
	//
	// If the step did not error, err will be nil.
	AfterExecution(ctx context.Context, call CallContext, result any, err error)

	// TransformOutput is called after a step finishes execution or a function
	// returns results.  It gives an opportunity to modify the output before it
	// is stored in function state or logs.
	TransformOutput(
		ctx context.Context,
		call CallContext,
		output *TransformableOutput,
	)

	// OnPanic is called if the function panics, with the recovered value and stack.
	OnPanic(ctx context.Context, call CallContext, recovered any, stack string)
}

// ensure that the MiddlewareManager implements Middleware at compile time.
var _ Middleware = &BaseMiddleware{}

// BaseMiddleware is a noop implementation that you can embed within custom
// middelware to implement the middleware.Middleware interface, making every
// unimplemented call in your struct a no-op.
type BaseMiddleware struct{}

func (m *BaseMiddleware) BeforeExecution(ctx context.Context, call CallContext) {
	// Noop.
}

func (m *BaseMiddleware) AfterExecution(ctx context.Context, call CallContext, result any, err error) {
	// Noop.
}

func (m *BaseMiddleware) TransformOutput(
	ctx context.Context,
	call CallContext,
	output *TransformableOutput,
) {
	//Noop
}

func (m *BaseMiddleware) OnPanic(ctx context.Context, call CallContext, recovered any, stack string) {
	// Noop.
}

func (m *BaseMiddleware) TransformInput(
	ctx context.Context,
	call CallContext,
	input *TransformableInput,
) {
	// Noop.
}

// TransformableOutput is passed to the TransformOutput middleware method as a
// pointer, allowing step and function return values to be modified after execution.
// This allows you to eg. implement data offloading to an external service.
type TransformableOutput struct {
	Result any
	Error  error
}

// TransformableInput is passed to the TransformInput middleware method as a
// pointer, allowing events and input data to be modified before function execution
// begins.  This allows you to eg. implement data offloading to an external service.
type TransformableInput struct {
	Event  *event.Event
	Events []*event.Event
	Steps  map[string]string

	context context.Context
}

// Context returns the context.
func (t *TransformableInput) Context() context.Context {
	return t.context
}

// WithContext sets the context.
func (t *TransformableInput) WithContext(ctx context.Context) {
	t.context = ctx
}
