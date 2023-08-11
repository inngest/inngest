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

/*
type Promise[T any] interface {
	// Await returns the value within the promise.  If the step hasn't ran,
	// this executes the step durably and reliably.
	Await() T

	// Name returns the name of the step that this promise refers to.
	Name() string
}

type future[T any] struct {
	f func() (T, error)

	name string
	value T
	err   error
}

// TODO: Await() (T, error)
func (f future[T]) Await() T {
	// TOOD: is invoked?
	return f.value
}

func All(f ...any) {
	// Assert that All is of type future.
}

func Race(f ...any) Promise[any] {
	return nil
}

func foo() {
	a := future[string]{
		f: func() (string, error) {
			return "hi", nil
		},
	}
	b := future[string]{
		f: func() (string, error) {
			return "wut", nil
		},
	}
	c := future[int]{
		f: func() (int, error) {
			return 1, nil
		},
	}

	All(a, b, c)
	a.Await()

	win := Race(a, b, c)
	win.Await()
}
*/
