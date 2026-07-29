package devserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
)

type cqrsAppProvider struct {
	reader appProviderReader
}

type appProviderReader interface {
	cqrs.AppReader
	cqrs.FunctionV2Reader
}

// NewAppProvider returns an AppProvider that looks up apps by external ID or
// UUID using the given AppReader.
func NewAppProvider(reader appProviderReader) apiv2.AppProvider {
	return &cqrsAppProvider{reader: reader}
}

func (p *cqrsAppProvider) GetApp(ctx context.Context, identifier string) (apiv2.App, error) {
	if appID, err := uuid.Parse(identifier); err == nil {
		if app, err := p.reader.GetAppByID(ctx, appID); err == nil {
			return p.toApp(ctx, app)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return apiv2.App{}, err
		}
	}

	app, err := p.reader.GetAppByName(ctx, consts.DevServerEnvID, identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiv2.App{}, fmt.Errorf("%w: %s", apiv2.ErrAppNotFound, identifier)
		}
		return apiv2.App{}, err
	}
	if app == nil {
		return apiv2.App{}, fmt.Errorf("%w: %s", apiv2.ErrAppNotFound, identifier)
	}
	return p.toApp(ctx, app)
}

func (p *cqrsAppProvider) toApp(ctx context.Context, app *cqrs.App) (apiv2.App, error) {
	fns, err := p.reader.GetFunctionsByApp(ctx, cqrs.GetFunctionsByAppOpts{
		AppID: app.ID,
	})
	if err != nil {
		return apiv2.App{}, err
	}

	//
	// the dev server stores the user-defined app ID as the app name
	result := apiv2.App{
		ID:            app.Name,
		InternalID:    app.ID,
		Name:          app.Name,
		AppVersion:    app.AppVersion,
		CreatedAt:     app.CreatedAt,
		ArchivedAt:    app.DeletedAt,
		FunctionCount: len(fns),
		LatestSync: &apiv2.AppSync{
			SdkLanguage: app.SdkLanguage,
			SdkVersion:  app.SdkVersion,
			URL:         app.Url,
			AppVersion:  app.AppVersion,
		},
	}
	if app.Framework.Valid {
		result.LatestSync.Framework = app.Framework.String
	}
	if app.Error.Valid {
		result.LatestSync.Error = app.Error.String
	}
	if method, err := enums.AppMethodString(app.Method); err == nil {
		result.Method = method
	}
	return result, nil
}
