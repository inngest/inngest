package sdkrequest

import (
	"context"

	"github.com/inngest/inngest/pkg/enums"
)

type ctxKey string

const (
	ParallelKey     = ctxKey("parallelKey")
	ParallelModeKey = ctxKey("parallelModeKey")
)

// IsParallel returns whether we're executing in the context of parallel steps.
//
// This will return true for code executing within group.Parallel.
func IsParallel(ctx context.Context) bool {
	if v := ctx.Value(ParallelKey); v != nil {
		if c, ok := v.(bool); ok {
			return c
		}
	}
	return false
}

// ParallelMode returns the type of parallelism, ie. whether discovery steps
// are enqueued or not.
func ParallelMode(ctx context.Context) enums.ParallelMode {
	if v := ctx.Value(ParallelModeKey); v != nil {
		if c, ok := v.(enums.ParallelMode); ok {
			return c
		}
	}
	return enums.ParallelModeNone
}
