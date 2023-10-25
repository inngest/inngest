package apiv1

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/dateutil"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

// GetEvents returns events in reverse chronological order for a workspace, with optional pagination
// and filtering params.
func (a api) GetEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID := a.opts.WorkspaceFinder(ctx)

	// NOTE (tonyhb): I'd love it if this was actually bounded by the schema above.
	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit == 0 {
		limit = 20
	}
	limit = util.Bound(limit, 1, 100)

	opts := cqrs.WorkspaceEventsOpts{
		Limit: limit,
	}

	if cursor := r.FormValue("cursor"); cursor != "" {
		parsed, err := ulid.Parse(cursor)
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid cursor query parameter"))
			return
		}
		opts.Cursor = &parsed
	}

	if after := r.FormValue("received_after"); after != "" {
		parsed, err := dateutil.Parse(after)
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid received_after query parameter"))
			return
		}
		opts.After = parsed
	}

	if a.opts.EventReader == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "No event reader specified"))
		return
	}

	events, err := a.opts.EventReader.WorkspaceEvents(ctx, wsID, r.FormValue("name"), opts)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to query events"))
		return
	}
	_ = json.NewEncoder(w).Encode(events)
}

// GetEvent returns a specific event for the given workspace.
func (a api) GetEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID := a.opts.WorkspaceFinder(ctx)

	eventID := chi.URLParam(r, "eventID")
	parsed, err := ulid.Parse(eventID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid event ID: %s", eventID))
		return
	}

	if a.opts.EventReader == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "No event reader specified"))
		return
	}

	event, err := a.opts.EventReader.FindEvent(ctx, wsID, parsed)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to query events"))
		return
	}
	_ = json.NewEncoder(w).Encode(event)
}

// GetEventRuns returns function runs given an event ID.
func (a api) GetEventRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID := a.opts.WorkspaceFinder(ctx)

	eventID := chi.URLParam(r, "eventID")
	parsed, err := ulid.Parse(eventID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid event ID: %s", eventID))
		return
	}

	fr, err := a.opts.FunctionRunReader.GetFunctionRunsFromEvents(ctx, []ulid.ULID{parsed})
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to query function runs"))
		return
	}

	result := []*cqrs.FunctionRun{}
	for _, item := range fr {
		if item.WorkspaceID == wsID {
			result = append(result, item)
		}
	}
	_ = json.NewEncoder(w).Encode(result)
}
