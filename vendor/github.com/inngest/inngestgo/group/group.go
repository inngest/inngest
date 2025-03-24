package group

import (
	"context"

	"github.com/inngest/inngestgo/step"
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
	ctx = context.WithValue(ctx, step.ParallelKey, true)

	results := Results{}
	isPlanned := false
	ch := make(chan struct{}, 1)
	var unexpectedPanic any
	for _, fn := range fns {
		fn := fn
		go func(fn func(ctx context.Context) (any, error)) {
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(step.ControlHijack); ok {
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
		panic(step.ControlHijack{})
	}

	return results
}
