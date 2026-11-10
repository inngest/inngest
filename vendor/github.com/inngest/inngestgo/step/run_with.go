package step

import "context"

// RunWith runs fn with arg as a durable step, the same as Run.  on replay,
// RunWith returns the stored result for id and does not use arg.  the
// expressions for fn and arg run on every request, outside the step.
//
// for example, this call:
//
//	b, err := step.RunWith(ctx, "load build", store.Get, id)
//
// is a shorthand for this call to Run:
//
//	b, err := step.Run(ctx, "load build", func(ctx context.Context) (Build, error) {
//		return store.Get(ctx, id)
//	})
func RunWith[A, R any](ctx context.Context, id string, fn func(context.Context, A) (R, error), arg A) (R, error) {
	return Run(ctx, id, func(ctx context.Context) (R, error) {
		return fn(ctx, arg)
	})
}
