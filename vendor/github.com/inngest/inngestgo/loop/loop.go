// Package loop holds the calls a loop handler makes to say what happens
// after the run.  a handler ends with return loop.Wake(ctx, id, d),
// return loop.End(ctx), or return nil.  none of the calls hijack control.
// the calls are collected during the run and sent once, in the LoopReturn
// opcode, when the handler returns nil.  the last call per id wins and End
// wins over Wake.
package loop

import (
	"context"
	"fmt"
	"time"

	sdkerrors "github.com/inngest/inngestgo/errors"
	"github.com/inngest/inngestgo/internal/fn"
	"github.com/inngest/inngestgo/internal/sdkrequest"
	"github.com/inngest/inngestgo/step"
)

// ErrNotInFunction is the panic value when a loop call runs outside of an
// Inngest function call context.
var ErrNotInFunction = step.ErrNotInFunction

// SubscribeOpts configures a subscription.  Event is the event name.  If
// is a CEL expression over the event.  TTL is how long the subscription
// lives.  a zero TTL has no limit.  a negative TTL fails the run.
type SubscribeOpts struct {
	Event string
	If    string
	TTL   time.Duration
}

// Wake runs the instance again after the given duration.  the run that
// follows has Loop.Kind set to wake and Loop.ID set to id.
func Wake(ctx context.Context, id string, after time.Duration) error {
	mgr := manager(ctx, "loop.Wake")
	if id == "" {
		fail(mgr, fmt.Errorf("loop.Wake requires a non-empty id"))
	}
	if after <= 0 {
		fail(mgr, fmt.Errorf("loop.Wake requires a duration above zero"))
	}
	mgr.Loop().Wake(id, after)
	return nil
}

// End finishes the instance after this run.  the state is still committed.
func End(ctx context.Context) error {
	mgr := manager(ctx, "loop.End")
	mgr.Loop().End()
	return nil
}

// Subscribe runs the instance again when an event matches opts.  the run
// that follows has Loop.Kind set to subscription and Loop.ID set to id.
// an fsm cannot subscribe.
func Subscribe(ctx context.Context, id string, opts SubscribeOpts) error {
	mgr := manager(ctx, "loop.Subscribe")
	if mgr.LoopKind() == fn.LoopKindFSM {
		fail(mgr, fmt.Errorf("loop.Subscribe called in an fsm.  an fsm cannot subscribe to events"))
	}
	if id == "" {
		fail(mgr, fmt.Errorf("loop.Subscribe requires a non-empty id"))
	}
	if opts.Event == "" {
		fail(mgr, fmt.Errorf("loop.Subscribe requires a non-empty event"))
	}
	if opts.TTL < 0 {
		fail(mgr, fmt.Errorf("loop.Subscribe requires a ttl of zero or above"))
	}
	mgr.Loop().Subscribe(id, opts.Event, opts.If, opts.TTL)
	return nil
}

// Unsubscribe removes the subscription with the given id.
func Unsubscribe(ctx context.Context, id string) error {
	mgr := manager(ctx, "loop.Unsubscribe")
	if id == "" {
		fail(mgr, fmt.Errorf("loop.Unsubscribe requires a non-empty id"))
	}
	mgr.Loop().Unsubscribe(id)
	return nil
}

// manager returns the invocation manager for a loop run.  it panics with
// ErrNotInFunction outside of a function and fails the run without retry
// when the function is not a loop or the call is inside a step callback.
func manager(ctx context.Context, name string) sdkrequest.InvocationManager {
	mgr, ok := sdkrequest.Manager(ctx)
	if !ok {
		panic(ErrNotInFunction)
	}
	if !mgr.IsLoop() {
		fail(mgr, fmt.Errorf("%s called outside a loop function", name))
	}
	if step.IsWithinStep(ctx) {
		fail(mgr, fmt.Errorf("%s called inside a step.  call it from the handler body", name))
	}
	return mgr
}

// fail ends the run with the error and no retry.
func fail(mgr sdkrequest.InvocationManager, err error) {
	mgr.SetErr(sdkerrors.NoRetryError(err))
	panic(sdkrequest.ControlHijack{})
}
