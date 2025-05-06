package resolvers

import (
	"context"

	"github.com/inngest/inngest/pkg/coreapi/generated"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/usage"
	"github.com/pkg/errors"
)

func (r *queryResolver) EventTypes(
	ctx context.Context,
	first int,
	after *string,
	filter cqrs.EventTypesFilter,
) (*cqrs.EventTypesConnection, error) {
	return r.Data.GetEventTypes(ctx, filter)
}

func (r *Resolver) EventType() generated.EventTypeResolver {
	return &eventTypeResolver{r}
}

type eventTypeResolver struct{ *Resolver }

func (r *eventTypeResolver) Functions(
	ctx context.Context,
	obj *cqrs.EventType,
	first int,
	after *string,
) (*models.FunctionsConnection, error) {
	if first <= 0 {
		return nil, errors.New("first must be greater than 0")
	}
	if first > defaultPageSize {
		return nil, errors.New("first must be less than or equal to 40")
	}
	return nil, nil
}

func (r *eventTypeResolver) Usage(
	ctx context.Context,
	obj *cqrs.EventType,
	opts *usage.UsageInput,
) (*usage.UsageResponse, error) {
	if opts == nil {
		return nil, errors.New("input is required")
	}
	if opts.Period == nil {
		return nil, errors.New("period is required")
	}
	if opts.Range != nil {
		return nil, errors.New("range is not supported")
	}

	return nil, nil
}
