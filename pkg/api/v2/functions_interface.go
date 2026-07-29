package apiv2

import (
	"context"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest"
)

type FunctionProvider interface {
	// GetFunction returns a function given its slug OR ID.
	GetFunction(ctx context.Context, identifier string) (inngest.DeployedFunction, error)
	// GetFunctionByApp returns a function given its app ID and user-defined function ID.
	GetFunctionByApp(ctx context.Context, appID string, functionID string) (inngest.DeployedFunction, error)
	// GetFunctions returns a stable page of functions within an app.
	GetFunctions(ctx context.Context, appID string, opts GetFunctionsOpts) (*GetFunctionsResult, error)
}

type FunctionConfigProvider interface {
	PlanConcurrencyLimit(ctx context.Context, fn inngest.DeployedFunction) int
}

type GetFunctionsOpts struct {
	Cursor uuid.UUID
	Limit  int
}

type GetFunctionsResult struct {
	Functions []inngest.DeployedFunction
	HasMore   bool
}
