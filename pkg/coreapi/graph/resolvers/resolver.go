package resolvers

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"github.com/inngest/inngest/pkg/coreapi/generated"
	"github.com/inngest/inngest/pkg/coredata"
	"github.com/inngest/inngest/pkg/execution/runner"
)

type Resolver struct {
	APIReadWriter coredata.APIReadWriter
	Runner        runner.Runner
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

func (r *Resolver) Event() generated.EventResolver { return &eventResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type eventResolver struct{ *Resolver }
