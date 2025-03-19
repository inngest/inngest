package cqrs

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Env struct {
	ArchivedAt         *time.Time
	AutoArchiveEnabled bool
	CreatedAt          time.Time
	EnvType            string
	ID                 uuid.UUID
	Name               string
	Slug               string
}

type EnvManager interface {
	EnvReader
	EnvWriter
}

type EnvReader interface {
	GetEnvByName(ctx context.Context, name string) (*Env, error)
}

type EnvWriter interface {
	ArchiveEnv(ctx context.Context, id uuid.UUID) error
	UnarchiveEnv(ctx context.Context, id uuid.UUID) error
}
