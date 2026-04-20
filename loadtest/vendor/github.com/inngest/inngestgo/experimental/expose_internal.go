package experimental

import "github.com/inngest/inngestgo/internal/middleware"

// BaseMiddleware is a noop implementation that you can embed within custom
// middleware to implement the middleware.Middleware interface, making every
// unimplemented call in your struct a no-op.
type BaseMiddleware = middleware.BaseMiddleware

// Middleware is the middleware interface that each implementation must fulfil.
// To avoid implementing each method as a noop, embed the BaseMiddleware struct
// in your struct implementation.
type Middleware = middleware.Middleware

// TransformableInput is passed to the TransformInput middleware method as a
// pointer, allowing events and input data to be modified before function execution
// begins.  This allows you to eg. implement data offloading to an external service.
type TransformableInput = middleware.TransformableInput

// TransformableOutput is passed to the TransformOutput middleware method as a
// pointer, allowing step and function return values to be modified after execution.
// This allows you to eg. implement data offloading to an external service.
type TransformableOutput = middleware.TransformableOutput

// LoggerFromContext returns the stdlib logger from the context.
// Returns an error if no logger is present.
var LoggerFromContext = middleware.LoggerFromContext

// CallContext is used in middleware, and represents information around the current
// execution request.
type CallContext = middleware.CallContext
