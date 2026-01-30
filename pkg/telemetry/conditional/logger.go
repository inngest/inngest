package conditional

import (
	"context"

	"github.com/inngest/inngest/pkg/logger"
)

// Logger returns a logger from context. If a scope is provided,
// the logger respects conditional observability settings for that scope.
// Returns VoidLogger if the scope is disabled.
//
// Usage with scope (recommended for conditional logging):
//
//	conditional.Logger(ctx, "queue.CapacityLease").Debug("extended capacity lease")
//
// Usage without scope (equivalent to logger.From(ctx)):
//
//	conditional.Logger(ctx).Info("message")
func Logger(ctx context.Context, scope ...string) logger.Logger {
	if len(scope) > 0 {
		return logger.From(WithScope(ctx, scope[0]))
	}
	return logger.From(ctx)
}

