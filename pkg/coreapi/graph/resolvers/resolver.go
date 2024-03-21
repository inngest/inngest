package resolvers

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/coreapi/generated"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/history_reader"
)

type Resolver struct {
	Data          cqrs.Manager
	HistoryReader history_reader.Reader
	Runner        runner.Runner
	Queue         queue.JobQueueReader
	EventHandler  api.EventHandler
	Executor      execution.Executor
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

func (r *Resolver) Event() generated.EventResolver { return &eventResolver{r} }

func (r *Resolver) FunctionRun() generated.FunctionRunResolver { return &functionRunResolver{r} }

func (r *Resolver) App() generated.AppResolver { return &appResolver{r} }

func (r *Resolver) Function() generated.FunctionResolver { return &functionResolver{r} }

func (r *Resolver) StreamItem() generated.StreamItemResolver { return &streamItemResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type eventResolver struct{ *Resolver }
type appResolver struct{ *Resolver }
type functionRunResolver struct{ *Resolver }
type functionResolver struct{ *Resolver }
type streamItemResolver struct{ *Resolver }
