package memory

import "sync"

// ttlMap is a striped map of hashed keys to values with an expiry.  a value is
// present while nowMS < expiresAtMS, which matches Redis EX.  expired entries
// stay until sweep removes them.
type ttlMap[V any] struct {
	stripes [ttlStripes]ttlStripe[V]
}

const ttlStripes = 256

// ttlStripe is padded so neighbouring stripes do not share a cache line.
type ttlStripe[V any] struct {
	mu sync.Mutex
	m  map[uint64]ttlEntry[V]
	// lastExpiryMS is the latest expiry an entry of this stripe ever had.
	lastExpiryMS int64
	_            [40]byte
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
	return &t.stripes[key%ttlStripes]
}

// get returns the value for key when it has not expired at nowMS.
func (t *ttlMap[V]) get(nowMS int64, key uint64) (V, bool) {
	s := t.stripe(key)
	s.mu.Lock()
	e, ok := s.m[key]
	s.mu.Unlock()
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
	s.lastExpiryMS = max(s.lastExpiryMS, expiresAtMS)
	s.mu.Unlock()
}

// lastExpiry returns the time at which every entry now in the map has
// expired.  it is zero for a map that never had an entry.
func (t *ttlMap[V]) lastExpiry() int64 {
	var last int64
	for i := range t.stripes {
		s := &t.stripes[i]
		s.mu.Lock()
		last = max(last, s.lastExpiryMS)
		s.mu.Unlock()
	}
	return last
}

// sweep deletes every entry expired at nowMS and returns how many it removed.
func (t *ttlMap[V]) sweep(nowMS int64) int {
	removed := 0
	for i := range t.stripes {
		removed += t.sweepStripe(i, nowMS)
	}
	return removed
}

// sweepStripe is sweep for stripe i only.
func (t *ttlMap[V]) sweepStripe(i int, nowMS int64) int {
	removed := 0
	s := &t.stripes[i]
	s.mu.Lock()
	for k, e := range s.m {
		if e.expiresAtMS <= nowMS {
			delete(s.m, k)
			removed++
		}
	}
	s.mu.Unlock()
	return removed
}

// len counts every entry, expired or not.
func (t *ttlMap[V]) len() int {
	n := 0
	for i := range t.stripes {
		s := &t.stripes[i]
		s.mu.Lock()
		n += len(s.m)
		s.mu.Unlock()
	}
	return n
}

// each calls fn for every entry that has not expired at nowMS.  fn runs
// under the stripe lock and must not call back into the map.
func (t *ttlMap[V]) each(nowMS int64, fn func(key uint64, v V, expiresAtMS int64)) {
	for i := range t.stripes {
		s := &t.stripes[i]
		s.mu.Lock()
		for k, e := range s.m {
			if e.expiresAtMS > nowMS {
				fn(k, e.v, e.expiresAtMS)
			}
		}
		s.mu.Unlock()
	}
}
