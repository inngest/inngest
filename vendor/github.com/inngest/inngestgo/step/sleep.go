package step

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	str2duration "github.com/xhit/go-str2duration/v2"
)

func Sleep(ctx context.Context, duration time.Duration) {
	mgr := preflight(ctx)
	name := str2duration.String(duration)
	op := mgr.NewOp(enums.OpcodeSleep, name, nil)
	if _, ok := mgr.Step(op); ok {
		// We've already slept.
		return
	}
	mgr.AppendOp(state.GeneratorOpcode{
		ID:   op.MustHash(),
		Op:   enums.OpcodeSleep,
		Name: name,
	})
	panic(ControlHijack{})
}
