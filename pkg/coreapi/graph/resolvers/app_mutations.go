package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/deploy"
)

func (r *mutationResolver) CreateApp(ctx context.Context, input models.CreateAppInput) (*cqrs.App, error) {
	if err := deploy.Ping(ctx, input.URL); err != nil {
		return nil, err
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
