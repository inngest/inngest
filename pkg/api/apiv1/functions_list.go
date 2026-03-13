package apiv1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/inngest/inngest/pkg/publicerr"
)

// FunctionResponse is the API representation of a function.
type FunctionResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Slug     string            `json:"slug"`
	AppID    string            `json:"app_id"`
	AppName  string            `json:"app_name,omitempty"`
	Triggers []TriggerResponse `json:"triggers,omitempty"`
}

// TriggerResponse is the API representation of a function trigger.
type TriggerResponse struct {
	Type  string `json:"type"`
	Value string `json:"value"`
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

	resp := make([]FunctionResponse, 0, limit)
	hasMore := false

	// TODO: FunctionReader lacks a bulk "get all functions by workspace" method,
	// so we query per-app. Consider adding a batch method to reduce round trips.
	for _, app := range apps {
		fns, err := a.opts.FunctionReader.GetFunctionsByAppInternalID(ctx, app.ID)
		if err != nil {
			continue
		}

		for _, fn := range fns {
			if len(resp) >= limit {
				hasMore = true
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
			}

			resp = append(resp, fr)
		}
		if hasMore {
			break
		}
	}

	_ = WriteCachedResponse(w, FunctionsListResponse{
		Functions: resp,
		HasMore:   hasMore,
	}, 2*time.Second)
}
