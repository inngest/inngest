package resolvers

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"context"

	"github.com/inngest/inngest-cli/pkg/coreapi/generated"
	model "github.com/inngest/inngest-cli/pkg/coreapi/graph/models"
)

type Resolver struct{}

// // foo
func (r *mutationResolver) DeployFunction(ctx context.Context, input model.DeployFunctionInput) (*model.FunctionVersion, error) {
	panic("not implemented")
}

// // foo
func (r *mutationResolver) UpsertActionVersion(ctx context.Context, input model.UpsertActionVersionInput) (*model.ActionVersion, error) {
	panic("not implemented")
}

// // foo
func (r *queryResolver) Config(ctx context.Context) (*model.Config, error) {
	dockerRegistry := "registry.inngest.com"
	dockerNamespace := "inngest" // TODO - Update w/ account DSN

	config := &model.Config{
		Execution: &model.ExecutionConfig{
			Drivers: &model.ExecutionDriversConfig{
				Docker: &model.ExecutionDockerDriverConfig{
					Registry:  &dockerRegistry,
					Namespace: &dockerNamespace,
				},
			},
		},
	}
	return config, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
