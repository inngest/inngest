package singleton

import (
	"context"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type SingletonStore interface {
	GetCurrentRunID(ctx context.Context, key string, accountID uuid.UUID) (*ulid.ULID, error)
	ReleaseSingleton(ctx context.Context, key string, accountID uuid.UUID) (*ulid.ULID, error)
}
