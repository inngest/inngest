package conditional

import (
	"context"

	"github.com/inngest/inngest/pkg/logger"
)

// Logger returns the logger from context, respecting conditional scope if set.
// This is a convenience wrapper that's equivalent to logger.From(ctx).
//
// For conditional logging, use the context-based approach:
//
//	logger.From(conditional.WithScope(ctx, "scope")).Info("message")
//
// Or the longer form:
//
//	ctx = conditional.WithScope(ctx, "scope")
//	logger.From(ctx).Info("message")
//
// The logger.From() function automatically checks for conditional scope in the
// context and returns a VoidLogger if the scope is disabled.
func Logger(ctx context.Context) logger.Logger {
	return logger.From(ctx)
}

