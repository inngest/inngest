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

type WorkspaceFinder func(ctx context.Context) (uuid.UUID, error)

func nilWorkspaceFinder(ctx context.Context) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}

// AddRoutes adds a new API handler to the given router.
func AddRoutes(r chi.Router, o Opts) http.Handler {
	instance := &api{Router: r}
	if o.AuthMiddleware != nil {
		instance.Use(o.AuthMiddleware)
	}
	if o.WorkspaceFinder == nil {
		o.WorkspaceFinder = nilWorkspaceFinder
	}

	instance.opts = o
	instance.setup()
	return instance
}

type api struct {
	chi.Router
	opts Opts
}

func (a *api) setup() {
	a.Get("/events", a.GetEvents)
	a.Get("/events/{eventID}", a.GetEvent)
	a.Get("/events/{eventID}/runs", a.GetEventRuns)
	a.Get("/runs/{runID}", a.GetFunctionRun)
	a.Get("/runs/{runID}/jobs", a.GetFunctionRunJobs)
	a.Delete("/runs/{runID}", a.CancelFunctionRun)
}
