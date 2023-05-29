package coredata

import (
	"context"

	"github.com/inngest/inngest/pkg/inngest"
)

type ReadWriter interface {
	APIReadWriter
	ExecutionLoader
}

// ExecutionLoader is an interface which specifies all functions required to run
// workers and executors.
type ExecutionLoader interface {
	ExecutionFunctionLoader
	ExecutionActionLoader
}

type ExecutionFunctionLoader interface {
	// Functions returns all functions.
	Functions(ctx context.Context) ([]inngest.Function, error)
	// FunctionsScheduled returns all scheduled functions available.
	FunctionsScheduled(ctx context.Context) ([]inngest.Function, error)
	// FunctionsByTrigger returns functions for the given trigger by event name.
	FunctionsByTrigger(ctx context.Context, eventName string) ([]inngest.Function, error)
}

type ExecutionActionLoader interface {
	// Action returns fully-defined action given a DSN and optional version constraint.  If the
	// versions within the constraint are nil we must return the latest version available.
	Action(ctx context.Context, dsn string, version *inngest.VersionConstraint) (*inngest.ActionVersion, error)
}

type APIReadWriter interface {
	APIFunctionReader
	APIFunctionWriter
	APIActionReader
	APIActionWriter
}

type APIFunctionReader interface {
	// Functions returns all functions.
	Functions(ctx context.Context) ([]inngest.Function, error)
}
type APIFunctionWriter interface {
}
type APIActionReader interface {
}
type APIActionWriter interface {
}
