package apiv2

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
)

type appSvc interface {
	DeleteApp(ctx context.Context, appID uuid.UUID) error

	GetAppByName(
		ctx context.Context,
		envInternalID uuid.UUID,
		name string,
	) (*cqrs.App, error)

	GetApps(
		ctx context.Context,
		envID uuid.UUID,
		filter *cqrs.FilterAppParam,
	) ([]*cqrs.App, error)

	UnarchiveApp(ctx context.Context, appID uuid.UUID) error
}

func (a api) ArchiveApp(
	w http.ResponseWriter,
	r *http.Request,
	envID string,
	appID string,
) {
	ctx := r.Context()

	env, err := a.opts.EnvSvc.GetEnvByName(ctx, envID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	app, err := a.opts.AppSvc.GetAppByName(ctx, env.ID, appID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	err = a.opts.AppSvc.DeleteApp(ctx, app.ID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}
	writeEmpty(w)
}

func (a api) GetApp(
	w http.ResponseWriter,
	r *http.Request,
	envID string,
	appID string,
) {
	ctx := r.Context()

	env, err := a.opts.EnvSvc.GetEnvByName(ctx, envID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	app, err := a.opts.AppSvc.GetAppByName(ctx, env.ID, appID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	json.NewEncoder(w).Encode(appFromCQRS(app))
}

func (a api) GetApps(
	w http.ResponseWriter,
	r *http.Request,
	envID string,
) {
	ctx := r.Context()

	env, err := a.opts.EnvSvc.GetEnvByName(ctx, envID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	apps, err := a.opts.AppSvc.GetApps(ctx, env.ID, nil)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	resp := AppsResponse{
		Data: make([]App, len(apps)),
	}
	for i, app := range apps {
		resp.Data[i] = appFromCQRS(app)
	}
	json.NewEncoder(w).Encode(resp)
}

func (a api) UnarchiveApp(
	w http.ResponseWriter,
	r *http.Request,
	envID string,
	appID string,
) {
	ctx := r.Context()

	env, err := a.opts.EnvSvc.GetEnvByName(ctx, envID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	app, err := a.opts.AppSvc.GetAppByName(ctx, env.ID, appID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	err = a.opts.AppSvc.UnarchiveApp(ctx, app.ID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}
	writeEmpty(w)
}
