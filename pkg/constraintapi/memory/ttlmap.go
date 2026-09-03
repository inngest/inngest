package memory

import "sync"

// ttlMap is a striped map of hashed keys to values with an expiry.  a value is
// present while nowMS < expiresAtMS, which matches Redis EX.  expired entries
// stay until sweep removes them.
type ttlMap[V any] struct {
	stripes [64]ttlStripe[V]
}

type ttlStripe[V any] struct {
	mu sync.RWMutex
	m  map[uint64]ttlEntry[V]
}

type ttlEntry[V any] struct {
	expiresAtMS int64
	v           V
}

func newTTLMap[V any]() *ttlMap[V] {
	t := &ttlMap[V]{}
	for i := range t.stripes {
		t.stripes[i].m = map[uint64]ttlEntry[V]{}
	}
	return t
}

func (t *ttlMap[V]) stripe(key uint64) *ttlStripe[V] {
	return &t.stripes[key&63]
}

// get returns the value for key when it has not expired at nowMS.
func (t *ttlMap[V]) get(nowMS int64, key uint64) (V, bool) {
	s := t.stripe(key)
	s.mu.RLock()
	e, ok := s.m[key]
	s.mu.RUnlock()
	if !ok || e.expiresAtMS <= nowMS {
		var zero V
		return zero, false
	}
	return e.v, true
}

// set stores v for key until expiresAtMS, replacing any earlier value.
func (t *ttlMap[V]) set(key uint64, v V, expiresAtMS int64) {
	s := t.stripe(key)
	s.mu.Lock()
	s.m[key] = ttlEntry[V]{expiresAtMS: expiresAtMS, v: v}
	s.mu.Unlock()
}

// sweep deletes every entry expired at nowMS and returns how many it removed.
func (t *ttlMap[V]) sweep(nowMS int64) int {
	removed := 0
	for i := range t.stripes {
		s := &t.stripes[i]
		s.mu.Lock()
		for k, e := range s.m {
			if e.expiresAtMS <= nowMS {
				delete(s.m, k)
				removed++
			}
		}
		s.mu.Unlock()
	}
	return removed
}

// len counts every entry, expired or not.
func (t *ttlMap[V]) len() int {
	n := 0
	for i := range t.stripes {
		s := &t.stripes[i]
		s.mu.RLock()
		n += len(s.m)
		s.mu.RUnlock()
	}
	return n
}
