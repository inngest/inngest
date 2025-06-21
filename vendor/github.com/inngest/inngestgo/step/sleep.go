package step

import (
	"context"
	"fmt"
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
	mgr := preflight(ctx)
	op := mgr.NewOp(enums.OpcodeSleep, id, nil)
	if _, ok := mgr.Step(ctx, op); ok {
		// We've already slept.
		return
	}

	mw, ok := internal.MiddlewareManagerFromContext(ctx)
	if !ok {
		mgr.SetErr(fmt.Errorf("no middleware manager found in context"))
		panic(ControlHijack{})
	}
	mw.BeforeExecution(ctx, mgr.MiddlewareCallCtx())
	mgr.AppendOp(sdkrequest.GeneratorOpcode{
		ID:   op.MustHash(),
		Op:   enums.OpcodeSleep,
		Name: id,
		Opts: map[string]any{
			"duration": str2duration.String(duration),
		},
	})

	panic(ControlHijack{})
}

// SleepUntil sleeps until a given time.  This halts function execution entirely,
// and Inngest will resume the function after the given time from this step.
func SleepUntil(ctx context.Context, id string, until time.Time) {
	duration := time.Until(until)
	Sleep(ctx, id, duration)
}
