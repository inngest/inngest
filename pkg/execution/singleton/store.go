package singleton

import (
	"context"
)

type SingletonStore interface {
	GetCurrentRunID(ctx context.Context, key string) (*string, error)
	ReleaseSingleton(ctx context.Context, key string) (*string, error)
}
