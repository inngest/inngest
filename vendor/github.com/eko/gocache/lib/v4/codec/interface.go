package codec

import (
	"context"
	"time"

	"github.com/eko/gocache/lib/v4/store"
)

// CodecInterface represents an instance of a cache codec
type CodecInterface interface {
	Get(ctx context.Context, key any) (any, error)
	GetWithTTL(ctx context.Context, key any) (any, time.Duration, error)
	Set(ctx context.Context, key any, value any, options ...store.Option) error
	Delete(ctx context.Context, key any) error
	Invalidate(ctx context.Context, options ...store.InvalidateOption) error
	Clear(ctx context.Context) error

	GetStore() store.StoreInterface
	GetStats() *Stats
}
