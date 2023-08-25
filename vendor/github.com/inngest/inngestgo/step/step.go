package step

import (
	"context"

	"github.com/inngest/inngestgo/internal/sdkrequest"
)

type ControlHijack struct{}

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
