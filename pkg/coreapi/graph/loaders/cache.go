package loader

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// LookupCache provides per-request caching with singleflight deduplication
// for frequently looked-up entities. When 100 concurrent goroutines request
// the same key, only one DB call is made and all others wait for the result.
type LookupCache struct {
	mu      sync.Mutex
	data    map[string]interface{}
	flights map[string]*flight
}

type flight struct {
	done chan struct{}
	val  interface{}
	err  error
}

type lookupCacheKey struct{}

func WithLookupCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, lookupCacheKey{}, &LookupCache{
		data:    make(map[string]interface{}),
		flights: make(map[string]*flight),
	})
}

func GetLookupCache(ctx context.Context) *LookupCache {
	if c, ok := ctx.Value(lookupCacheKey{}).(*LookupCache); ok {
		return c
	}
	return nil
}

// GetOrLoad returns a cached value or calls the loader function exactly once
// per key, even under concurrent access. All concurrent callers for the same
// key block until the single loader completes.
func (c *LookupCache) GetOrLoad(kind string, id uuid.UUID, loader func() (interface{}, error)) (interface{}, error) {
	key := kind + ":" + id.String()

	c.mu.Lock()
	// Check cache first
	if v, ok := c.data[key]; ok {
		c.mu.Unlock()
		return v, nil
	}
	// Check if there's an in-flight request
	if f, ok := c.flights[key]; ok {
		c.mu.Unlock()
		<-f.done
		return f.val, f.err
	}
	// Start a new flight
	f := &flight{done: make(chan struct{})}
	c.flights[key] = f
	c.mu.Unlock()

	// Execute the loader
	f.val, f.err = loader()
	if f.err == nil {
		c.mu.Lock()
		c.data[key] = f.val
		c.mu.Unlock()
	}

	// Unblock all waiters
	close(f.done)

	// Clean up flight
	c.mu.Lock()
	delete(c.flights, key)
	c.mu.Unlock()

	return f.val, f.err
}
