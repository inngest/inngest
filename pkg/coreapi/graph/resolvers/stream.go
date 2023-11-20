package resolvers

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (r *queryResolver) Stream(ctx context.Context, q models.StreamQuery) ([]*models.StreamItem, error) {
	tb := cqrs.Timebound{
		Before: q.Before,
		After:  q.After,
	}

	evts, err := r.Data.GetEventsTimebound(ctx, tb, q.Limit)
	if err != nil {
		return nil, err
	}

	ids := make([]ulid.ULID, len(evts))
	for n, evt := range evts {
		ids[n] = evt.InternalID()
	}

	// Fetch all function runs by event
	fns, err := r.Data.GetFunctionRunsFromEvents(
		ctx,
		uuid.UUID{},
		uuid.UUID{},
		ids,
	)
	if err != nil {
		return nil, err
	}
	fnsByID := map[ulid.ULID][]*models.FunctionRun{}
	for _, fn := range fns {
		run := models.MakeFunctionRun(fn)
		_, err := r.Data.GetFunctionByID(ctx, uuid.MustParse(run.FunctionID))
		if err == sql.ErrNoRows {
			// Skip run since its function doesn't exist. This can happen when
			// deleting a function or changing its ID.
			continue
		}

		fnsByID[fn.EventID] = append(fnsByID[fn.EventID], run)
	}

	items := make([]*models.StreamItem, len(evts))
	for n, i := range evts {
		items[n] = &models.StreamItem{
			ID:        i.ID.String(),
			Trigger:   i.EventName,
			Type:      models.StreamTypeEvent,
			CreatedAt: time.UnixMilli(i.EventTS),
			Runs:      []*models.FunctionRun{},
		}
		if len(fnsByID[i.ID]) > 0 {
			items[n].Runs = fnsByID[i.ID]
		}
	}

	// Query all function runs received, and filter by crons.
	fns, err = r.Data.GetFunctionRunsTimebound(ctx, tb, q.Limit)
	if err != nil {
		return nil, err
	}
	for _, i := range fns {
		if i.Cron == nil {
			// These are children of events.
			continue
		}

		var trigger string
		if i.Cron != nil {
			trigger = *i.Cron
		}

		runs := []*models.FunctionRun{models.MakeFunctionRun(i)}
		_, err := r.Data.GetFunctionByID(ctx, uuid.MustParse(runs[0].FunctionID))
		if err == sql.ErrNoRows {
			// Skip run since its function doesn't exist. This can happen when
			// deleting a function or changing its ID.
			runs = []*models.FunctionRun{}
		}

		items = append(items, &models.StreamItem{
			ID:        i.RunID.String(),
			Trigger:   trigger,
			Type:      models.StreamTypeCron,
			CreatedAt: i.RunStartedAt,
			Runs:      runs,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	if len(items) > q.Limit {
		return items[0:q.Limit], nil
	}

	return items, nil
}
