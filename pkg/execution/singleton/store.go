package singleton

import (
	"context"

	"github.com/oklog/ulid/v2"
)

type SingletonStore interface {
	GetCurrentRunID(ctx context.Context, key string) (*ulid.ULID, error)
	ReleaseSingleton(ctx context.Context, key string) (*ulid.ULID, error)
}
