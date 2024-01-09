package apiv1

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/publicerr"
)

type FunctionConfig struct {
	Priority    *inngest.Priority          `json:"priority,omitempty"`
	Concurrency *inngest.ConcurrencyLimits `json:"concurrency,omitempty"`
	Debounce    *inngest.Debounce          `json:"debounce,omitempty"`
	Triggers    []inngest.Trigger          `json:"triggers"`
	Batching    *inngest.EventBatchConfig  `json:"batching,omitempty"`
	RateLimit   *inngest.RateLimit         `json:"ratelimit,omitempty"`
	Cancel      []inngest.Cancel           `json:"cancel"`
	Version     int                        `json:"version"`
}

type FunctionResponse struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	Config    FunctionConfig `json:"config"`
	AppID     uuid.UUID      `json:"app_id"`
	CreatedAt time.Time      `json:"created_at"`
}

func (a api) GetFunctions(w http.ResponseWriter, r *http.Request) {}

func (a api) GetFunction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	appID, err := uuid.Parse(chi.URLParam(r, "app_id"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, err.Error()))
		return
	}

	id := chi.URLParam(r, "id")
	fn, err := a.findFnByID(ctx, appID, id)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 404, err.Error()))
		return
	}

	// TODO: authorize if fn accountID matches auth accountID
	if auth.AccountID() != uuid.Nil {
		fmt.Println("Check to make sure account ID matches the auth account ID")
	}

	resp, err := toAPIResponse(fn)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, err.Error()))
	}

	_ = WriteCachedResponse(w, resp, 0) // no caching
}

func (a api) findFnByID(ctx context.Context, appID uuid.UUID, id string) (*cqrs.Function, error) {
	fnID, err := uuid.Parse(id)
	if err != nil {
		// If not parsable as UUID, assume it is a slug so it's not an error
		// NOTE: should this be logged or reported?
		log.From(ctx).
			Error().
			Str("ID", id).
			Err(err).
			Msg("could not parse ID as UUID")
	}

	var fn *cqrs.Function
	if fnID != uuid.Nil {
		// Query function by UUID
		if fn, err = a.opts.FunctionReader.GetFunctionByID(ctx, fnID); err != nil {
			return nil, fmt.Errorf("error finding function with ID '%s': %w", fnID, err)
		}
	} else {
		// Query function by slug
		if fn, err = a.opts.FunctionReader.GetFunctionBySlug(ctx, id); err != nil {
			return nil, fmt.Errorf("error finding function with ID '%s': %w", id, err)
		}
	}

	// Check to make sure fn appID matches the passed in appID
	if fn.AppID != appID {
		return nil, fmt.Errorf("function not found in app: %s", appID)
	}

	return fn, nil
}

func toAPIResponse(fn *cqrs.Function) (*FunctionResponse, error) {
	ingFn, err := fn.InngestFunction()
	if err != nil {
		return nil, err
	}

	resp := FunctionResponse{
		Name: fn.Name,
		ID:   fn.ID,
		Slug: fn.Slug,
		Config: FunctionConfig{
			Priority:    ingFn.Priority,
			Concurrency: ingFn.Concurrency,
			Debounce:    ingFn.Debounce,
			Triggers:    ingFn.Triggers,
			Batching:    ingFn.EventBatch,
			RateLimit:   ingFn.RateLimit,
			Cancel:      ingFn.Cancel,
			Version:     ingFn.FunctionVersion,
		},
		AppID:     fn.AppID,
		CreatedAt: fn.CreatedAt,
	}

	return &resp, nil
}
