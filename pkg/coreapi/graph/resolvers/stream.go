package resolvers

import (
	"context"
	"sort"
	"time"

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
	fns, err := r.Data.GetFunctionRunsFromEvents(ctx, ids)
	if err != nil {
		return nil, err
	}
	fnsByID := map[ulid.ULID][]*models.FunctionRun{}
	for _, fn := range fns {
		fnsByID[fn.EventID] = append(fnsByID[fn.EventID], models.MakeFunctionRun(fn))
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
		if i.TriggerType != "cron" {
			// These are children of events.
			continue
		}

		var trigger string
		if fn, err := r.Data.GetFunctionByID(ctx, i.FunctionID); err == nil {
			if fn, err := fn.InngestFunction(); err == nil {
				// Should always have at least 1 trigger, but we'll check anyway
				// to avoid a panic just in case.
				if (len(fn.Triggers)) >= 0 {
					// This is a flawed way to get the cron, since this value is
					// always the latest cron schedule. In other words, if a
					// user updates the cron schedule for the function then
					// preexisting runs will have the new cron schedule instead
					// of the one when they ran.
					trigger = fn.Triggers[0].Cron
				}
			}
		}

		items = append(items, &models.StreamItem{
			ID:        i.RunID.String(),
			Trigger:   trigger,
			Type:      models.StreamTypeCron,
			CreatedAt: i.RunStartedAt,
			Runs:      []*models.FunctionRun{models.MakeFunctionRun(i)},
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
