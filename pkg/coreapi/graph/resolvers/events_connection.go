package resolvers

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
)

func (r *eventsConnectionResolver) TotalCount(
	ctx context.Context,
	conn *models.EventsConnection,
) (int, error) {
	filter := graphql.GetFieldContext(ctx).Parent.Args["filter"].(models.EventsFilter)

	opts := cqrs.WorkspaceEventsOpts{
		Limit:                 cqrs.MaxEvents, // pass in dummy value to pass validation, but won't be used in actual count query
		Names:                 filter.EventNames,
		IncludeInternalEvents: filter.IncludeInternalEvents,
		Oldest:                filter.From,
	}

	// this relies on the frontend not requesting TotalCount except on loading the first page
	opts.Newest = time.Now()
	if filter.Until != nil {
		opts.Newest = *filter.Until
	}

	totalCount, err := r.Data.GetEventsCount(ctx, consts.DevServerAccountID, consts.DevServerEnvID, opts)

	if err != nil {
		return 0, err
	}
	return int(totalCount), nil
}
