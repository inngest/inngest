package resolvers

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"github.com/inngest/inngest/pkg/coreapi/generated"
	"github.com/inngest/inngest/pkg/coredata"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/runner"
)

type Resolver struct {
	Data          cqrs.Manager
	APIReadWriter coredata.APIReadWriter
	Runner        runner.Runner
}

// Mutation returns generated.MutationResolver implementation. Only necessary when mutations exist.
// func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

func (r *Resolver) Event() generated.EventResolver { return &eventResolver{r} }

func (r *Resolver) FunctionRun() generated.FunctionRunResolver { return &functionRunResolver{r} }

func (r *Resolver) App() generated.AppResolver { return &appResolver{r} }

// type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type eventResolver struct{ *Resolver }
type appResolver struct{ *Resolver }
type functionRunResolver struct{ *Resolver }
