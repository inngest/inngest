package resolvers

import (
	"context"

	"github.com/inngest/inngest-cli/pkg/coreapi/graph/models"
	"github.com/inngest/inngest-cli/pkg/function"
)

func (r *mutationResolver) DeployFunction(ctx context.Context, input models.DeployFunctionInput) (*models.FunctionVersion, error) {
	// Parse function CUE or JSON string
	f, err := function.Unmarshal(ctx, []byte(input.Config), "")
	if err != nil {
		return nil, err
	}

	fv, err := r.APILoader.CreateFunctionVersion(ctx, *f, *input.Live)
	if err != nil {
		return nil, err
	}

	// TODO convert function to cue config
	config, err := function.MarshalJSON(fv.Function)
	if err != nil {
		return nil, err
	}

	return &models.FunctionVersion{
		FunctionID: fv.FunctionID,
		Version:    int(fv.Version),
		Config:     string(config),
		ValidFrom:  &fv.ValidFrom,
		ValidTo:    &fv.ValidTo,
		CreatedAt:  fv.CreatedAt,
		UpdatedAt:  fv.UpdatedAt,
	}, nil
}
