package resolvers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/run"
	"go.opentelemetry.io/otel/attribute"
)

func (r *mutationResolver) CreateApp(ctx context.Context, input models.CreateAppInput) (*cqrs.App, error) {
	// URLs must contain a protocol. If not, add http since very few apps use
	// https during development
	if !strings.Contains(input.URL, "://") {
		input.URL = "http://" + input.URL
	}

	// Create a new app which holds the error message.
	params := cqrs.UpsertAppParams{
		ID:  inngest.DeterministicAppUUID(input.URL),
		Url: input.URL,
		Error: sql.NullString{
			Valid:  true,
			String: deploy.DeployErrUnreachable.Error(),
		},
	}
	app, _ := r.Data.UpsertApp(ctx, params)

	if res := deploy.Ping(ctx, input.URL, r.ServerKind, r.LocalSigningKey, r.RequireKeys); res.Err != nil {
		return app, res.Err
	}

	<-time.After(100 * time.Millisecond)
	apps, err := r.Data.GetAllApps(ctx, consts.DevServerEnvId)
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
	apps, err := r.Data.GetApps(ctx, consts.DevServerEnvId)
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

func (r *mutationResolver) InvokeFunction(
	ctx context.Context,
	data map[string]any,
	functionSlug string,
	user map[string]any,
) (*bool, error) {
	evt := event.NewInvocationEvent(event.NewInvocationEventOpts{
		Event: event.Event{
			Data: data,
			User: user,
		},
		FnID: functionSlug,
	})

	ctx, span := run.NewSpan(ctx,
		run.WithName(consts.OtelSpanInvoke),
		run.WithScope(consts.OtelScopeInvoke),
		run.WithNewRoot(),
		run.WithSpanAttributes(
			attribute.String(consts.OtelSysFunctionSlug, functionSlug),
		),
	)
	defer span.End()

	sent := false
	_, err := r.EventHandler(ctx, &evt)
	if err != nil {
		return &sent, err
	}

	sent = true
	return &sent, nil
}
