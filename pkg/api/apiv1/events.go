package apiv1

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/dateutil"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

const (
	DefaultEvents = 20
)

// GetEvents returns events in reverse chronological order for a workspace, with optional pagination
// and filtering params.
func (a api) GetEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit == 0 {
		limit = DefaultEvents
	}
	limit = util.Bound(limit, 1, cqrs.MaxEvents)

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

	if before := r.FormValue("received_before"); before != "" {
		parsed, err := dateutil.Parse(before)
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid received_after query parameter"))
			return
		}
		opts.Newest = parsed
	}

	if after := r.FormValue("received_after"); after != "" {
		parsed, err := dateutil.Parse(after)
		if err != nil {
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid received_after query parameter"))
			return
		}
		opts.Oldest = parsed
	}

	if name := r.FormValue("name"); name != "" {
		opts.Name = &name
	}

	if a.opts.EventReader == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "No event reader specified"))
		return
	}

	events, err := a.opts.EventReader.WorkspaceEvents(ctx, auth.WorkspaceID(), &opts)
	if err != nil {
		logger.StdlibLogger(ctx).Error("error querying events", "error", err)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to query events"))
		return
	}

	// Do not cache this response.
	_ = WriteResponse(w, events)
}

// GetEvent returns a specific event for the given workspace.
func (a api) GetEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

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

	event, err := a.opts.EventReader.FindEvent(ctx, auth.WorkspaceID(), parsed)
	if err == sql.ErrNoRows {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 404, "Event not found"))
		return
	}
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to query events"))
		return
	}
	_ = WriteCachedResponse(w, event, 5*time.Second)
}

// GetEventRuns returns function runs given an event ID.
func (a api) GetEventRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	eventID := chi.URLParam(r, "eventID")
	parsed, err := ulid.Parse(eventID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid event ID: %s", eventID))
		return
	}

	fr, err := a.opts.FunctionRunReader.GetFunctionRunsFromEvents(
		ctx,
		auth.AccountID(),
		auth.WorkspaceID(),
		[]ulid.ULID{parsed},
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to query function runs"))
		return
	}

	result := []*cqrs.FunctionRun{}
	for _, item := range fr {
		if item.WorkspaceID == auth.WorkspaceID() {
			result = append(result, item)
		}
	}
	_ = WriteCachedResponse(w, result, 5*time.Second)
}
