package coredata

import (
	"context"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/function"
)

// ExecutionLoader is an interface which specifies all functions required to run
// workers and executors.
type ExecutionLoader interface {
	ExecutionFunctionLoader
	ExecutionActionLoader
}

type ExecutionFunctionLoader interface {
	// Functions returns all functions.
	Functions(ctx context.Context) ([]function.Function, error)
	// FunctionsScheduled returns all scheduled functions available.
	FunctionsScheduled(ctx context.Context) ([]function.Function, error)
	// FunctionsByTrigger returns functions for the given trigger by event name.
	FunctionsByTrigger(ctx context.Context, eventName string) ([]function.Function, error)
}

type ExecutionActionLoader interface {
	// Action returns fully-defined action given a DSN and optional version constraint.  If the
	// versions within the constraint are nil we must return the latest version available.
	Action(ctx context.Context, dsn string, version *inngest.VersionConstraint) (*inngest.ActionVersion, error)
}
