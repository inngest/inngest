package resolvers

import (
	"context"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/internal/cuedefs"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
)

func (r *queryResolver) ActionVersion(ctx context.Context, query models.ActionVersionQuery) (*client.ActionVersion, error) {
	vc := &inngest.VersionConstraint{}
	if query.VersionMajor != nil {
		major := uint(*query.VersionMajor)
		vc.Major = &major
		if query.VersionMinor != nil {
			minor := uint(*query.VersionMinor)
			vc.Minor = &minor
		}
	}
	av, err := r.APIReadWriter.ActionVersion(ctx, query.Dsn, vc)
	if err != nil {
		return nil, err
	}
	return &av, nil
}

func (r *mutationResolver) CreateActionVersion(ctx context.Context, input models.CreateActionVersionInput) (*client.ActionVersion, error) {
	// TODO - Do we need additional validation beyond parsing the cue string?
	parsed, err := cuedefs.ParseAction(input.Config)
	if err != nil {
		return nil, err
	}
	created, err := r.APIReadWriter.CreateActionVersion(ctx, *parsed)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *mutationResolver) UpdateActionVersion(ctx context.Context, input models.UpdateActionVersionInput) (*client.ActionVersion, error) {
	version := inngest.VersionInfo{
		Major: uint(input.VersionMajor),
		Minor: uint(input.VersionMinor),
	}
	updated, err := r.APIReadWriter.UpdateActionVersion(ctx, input.Dsn, version, *input.Enabled)
	if err != nil {
		return nil, err
	}
	return &updated, nil
}
