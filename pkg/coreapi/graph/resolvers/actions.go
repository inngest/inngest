package resolvers

import (
	"context"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/inngest/client"
	"github.com/inngest/inngest-cli/internal/cuedefs"
	"github.com/inngest/inngest-cli/pkg/coreapi/graph/models"
)

func internalToGraphQLModel(av client.ActionVersion) *models.ActionVersion {
	return &models.ActionVersion{
		Dsn:          av.DSN,
		Name:         av.Name,
		VersionMajor: int(av.Version.Major),
		VersionMinor: int(av.Version.Minor),
		Config:       av.Config,
		ValidFrom:    av.ValidFrom,
		ValidTo:      av.ValidTo,
		// TODO - Add missing fields to client model
	}
}

func (r *mutationResolver) CreateActionVersion(ctx context.Context, input models.CreateActionVersionInput) (*models.ActionVersion, error) {
	// TODO - Do we need additional validation beyond parsing the cue string?
	parsed, err := cuedefs.ParseAction(input.Config)
	if err != nil {
		return nil, err
	}
	created, err := r.APILoader.CreateActionVersion(ctx, *parsed)
	if err != nil {
		return nil, err
	}
	return internalToGraphQLModel(created), nil
}

func (r *mutationResolver) UpdateActionVersion(ctx context.Context, input models.UpdateActionVersionInput) (*models.ActionVersion, error) {
	version := inngest.VersionExact{
		Major: uint(input.VersionMajor),
		Minor: uint(input.VersionMinor),
	}
	updated, err := r.APILoader.UpdateActionVersion(ctx, input.Dsn, version, *input.Enabled)
	if err != nil {
		return nil, err
	}
	return internalToGraphQLModel(updated), nil
}
