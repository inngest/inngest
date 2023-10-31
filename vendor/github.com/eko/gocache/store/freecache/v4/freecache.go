package freecache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	lib_store "github.com/eko/gocache/lib/v4/store"
)

const (
	// FreecacheType represents the storage type as a string value
	FreecacheType = "freecache"
	// FreecacheTagPattern represents the tag pattern to be used as a key in specified storage
	FreecacheTagPattern = "freecache_tag_%s"
)

// FreecacheClientInterface represents a coocood/freecache client
type FreecacheClientInterface interface {
	Get(key []byte) (value []byte, err error)
	GetInt(key int64) (value []byte, err error)
	TTL(key []byte) (timeLeft uint32, err error)
	Set(key, value []byte, expireSeconds int) (err error)
	SetInt(key int64, value []byte, expireSeconds int) (err error)
	Del(key []byte) (affected bool)
	DelInt(key int64) (affected bool)
	Clear()
}

// FreecacheStore is a store for freecache
type FreecacheStore struct {
	client  FreecacheClientInterface
	options *lib_store.Options
}

// NewFreecache creates a new store to freecache instance(s)
func NewFreecache(client FreecacheClientInterface, options ...lib_store.Option) *FreecacheStore {
	return &FreecacheStore{
		client:  client,
		options: lib_store.ApplyOptions(options...),
	}
}

// Get returns data stored from a given key. It returns the value or not found error
func (f *FreecacheStore) Get(_ context.Context, key any) (any, error) {
	var err error
	var result any
	if k, ok := key.(string); ok {
		result, err = f.client.Get([]byte(k))
		if err != nil {
			return nil, lib_store.NotFoundWithCause(errors.New("value not found in Freecache store"))
		}
		return result, err
	}

	return nil, errors.New("key type not supported by Freecache store")
}

// GetWithTTL returns data stored from a given key and its corresponding TTL
func (f *FreecacheStore) GetWithTTL(_ context.Context, key any) (any, time.Duration, error) {
	if k, ok := key.(string); ok {
		result, err := f.client.Get([]byte(k))
		if err != nil {
			return nil, 0, lib_store.NotFoundWithCause(errors.New("value not found in Freecache store"))
		}

		ttl, err := f.client.TTL([]byte(k))
		if err != nil {
			return nil, 0, lib_store.NotFoundWithCause(errors.New("value not found in Freecache store"))
		}

		return result, time.Duration(ttl) * time.Second, err
	}

	return nil, 0, errors.New("key type not supported by Freecache store")
}

// Set sets a key, value and expiration for a cache entry and stores it in the cache.
// If the key is larger than 65535 or value is larger than 1/1024 of the cache size,
// the entry will not be written to the cache. expireSeconds <= 0 means no expire,
// but it can be evicted when cache is full.
func (f *FreecacheStore) Set(ctx context.Context, key any, value any, options ...lib_store.Option) error {
	var err error
	var val []byte

	// Using default options set during cache initialization
	opts := lib_store.ApplyOptionsWithDefault(f.options, options...)

	// type check for value, as freecache only supports value of type []byte
	switch v := value.(type) {
	case []byte:
		val = v
	default:
		return errors.New("value type not supported by Freecache store")
	}

	if k, ok := key.(string); ok {
		err = f.client.Set([]byte(k), val, int(opts.Expiration.Seconds()))
		if err != nil {
			return fmt.Errorf("size of key: %v, value: %v, err: %v", k, len(val), err)
		}
		if tags := opts.Tags; len(tags) > 0 {
			f.setTags(ctx, key, tags)
		}
		return nil
	}
	return errors.New("key type not supported by Freecache store")
}

func (f *FreecacheStore) setTags(ctx context.Context, key any, tags []string) {
	for _, tag := range tags {
		tagKey := fmt.Sprintf(FreecacheTagPattern, tag)
		cacheKeys := f.getCacheKeysForTag(ctx, tagKey)

		alreadyInserted := false
		for _, cacheKey := range cacheKeys {
			if cacheKey == key.(string) {
				alreadyInserted = true
				break
			}
		}

		if !alreadyInserted {
			cacheKeys = append(cacheKeys, key.(string))
		}

		f.Set(ctx, tagKey, []byte(strings.Join(cacheKeys, ",")), lib_store.WithExpiration(720*time.Hour))
	}
}

func (f *FreecacheStore) getCacheKeysForTag(ctx context.Context, tagKey string) []string {
	cacheKeys := []string{}
	if result, err := f.Get(ctx, tagKey); err == nil && result != nil {
		if str, ok := result.([]byte); ok {
			cacheKeys = strings.Split(string(str), ",")
		}
	}
	return cacheKeys
}

// Delete deletes an item in the cache by key and returns err or nil if a delete occurred
func (f *FreecacheStore) Delete(_ context.Context, key any) error {
	if v, ok := key.(string); ok {
		if f.client.Del([]byte(v)) {
			return nil
		}
		return fmt.Errorf("failed to delete key %v", key)
	}
	return errors.New("key type not supported by Freecache store")
}

// Invalidate invalidates some cache data in freecache for given options
func (f *FreecacheStore) Invalidate(ctx context.Context, options ...lib_store.InvalidateOption) error {
	opts := lib_store.ApplyInvalidateOptions(options...)

	if tags := opts.Tags; len(tags) > 0 {
		for _, tag := range tags {
			tagKey := fmt.Sprintf(FreecacheTagPattern, tag)
			cacheKeys := f.getCacheKeysForTag(ctx, tagKey)

			for _, cacheKey := range cacheKeys {
				err := f.Delete(ctx, cacheKey)
				if err != nil {
					return err
				}
			}

			err := f.Delete(ctx, tagKey)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Clear resets all data in the store
func (f *FreecacheStore) Clear(_ context.Context) error {
	f.client.Clear()
	return nil
}

// GetType returns the store type
func (f *FreecacheStore) GetType() string {
	return FreecacheType
}
