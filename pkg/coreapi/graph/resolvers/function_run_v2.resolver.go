package resolvers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
)

func (r *functionRunV2Resolver) Function(ctx context.Context, fn *models.FunctionRunV2) (*models.Function, error) {
	fun, err := r.Data.GetFunctionByInternalUUID(ctx, uuid.UUID{}, fn.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving function: %w", err)
	}

	return models.MakeFunction(fun)
}

func (r *functionRunV2Resolver) Triggers(ctx context.Context, fn *models.FunctionRunV2) ([]string, error) {
	events, err := r.Data.GetEventsByInternalIDs(ctx, fn.TriggerIDs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving triggers: %w", err)
	}

	res := []string{}
	for _, evt := range events {
		byt, err := json.Marshal(evt.GetEvent())
		if err != nil {
			return nil, fmt.Errorf("invalid event format: %w", err)
		}

		sevt := string(byt)
		if sevt != "" {
			res = append(res, string(byt))
		}
	}

	return res, nil
}

func (r *functionRunV2Resolver) Trace(ctx context.Context, fn *models.FunctionRunV2) (*models.RunTraceSpan, error) {
	// TODO: handle the case when it's Scheduled
	// there's no run so it should return empty but not error

	return loader.LoadOne[models.RunTraceSpan](
		ctx,
		loader.FromCtx(ctx).RunTraceLoader,
		&loader.TraceRequestKey{
			TraceRunIdentifier: &cqrs.TraceRunIdentifier{
				AppID:      fn.AppID,
				FunctionID: fn.FunctionID,
				RunID:      fn.ID,
				TraceID:    fn.TraceID,
			},
		},
	)
}
