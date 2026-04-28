package resolvers

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/inngest"
)

func (r *functionResolver) App(ctx context.Context, obj *models.Function) (*cqrs.App, error) {
	appID := uuid.MustParse(obj.AppID)
	return r.Data.GetAppByID(ctx, appID)
}

func (qr *queryResolver) FunctionBySlug(ctx context.Context, query models.FunctionQuery) (*models.Function, error) {
	fn, err := qr.Data.GetFunctionByExternalID(ctx, consts.DevServerEnvID, "local", query.FunctionSlug)
	if err != nil {
		return nil, err
	}
	return models.MakeFunction(fn)
}

func (r *functionResolver) FailureHandler(ctx context.Context, obj *models.Function) (*models.Function, error) {
	slug := inngest.GetFailureHandlerSlug(obj.Slug)

	failureFn, err := r.Data.GetFunctionByExternalID(ctx, consts.DevServerEnvID, "local", slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return models.MakeFunction(failureFn)
}
