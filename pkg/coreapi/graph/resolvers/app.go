package resolvers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/devserver/discovery"
)

func (qr queryResolver) Apps(ctx context.Context, filter *models.AppsFilterV1) ([]*cqrs.App, error) {
	cqrsFilter, err := models.FromAppsFilter(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse filter: %w", err)
	}
	return qr.Data.GetApps(ctx, consts.DevServerEnvID, cqrsFilter)
}

func (qr queryResolver) App(ctx context.Context, id uuid.UUID) (*cqrs.App, error) {
	return qr.Data.GetAppByID(ctx, id)
}

func (a appResolver) ID(ctx context.Context, obj *cqrs.App) (string, error) {
	if obj == nil {
		return "", fmt.Errorf("no app defined")
	}
	return obj.ID.String(), nil
}

func (a appResolver) ExternalID(ctx context.Context, obj *cqrs.App) (string, error) {
	if obj == nil {
		return "", fmt.Errorf("no app defined")
	}

	// Name is currently the same as external ID, but we'll eventually allow
	// apps to have names (similar to functions)
	return obj.Name, nil
}

func (a appResolver) Framework(ctx context.Context, obj *cqrs.App) (*string, error) {
	if obj == nil {
		return nil, fmt.Errorf("no app defined")
	}
	if obj.Framework.Valid {
		return &obj.Framework.String, nil
	}
	return nil, nil
}

func (a appResolver) Error(ctx context.Context, obj *cqrs.App) (*string, error) {
	if obj == nil {
		return nil, fmt.Errorf("no app defined")
	}
	if obj.Error.Valid {
		return &obj.Error.String, nil
	}
	return nil, nil
}

func (a appResolver) Functions(ctx context.Context, obj *cqrs.App) ([]*models.Function, error) {
	if obj == nil {
		return nil, fmt.Errorf("no app defined")
	}
	// Local dev doesn't have a workspace ID.
	funcs, err := a.Data.GetFunctionsByAppInternalID(ctx, obj.ID)
	if err != nil {
		return nil, err
	}
	res := make([]*models.Function, len(funcs))
	for n, f := range funcs {
		res[n], err = models.MakeFunction(f)
		if err != nil {
			return nil, err
		}

	}
	return res, nil
}

func (a appResolver) Connected(ctx context.Context, obj *cqrs.App) (bool, error) {
	return !obj.Error.Valid, nil
}

func (a appResolver) Autodiscovered(ctx context.Context, obj *cqrs.App) (bool, error) {
	urls := discovery.URLs()
	_, ok := urls[obj.Url]
	return ok, nil
}

func (a appResolver) FunctionCount(ctx context.Context, obj *cqrs.App) (int, error) {
	funcs, err := a.Data.GetFunctionsByAppInternalID(ctx, obj.ID)
	if err != nil {
		return 0, err
	}
	return len(funcs), nil
}

func (a appResolver) Method(ctx context.Context, obj *cqrs.App) (models.AppMethod, error) {
	method, err := enums.AppMethodString(obj.Method)
	if err != nil {
		return models.AppMethodServe, fmt.Errorf("unknown connection type")
	}

	return models.ToAppMethod(method), nil
}
