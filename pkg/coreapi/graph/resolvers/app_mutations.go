package resolvers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/deploy"
)

func (r *mutationResolver) CreateApp(ctx context.Context, input models.CreateAppInput) (*cqrs.App, error) {
	// If we already have the app, return it.
	if app, err := r.Data.GetAppByURL(ctx, input.URL); err == nil && app != nil {
		return app, nil
	}

	// Create a new app which holds the error message.
	params := cqrs.InsertAppParams{
		ID:  uuid.New(),
		Url: input.URL,
		Error: sql.NullString{
			Valid:  true,
			String: deploy.DeployErrUnreachable.Error(),
		},
	}
	app, _ := r.Data.InsertApp(ctx, params)

	if err := deploy.Ping(ctx, input.URL); err != nil {
		return app, err
	}

	<-time.After(100 * time.Millisecond)
	apps, err := r.Data.GetAllApps(ctx)
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		if app.Url == input.URL {
			return app, nil
		}
	}
	return nil, fmt.Errorf("There was an error creating your app")
}

func (r *mutationResolver) UpdateApp(ctx context.Context, input models.UpdateAppInput) (*cqrs.App, error) {
	return r.Data.UpdateAppURL(ctx, cqrs.UpdateAppURLParams{
		ID:  uuid.MustParse(input.ID),
		Url: input.URL,
	})
}

func (r *mutationResolver) DeleteApp(ctx context.Context, idstr string) (string, error) {
	id, err := uuid.Parse(idstr)
	if err != nil {
		return "", err
	}
	if err = r.Data.DeleteApp(ctx, id); err != nil {
		return "", err
	}
	return idstr, nil
}

func (r *mutationResolver) DeleteAppByName(
	ctx context.Context,
	name string,
) (bool, error) {
	apps, err := r.Data.GetApps(ctx)
	if err != nil {
		return false, err
	}

	for _, app := range apps {
		if app.Name == name {
			return true, r.Data.DeleteApp(ctx, app.ID)
		}
	}

	return false, nil
}
