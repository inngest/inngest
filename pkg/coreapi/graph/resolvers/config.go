package resolvers

import (
	"context"

	model "github.com/inngest/inngest/pkg/coreapi/graph/models"
)

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
