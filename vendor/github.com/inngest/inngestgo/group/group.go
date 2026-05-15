package group

import (
	"context"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo/internal/sdkrequest"
)

type Result struct {
	Error error
	Value any
}

type Results []Result

// AnyError returns an error if any of the results have an error.
func (r Results) AnyError() error {
	for _, result := range r {
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

// Parallel runs steps in parallel. Its arguments are callbacks that include
// steps.
func Parallel(
	ctx context.Context,
	fns ...func(ctx context.Context) (any, error),
) Results {
	return ParallelWithOpts(ctx, ParallelOpts{}, fns...)
}

type ParallelOpts struct {
	// ParallelMode controls "discovery request" scheduling after a parallel
	// step ends. Defaults to ParallelModeWait.
	ParallelMode enums.ParallelMode
}

func ParallelWithOpts(
	ctx context.Context,
	opts ParallelOpts,
	fns ...func(ctx context.Context) (any, error),
) Results {
	ctx = context.WithValue(ctx, sdkrequest.ParallelKey, true)
	ctx = context.WithValue(ctx, sdkrequest.ParallelModeKey, opts.ParallelMode)

	results := Results{}
	isPlanned := false
	ch := make(chan struct{}, 1)
	var unexpectedPanic any

	for _, fn := range fns {
		fn := fn
		go func(fn func(ctx context.Context) (any, error)) {
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(sdkrequest.ControlHijack); ok {
						isPlanned = true
					} else {
						unexpectedPanic = r
					}
				}
				ch <- struct{}{}
			}()

			value, err := fn(ctx)
			results = append(results, Result{Error: err, Value: value})
		}(fn)
		<-ch
	}

	if unexpectedPanic != nil {
		// Repanic to let our normal panic recovery handle it
		panic(unexpectedPanic)
	}

	if isPlanned {
		panic(sdkrequest.ControlHijack{})
	}
	return results
}
