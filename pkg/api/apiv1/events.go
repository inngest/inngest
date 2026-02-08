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
	"github.com/inngest/inngest/pkg/enums"
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
func (a API) GetEvents(ctx context.Context, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		return nil, publicerr.Wrap(err, 401, "No auth found")
	}

	if a.opts.TraceReader == nil {
		return nil, publicerr.Errorf(500, "No trace reader specified")
	}

	events, err := a.opts.TraceReader.GetEvents(ctx, auth.AccountID(), auth.WorkspaceID(), opts)
	if err != nil {
		logger.StdlibLogger(ctx).Error("error querying events", "error", err)
		return nil, publicerr.Wrap(err, 500, "Unable to query events")
	}
	return events, nil
}

func (a router) getEvents(w http.ResponseWriter, r *http.Request) {
	if a.opts.RateLimited(r, w, "/v1/events") {
		return
	}

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
		opts.Names = []string{name}
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
	if a.opts.TraceReader == nil {
		return nil, publicerr.Errorf(500, "No trace reader specified")
	}
	event, err := a.opts.TraceReader.GetEvent(ctx, eventID, auth.AccountID(), auth.WorkspaceID())
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
	if a.opts.RateLimited(r, w, "/v1/events/{eventID}") {
		return
	}

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

	result, err := a.opts.TraceReader.GetEventRuns(ctx, eventID, auth.AccountID(), auth.WorkspaceID())
	if err != nil {
		return nil, publicerr.Wrap(err, 500, "Unable to query event runs")
	}
	return result, nil
}

func (a router) getEventRuns(w http.ResponseWriter, r *http.Request) {
	if a.opts.RateLimited(r, w, "/v1/events/{eventID}/runs") {
		return
	}

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

	// XXX (tonyhb, 2025-10-17): Moving to the new trace pipeline means that we're going to query
	// for the runs found from each event, then fetch the status directly.
	{
		for _, run := range runs {
			rootSpan, err := a.opts.TraceReader.GetSpansByRunID(ctx, run.RunID)
			if err != nil {
				_ = publicerr.WriteHTTP(w, err) // return with error since user can leave out trace_preview flag
				return
			}
			if rootSpan == nil {
				continue
			}
			run.Status = enums.StepStatusToRunStatus(rootSpan.Status)
		}
	}

	_ = WriteCachedResponse(w, runs, 15*time.Second)
}
