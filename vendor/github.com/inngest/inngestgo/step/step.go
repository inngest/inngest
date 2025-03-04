package step

import (
	"context"

	"github.com/inngest/inngestgo/internal/sdkrequest"
)

type ControlHijack struct{}

type ctxKey string

const (
	targetStepIDKey = ctxKey("stepID")
	ParallelKey     = ctxKey("parallelKey")
	isWithinStepKey = ctxKey("in-step")
)

var (
	// ErrNotInFunction is called when a step tool is executed outside of an Inngest
	// function call context.
	//
	// If this is thrown, you're likely executing an Inngest function manually instead
	// of it being invoked by the scheduler.
	ErrNotInFunction = &errNotInFunction{}
)

type errNotInFunction struct{}

func (errNotInFunction) Error() string {
	return "step called without function context"
}

func getTargetStepID(ctx context.Context) *string {
	if v := ctx.Value(targetStepIDKey); v != nil {
		if c, ok := v.(string); ok {
			return &c
		}
	}
	return nil
}

func SetTargetStepID(ctx context.Context, id string) context.Context {
	if id == "" || id == "step" {
		return ctx
	}

	return context.WithValue(ctx, targetStepIDKey, id)
}

func isParallel(ctx context.Context) bool {
	if v := ctx.Value(ParallelKey); v != nil {
		if c, ok := v.(bool); ok {
			return c
		}
	}
	return false
}

func preflight(ctx context.Context) sdkrequest.InvocationManager {
	if ctx.Err() != nil {
		// Another tool has already ran and the context is closed.  Return
		// and do nothing.
		panic(ControlHijack{})
	}
	mgr, ok := sdkrequest.Manager(ctx)
	if !ok {
		panic(ErrNotInFunction)
	}
	return mgr
}

func IsWithinStep(ctx context.Context) bool {
	if v := ctx.Value(isWithinStepKey); v != nil {
		return true
	}
	return false
}

func setWithinStep(ctx context.Context) context.Context {
	return context.WithValue(ctx, isWithinStepKey, &struct{}{})
}
