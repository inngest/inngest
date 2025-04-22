// This is a fork of https://github.com/rs/dnscache. It makes 1 change: use
// ccache for the underlying cache.

package dnscache

import (
	"time"

	"github.com/karlseguin/ccache/v2"
)

func newCache(ttl time.Duration) *cache {
	return &cache{
		cache: ccache.New(ccache.Configure().MaxSize(10_000).ItemsToPrune(500)),
		ttl:   ttl,
	}
}

type cache struct {
	cache *ccache.Cache
	ttl   time.Duration
}

func (c *cache) Delete(key string) bool {
	return c.cache.Delete(key)
}

func (c *cache) ForEachFunc(matches func(key string, item *cacheEntry) bool) {
	c.cache.ForEachFunc(func(key string, item *ccache.Item) bool {
		entry, ok := item.Value().(*cacheEntry)
		if !ok {
			// Unreachable.
			return false
		}

		return matches(key, entry)
	})
}

func (c *cache) Get(key string) (*cacheEntry, bool) {
	item := c.cache.Get(key)
	if item == nil {
		return nil, false
	}

	entry, ok := item.Value().(*cacheEntry)
	if !ok {
		// Unreachable.
		return nil, false
	}
	if item.Expired() {
		// ignore expired items.
		return nil, false
	}

	return entry, true
}

func (c *cache) Set(key string, entry *cacheEntry) {
	c.cache.Set(key, entry, c.ttl)
}
