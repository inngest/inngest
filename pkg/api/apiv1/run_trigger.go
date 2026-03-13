package apiv1

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
)

// RunTriggerResponse is the API representation of a run's trigger.
type RunTriggerResponse struct {
	EventName string            `json:"event_name,omitempty"`
	IDs       []string          `json:"ids,omitempty"`
	Payloads  []json.RawMessage `json:"payloads,omitempty"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
	IsBatch   bool              `json:"is_batch,omitempty"`
	Cron      *string           `json:"cron,omitempty"`
	BatchID   *string           `json:"batch_id,omitempty"`
}

func (a router) getRunTrigger(w http.ResponseWriter, r *http.Request) {
	if a.opts.RateLimited(r, w, "/v1/runs/{runID}/trigger") {
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

	runID, err := ulid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid run ID: %s", chi.URLParam(r, "runID")))
		return
	}

	// Get the trace run to access trigger IDs
	run, err := a.opts.TraceReader.GetTraceRun(ctx, cqrs.TraceRunIdentifier{
		AccountID:   auth.AccountID(),
		WorkspaceID: auth.WorkspaceID(),
		RunID:       runID,
	})
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load run: %s", runID))
		return
	}
	if run == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(404, "Run not found: %s", runID))
		return
	}

	resp := RunTriggerResponse{
		Cron:    run.CronSchedule,
		IsBatch: run.BatchID != nil,
	}

	if run.BatchID != nil {
		s := run.BatchID.String()
		resp.BatchID = &s
	}

	// Parse trigger IDs and fetch events concurrently
	n := len(run.TriggerIDs)
	resp.IDs = make([]string, n)
	resp.Payloads = make([]json.RawMessage, n)

	var (
		mu        sync.Mutex
		ts        time.Time
		eventName string
	)

	eg, egCtx := errgroup.WithContext(ctx)
	for i, id := range run.TriggerIDs {
		resp.IDs[i] = id

		evtID, err := ulid.Parse(id)
		if err != nil {
			continue
		}

		// Track earliest timestamp from event IDs
		evtTime := ulid.Time(evtID.Time())
		mu.Lock()
		if ts.IsZero() || evtTime.Before(ts) {
			ts = evtTime
		}
		mu.Unlock()

		eg.Go(func() error {
			evt, err := a.opts.TraceReader.GetEvent(egCtx, evtID, auth.AccountID(), auth.WorkspaceID())
			if err != nil || evt == nil {
				return nil
			}

			payload, err := json.Marshal(evt.GetEvent())
			if err != nil {
				return nil
			}

			resp.Payloads[i] = payload

			mu.Lock()
			if run.CronSchedule == nil && eventName == "" {
				eventName = evt.EventName
			}
			mu.Unlock()

			return nil
		})
	}

	_ = eg.Wait()
	resp.EventName = eventName

	if !ts.IsZero() {
		resp.Timestamp = &ts
	}

	_ = WriteResponse(w, resp)
}
