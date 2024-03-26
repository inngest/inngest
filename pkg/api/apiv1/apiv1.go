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
	"github.com/inngest/inngest/pkg/headers"
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
	// FunctionReader reads functions from a backing store.
	FunctionReader cqrs.FunctionReader
	// FunctionRunReader reads function runs, history, etc. from backing storage
	FunctionRunReader cqrs.APIV1FunctionRunReader
	// JobQueueReader reads information around a function run's job queues.
	JobQueueReader queue.JobQueueReader
	// CancellationReadWriter reads and writes cancellations to/from a backing store.
	CancellationReadWriter cqrs.CancellationReadWriter
}

// AddRoutes adds a new API handler to the given router.
func AddRoutes(r chi.Router, o Opts) http.Handler {
	if o.AuthFinder == nil {
		o.AuthFinder = nilAuthFinder
	}

	// Create the HTTP implementation, which wraps the handler.  We do ths to code
	// share and split the HTTP concerns from the actual logic, eg. to share to GQL.
	impl := &API{opts: o}

	instance := &router{
		Router: r,
		API:    impl,
	}
	// Add the auth middleware, if specified.
	if o.AuthMiddleware != nil {
		instance.Use(o.AuthMiddleware)
	}
	instance.setup()
	return instance
}

type API struct {
	opts Opts
}

type router struct {
	*API
	chi.Router
}

func (a *router) setup() {
	a.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer)

		if a.opts.CachingMiddleware != nil {
			r.Use(a.opts.CachingMiddleware.Middleware)
		}

		r.Use(headers.ContentTypeJsonResponse())

		r.Get("/events", a.getEvents)
		r.Get("/events/{eventID}", a.getEvent)
		r.Get("/events/{eventID}/runs", a.getEventRuns)
		r.Get("/runs/{runID}", a.GetFunctionRun)
		r.Delete("/runs/{runID}", a.cancelFunctionRun)
		r.Get("/runs/{runID}/jobs", a.GetFunctionRunJobs)

		r.Get("/apps/{appName}/functions", a.GetAppFunctions) // Returns an app and all of its functions.

		r.Post("/cancellations", a.createCancellation)
		r.Get("/cancellations", a.getCancellations)
		r.Delete("/cancellations/{id}", a.deleteCancellation)
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
	FetchedAt   time.Time  `json:"fetched_at,omitempty"`
	CachedUntil *time.Time `json:"cached_until,omitempty"`
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
