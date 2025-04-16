package util

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

func Crit(ctx context.Context, name string, f func(ctx context.Context) error, withinBounds ...time.Duration) error {
	_, err := CritT(ctx, name, func(ctx context.Context) (any, error) { return nil, f(ctx) }, withinBounds...)
	return err
}

// Crit is a util to wrap a lambda with a non-cancellable context.  It allows an optional time boundary
// for checking context deadlines;  if the parent ctx has a deadline shorter than the boundary we exit
// immediately with an error.
func CritT[T any](ctx context.Context, name string, f func(ctx context.Context) (T, error), withinBounds ...time.Duration) (resp T, err error) {
	// If withinBounds is set, we have some time period in which we must complete the Crit
	// section.
	//
	// Check the parent context to see if there's a deadline, and if the deadline < withinBounds
	// don't even bother.  The crit section must exist within some retryable process.
	if len(withinBounds) == 1 {
		if dl, ok := ctx.Deadline(); ok && time.Until(dl) < withinBounds[0] {
			return resp, fmt.Errorf("context deadline shorter than critical bounds: %s", name)
		}
	}

	if ctx.Err() == context.Canceled {
		return resp, fmt.Errorf("context canceled before entering crit: %s", name)
	}

	pre := time.Now()
	resp, err = f(context.WithoutCancel(ctx))

	// XXX: Instrument critical section durations and error responses via the names.

	if len(withinBounds) == 1 {
		actual := time.Since(pre)
		if actual > withinBounds[0] {
			// This took longer than the predefined boundaries, so log a fat warning.
			logger.StdlibLogger(ctx).Warn("critical section took longer than boundaries", "name", name, "duration_ms", actual.Milliseconds())
		}
	}

	return resp, err
}
