package util

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

type critctx struct {
	boundary time.Duration
	maxDur   time.Duration
}

type CritOpt func(c *critctx)

func WithBoundaries(b time.Duration) CritOpt {
	return func(c *critctx) {
		c.boundary = b
	}
}

func WithMaxDuration(dur time.Duration) CritOpt {
	return func(c *critctx) {
		c.maxDur = dur
	}
}

func Crit(ctx context.Context, name string, f func(ctx context.Context) error, opts ...CritOpt) error {
	_, err := CritT(ctx, name, func(ctx context.Context) (any, error) { return nil, f(ctx) }, opts...)
	return err
}

// Crit is a util to wrap a lambda with a non-cancellable context.  It allows an optional time boundary
// for checking context deadlines;  if the parent ctx has a deadline shorter than the boundary we exit
// immediately with an error.
func CritT[T any](ctx context.Context, name string, f func(ctx context.Context) (T, error), opts ...CritOpt) (resp T, err error) {
	cr := critctx{}

	for _, apply := range opts {
		apply(&cr)
	}

	pre := time.Now()

	// If withinBounds is set, we have some time period in which we must complete the Crit
	// section.
	//
	// Check the parent context to see if there's a deadline, and if the deadline < withinBounds
	// don't even bother.  The crit section must exist within some retryable process.
	if cr.boundary > 0 {
		if dl, ok := ctx.Deadline(); ok && time.Until(dl) < cr.boundary {
			return resp, fmt.Errorf("context deadline shorter than critical bounds: %s", name)
		}

		// XXX: Instrument critical section durations and error responses via the names.
		defer func() {
			actual := time.Since(pre)
			if actual > cr.boundary {
				// This took longer than the predefined boundary, so log a fat warning.
				logger.StdlibLogger(ctx).Warn("critical section took longer than boundary", "name", name, "duration_ms", actual.Milliseconds())
			}
		}()
	}

	if ctx.Err() == context.Canceled {
		logger.StdlibLogger(ctx).Warn("context canceled before entering crit", "name", name)
	}

	return f(context.WithoutCancel(ctx))
}
