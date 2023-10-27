package apiv1

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
)

type Opts struct {
	// AuthMiddleware authenticates the incoming API request.
	AuthMiddleware func(http.Handler) http.Handler
	// WorkspaceFinder returns the authenticated workspace given the current context.
	AuthFinder AuthFinder

	Executor execution.Executor

	// EventReader allows reading of events from storage.
	EventReader       EventReader
	FunctionRunReader cqrs.APIV1FunctionRunReader
	JobQueueReader    queue.JobQueueReader
}

// AddRoutes adds a new API handler to the given router.
func AddRoutes(r chi.Router, o Opts) http.Handler {
	instance := &api{Router: r}
	if o.AuthMiddleware != nil {
		instance.Use(o.AuthMiddleware)
	}
	if o.AuthFinder == nil {
		o.AuthFinder = nilAuthFinder
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

// TODO (tonyhb) Open source the auth context.

// AuthFinder returns auth information from the current context.
type AuthFinder func(ctx context.Context) (V1Auth, error)

// V1Auth represents an object that returns the account and worskpace currently authed.
type V1Auth interface {
	AccountID() uuid.UUID
	WorkspaceID() uuid.UUID
}

// nilAuthFinder is used in the dev server, returning zero auth.
func nilAuthFinder(ctx context.Context) (V1Auth, error) {
	return nilAuth{}, nil
}

type nilAuth struct{}

func (nilAuth) AccountID() uuid.UUID {
	return uuid.UUID{}
}

func (nilAuth) WorkspaceID() uuid.UUID {
	return uuid.UUID{}
}
