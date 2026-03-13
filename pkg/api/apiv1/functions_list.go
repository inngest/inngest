package apiv1

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/inngest/inngest/pkg/publicerr"
)

// FunctionResponse is the API representation of a function.
type FunctionResponse struct {
	ID       string                  `json:"id"`
	Name     string                  `json:"name"`
	Slug     string                  `json:"slug"`
	AppID    string                  `json:"app_id"`
	AppName  string                  `json:"app_name,omitempty"`
	Triggers []TriggerResponse       `json:"triggers,omitempty"`
	Config   *FunctionConfigResponse `json:"config,omitempty"`
}

// TriggerResponse is the API representation of a function trigger.
type TriggerResponse struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// FunctionConfigResponse holds function configuration for the API.
type FunctionConfigResponse struct {
	Retries     *int                  `json:"retries,omitempty"`
	Concurrency []ConcurrencyResponse `json:"concurrency,omitempty"`
	RateLimit   *RateLimitResponse    `json:"rate_limit,omitempty"`
	BatchSize   *int                  `json:"batch_size,omitempty"`
}

// ConcurrencyResponse is the API representation of a concurrency limit.
type ConcurrencyResponse struct {
	Scope string `json:"scope,omitempty"`
	Limit int    `json:"limit"`
	Key   string `json:"key,omitempty"`
}

// RateLimitResponse is the API representation of a rate limit.
type RateLimitResponse struct {
	Limit  int    `json:"limit"`
	Period string `json:"period"`
	Key    string `json:"key,omitempty"`
}

// FunctionsListResponse is the paginated list of functions.
type FunctionsListResponse struct {
	Functions []FunctionResponse `json:"functions"`
	HasMore   bool               `json:"has_more"`
}

func (a router) listFunctions(w http.ResponseWriter, r *http.Request) {
	if a.opts.RateLimited(r, w, "/v1/functions") {
		return
	}

	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	if a.opts.AppReader == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(501, "Function listing not available"))
		return
	}
	if a.opts.FunctionReader == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(501, "Function listing not available"))
		return
	}

	// Parse pagination params
	limit, _ := strconv.Atoi(r.FormValue("first"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 400 {
		limit = 400
	}

	apps, err := a.opts.AppReader.GetAllApps(ctx, auth.WorkspaceID())
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to query apps"))
		return
	}

	// Collect up to limit+1 items so we can accurately determine hasMore
	collectLimit := limit + 1
	resp := make([]FunctionResponse, 0, collectLimit)

	// TODO: FunctionReader lacks a bulk "get all functions by workspace" method,
	// so we query per-app. Consider adding a batch method to reduce round trips.
	for _, app := range apps {
		fns, err := a.opts.FunctionReader.GetFunctionsByAppInternalID(ctx, app.ID)
		if err != nil {
			slog.WarnContext(ctx, "failed to list functions for app", "app_id", app.ID, "error", err)
			continue
		}

		for _, fn := range fns {
			if len(resp) >= collectLimit {
				break
			}

			fr := FunctionResponse{
				ID:      fn.ID.String(),
				Name:    fn.Name,
				Slug:    fn.Slug,
				AppID:   fn.AppID.String(),
				AppName: app.Name,
			}

			inngestFn, err := fn.InngestFunction()
			if err == nil && inngestFn != nil {
				for _, t := range inngestFn.Triggers {
					tr := TriggerResponse{}
					if t.EventTrigger != nil {
						tr.Type = "event"
						tr.Value = t.Event
					} else if t.CronTrigger != nil {
						tr.Type = "cron"
						tr.Value = t.Cron
					}
					if tr.Value != "" {
						fr.Triggers = append(fr.Triggers, tr)
					}
				}

				// Populate config
				config := &FunctionConfigResponse{}
				hasConfig := false

				if len(inngestFn.Steps) > 0 {
					retries := inngestFn.Steps[0].RetryCount()
					config.Retries = &retries
					hasConfig = true
				}

				if inngestFn.Concurrency != nil {
					for _, c := range inngestFn.Concurrency.Limits {
						cr := ConcurrencyResponse{
							Limit: c.Limit,
							Scope: c.Scope.String(),
						}
						if c.Key != nil {
							cr.Key = *c.Key
						}
						config.Concurrency = append(config.Concurrency, cr)
					}
					hasConfig = true
				}

				if inngestFn.RateLimit != nil {
					rl := &RateLimitResponse{
						Limit:  int(inngestFn.RateLimit.Limit),
						Period: inngestFn.RateLimit.Period,
					}
					if inngestFn.RateLimit.Key != nil {
						rl.Key = *inngestFn.RateLimit.Key
					}
					config.RateLimit = rl
					hasConfig = true
				}

				if inngestFn.EventBatch != nil {
					config.BatchSize = &inngestFn.EventBatch.MaxSize
					hasConfig = true
				}

				if hasConfig {
					fr.Config = config
				}
			}

			resp = append(resp, fr)
		}
		if len(resp) >= collectLimit {
			break
		}
	}

	hasMore := len(resp) > limit
	if hasMore {
		resp = resp[:limit]
	}

	_ = WriteCachedResponse(w, FunctionsListResponse{
		Functions: resp,
		HasMore:   hasMore,
	}, 2*time.Second)
}
