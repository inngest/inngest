package resolvers

import (
	"context"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/function"
)

// Deploy a function creating a new function version
func (r *mutationResolver) DeployFunction(ctx context.Context, input models.DeployFunctionInput) (*function.FunctionVersion, error) {
	// Parse function CUE or JSON string - This also validates the function
	f, err := function.Unmarshal(ctx, []byte(input.Config), "")
	if err != nil {
		return nil, err
	}

	// TODO - Move default environment to config
	env := "prod"
	if input.Env != nil {
		env = input.Env.String()
	}
	fv, err := r.APIReadWriter.CreateFunctionVersion(ctx, *f, *input.Live, env)
	if err != nil {
		return nil, err
	}

	config, err := function.MarshalCUE(fv.Function)
	if err != nil {
		return nil, err
	}

	fv.Config = string(config)
	return &fv, nil
}
