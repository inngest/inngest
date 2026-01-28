package conditional

import (
	"context"

	"github.com/inngest/inngest/pkg/logger"
)

// ConditionalDebug logs a debug message if logging is enabled for the given scope.
// Uses the logger from the context.
func ConditionalDebug(ctx context.Context, scope string, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, scope) {
		return
	}
	logger.From(ctx).DebugContext(ctx, msg, args...)
}

// ConditionalInfo logs an info message if logging is enabled for the given scope.
// Uses the logger from the context.
func ConditionalInfo(ctx context.Context, scope string, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, scope) {
		return
	}
	logger.From(ctx).InfoContext(ctx, msg, args...)
}

// ConditionalWarn logs a warning message if logging is enabled for the given scope.
// Uses the logger from the context.
func ConditionalWarn(ctx context.Context, scope string, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, scope) {
		return
	}
	logger.From(ctx).WarnContext(ctx, msg, args...)
}

// ConditionalError logs an error message if logging is enabled for the given scope.
// Uses the logger from the context.
func ConditionalError(ctx context.Context, scope string, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, scope) {
		return
	}
	logger.From(ctx).ErrorContext(ctx, msg, args...)
}

// ConditionalTrace logs a trace message if logging is enabled for the given scope.
// Uses the logger from the context.
func ConditionalTrace(ctx context.Context, scope string, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, scope) {
		return
	}
	logger.From(ctx).TraceContext(ctx, msg, args...)
}

// ConditionalLogger is a scoped logger that conditionally logs based on feature flags.
type ConditionalLogger struct {
	logger logger.Logger
	scope  string
}

// NewConditionalLogger creates a new ConditionalLogger with the given logger and scope.
func NewConditionalLogger(l logger.Logger, scope string) *ConditionalLogger {
	return &ConditionalLogger{
		logger: l,
		scope:  scope,
	}
}

// NewConditionalLoggerFromContext creates a new ConditionalLogger using the logger from
// the context and the given scope.
func NewConditionalLoggerFromContext(ctx context.Context, scope string) *ConditionalLogger {
	return NewConditionalLogger(logger.From(ctx), scope)
}

// Scope returns the scope of the conditional logger.
func (l *ConditionalLogger) Scope() string {
	return l.scope
}

// Debug logs a debug message if logging is enabled for this logger's scope.
func (l *ConditionalLogger) Debug(ctx context.Context, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, l.scope) {
		return
	}
	l.logger.DebugContext(ctx, msg, args...)
}

// Info logs an info message if logging is enabled for this logger's scope.
func (l *ConditionalLogger) Info(ctx context.Context, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, l.scope) {
		return
	}
	l.logger.InfoContext(ctx, msg, args...)
}

// Warn logs a warning message if logging is enabled for this logger's scope.
func (l *ConditionalLogger) Warn(ctx context.Context, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, l.scope) {
		return
	}
	l.logger.WarnContext(ctx, msg, args...)
}

// Error logs an error message if logging is enabled for this logger's scope.
func (l *ConditionalLogger) Error(ctx context.Context, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, l.scope) {
		return
	}
	l.logger.ErrorContext(ctx, msg, args...)
}

// Trace logs a trace message if logging is enabled for this logger's scope.
func (l *ConditionalLogger) Trace(ctx context.Context, msg string, args ...any) {
	if !IsLoggingEnabled(ctx, l.scope) {
		return
	}
	l.logger.TraceContext(ctx, msg, args...)
}

// With returns a new ConditionalLogger with additional attributes.
func (l *ConditionalLogger) With(args ...any) *ConditionalLogger {
	return &ConditionalLogger{
		logger: l.logger.With(args...),
		scope:  l.scope,
	}
}

// WithScope returns a new ConditionalLogger with a different scope.
func (l *ConditionalLogger) WithScope(scope string) *ConditionalLogger {
	return &ConditionalLogger{
		logger: l.logger,
		scope:  scope,
	}
}

// Logger returns the underlying logger.
func (l *ConditionalLogger) Logger() logger.Logger {
	return l.logger
}
