package memory

import "sync"

// ttlMap is a striped map of hashed keys to values with an expiry.  a value is
// present while nowMS < expiresAtMS, which matches Redis EX.  expired entries
// stay until rotate drops the map they are in, which is at most two times
// their time to live after they were set.
type ttlMap[V any] struct {
	stripes [ttlStripes]ttlStripe[V]
}

const ttlStripes = 256

// ttlStripe is padded so neighbouring stripes do not share a cache line.
type ttlStripe[V any] struct {
	mu sync.Mutex
	// cur receives every set.
	cur map[uint64]ttlEntry[V]
	// old is an earlier cur.  it is only read, and it is nil when the stripe
	// has none.
	old map[uint64]ttlEntry[V]
	// curLastMS and oldLastMS are the latest expiry of an entry in cur and in
	// old.
	curLastMS int64
	oldLastMS int64
	_         [24]byte
}

type ttlEntry[V any] struct {
	expiresAtMS int64
	v           V
}

func newTTLMap[V any]() *ttlMap[V] {
	t := &ttlMap[V]{}
	for i := range t.stripes {
		t.stripes[i].cur = map[uint64]ttlEntry[V]{}
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
	e, ok := s.cur[key]
	if !ok && s.old != nil {
		e, ok = s.old[key]
	}
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
	s.cur[key] = ttlEntry[V]{
		expiresAtMS: expiresAtMS,
		v:           v,
	}
	s.curLastMS = max(s.curLastMS, expiresAtMS)
	s.mu.Unlock()
}

// lastExpiry returns the time at which every entry now in the map has
// expired.  it is zero for a map with no entry.
func (t *ttlMap[V]) lastExpiry() int64 {
	var last int64
	for i := range t.stripes {
		s := &t.stripes[i]
		s.mu.Lock()
		last = max(last, s.curLastMS, s.oldLastMS)
		s.mu.Unlock()
	}
	return last
}

// rotate frees expired entries without a scan.  a stripe drops its old map
// when every entry in it has expired at nowMS.  cur then becomes the old
// map, or is dropped too when all of it has expired.  a stripe is locked for
// the same short time whatever its size, so a get or a set never waits for
// the entries to be read.
func (t *ttlMap[V]) rotate(nowMS int64) {
	for i := range t.stripes {
		s := &t.stripes[i]
		s.mu.Lock()
		if s.oldLastMS <= nowMS {
			s.old = nil
			s.oldLastMS = 0
		}
		if s.old == nil && len(s.cur) > 0 {
			if s.curLastMS > nowMS {
				s.old = s.cur
				s.oldLastMS = s.curLastMS
			}
			s.cur = map[uint64]ttlEntry[V]{}
			s.curLastMS = 0
		}
		s.mu.Unlock()
	}
}

// len counts every entry of both maps, expired or not.  a key that was set
// again after a rotate counts two times.
func (t *ttlMap[V]) len() int {
	n := 0
	for i := range t.stripes {
		s := &t.stripes[i]
		s.mu.Lock()
		n += len(s.cur) + len(s.old)
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
		for k, e := range s.cur {
			if e.expiresAtMS > nowMS {
				fn(k, e.v, e.expiresAtMS)
			}
		}
		for k, e := range s.old {
			// a key that is also in cur was set again, and cur has its value
			if _, again := s.cur[k]; !again && e.expiresAtMS > nowMS {
				fn(k, e.v, e.expiresAtMS)
			}
		}
		s.mu.Unlock()
	}
}
