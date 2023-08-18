package resolvers

import (
	"context"
	"sort"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
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

	// Query all function runs received, and filter by crons.
	fns, err := r.Data.GetFunctionRunsTimebound(ctx, tb, q.Limit)
	if err != nil {
		return nil, err
	}

	items := make([]*models.StreamItem, len(evts))
	for n, i := range evts {
		items[n] = &models.StreamItem{
			ID:        i.ID.String(),
			Trigger:   i.EventName,
			Type:      models.StreamTypeEvent,
			CreatedAt: time.UnixMilli(i.EventTS),
		}

		// XXX: Optimize this
	}

	for _, i := range fns {
		if i.TriggerType != "cron" {
			// These are children of events.
			continue
		}

		items = append(items, &models.StreamItem{
			ID:        i.RunID.String(),
			Trigger:   "", // TODO: Load
			Type:      models.StreamTypeCron,
			CreatedAt: i.RunStartedAt,
			Runs: []*models.FunctionRun{
				{
					ID: i.RunID.String(),
					// TODO:...
				},
			},
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
