package freecache

import (
	"encoding/binary"
	"sync"
	"sync/atomic"

	"github.com/cespare/xxhash/v2"
)

const (
	// segmentCount represents the number of segments within a freecache instance.
	segmentCount = 256
	// segmentAndOpVal is bitwise AND applied to the hashVal to find the segment id.
	segmentAndOpVal = 255
	minBufSize      = 512 * 1024
)

// Cache is a freecache instance.
type Cache struct {
	locks    [segmentCount]sync.Mutex
	segments [segmentCount]segment
}

type Updater func(value []byte, found bool) (newValue []byte, replace bool, expireSeconds int)

func hashFunc(data []byte) uint64 {
	return xxhash.Sum64(data)
}

// NewCache returns a newly initialize cache by size.
// The cache size will be set to 512KB at minimum.
// If the size is set relatively large, you should call
// `debug.SetGCPercent()`, set it to a much smaller value
// to limit the memory consumption and GC pause time.
func NewCache(size int) (cache *Cache) {
	return NewCacheCustomTimer(size, defaultTimer{})
}

// NewCacheCustomTimer returns new cache with custom timer.
func NewCacheCustomTimer(size int, timer Timer) (cache *Cache) {
	if size < minBufSize {
		size = minBufSize
	}
	if timer == nil {
		timer = defaultTimer{}
	}
	cache = new(Cache)
	for i := 0; i < segmentCount; i++ {
		cache.segments[i] = newSegment(size/segmentCount, i, timer)
	}
	return
}

// Set sets a key, value and expiration for a cache entry and stores it in the cache.
// If the key is larger than 65535 or value is larger than 1/1024 of the cache size,
// the entry will not be written to the cache. expireSeconds <= 0 means no expire,
// but it can be evicted when cache is full.
func (cache *Cache) Set(key, value []byte, expireSeconds int) (err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	err = cache.segments[segID].set(key, value, hashVal, expireSeconds)
	cache.locks[segID].Unlock()
	return
}

// Touch updates the expiration time of an existing key. expireSeconds <= 0 means no expire,
// but it can be evicted when cache is full.
func (cache *Cache) Touch(key []byte, expireSeconds int) (err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	err = cache.segments[segID].touch(key, hashVal, expireSeconds)
	cache.locks[segID].Unlock()
	return
}

// Get returns the value or not found error.
func (cache *Cache) Get(key []byte) (value []byte, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	value, _, err = cache.segments[segID].get(key, nil, hashVal, false)
	cache.locks[segID].Unlock()
	return
}

// MultiGet returns values and errors for the given keys. The returned slices
// have the same length as keys; values[i] and errs[i] correspond to keys[i].
// A miss is represented by values[i] == nil and errs[i] == ErrNotFound.
// MultiGet reduces lock contention by grouping keys by segment and acquiring
// each segment lock at most once.
// Note that MultiGet holds each segment lock longer than a single Get (for
// the duration of all keys in that segment), which can increase Get tail
// latency when MultiGet and Get run concurrently.
func (cache *Cache) MultiGet(keys [][]byte) (values [][]byte, errs []error) {
	n := len(keys)
	if n == 0 {
		return nil, nil
	}
	values = make([][]byte, n)
	errs = make([]error, n)
	type keyLoc struct {
		idx     int
		hashVal uint64
	}
	var groups [segmentCount][]keyLoc
	for i, key := range keys {
		hashVal := hashFunc(key)
		segID := hashVal & segmentAndOpVal
		groups[segID] = append(groups[segID], keyLoc{idx: i, hashVal: hashVal})
	}
	for segID := 0; segID < segmentCount; segID++ {
		batch := groups[segID]
		if len(batch) == 0 {
			continue
		}
		cache.locks[segID].Lock()
		for _, loc := range batch {
			value, _, err := cache.segments[segID].get(keys[loc.idx], nil, loc.hashVal, false)
			values[loc.idx] = value
			errs[loc.idx] = err
		}
		cache.locks[segID].Unlock()
	}
	return values, errs
}

// GetFn is equivalent to Get or GetWithBuf, but it attempts to be zero-copy,
// calling the provided function with slice view over the current underlying
// value of the key in memory. The slice is constrained in length and capacity.
//
// In moth cases, this method will not alloc a byte buffer. The only exception
// is when the value wraps around the underlying segment ring buffer.
//
// The method will return ErrNotFound is there's a miss, and the function will
// not be called. Errors returned by the function will be propagated.
func (cache *Cache) GetFn(key []byte, fn func([]byte) error) (err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	err = cache.segments[segID].view(key, fn, hashVal, false)
	cache.locks[segID].Unlock()
	return
}

