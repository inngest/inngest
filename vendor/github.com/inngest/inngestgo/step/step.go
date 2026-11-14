package step

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	sdkerrors "github.com/inngest/inngestgo/errors"
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
	if ok {
		rejectInLoop(ctx, mgr, op)
	}
	return mgr
}

// loopRejected maps the step tools a loop handler cannot call to the name
// in the error text.
var loopRejected = map[enums.Opcode]string{
	enums.OpcodeSleep:         "step.Sleep",
	enums.OpcodeWaitForEvent:  "step.WaitForEvent",
	enums.OpcodeWaitForSignal: "step.WaitForSignal",
}

// rejectInLoop fails the run without retry when a loop handler calls a
// step tool that waits.  a loop waits with loop.Wake or loop.Subscribe.
func rejectInLoop(ctx context.Context, mgr sdkrequest.InvocationManager, op enums.Opcode) {
	name, ok := loopRejected[op]
	if !ok || !mgr.IsLoop() {
		return
	}
	mgr.SetErr(sdkerrors.NoRetryError(fmt.Errorf("%s is not allowed in a loop handler.  return loop.Wake or loop.Subscribe instead", name)))
	panic(sdkrequest.ControlHijack{})
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
