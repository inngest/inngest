package apiv1

import (
	"context"
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
//
// This has no API
func (a API) GetEvents(ctx context.Context, opts *cqrs.WorkspaceEventsOpts) ([]cqrs.Event, error) {
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		return nil, publicerr.Wrap(err, 401, "No auth found")
	}

	if a.opts.EventReader == nil {
		return nil, publicerr.Errorf(500, "No event reader specified")
	}

	events, err := a.opts.EventReader.WorkspaceEvents(ctx, auth.WorkspaceID(), opts)
	if err != nil {
		logger.StdlibLogger(ctx).Error("error querying events", "error", err)
		return nil, publicerr.Wrap(err, 500, "Unable to query events")
	}
	return events, nil
}

func (a router) getEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	opts := cqrs.WorkspaceEventsOpts{}

	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit == 0 {
		limit = DefaultEvents
	}
	opts.Limit = util.Bound(limit, 1, cqrs.MaxEvents)

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

	events, err := a.API.GetEvents(ctx, &opts)
	if err != nil {
		logger.StdlibLogger(ctx).Error("error querying events", "error", err)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Unable to query events"))
		return
	}

	// Do not cache this response.
	_ = WriteResponse(w, events)
}

// GetEvent returns a specific event for the given workspace.
func (a API) GetEvent(ctx context.Context, eventID ulid.ULID) (*cqrs.Event, error) {
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		return nil, publicerr.Wrap(err, 401, "No auth found")
	}
	if a.opts.EventReader == nil {
		return nil, publicerr.Errorf(500, "No event reader specified")
	}
	event, err := a.opts.EventReader.FindEvent(ctx, auth.WorkspaceID(), eventID)
	if err == sql.ErrNoRows {
		return nil, publicerr.Wrap(err, 404, "Event not found")
	}
	if err != nil {
		return nil, publicerr.Wrap(err, 500, "Unable to query events")
	}
	return event, nil
}

// GetEvent is the HTTP implementation for retrieving events.
func (a router) getEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventID := chi.URLParam(r, "eventID")
	parsed, err := ulid.Parse(eventID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid event ID: %s", eventID))
		return
	}
	event, err := a.API.GetEvent(ctx, parsed)
	if err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}
	_ = WriteCachedResponse(w, event, 5*time.Second)
}

// GetEventRuns returns function runs given an event ID.
func (a API) GetEventRuns(ctx context.Context, eventID ulid.ULID) ([]*cqrs.FunctionRun, error) {
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		return nil, publicerr.Wrap(err, 401, "No auth found")
	}
	fr, err := a.opts.FunctionRunReader.GetFunctionRunsFromEvents(
		ctx,
		auth.AccountID(),
		auth.WorkspaceID(),
		[]ulid.ULID{eventID},
	)
	if err != nil {
		return nil, publicerr.Wrap(err, 500, "Unable to query function runs")
	}

	result := []*cqrs.FunctionRun{}
	for _, item := range fr {
		if item.WorkspaceID == auth.WorkspaceID() {
			result = append(result, item)
		}
	}
	return result, nil
}

func (a router) getEventRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventID := chi.URLParam(r, "eventID")
	parsed, err := ulid.Parse(eventID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid event ID: %s", eventID))
		return
	}
	runs, err := a.GetEventRuns(ctx, parsed)
	if err != nil {
		_ = publicerr.WriteHTTP(w, err)
		return
	}
	_ = WriteCachedResponse(w, runs, 5*time.Second)
}
