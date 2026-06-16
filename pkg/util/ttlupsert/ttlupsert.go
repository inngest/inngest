package ttlupsert

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"golang.org/x/sync/singleflight"
)

const (
	defaultTTL      = 10 * time.Minute
	defaultCapacity = 10_000
)

var (
	ErrEmptyKey = errors.New("ttlupsert: empty key")
	ErrNilTx    = errors.New("ttlupsert: nil transaction")
)

// Upserter is an upserter that reduces load on the DB by checking whether
// we've upserted a record with the given Key() recently.
type Upserter[T any] interface {
	// Key returns a key for the given type.
	Key(T) string
	// Upsert runs tx unless the item was successfully upserted recently.
	// The bool is true only for the caller that executed tx.
	Upsert(ctx context.Context, item T, tx func(ctx context.Context) error) (bool, error)
}

type config struct {
	ttl      time.Duration
	capacity int
	clock    clockwork.Clock
}

// Option configures an Upserter.
type Option func(*config)

// WithTTL sets how long successful upsert keys remain skippable.
func WithTTL(ttl time.Duration) Option {
	return func(c *config) {
		c.ttl = ttl
	}
}

// WithCapacity sets the maximum number of successful upsert keys to remember.
func WithCapacity(capacity int) Option {
	return func(c *config) {
		c.capacity = capacity
	}
}

// WithClock sets the clock used for expiry checks.
func WithClock(clock clockwork.Clock) Option {
	return func(c *config) {
		if clock != nil {
			c.clock = clock
		}
	}
}

func New[T any](opts ...Option) Upserter[T] {
	return NewWithKey(func(t T) string {
		return util.XXHash(t)
	}, opts...)
}

func NewWithKey[T any](keyFn func(T) string, opts ...Option) Upserter[T] {
	conf := config{
		ttl:      defaultTTL,
		capacity: defaultCapacity,
		clock:    clockwork.NewRealClock(),
	}
	for _, opt := range opts {
		opt(&conf)
	}
	if keyFn == nil {
		keyFn = func(t T) string {
			return util.XXHash(t)
		}
	}

	return &upserter[T]{
		keyFn:       keyFn,
		ttl:         conf.ttl,
		capacity:    conf.capacity,
		clock:       conf.clock,
		entries:     map[string]*expiryEntry{},
		expirations: expiryHeap{},
	}
}

type upserter[T any] struct {
	keyFn func(T) string
	ttl   time.Duration

	// capacity is how many keys wiull be tracked to skip uypserting
	capacity int
	clock    clockwork.Clock

	// mu locks entries and expirations, a basic LRU cache
	mu          sync.Mutex
	entries     map[string]*expiryEntry
	expirations expiryHeap

	// group prevents more than 1 upsert per key from occurring at once (on this
	// particular instance)
	group singleflight.Group
}

func (s *upserter[T]) Key(t T) string {
	return s.keyFn(t)
}

func (s *upserter[T]) Upsert(ctx context.Context, item T, tx func(context.Context) error) (bool, error) {
	if tx == nil {
		return false, ErrNilTx
	}

	key := s.Key(item)
	if key == "" {
		return false, ErrEmptyKey
	}

	if s.cached(key) {
		return false, nil
	}

	var ran bool
	_, err, _ := s.group.Do(key, func() (any, error) {
		// Re-check inside singleflight in case another caller populated the
		// cache after this caller's first check but before this closure won.
		if s.cached(key) {
			return nil, nil
		}

		ran = true
		err := tx(ctx)
		if err != nil {
			return nil, err
		}

		s.record(key)
		return nil, nil
	})
	return ran, err
}

func (s *upserter[T]) cached(key string) bool {
	if !s.cacheEnabled() {
		return false
	}

	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.entries[key]
	if !ok {
		return false
	}
	if !item.expiresAt.After(now) {
		s.removeLocked(item)
		return false
	}
	s.refreshLocked(item, now)
	return true
}

func (s *upserter[T]) record(key string) {
	if !s.cacheEnabled() {
		return
	}

	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.recordLocked(key, now)
}

func (s *upserter[T]) cacheEnabled() bool {
	return s.ttl > 0 && s.capacity > 0
}

func (s *upserter[T]) recordLocked(key string, now time.Time) {
	if item, ok := s.entries[key]; ok {
		if !item.expiresAt.After(now) {
			s.removeLocked(item)
		} else {
			s.refreshLocked(item, now)
			return
		}
	}

	item := &expiryEntry{key: key}
	s.refreshItem(item, now)
	s.entries[key] = item
	heap.Push(&s.expirations, item)
	s.trimCapacityLocked()
}

func (s *upserter[T]) refreshLocked(item *expiryEntry, now time.Time) {
	s.refreshItem(item, now)
	heap.Fix(&s.expirations, item.index)
}

func (s *upserter[T]) removeLocked(item *expiryEntry) {
	heap.Remove(&s.expirations, item.index)
	delete(s.entries, item.key)
}

func (s *upserter[T]) trimCapacityLocked() {
	for s.capacity > 0 && len(s.entries) > s.capacity && s.expirations.Len() > 0 {
		next := heap.Pop(&s.expirations).(*expiryEntry)
		delete(s.entries, next.key)
	}
}

func (s *upserter[T]) refreshItem(item *expiryEntry, now time.Time) {
	item.expiresAt = now.Add(s.ttl)
}

// expiryEntry tracks an entry in our upserter lru
type expiryEntry struct {
	key       string
	expiresAt time.Time
	index     int
}

// expiryHeap tracks lru expiry times so we can pop the oldest when our cache
// is full
type expiryHeap []*expiryEntry

func (h expiryHeap) Len() int {
	return len(h)
}

func (h expiryHeap) Less(i, j int) bool {
	return h[i].expiresAt.Before(h[j].expiresAt)
}

func (h expiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *expiryHeap) Push(x any) {
	item := x.(*expiryEntry)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *expiryHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	item.index = -1
	*h = old[:n-1]
	return item
}
