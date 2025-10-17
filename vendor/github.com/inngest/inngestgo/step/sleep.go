package step

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo/internal"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	str2duration "github.com/xhit/go-str2duration/v2"
)

type SleepOpts struct {
	ID string
	// Name represents the optional step name.
	Name string
}

func Sleep(ctx context.Context, id string, duration time.Duration) {
	targetID := getTargetStepID(ctx)
	mgr := preflight(ctx, enums.OpcodeSleep)
	op := mgr.NewOp(enums.OpcodeSleep, id)
	if _, ok := mgr.Step(ctx, op); ok {
		// We've already slept.
		return
	}

	if targetID != nil && *targetID != op.MustHash() {
		// Don't report this step since targeting is happening and it isn't
		// targeted
		panic(sdkrequest.ControlHijack{})
	}

	mw := internal.MiddlewareFromContext(ctx)
	mw.BeforeExecution(ctx, mgr.CallContext())

	plannedOp := sdkrequest.GeneratorOpcode{
		ID:   op.MustHash(),
		Op:   enums.OpcodeSleep,
		Name: id,
		Opts: map[string]any{
			"duration": str2duration.String(duration),
		},
	}
	mgr.AppendOp(ctx, plannedOp)

	panic(sdkrequest.ControlHijack{})
}

// SleepUntil sleeps until a given time.  This halts function execution entirely,
// and Inngest will resume the function after the given time from this step.
func SleepUntil(ctx context.Context, id string, until time.Time) {
	duration := time.Until(until)
	Sleep(ctx, id, duration)
}
