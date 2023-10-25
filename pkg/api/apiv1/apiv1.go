package apiv1

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
)

type Opts struct {
	// Docs shows API docs inline.
	HideDocs bool
	// AuthMiddleware authenticates the incoming API request.
	AuthMiddleware func(http.Handler) http.Handler
	// WorkspaceFinder returns the authenticated workspace given the current context.
	WorkspaceFinder WorkspaceFinder

	Executor execution.Executor

	// EventReader allows reading of events from storage.
	EventReader       EventReader
	FunctionRunReader FunctionRunReader
	JobQueueReader    queue.JobQueueReader
}

type WorkspaceFinder func(ctx context.Context) uuid.UUID

func nilWorkspaceFinder(ctx context.Context) uuid.UUID {
	return uuid.UUID{}
}

// AddRoutes adds a new API handler to the given router.
func AddRoutes(r chi.Router, o Opts) (http.Handler, error) {
	instance := &api{Router: r}
	if o.AuthMiddleware != nil {
		instance.Use(o.AuthMiddleware)
	}
	if o.WorkspaceFinder == nil {
		o.WorkspaceFinder = nilWorkspaceFinder
	}

	instance.opts = o
	err := instance.setup()
	return instance, err
}

type api struct {
	chi.Router
	opts Opts
}

func (a *api) setup() error {
	a.Get("/v1/events", a.GetEvents)
	a.Get("/v1/events/{eventID}", a.GetEvent)
	a.Get("/v1/events/{eventID}/runs", a.GetEventRuns)
	a.Get("/v1/runs/{runID}", a.GetFunctionRun)
	a.Get("/v1/runs/{runID}/jobs", a.GetFunctionRunJobs)
	a.Delete("/v1/runs/{runID}", a.CancelFunctionRun)
	return nil
}
