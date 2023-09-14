package ccache

import (
	"sync"
	"time"
)

type layeredBucket struct {
	sync.RWMutex
	buckets map[string]*bucket
}

func (b *layeredBucket) itemCount() int {
	count := 0
	b.RLock()
	defer b.RUnlock()
	for _, b := range b.buckets {
		count += b.itemCount()
	}
	return count
}

func (b *layeredBucket) get(primary, secondary string) *Item {
	bucket := b.getSecondaryBucket(primary)
	if bucket == nil {
		return nil
	}
	return bucket.get(secondary)
}

func (b *layeredBucket) getSecondaryBucket(primary string) *bucket {
	b.RLock()
	bucket, exists := b.buckets[primary]
	b.RUnlock()
	if exists == false {
		return nil
	}
	return bucket
}

func (b *layeredBucket) set(primary, secondary string, value interface{}, duration time.Duration, track bool) (*Item, *Item) {
	b.Lock()
	bkt, exists := b.buckets[primary]
	if exists == false {
		bkt = &bucket{lookup: make(map[string]*Item)}
		b.buckets[primary] = bkt
	}
	b.Unlock()
	item, existing := bkt.set(secondary, value, duration, track)
	item.group = primary
	return item, existing
}

func (b *layeredBucket) delete(primary, secondary string) *Item {
	b.RLock()
	bucket, exists := b.buckets[primary]
	b.RUnlock()
	if exists == false {
		return nil
	}
	return bucket.delete(secondary)
}

func (b *layeredBucket) deletePrefix(primary, prefix string, deletables chan *Item) int {
	b.RLock()
	bucket, exists := b.buckets[primary]
	b.RUnlock()
	if exists == false {
		return 0
	}
	return bucket.deletePrefix(prefix, deletables)
}

func (b *layeredBucket) deleteFunc(primary string, matches func(key string, item *Item) bool, deletables chan *Item) int {
	b.RLock()
	bucket, exists := b.buckets[primary]
	b.RUnlock()
	if exists == false {
		return 0
	}
	return bucket.deleteFunc(matches, deletables)
}

func (b *layeredBucket) deleteAll(primary string, deletables chan *Item) bool {
	b.RLock()
	bucket, exists := b.buckets[primary]
	b.RUnlock()
	if exists == false {
		return false
	}

	bucket.Lock()
	defer bucket.Unlock()

	if l := len(bucket.lookup); l == 0 {
		return false
	}
	for key, item := range bucket.lookup {
		delete(bucket.lookup, key)
		deletables <- item
	}
	return true
}

func (b *layeredBucket) forEachFunc(primary string, matches func(key string, item *Item) bool) {
	b.RLock()
	bucket, exists := b.buckets[primary]
	b.RUnlock()
	if exists {
		bucket.forEachFunc(matches)
	}
}

func (b *layeredBucket) clear() {
	b.Lock()
	defer b.Unlock()
	for _, bucket := range b.buckets {
		bucket.clear()
	}
	b.buckets = make(map[string]*bucket)
}