// GetOrSet returns existing value or if record doesn't exist
// it sets a new key, value and expiration for a cache entry and stores it in the cache, returns nil in that case
func (cache *Cache) GetOrSet(key, value []byte, expireSeconds int) (retValue []byte, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	defer cache.locks[segID].Unlock()

	retValue, _, err = cache.segments[segID].get(key, nil, hashVal, false)
	if err != nil {
		err = cache.segments[segID].set(key, value, hashVal, expireSeconds)
	}
	return
}

// SetAndGet sets a key, value and expiration for a cache entry and stores it in the cache.
// If the key is larger than 65535 or value is larger than 1/1024 of the cache size,
// the entry will not be written to the cache. expireSeconds <= 0 means no expire,
// but it can be evicted when cache is full.  Returns existing value if record exists
// with a bool value to indicate whether an existing record was found
func (cache *Cache) SetAndGet(key, value []byte, expireSeconds int) (retValue []byte, found bool, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	defer cache.locks[segID].Unlock()

	retValue, _, err = cache.segments[segID].get(key, nil, hashVal, false)
	if err == nil {
		found = true
	}
	err = cache.segments[segID].set(key, value, hashVal, expireSeconds)
	return
}

// Update gets value for a key, passes it to updater function that decides if set should be called as well
// This allows for an atomic Get plus Set call using the existing value to decide on whether to call Set.
// If the key is larger than 65535 or value is larger than 1/1024 of the cache size,
// the entry will not be written to the cache. expireSeconds <= 0 means no expire,
// but it can be evicted when cache is full. Returns bool value to indicate if existing record was found along with bool
// value indicating the value was replaced and error if any
func (cache *Cache) Update(key []byte, updater Updater) (found bool, replaced bool, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	defer cache.locks[segID].Unlock()

	retValue, _, err := cache.segments[segID].get(key, nil, hashVal, false)
	if err == nil {
		found = true
	} else {
		err = nil // Clear ErrNotFound error since we're returning found flag
	}
	value, replaced, expireSeconds := updater(retValue, found)
	if !replaced {
		return
	}
	err = cache.segments[segID].set(key, value, hashVal, expireSeconds)
	return
}

// Peek returns the value or not found error, without updating access time or counters.
// Warning: No expiry check is performed so if an expired value is found, it will be
// returned without error
func (cache *Cache) Peek(key []byte) (value []byte, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	value, _, err = cache.segments[segID].get(key, nil, hashVal, true)
	cache.locks[segID].Unlock()
	return
}

// PeekWithExpiration returns the value and expiration time, without updating access time or counters.
// Warning: No expiry check is performed so if an expired value is found, it will be
// returned without error
func (cache *Cache) PeekWithExpiration(key []byte) (value []byte, expireAt uint32, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	value, expireAt, err = cache.segments[segID].get(key, nil, hashVal, true)
	cache.locks[segID].Unlock()
	return
}

// PeekFn is equivalent to Peek, but it attempts to be zero-copy, calling the
// provided function with slice view over the current underlying value of the
// key in memory. The slice is constrained in length and capacity.
//
// In most cases, this method will not alloc a byte buffer. The only exception
// is when the value wraps around the underlying segment ring buffer.
//
// The method will return ErrNotFound is there's a miss, and the function will
// not be called. Errors returned by the function will be propagated.
// Warning: No expiry check is performed so if an expired value is found, it will be
// returned without error
func (cache *Cache) PeekFn(key []byte, fn func([]byte) error) (err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	err = cache.segments[segID].view(key, fn, hashVal, true)
	cache.locks[segID].Unlock()
	return
}

// GetWithBuf copies the value to the buf or returns not found error.
// This method doesn't allocate memory when the capacity of buf is greater or equal to value.
func (cache *Cache) GetWithBuf(key, buf []byte) (value []byte, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	value, _, err = cache.segments[segID].get(key, buf, hashVal, false)
	cache.locks[segID].Unlock()
	return
}

// GetWithExpiration returns the value with expiration or not found error.
func (cache *Cache) GetWithExpiration(key []byte) (value []byte, expireAt uint32, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	value, expireAt, err = cache.segments[segID].get(key, nil, hashVal, false)
	cache.locks[segID].Unlock()
	return
}

// GetWithExpirationAndBuf copies the value to the buf and gets with expiration or returns a not found error.
// This method doesn't allocate memory when the capacity of buf is greater or equal to value.
func (cache *Cache) GetWithExpirationAndBuf(key []byte, buf []byte) (value []byte, expireAt uint32, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	value, expireAt, err = cache.segments[segID].get(key, buf, hashVal, false)
	cache.locks[segID].Unlock()
	return
}

// TTL returns the TTL time left for a given key or a not found error.
func (cache *Cache) TTL(key []byte) (timeLeft uint32, err error) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	timeLeft, err = cache.segments[segID].ttl(key, hashVal)
	cache.locks[segID].Unlock()
	return
}

