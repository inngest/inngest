package group

import (
	"context"

	"github.com/inngest/inngestgo/step"
)

type Result struct {
	Error error
	Value any
}

func Parallel(
	ctx context.Context,
	fns ...func(ctx context.Context,
	) (any, error)) []Result {
	ctx = context.WithValue(ctx, step.ParallelKey, true)

	results := []Result{}
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
