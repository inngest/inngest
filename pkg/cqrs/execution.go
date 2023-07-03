package cqrs

import (
	"context"

	"github.com/inngest/inngest/pkg/inngest"
)

// ExecutionLoader is an interface which specifies all functions required to run
// workers and executors.
type ExecutionLoader interface {
	// Functions returns all functions.
	Functions(ctx context.Context) ([]inngest.Function, error)
	// FunctionsScheduled returns all scheduled functions available.
	FunctionsScheduled(ctx context.Context) ([]inngest.Function, error)
	// FunctionsByTrigger returns functions for the given trigger by event name.
	FunctionsByTrigger(ctx context.Context, eventName string) ([]inngest.Function, error)
}
