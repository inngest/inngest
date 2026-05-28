package resolvers

import (
	"context"
	"encoding/json"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

func (qr *queryResolver) EventV2(ctx context.Context, id ulid.ULID) (*models.EventV2, error) {
	targetLoader := loader.FromCtx(ctx).EventLoader

	event, err := loader.LoadOneWithString[cqrs.Event](
		ctx,
		targetLoader,
		id.String(),
	)

	if err != nil {
		return nil, err
	}

	return cqrsEventToGQLEvent(event), nil
}

func (er eventV2Resolver) Runs(ctx context.Context, obj *models.EventV2) ([]*models.FunctionRunV2, error) {
	// convert cqrs TraceRun to FunctionRunV2

	// This is an N+1, currently also an N+1 on cloud in the form of multiple calls to the metrics service
	// we cannot currently use a data loader due to current db schema and https://github.com/sqlc-dev/sqlc/issues/1830
	traceRuns, err := er.Data.GetTraceRunsByTriggerID(ctx, obj.ID)
	if err != nil {
		return nil, err
	}

	functionRuns := make([]*models.FunctionRunV2, 0, len(traceRuns))
	for _, r := range traceRuns {
		fr, err := models.MakeFunctionRunV2(r)
		if err != nil {
			continue
		}
		functionRuns = append(functionRuns, fr)
	}
	return functionRuns, nil
}

func (er eventV2Resolver) Raw(ctx context.Context, obj *models.EventV2) (string, error) {
	targetLoader := loader.FromCtx(ctx).EventLoader

	event, err := loader.LoadOneWithString[cqrs.Event](
		ctx,
		targetLoader,
		obj.ID.String(),
	)
	if err != nil {
		return "", err
	}

	raw, err := marshalRaw(event)
	if err != nil {
		return "", err
	}

	return raw, nil
}

func marshalRaw(e *cqrs.Event) (string, error) {
	data := e.EventData
	if data == nil {
		data = make(map[string]any)
	}

	var version *string
	if len(e.EventVersion) > 0 {
		version = &e.EventVersion
	}

	id := e.InternalID().String()
	if len(e.EventID) > 0 {
		id = e.EventID
	}

	byt, err := json.Marshal(map[string]any{
		"data": data,
		"id":   id,
		"name": e.EventName,
		"ts":   e.EventTS,
		"v":    version,
	})
	if err != nil {
		return "", err
	}
	return string(byt), nil
}
