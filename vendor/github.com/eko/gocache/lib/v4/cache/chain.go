package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eko/gocache/lib/v4/store"
)

const (
	// ChainType represents the chain cache type as a string value
	ChainType = "chain"
)

type chainKeyValue[T any] struct {
	key       any
	value     T
	ttl       time.Duration
	storeType *string
}

// ChainCache represents the configuration needed by a cache aggregator
type ChainCache[T any] struct {
	caches     []SetterCacheInterface[T]
	setChannel chan *chainKeyValue[T]
}

// NewChain instantiates a new cache aggregator
func NewChain[T any](caches ...SetterCacheInterface[T]) *ChainCache[T] {
	chain := &ChainCache[T]{
		caches:     caches,
		setChannel: make(chan *chainKeyValue[T], 10000),
	}

	go chain.setter()

	return chain
}

// setter sets a value in available caches, until a given cache layer
func (c *ChainCache[T]) setter() {
	for item := range c.setChannel {
		for _, cache := range c.caches {
			if item.storeType != nil && *item.storeType == cache.GetCodec().GetStore().GetType() {
				break
			}

			cache.Set(context.Background(), item.key, item.value, store.WithExpiration(item.ttl))
		}
	}
}

// Get returns the object stored in cache if it exists
func (c *ChainCache[T]) Get(ctx context.Context, key any) (T, error) {
	var object T
	var err error
	var ttl time.Duration

	for _, cache := range c.caches {
		storeType := cache.GetCodec().GetStore().GetType()
		object, ttl, err = cache.GetWithTTL(ctx, key)
		if err == nil {
			// Set the value back until this cache layer
			c.setChannel <- &chainKeyValue[T]{key, object, ttl, &storeType}
			return object, nil
		}
	}

	return object, err
}

// Set sets a value in available caches
func (c *ChainCache[T]) Set(ctx context.Context, key any, object T, options ...store.Option) error {
	errs := []error{}
	for _, cache := range c.caches {
		err := cache.Set(ctx, key, object, options...)
		if err != nil {
			storeType := cache.GetCodec().GetStore().GetType()
			errs = append(errs, fmt.Errorf("Unable to set item into cache with store '%s': %v", storeType, err))
		}
	}
	if len(errs) > 0 {
		errStr := ""
		for k, v := range errs {
			errStr += fmt.Sprintf("error %d of %d: %v", k+1, len(errs), v.Error())
		}
		return errors.New(errStr)
	}

	return nil
}

// Delete removes a value from all available caches
func (c *ChainCache[T]) Delete(ctx context.Context, key any) error {
	for _, cache := range c.caches {
		cache.Delete(ctx, key)
	}

	return nil
}

// Invalidate invalidates cache item from given options
func (c *ChainCache[T]) Invalidate(ctx context.Context, options ...store.InvalidateOption) error {
	for _, cache := range c.caches {
		cache.Invalidate(ctx, options...)
	}

	return nil
}

// Clear resets all cache data
func (c *ChainCache[T]) Clear(ctx context.Context) error {
	for _, cache := range c.caches {
		cache.Clear(ctx)
	}

	return nil
}

// GetCaches returns all Chained caches
func (c *ChainCache[T]) GetCaches() []SetterCacheInterface[T] {
	return c.caches
}

// GetType returns the cache type
func (c *ChainCache[T]) GetType() string {
	return ChainType
}
