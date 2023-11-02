package rueidis

import (
	"context"
	"fmt"
	"time"

	lib_store "github.com/eko/gocache/lib/v4/store"
	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidiscompat"
)

const (
	// RueidisType represents the storage type as a string value
	RueidisType = "rueidis"
	// RueidisTagPattern represents the tag pattern to be used as a key in specified storage
	RueidisTagPattern = "gocache_tag_%s"

	defaultClientSideCacheExpiration = 10 * time.Second
)

// RueidisStore is a store for Redis
type RueidisStore struct {
	client  rueidis.Client
	options *lib_store.Options
}

// NewRueidis creates a new store to Redis instance(s)
func NewRueidis(client rueidis.Client, options ...lib_store.Option) *RueidisStore {
	// defaults client side cache expiration to 10s
	appliedOptions := lib_store.ApplyOptions(options...)

	if appliedOptions.ClientSideCacheExpiration == 0 {
		appliedOptions.ClientSideCacheExpiration = defaultClientSideCacheExpiration
	}

	return &RueidisStore{
		client:  client,
		options: appliedOptions,
	}
}

// Get returns data stored from a given key
func (s *RueidisStore) Get(ctx context.Context, key any) (any, error) {
	cmd := s.client.B().Get().Key(key.(string)).Cache()
	res := s.client.DoCache(ctx, cmd, s.options.ClientSideCacheExpiration)
	str, err := res.ToString()
	if rueidis.IsRedisNil(err) {
		err = lib_store.NotFoundWithCause(err)
	}
	return str, err
}

// GetWithTTL returns data stored from a given key and its corresponding TTL
func (s *RueidisStore) GetWithTTL(ctx context.Context, key any) (any, time.Duration, error) {
	cmd := s.client.B().Get().Key(key.(string)).Cache()
	res := s.client.DoCache(ctx, cmd, s.options.ClientSideCacheExpiration)
	str, err := res.ToString()
	if rueidis.IsRedisNil(err) {
		err = lib_store.NotFoundWithCause(err)
	}
	return str, time.Duration(res.CacheTTL()) * time.Second, err
}

// Set defines data in Redis for given key identifier
func (s *RueidisStore) Set(ctx context.Context, key any, value any, options ...lib_store.Option) error {
	opts := lib_store.ApplyOptionsWithDefault(s.options, options...)
	ttl := int64(opts.Expiration.Seconds())
	cmd := s.client.B().Set().Key(key.(string)).Value(value.(string)).ExSeconds(ttl).Build()
	err := s.client.Do(ctx, cmd).Error()
	if err != nil {
		return err
	}

	if tags := opts.Tags; len(tags) > 0 {
		s.setTags(ctx, key, tags)
	}

	return nil
}

func (s *RueidisStore) setTags(ctx context.Context, key any, tags []string) {
	ttl := 720 * time.Hour
	for _, tag := range tags {
		tagKey := fmt.Sprintf(RueidisTagPattern, tag)
		s.client.DoMulti(ctx,
			s.client.B().Sadd().Key(tagKey).Member(key.(string)).Build(),
			s.client.B().Expire().Key(tagKey).Seconds(int64(ttl.Seconds())).Build(),
		)
	}
}

// Delete removes data from Redis for given key identifier
func (s *RueidisStore) Delete(ctx context.Context, key any) error {
	return s.client.Do(ctx, s.client.B().Del().Key(key.(string)).Build()).Error()
}

// Invalidate invalidates some cache data in Redis for given options
func (s *RueidisStore) Invalidate(ctx context.Context, options ...lib_store.InvalidateOption) error {
	opts := lib_store.ApplyInvalidateOptions(options...)

	if tags := opts.Tags; len(tags) > 0 {
		for _, tag := range tags {
			tagKey := fmt.Sprintf(RueidisTagPattern, tag)

			cacheKeys, err := s.client.Do(ctx, s.client.B().Smembers().Key(tagKey).Build()).AsStrSlice()
			if err != nil {
				continue
			}

			for _, cacheKey := range cacheKeys {
				s.Delete(ctx, cacheKey)
			}

			s.Delete(ctx, tagKey)
		}
	}

	return nil
}

// GetType returns the store type
func (s *RueidisStore) GetType() string {
	return RueidisType
}

// Clear resets all data in the store
func (s *RueidisStore) Clear(ctx context.Context) error {
	return rueidiscompat.NewAdapter(s.client).FlushAll(ctx).Err()
}
