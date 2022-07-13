package resolvers

import (
	"context"

	"github.com/inngest/inngest-cli/pkg/coreapi/graph/models"
	"github.com/inngest/inngest-cli/pkg/function"
)

// Deploy a function creating a new function version
func (r *mutationResolver) DeployFunction(ctx context.Context, input models.DeployFunctionInput) (*function.FunctionVersion, error) {
	// Parse function CUE or JSON string
	f, err := function.Unmarshal(ctx, []byte(input.Config), "")
	if err != nil {
		return nil, err
	}

	fv, err := r.APILoader.CreateFunctionVersion(ctx, *f, *input.Live, input.Env.String())
	if err != nil {
		return nil, err
	}

	// TODO convert function to cue config
	config, err := function.MarshalJSON(fv.Function)
	if err != nil {
		return nil, err
	}

	fv.Config = string(config)
	return &fv, nil
}
