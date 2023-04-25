package coredata

import (
	"context"
	"errors"

	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/inngest/client"
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
	Functions(ctx context.Context) ([]function.Function, error)
}
type APIFunctionWriter interface {
	// Create a new function
	CreateFunctionVersion(ctx context.Context, f function.Function, live bool, env string) (function.FunctionVersion, error)
}
type APIActionReader interface {
	// Find a given action by an exact version number
	ActionVersion(ctx context.Context, dsn string, version *inngest.VersionConstraint) (client.ActionVersion, error)
}
type APIActionWriter interface {
	// Create a a new action version
	CreateActionVersion(ctx context.Context, av inngest.ActionVersion) (client.ActionVersion, error)
	// Update an action version, e.g. when updating a Docker image has been successfully pushed to the registry
	UpdateActionVersion(ctx context.Context, dsn string, version inngest.VersionInfo, enabled bool) (client.ActionVersion, error)
}

var ErrActionVersionNotFound error = errors.New("action version not found")
