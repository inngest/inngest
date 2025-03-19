package apiv2

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
)

type envSvc interface {
	ArchiveEnv(ctx context.Context, id uuid.UUID) error
	GetEnvByName(ctx context.Context, envName string) (*cqrs.Env, error)
	UnarchiveEnv(ctx context.Context, id uuid.UUID) error
}

func (a api) ArchiveEnv(
	w http.ResponseWriter,
	r *http.Request,
	envName string,
) {
	ctx := r.Context()

	env, err := a.opts.EnvSvc.GetEnvByName(ctx, envName)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	err = a.opts.EnvSvc.ArchiveEnv(ctx, env.ID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}
	writeEmpty(w)
}

func (a api) GetEnv(
	w http.ResponseWriter,
	r *http.Request,
	envName string,
) {
	ctx := r.Context()

	env, err := a.opts.EnvSvc.GetEnvByName(ctx, envName)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}
	json.NewEncoder(w).Encode(Env{
		ArchivedAt:         env.ArchivedAt,
		AutoArchiveEnabled: env.AutoArchiveEnabled,
		CreatedAt:          env.CreatedAt,
		EnvType:            env.EnvType,
		InternalID:         env.ID,
		Name:               env.Name,
		Slug:               env.Slug,
	})
}

func (a api) UnarchiveEnv(
	w http.ResponseWriter,
	r *http.Request,
	envName string,
) {
	ctx := r.Context()

	env, err := a.opts.EnvSvc.GetEnvByName(ctx, envName)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}

	err = a.opts.EnvSvc.UnarchiveEnv(ctx, env.ID)
	if err != nil {
		writeFailBody(ctx, w, err)
		return
	}
	writeEmpty(w)
}
