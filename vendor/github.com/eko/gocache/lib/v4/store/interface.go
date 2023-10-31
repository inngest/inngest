package store

import (
	"context"
	"time"
)

// StoreInterface is the interface for all available stores
type StoreInterface interface {
	Get(ctx context.Context, key any) (any, error)
	GetWithTTL(ctx context.Context, key any) (any, time.Duration, error)
	Set(ctx context.Context, key any, value any, options ...Option) error
	Delete(ctx context.Context, key any) error
	Invalidate(ctx context.Context, options ...InvalidateOption) error
	Clear(ctx context.Context) error
	GetType() string
}
