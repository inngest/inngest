package resolvers

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (r *queryResolver) Stream(ctx context.Context, q models.StreamQuery) ([]*models.StreamItem, error) {
	var before *ulid.ULID
	var after *ulid.ULID

	if q.Before != nil {
		val := ulid.MustParse(*q.Before)
		before = &val
	}

	if q.After != nil {
		val := ulid.MustParse(*q.After)
		after = &val
	}

	bound := cqrs.IDBound{
		Before: before,
		After:  after,
	}

	includeInternalEvents := false
	if q.IncludeInternalEvents != nil {
		includeInternalEvents = *q.IncludeInternalEvents
	}

	evts, err := r.Data.GetEventsIDbound(
		ctx,
		bound,
		q.Limit,
		includeInternalEvents,
	)
	if err != nil {
		return nil, err
	}

	ids := make([]ulid.ULID, len(evts))
	for n, evt := range evts {
		ids[n] = evt.InternalID()
	}

	// Values don't matter in the Dev Server
	accountID := uuid.New()
	workspaceID := uuid.New()

	fns, err := r.HistoryReader.GetFunctionRunsFromEvents(
		ctx,
		accountID,
		workspaceID,
		ids,
	)
	if err != nil {
		return nil, err
	}
	fnsByID := map[ulid.ULID][]*models.FunctionRun{}
	for _, fn := range fns {
		run := models.MakeFunctionRun(fn)
		_, err := r.Data.GetFunctionByInternalUUID(ctx, consts.DevServerEnvId, uuid.MustParse(run.FunctionID))
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

		runs := fnsByID[i.ID]
		if len(runs) > 0 {
			// If any of the runs is a cron, then the stream item is a cron
			for _, run := range runs {
				if run.Cron != nil && *run.Cron != "" {
					items[n].Trigger = *run.Cron
					items[n].Type = models.StreamTypeCron
					break
				}
			}

			items[n].Runs = runs
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	if len(items) > q.Limit {
		return items[0:q.Limit], nil
	}

	return items, nil
}
