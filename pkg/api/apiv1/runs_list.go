package apiv1

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/publicerr"
	"golang.org/x/sync/errgroup"
)

// RunResponse is the API representation of a function run.
type RunResponse struct {
	ID           string          `json:"id"`
	Status       string          `json:"status"`
	QueuedAt     time.Time       `json:"queued_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	EndedAt      *time.Time      `json:"ended_at,omitempty"`
	TraceID      string          `json:"trace_id,omitempty"`
	FunctionID   string          `json:"function_id"`
	FunctionName string          `json:"function_name,omitempty"`
	FunctionSlug string          `json:"function_slug,omitempty"`
	AppID        string          `json:"app_id"`
	AppName      string          `json:"app_name,omitempty"`
	EventName    string          `json:"event_name,omitempty"`
	CronSchedule *string         `json:"cron_schedule,omitempty"`
	IsBatch      bool            `json:"is_batch,omitempty"`
	Output       json.RawMessage `json:"output,omitempty"`
}

// RunsListResponse is the paginated list of runs.
type RunsListResponse struct {
	Runs       []RunResponse `json:"runs"`
	TotalCount int           `json:"total_count"`
	Cursor     string        `json:"cursor,omitempty"`
	HasMore    bool          `json:"has_more"`
}

// RunsCountResponse wraps a count result.
type RunsCountResponse struct {
	Count int `json:"count"`
}

// parseTimeRangeFilters parses shared status and time range filters from the request.
func parseTimeRangeFilters(r *http.Request, filter *cqrs.GetTraceRunFilter) error {
	for _, s := range r.URL.Query()["status"] {
		status, err := enums.RunStatusString(s)
		if err != nil {
			return publicerr.Wrapf(err, 400, "Invalid status: %s", s)
		}
		filter.Status = append(filter.Status, status)
	}

	if from := r.FormValue("from"); from != "" {
		parsed, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return publicerr.Wrap(err, 400, "Invalid from timestamp")
		}
		filter.From = parsed
	} else {
		filter.From = time.Now().Add(-24 * time.Hour)
	}

	if until := r.FormValue("until"); until != "" {
		parsed, err := time.Parse(time.RFC3339, until)
		if err != nil {
			return publicerr.Wrap(err, 400, "Invalid until timestamp")
		}
		filter.Until = parsed
	} else {
		filter.Until = time.Now()
	}

	return nil
}

// parsedFunction caches both the raw function and its parsed inngest config.
type parsedFunction struct {
	fn        *cqrs.Function
	inngestFn *inngest.Function
}

func (a router) listRuns(w http.ResponseWriter, r *http.Request) {
	if a.opts.RateLimited(r, w, "/v1/runs") {
		return
	}

	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	if a.opts.TraceReader == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "No trace reader specified"))
		return
	}

	// Parse query params
	limit, _ := strconv.Atoi(r.FormValue("first"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 400 {
		limit = 400
	}

	filter := cqrs.GetTraceRunFilter{
		AccountID:   auth.AccountID(),
		WorkspaceID: auth.WorkspaceID(),
	}

	if err := parseTimeRangeFilters(r, &filter); err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	// Parse function_id filters
	for _, id := range r.URL.Query()["function_id"] {
		parsed, err := uuid.Parse(id)
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid function_id: %s", id))
			return
		}
		filter.FunctionID = append(filter.FunctionID, parsed)
	}

	// Parse app_id filters
	for _, id := range r.URL.Query()["app_id"] {
		parsed, err := uuid.Parse(id)
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid app_id: %s", id))
			return
		}
		filter.AppID = append(filter.AppID, parsed)
	}

	opts := cqrs.GetTraceRunOpt{
		Filter: filter,
		Order: []cqrs.GetTraceRunOrder{
			{Field: enums.TraceRunTimeQueuedAt, Direction: enums.TraceRunOrderDesc},
		},
		Cursor: r.FormValue("cursor"),
		Items:  uint(limit),
	}

	// Fetch runs and count concurrently
	var (
		runs  []*cqrs.TraceRun
		count int
	)
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		runs, err = a.opts.TraceReader.GetTraceRuns(egCtx, opts)
		return err
	})
	eg.Go(func() error {
		var err error
		count, err = a.opts.TraceReader.GetTraceRunsCount(egCtx, opts)
		return err
	})
	if err := eg.Wait(); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to query runs"))
		return
	}

	// Collect unique IDs for enrichment
	fnIDs := map[uuid.UUID]struct{}{}
	appIDs := map[uuid.UUID]struct{}{}
	for _, run := range runs {
		fnIDs[run.FunctionID] = struct{}{}
		appIDs[run.AppID] = struct{}{}
	}

	// Fetch function and app data concurrently
	var mu sync.Mutex
	fnCache := make(map[uuid.UUID]*parsedFunction, len(fnIDs))
	appCache := make(map[uuid.UUID]string, len(appIDs))

	eg2, egCtx2 := errgroup.WithContext(ctx)
	for fnID := range fnIDs {
		eg2.Go(func() error {
			fn, err := a.opts.FunctionReader.GetFunctionByInternalUUID(egCtx2, fnID)
			if err != nil || fn == nil {
				return nil
			}
			pf := &parsedFunction{fn: fn}
			if inngestFn, err := fn.InngestFunction(); err == nil {
				pf.inngestFn = inngestFn
			}
			mu.Lock()
			fnCache[fnID] = pf
			mu.Unlock()
			return nil
		})
	}
	if a.opts.AppReader != nil {
		for appID := range appIDs {
			eg2.Go(func() error {
				app, err := a.opts.AppReader.GetAppByID(egCtx2, appID)
				if err != nil || app == nil {
					return nil
				}
				mu.Lock()
				appCache[appID] = app.Name
				mu.Unlock()
				return nil
			})
		}
	}
	_ = eg2.Wait() // Best-effort enrichment; don't fail the request

	resp := RunsListResponse{
		Runs:       make([]RunResponse, 0, len(runs)),
		TotalCount: count,
	}

	for _, run := range runs {
		rr := RunResponse{
			ID:           run.RunID,
			Status:       run.Status.String(),
			QueuedAt:     run.QueuedAt,
			TraceID:      run.TraceID,
			FunctionID:   run.FunctionID.String(),
			AppID:        run.AppID.String(),
			CronSchedule: run.CronSchedule,
			IsBatch:      run.IsBatch,
			Output:       run.Output,
		}

		if !run.StartedAt.IsZero() {
			rr.StartedAt = &run.StartedAt
		}
		if !run.EndedAt.IsZero() {
			rr.EndedAt = &run.EndedAt
		}

		// Resolve function name/slug and event name
		if pf, ok := fnCache[run.FunctionID]; ok {
			rr.FunctionName = pf.fn.Name
			rr.FunctionSlug = pf.fn.Slug

			if pf.inngestFn != nil {
				for _, t := range pf.inngestFn.Triggers {
					if t.EventTrigger != nil {
						rr.EventName = t.Event
					}
				}
			}
		}

		// Resolve app name
		if name, ok := appCache[run.AppID]; ok {
			rr.AppName = name
		}

		resp.Runs = append(resp.Runs, rr)
	}

	// Use the last run's cursor for pagination
	if len(runs) > 0 && runs[len(runs)-1].Cursor != "" {
		resp.Cursor = runs[len(runs)-1].Cursor
	}

	resp.HasMore = len(runs) == limit

	_ = WriteCachedResponse(w, resp, 2*time.Second)
}

func (a router) getRunsCounts(w http.ResponseWriter, r *http.Request) {
	if a.opts.RateLimited(r, w, "/v1/runs/counts") {
		return
	}

	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	if a.opts.TraceReader == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "No trace reader specified"))
		return
	}

	filter := cqrs.GetTraceRunFilter{
		AccountID:   auth.AccountID(),
		WorkspaceID: auth.WorkspaceID(),
	}

	if err := parseTimeRangeFilters(r, &filter); err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}

	opts := cqrs.GetTraceRunOpt{Filter: filter}

	count, err := a.opts.TraceReader.GetTraceRunsCount(ctx, opts)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to count runs"))
		return
	}

	_ = WriteCachedResponse(w, RunsCountResponse{Count: count}, 2*time.Second)
}
