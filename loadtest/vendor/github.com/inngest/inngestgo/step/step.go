package step

import (
	"context"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo/internal/sdkrequest"
)

type ctxKey string

const (
	targetStepIDKey = ctxKey("stepID")
	isWithinStepKey = ctxKey("in-step")
)

// ErrNotInFunction is called when a step tool is executed outside of an Inngest
// function call context.
//
// If this is thrown, you're likely executing an Inngest function manually instead
// of it being invoked by the scheduler.
var ErrNotInFunction = &errNotInFunction{}

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

func preflight(ctx context.Context, op enums.Opcode) sdkrequest.InvocationManager {
	if ctx.Err() != nil {
		// Another tool has already ran and the context is closed.  Return
		// and do nothing.
		panic(sdkrequest.ControlHijack{})
	}
	mgr, ok := sdkrequest.Manager(ctx)
	if !ok && enums.OpcodeIsAsync(op) {
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
