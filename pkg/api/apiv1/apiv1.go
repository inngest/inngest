package apiv1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
)

// Opts represents options for the APIv1 router.
type Opts struct {
	// AuthMiddleware authenticates the incoming API request.
	AuthMiddleware func(http.Handler) http.Handler
	// CachingMiddleware caches API responses, if the handler specifies
	// a max-age.
	CachingMiddleware CachingMiddleware
	// WorkspaceFinder returns the authenticated workspace given the current context.
	AuthFinder AuthFinder
	// Executor is required to cancel and manage function executions.
	Executor execution.Executor
	// EventReader allows reading of events from storage.
	EventReader EventReader
	// FunctionRunReader reads function runs, history, etc. from backing storage
	FunctionRunReader cqrs.APIV1FunctionRunReader
	// JobQueueReader reads information around a function run's job queues.
	JobQueueReader queue.JobQueueReader
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
	a.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)

		if a.opts.CachingMiddleware != nil {
			r.Use(a.opts.CachingMiddleware.Middleware)
		}

		r.Get("/events", a.GetEvents)
		r.Get("/events/{eventID}", a.GetEvent)
		r.Get("/events/{eventID}/runs", a.GetEventRuns)
		r.Get("/runs/{runID}", a.GetFunctionRun)
		r.Delete("/runs/{runID}", a.CancelFunctionRun)
		r.Get("/runs/{runID}/jobs", a.GetFunctionRunJobs)
	})
}

func WriteResponse[T any](w http.ResponseWriter, data T) error {
	return WriteCachedResponse(w, data, 0)
}

func WriteCachedResponse[T any](w http.ResponseWriter, data T, cachePeriod time.Duration) error {
	resp := Response[T]{
		Data: data,
		Metadata: ResponseMetadata{
			FetchedAt: time.Now(),
		},
	}

	if cachePeriod.Seconds() > 0 {
		cachedUntil := time.Now().Add(cachePeriod)
		resp.Metadata.CachedUntil = &cachedUntil
		// Set a max-age header if the response is cacheable.  This instructs
		// our caching middleware to cache the result for this period of time.
		w.Header().Set("Cache-Control", fmt.Sprintf("private, max-age=%d", int(cachePeriod.Seconds())))
	}

	return json.NewEncoder(w).Encode(resp)
}

// Response represents
type Response[T any] struct {
	Data     T                `json:"data"`
	Metadata ResponseMetadata `json:"metadata"`
}

// ResponseMetadata represents metadata regarding the response.
type ResponseMetadata struct {
	FetchedAt   time.Time  `json:"fetchedAt,omitempty"`
	CachedUntil *time.Time `json:"cachedUntil,omitempty"`
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
