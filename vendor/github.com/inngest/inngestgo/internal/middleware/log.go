package middleware

import (
	"context"
	"log/slog"
	"sync/atomic"
)

type logContextKeyType struct{}

var logContextKey = logContextKeyType{}

// LoggerFromContext returns the logger from the context.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	value := ctx.Value(logContextKey)
	if value == nil {
		// Unreachable if the middleware is used correctly.
		return slog.Default()
	}

	l, ok := value.(*slog.Logger)
	if !ok {
		// Unreachable if the middleware is used correctly.
		return slog.Default()
	}

	return l
}

// withLogger adds the logger to the context.
func withLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, logContextKey, l)
}

type logMiddlewareRequest struct {
	BaseMiddleware

	enableLogger func()
	logger       *slog.Logger
}

func (l *logMiddlewareRequest) BeforeExecution(ctx context.Context, call CallContext) {
	// We're encountering "new code", so enable the logger.
	l.enableLogger()
}

func (l *logMiddlewareRequest) TransformInput(
	ctx context.Context,
	call CallContext,
	input *TransformableInput,
) {
	// Add the logger to the context so that it can be used by the
	// Inngest function.
	input.WithContext(withLogger(input.Context(), l.logger))
}

// LogMiddleware returns a middleware that adds an idempotent logger to the
// context.
func LogMiddleware(l *slog.Logger) func() Middleware {
	return func() Middleware {
		// IMPORTANT: Wrapper must be created within the callback to ensure its
		// lifetime is a single execution request.
		wrapper := &handlerWrapper{
			enabled:     &atomic.Bool{},
			userHandler: l.Handler(),
		}

		logger := slog.New(wrapper)
		return &logMiddlewareRequest{
			enableLogger: func() {
				// We're encountering "new code", so enable the logger.
				wrapper.enable()
			},
			logger: logger,
		}
	}
}

// handlerWrapper is a wrapper around a slog.Handler that allows it to be
// enabled or disabled.
type handlerWrapper struct {
	enabled     *atomic.Bool
	userHandler slog.Handler
}

func (t *handlerWrapper) Enabled(ctx context.Context, level slog.Level) bool {
	if !t.enabled.Load() {
		// Logging is disabled.
		return false
	}

	// Pass through to the user handler.
	return t.userHandler.Enabled(ctx, level)
}

func (t *handlerWrapper) Handle(ctx context.Context, r slog.Record) error {
	// Pass through to the user handler.
	return t.userHandler.Handle(ctx, r)
}

func (t *handlerWrapper) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Pass through to the user handler.
	return t.userHandler.WithAttrs(attrs)
}

func (t *handlerWrapper) WithGroup(name string) slog.Handler {
	// Pass through to the user handler.
	return t.userHandler.WithGroup(name)
}

// enable enables logging to the user handler.
func (t *handlerWrapper) enable() {
	t.enabled.Store(true)
}