// Del deletes an item in the cache by key and returns true or false if a delete occurred.
func (cache *Cache) Del(key []byte) (affected bool) {
	hashVal := hashFunc(key)
	segID := hashVal & segmentAndOpVal
	cache.locks[segID].Lock()
	affected = cache.segments[segID].del(key, hashVal)
	cache.locks[segID].Unlock()
	return
}

// SetInt stores in integer value in the cache.
func (cache *Cache) SetInt(key int64, value []byte, expireSeconds int) (err error) {
	var bKey [8]byte
	binary.LittleEndian.PutUint64(bKey[:], uint64(key))
	return cache.Set(bKey[:], value, expireSeconds)
}

// GetInt returns the value for an integer within the cache or a not found error.
func (cache *Cache) GetInt(key int64) (value []byte, err error) {
	var bKey [8]byte
	binary.LittleEndian.PutUint64(bKey[:], uint64(key))
	return cache.Get(bKey[:])
}

// GetIntWithExpiration returns the value and expiration or a not found error.
func (cache *Cache) GetIntWithExpiration(key int64) (value []byte, expireAt uint32, err error) {
	var bKey [8]byte
	binary.LittleEndian.PutUint64(bKey[:], uint64(key))
	return cache.GetWithExpiration(bKey[:])
}

// DelInt deletes an item in the cache by int key and returns true or false if a delete occurred.
func (cache *Cache) DelInt(key int64) (affected bool) {
	var bKey [8]byte
	binary.LittleEndian.PutUint64(bKey[:], uint64(key))
	return cache.Del(bKey[:])
}

// EvacuateCount is a metric indicating the number of times an eviction occurred.
func (cache *Cache) EvacuateCount() (count int64) {
	for i := range cache.segments {
		count += atomic.LoadInt64(&cache.segments[i].totalEvacuate)
	}
	return
}

// ExpiredCount is a metric indicating the number of times an expire occurred.
func (cache *Cache) ExpiredCount() (count int64) {
	for i := range cache.segments {
		count += atomic.LoadInt64(&cache.segments[i].totalExpired)
	}
	return
}

// EntryCount returns the number of items currently in the cache.
func (cache *Cache) EntryCount() (entryCount int64) {
	for i := range cache.segments {
		entryCount += atomic.LoadInt64(&cache.segments[i].entryCount)
	}
	return
}

// AverageAccessTime returns the average unix timestamp when a entry being accessed.
// Entries have greater access time will be evacuated when it
// is about to be overwritten by new value.
func (cache *Cache) AverageAccessTime() int64 {
	var entryCount, totalTime int64
	for i := range cache.segments {
		totalTime += atomic.LoadInt64(&cache.segments[i].totalTime)
		entryCount += atomic.LoadInt64(&cache.segments[i].totalCount)
	}
	if entryCount == 0 {
		return 0
	} else {
		return totalTime / entryCount
	}
}

// HitCount is a metric that returns number of times a key was found in the cache.
func (cache *Cache) HitCount() (count int64) {
	for i := range cache.segments {
		count += atomic.LoadInt64(&cache.segments[i].hitCount)
	}
	return
}

// MissCount is a metric that returns the number of times a miss occurred in the cache.
func (cache *Cache) MissCount() (count int64) {
	for i := range cache.segments {
		count += atomic.LoadInt64(&cache.segments[i].missCount)
	}
	return
}

// LookupCount is a metric that returns the number of times a lookup for a given key occurred.
func (cache *Cache) LookupCount() int64 {
	return cache.HitCount() + cache.MissCount()
}

// HitRate is the ratio of hits over lookups.
func (cache *Cache) HitRate() float64 {
	hitCount, missCount := cache.HitCount(), cache.MissCount()
	lookupCount := hitCount + missCount
	if lookupCount == 0 {
		return 0
	} else {
		return float64(hitCount) / float64(lookupCount)
	}
}

// OverwriteCount indicates the number of times entries have been overridden.
func (cache *Cache) OverwriteCount() (overwriteCount int64) {
	for i := range cache.segments {
		overwriteCount += atomic.LoadInt64(&cache.segments[i].overwrites)
	}
	return
}

// TouchedCount indicates the number of times entries have had their expiration time extended.
func (cache *Cache) TouchedCount() (touchedCount int64) {
	for i := range cache.segments {
		touchedCount += atomic.LoadInt64(&cache.segments[i].touched)
	}
	return
}

// Clear clears the cache.
func (cache *Cache) Clear() {
	for i := range cache.segments {
		cache.locks[i].Lock()
		cache.segments[i].clear()
		cache.locks[i].Unlock()
	}
}

// ResetStatistics refreshes the current state of the statistics.
func (cache *Cache) ResetStatistics() {
	for i := range cache.segments {
		cache.locks[i].Lock()
		cache.segments[i].resetStatistics()
		cache.locks[i].Unlock()
	}
}
