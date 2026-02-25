// Copyright 2024 The Cockroach Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package swiss is a Go implementation of Swiss Tables as described in
// https://abseil.io/about/design/swisstables. See also:
// https://faultlore.com/blah/hashbrown-tldr/.
//
// Google's C++ implementation:
//
//	https://github.com/abseil/abseil-cpp/blob/master/absl/container/internal/raw_hash_set.h
//
// # Swiss Tables
//
// Swiss tables are hash tables that map keys to values, similar to Go's
// builtin map type. Swiss tables use open-addressing rather than chaining to
// handle collisions. If you're not familiar with open-addressing see
// https://en.wikipedia.org/wiki/Open_addressing. A hybrid between linear and
// quadratic probing is used - linear probing within groups of small fixed
// size and quadratic probing at the group level. The key design choice of
// Swiss tables is the usage of a separate metadata array that stores 1 byte
// per slot in the table. 7-bits of this "control byte" are taken from
// hash(key) and the remaining bit is used to indicate whether the slot is
// empty, full, or deleted. The metadata array allows quick probes. The Google
// implementation of Swiss tables uses SIMD on x86 CPUs in order to quickly
// check 16 slots at a time for a match. Neon on arm64 CPUs is apparently too
// high latency, but the generic version is still able to compare 8 bytes at
// time through bit tricks (SWAR, SIMD Within A Register).
//
// Google's Swiss Tables layout is N-1 slots where N is a power of 2 and
// N+groupSize control bytes. The [N:N+groupSize] control bytes mirror the
// first groupSize control bytes so that probe operations at the end of the
// control bytes array do not have to perform additional checks. The
// separation of control bytes and slots implies 2 cache misses for a large
// map (larger than L2 cache size) or a cold map. The swiss.Map implementation
// differs from Google's layout: it groups together 8 control bytes and 8
// slots which often results in 1 cache miss for a large or cold map rather
// than separate accesses for the controls and slots. The mirrored control
// bytes are no longer needed and and groups no longer start at arbitrary slot
// index, but only at those that are multiples of 8.
//
// Probing is done by taking the top 57 bits of hash(key)%N as the index into
// the groups slice and then performing a check of the groupSize control bytes
// within the group. Probing walks through groups in the table using quadratic
// probing until it finds a group that has at least one empty slot. See the
// comments on probeSeq for more details on the order in which groups are
// probed and the guarantee that every group is examined which means that in
// the worst case probing will end when an empty slot is encountered (the map
// can never be 100% full).
//
// Deletion is performed using tombstones (ctrlDeleted) with an optimization
// to mark a slot as empty if we can prove that doing so would not violate the
// probing behavior that a group of full slots causes probing to continue. It
// is invalid to take a group of full slots and mark one as empty as doing so
// would cause subsequent lookups to terminate at that group rather than
// continue to probe. We prove a slot was never part of a full group by
// looking for whether any of the groupSize-1 neighbors to the left and right
// of the deleting slot are empty which indicates that the slot was never part
// of a full group.
//
// # Extendible Hashing
//
// The Swiss table design has a significant caveat: resizing of the table is
// done all at once rather than incrementally. This can cause long-tail
// latency blips in some use cases. To address this caveat, extendible hashing
// (https://en.wikipedia.org/wiki/Extendible_hashing) is applied on top of the
// Swiss table foundation. In extendible hashing, there is a top-level
// directory containing entries pointing to buckets. In swiss.Map each bucket
// is a Swiss table as described above.
//
// The high bits of hash(key) are used to index into the bucket directory
// which is effectively a trie. The number of bits used is the globalDepth,
// resulting in 2^globalDepth directory entries. Adjacent entries in the
// directory are allowed to point to the same bucket which enables resizing to
// be done incrementally, one bucket at a time. Each bucket has a localDepth
// which is less than or equal to the globalDepth. If the localDepth for a
// bucket equals the globalDepth then only a single directory entry points to
// the bucket. Otherwise, more than one directory entry points to the bucket.
//
// The diagram below shows one possible scenario for the directory and
// buckets. With a globalDepth of 2 the directory contains 4 entries. The
// first 2 entries point to the same bucket which has a localDepth of 1, while
// the last 2 entries point to different buckets.
//
//	 dir(globalDepth=2)
//	+----+
//	| 00 | --\
//	+----+    +--> bucket[localDepth=1]
//	| 01 | --/
//	+----+
//	| 10 | ------> bucket[localDepth=2]
//	+----+
//	| 11 | ------> bucket[localDepth=2]
//	+----+
//
// The index into the directory is "hash(key) >> (64 - globalDepth)".
//
// When a bucket gets too large (specified by a configurable threshold) it is
// split. When a bucket is split its localDepth is incremented. If its
// localDepth is less than or equal to its globalDepth then the newly split
// bucket can be installed in the directory. If the bucket's localDepth is
// greater than the globalDepth then the globalDepth is incremented and the
// directory is reallocated at twice its current size. In the diagram above,
// consider what happens if the bucket at dir[3] is split:
//
//	 dir(globalDepth=3)
//	+-----+
//	| 000 | --\
//	+-----+    \
//	| 001 | ----\
//	+-----+      +--> bucket[localDepth=1]
//	| 010 | ----/
//	+-----+    /
//	| 011 | --/
//	+-----+
//	| 100 | --\
//	+-----+    +----> bucket[localDepth=2]
//	| 101 | --/
//	+-----+
//	| 110 | --------> bucket[localDepth=3]
//	+-----+
//	| 111 | --------> bucket[localDepth=3]
//	+-----+
//
// Note that the diagram above is very unlikely with a good hash function as
// the buckets will tend to fill at a similar rate.
//
// The split operation redistributes the records in a bucket into two buckets.
// This is done by walking over the records in the bucket to be split,
// computing hash(key) and using localDepth to extract the bit which
// determines whether to leave the record in the current bucket or to move it
// to the new bucket.
//
// Maps containing only a single bucket are optimized to avoid the directory
// indexing resulting in performance that is equivalent to a Swiss table
// without extendible hashing. A single bucket can be guaranteed by
// configuring a very large bucket size threshold via the
// WithMaxBucketCapacity option.
//
// In order to avoid a level of indirection when accessing a bucket, the
// bucket directory points to buckets by value rather than by pointer.
// Adjacent bucket[K,V]'s which share are logically the same bucket share the
// bucket.groups slice and have the same values for
// bucket.{groupMask,localDepth,index}. The other fields of a bucket are only
// valid for buckets where &m.dir[bucket.index] = &bucket (i.e. the first
// bucket in the directory with the specified index). During Get operations,
// any of the buckets with the same index may be used for retrieval. During
// Put and Delete operations an additional indirection is performed, though
// the common case is that this indirection is within the same cache line as
// it is to the immediately preceding bucket in the directory.
//
// # Implementation
//
// The implementation follows Google's Abseil implementation of Swiss Tables,
// and is heavily tuned, using unsafe and raw pointer arithmentic rather than
// Go slices to squeeze out every drop of performance. In order to support
// hashing of arbitrary keys, a hack is performed to extract the hash function
// from Go's implementation of map[K]struct{} by reaching into the internals
// of the type. (This might break in future version of Go, but is likely
// fixable unless the Go runtime does something drastic).
//
// # Performance
//
// A swiss.Map has similar or slightly better performance than Go's builtin
// map for small map sizes, and is much faster at large map sizes. See
// [README.md] for details.
//
// [README.md] https://github.com/cockroachdb/swiss/blob/main/README.md
package swiss

import (
	"fmt"
	"io"
	"math"
	"math/bits"
	"strings"
	"unsafe"
)

const (
	groupSize       = 8
	maxAvgGroupLoad = 7

	ctrlEmpty   ctrl = 0b10000000
	ctrlDeleted ctrl = 0b11111110

	bitsetLSB     = 0x0101010101010101
	bitsetMSB     = 0x8080808080808080
	bitsetEmpty   = bitsetLSB * uint64(ctrlEmpty)
	bitsetDeleted = bitsetLSB * uint64(ctrlDeleted)

	// The default maximum capacity a bucket is allowed to grow to before it
	// will be split.
	defaultMaxBucketCapacity uint32 = 4096

	// ptrSize and shiftMask are used to optimize code generation for
	// Map.bucket(), Map.bucketCount(), and bucketStep(). This technique was
	// lifted from the Go runtime's runtime/map.go:bucketShift() routine. Note
	// that ptrSize will be either 4 on 32-bit archs or 8 on 64-bit archs.
	ptrSize   = 4 << (^uintptr(0) >> 63)
	ptrBits   = ptrSize * 8
	shiftMask = ptrSize*8 - 1

	expectedBucketSize = ptrSize + 6*4
)

// Don't add fields to the bucket unnecessarily. It is packed for efficiency so
// that we can fit 2 buckets into a 64-byte cache line on 64-bit architectures.
// This will cause a type error if the size of a bucket changes.
var _ [0]struct{} = [unsafe.Sizeof(bucket[int, int]{}) - expectedBucketSize]struct{}{}

// slot holds a key and value.
type slot[K comparable, V any] struct {
	key   K
	value V
}

// Group holds groupSize control bytes and slots.
type Group[K comparable, V any] struct {
	ctrls ctrlGroup
	slots slotGroup[K, V]
}

// bucket implements Google's Swiss Tables hash table design. A Map is
// composed of 1 or more buckets that are addressed using extendible hashing.
type bucket[K comparable, V any] struct {
	// groups is groupMask+1 in length and holds groupSize key/value slots and
	// their control bytes.
	groups unsafeSlice[Group[K, V]]
	// groupMask is the number of groups minus 1 which is used to quickly
	// compute i%N using a bitwise & operation. The groupMask only changes
	// when a bucket is resized.
	groupMask uint32

	// Capacity, used, and growthLeft are only updated on mutation operations
	// (Put, Delete). Read operations (Get) only access the groups and
	// groupMask fields.

	// The total number (always 2^N). Equal to `(groupMask+1)*groupSize`
	// (unless the bucket is empty, when capacity is 0).
	capacity uint32
	// The number of filled slots (i.e. the number of elements in the bucket).
	used uint32
	// The number of slots we can still fill without needing to rehash.
	//
	// This is stored separately due to tombstones: we do not include
	// tombstones in the growth capacity because we'd like to rehash when the
	// table is filled with tombstones as otherwise probe sequences might get
	// unacceptably long without triggering a rehash.
	growthLeft uint32

	// localDepth is the number of high bits from hash(key) used to generate
	// an index for the global directory to locate this bucket. If localDepth
	// is 0 this bucket is Map.bucket0. LocalDepth is only updated when a
	// bucket splits.
	localDepth uint32
	// The index of the bucket within Map.dir. The buckets in
	// Map.dir[index:index+2^(globalDepth-localDepth)] all share the same
	// groups (and are logically the same bucket). Only the bucket at
	// Map.dir[index] can be used for mutation operations (Put, Delete). The
	// other buckets can be used for Get operations. Index is only updated
	// when a bucket splits or the directory grows.
	index uint32
}

// Map is an unordered map from keys to values with Put, Get, Delete, and All
// operations. Map is inspired by Google's Swiss Tables design as implemented
// in Abseil's flat_hash_map, combined with extendible hashing. By default, a
// Map[K,V] uses the same hash function as Go's builtin map[K]V, though a
// different hash function can be specified using the WithHash option.
//
// A Map is NOT goroutine-safe.
type Map[K comparable, V any] struct {
	// The hash function to each keys of type K. The hash function is
	// extracted from the Go runtime's implementation of map[K]struct{}.
	hash hashFn
	seed uintptr
	// The allocator to use for the ctrls and slots slices.
	allocator Allocator[K, V]
	// bucket0 is always present and inlined in the Map to avoid a pointer
	// indirection during the common case that the map contains a single
	// bucket. bucket0 is also used during split operations as a temporary
	// bucket to split into before the bucket is installed in the directory.
	bucket0 bucket[K, V]
	// The directory of buckets. See the comment on bucket.index for details
	// on how the physical bucket values map to logical buckets.
	dir unsafeSlice[bucket[K, V]]
	// The number of filled slots across all buckets (i.e. the number of
	// elements in the map).
	used int
	// globalShift is the number of bits to right shift a hash value to
	// generate an index for the global directory. As a special case, if
	// globalShift==0 then bucket0 is used and the directory is not accessed.
	// Note that globalShift==(64-globalDepth). globalShift is used rather
	// than globalDepth because the shifting is the more common operation than
	// needing to compare globalDepth to a bucket's localDepth.
	globalShift uint32
	// The maximum capacity a bucket is allowed to grow to before it will be
	// split.
	maxBucketCapacity uint32
	_                 noCopy
}

// normalizeCapacity rounds capacity to the next power of 2.
func normalizeCapacity(capacity uint32) uint32 {
	return uint32(1) << min(bits.Len32(capacity-1), 31)
}

// New constructs a new Map with the specified initial capacity. If
// initialCapacity is 0 the map will start out with zero capacity and will
// grow on the first insert. The zero value for a Map is not usable.
func New[K comparable, V any](initialCapacity int, options ...Option[K, V]) *Map[K, V] {
	m := &Map[K, V]{}
	m.Init(initialCapacity, options...)
	return m
}

// Init initializes a Map with the specified initial capacity. If
// initialCapacity is 0 the map will start out with zero capacity and will
// grow on the first insert. The zero value for a Map is not usable and Init
// must be called before using the map.
//
// Init is intended for usage when a Map is embedded by value in another
// structure.
func (m *Map[K, V]) Init(initialCapacity int, options ...Option[K, V]) {
	*m = Map[K, V]{
		hash:      getRuntimeHasher[K](),
		seed:      uintptr(fastrand64()),
		allocator: defaultAllocator[K, V]{},
		bucket0: bucket[K, V]{
			// The groups slice for bucket0 in an empty map points to a single
			// group where the controls are all marked as empty. This
			// simplifies the logic for probing in Get, Put, and Delete. The
			// empty controls will never match a probe operation, and if
			// insertion is performed growthLeft==0 will trigger a resize of
			// the bucket.
			groups: makeUnsafeSlice(unsafeConvertSlice[Group[K, V]](emptyCtrls[:])),
		},
		maxBucketCapacity: defaultMaxBucketCapacity,
	}

	// Initialize the directory to point to bucket0.
	m.dir = makeUnsafeSlice(unsafe.Slice(&m.bucket0, 1))

	for _, op := range options {
		op.apply(m)
	}

	if m.maxBucketCapacity < groupSize {
		m.maxBucketCapacity = groupSize
	}
	m.maxBucketCapacity = normalizeCapacity(m.maxBucketCapacity)

	if initialCapacity > 0 {
		// We consider initialCapacity to be an indication from the caller
		// about the number of records the map should hold. The realized
		// capacity of a map is 7/8 of the number of slots, so we set the
		// target capacity to initialCapacity*8/7.
		targetCapacity := uintptr((initialCapacity * groupSize) / maxAvgGroupLoad)
		if targetCapacity <= uintptr(m.maxBucketCapacity) {
			// Normalize targetCapacity to the smallest value of the form 2^k.
			m.bucket0.init(m, normalizeCapacity(uint32(targetCapacity)))
		} else {
			// If targetCapacity is larger than maxBucketCapacity we need to
			// size the directory appropriately. We'll size each bucket to
			// maxBucketCapacity and create enough buckets to hold
			// initialCapacity.
			nBuckets := (targetCapacity + uintptr(m.maxBucketCapacity) - 1) / uintptr(m.maxBucketCapacity)
			globalDepth := uint32(bits.Len32(uint32(nBuckets) - 1))
			m.growDirectory(globalDepth, 0 /* index */)

			n := m.bucketCount()
			for i := uint32(0); i < n; i++ {
				b := m.dir.At(uintptr(i))
				b.init(m, m.maxBucketCapacity)
				b.localDepth = globalDepth
				b.index = i
			}

			m.checkInvariants()
		}
	}

	m.buckets(0, func(b *bucket[K, V]) bool {
		b.checkInvariants(m)
		return true
	})
}

// Close closes the map, releasing any memory back to its configured
// allocator. It is unnecessary to close a map using the default allocator. It
// is invalid to use a Map after it has been closed, though Close itself is
// idempotent.
func (m *Map[K, V]) Close() {
	m.buckets(0, func(b *bucket[K, V]) bool {
		b.close(m.allocator)
		return true
	})

	m.allocator = nil
}

// Put inserts an entry into the map, overwriting an existing value if an
// entry with the same key already exists.
func (m *Map[K, V]) Put(key K, value V) {
	// Put is find composed with uncheckedPut. We perform find to see if the
	// key is already present. If it is, we're done and overwrite the existing
	// value. If the value isn't present we perform an uncheckedPut which
	// inserts an entry known not to be in the table (violating this
	// requirement will cause the table to behave erratically).
	h := m.hash(noescape(unsafe.Pointer(&key)), m.seed)
	b := m.mutableBucket(h)

	// NB: Unlike the abseil swiss table implementation which uses a common
	// find routine for Get, Put, and Delete, we have to manually inline the
	// find routine for performance.
	seq := makeProbeSeq(h1(h), b.groupMask)
	startOffset := seq.offset

	for ; ; seq = seq.next() {
		g := b.groups.At(uintptr(seq.offset))
		match := g.ctrls.matchH2(h2(h))

		for match != 0 {
			i := match.first()
			slot := g.slots.At(i)
			if key == slot.key {
				slot.value = value
				b.checkInvariants(m)
				return
			}
			match = match.removeFirst()
		}

		match = g.ctrls.matchEmpty()
		if match != 0 {
			// Finding an empty slot means we've reached the end of the probe
			// sequence.

			// If there is room left to grow in the bucket and we're at the
			// start of the probe sequence we can just insert the new entry.
			if b.growthLeft > 0 && seq.offset == startOffset {
				i := match.first()
				slot := g.slots.At(i)
				slot.key = key
				slot.value = value
				g.ctrls.Set(i, ctrl(h2(h)))
				b.growthLeft--
				b.used++
				m.used++
				b.checkInvariants(m)
				return
			}

			// Find the first empty or deleted slot in the key's probe
			// sequence.
			seq := makeProbeSeq(h1(h), b.groupMask)
			for ; ; seq = seq.next() {
				g := b.groups.At(uintptr(seq.offset))
				match = g.ctrls.matchEmptyOrDeleted()
				if match != 0 {
					i := match.first()
					// If there is room left to grow in the table or the slot
					// is deleted (and thus we're overwriting it and not
					// changing growthLeft) we can insert the entry here.
					// Otherwise we need to rehash the bucket.
					if b.growthLeft > 0 || g.ctrls.Get(i) == ctrlDeleted {
						slot := g.slots.At(i)
						slot.key = key
						slot.value = value
						if g.ctrls.Get(i) == ctrlEmpty {
							b.growthLeft--
						}
						g.ctrls.Set(i, ctrl(h2(h)))
						b.used++
						m.used++
						b.checkInvariants(m)
						return
					}
					break
				}
			}

			if invariants && b.growthLeft != 0 {
				panic(fmt.Sprintf("invariant failed: growthLeft is unexpectedly non-zero: %d\n%#v", b.growthLeft, b))
			}

			b.rehash(m)

			// We may have split the bucket in which case we have to
			// re-determine which bucket the key resides on. This
			// determination is quick in comparison to rehashing, resizing,
			// and splitting, so just always do it.
			b = m.mutableBucket(h)

			// Note that we don't have to restart the entire Put process as we
			// know the key doesn't exist in the map.
			b.uncheckedPut(h, key, value)
			b.used++
			m.used++
			b.checkInvariants(m)
			return
		}
	}
}

// Get retrieves the value from the map for the specified key, returning
// ok=false if the key is not present.
func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	h := m.hash(noescape(unsafe.Pointer(&key)), m.seed)
	b := m.bucket(h)

	// NB: Unlike the abseil swiss table implementation which uses a common
	// find routine for Get, Put, and Delete, we have to manually inline the
	// find routine for performance.

	// To find the location of a key in the table, we compute hash(key). From
	// h1(hash(key)) and the capacity, we construct a probeSeq that visits
	// every group of slots in some interesting order.
	//
	// We walk through these indices. At each index, we select the entire group
	// starting with that index and extract potential candidates: occupied slots
	// with a control byte equal to h2(hash(key)). If we find an empty slot in the
	// group, we stop and return an error. The key at candidate slot y is compared
	// with key; if key == m.slots[y].key we are done and return y; otherwise we
	// continue to the next probe index. Tombstones (ctrlDeleted) effectively
	// behave like full slots that never match the value we're looking for.
	//
	// The h2 bits ensure when we compare a key we are likely to have actually
	// found the object. That is, the chance is low that keys compare false. Thus,
	// when we search for an object, we are unlikely to call == many times. This
	// likelyhood can be analyzed as follows (assuming that h2 is a random enough
	// hash function).
	//
	// Let's assume that there are k "wrong" objects that must be examined in a
	// probe sequence. For example, when doing a find on an object that is in the
	// table, k is the number of objects between the start of the probe sequence
	// and the final found object (not including the final found object). The
	// expected number of objects with an h2 match is then k/128. Measurements and
	// analysis indicate that even at high load factors, k is less than 32,
	// meaning that the number of false positive comparisons we must perform is
	// less than 1/8 per find.
	seq := makeProbeSeq(h1(h), b.groupMask)
	for ; ; seq = seq.next() {
		g := b.groups.At(uintptr(seq.offset))
		match := g.ctrls.matchH2(h2(h))

		for match != 0 {
			i := match.first()
			slot := g.slots.At(i)
			if key == slot.key {
				return slot.value, true
			}
			match = match.removeFirst()
		}

		match = g.ctrls.matchEmpty()
		if match != 0 {
			return value, false
		}
	}
}

// Delete deletes the entry corresponding to the specified key from the map.
// It is a noop to delete a non-existent key.
func (m *Map[K, V]) Delete(key K) {
	// Delete is find composed with "deleted at": we perform find(key), and
	// then delete at the resulting slot if found.
	h := m.hash(noescape(unsafe.Pointer(&key)), m.seed)
	b := m.mutableBucket(h)

	// NB: Unlike the abseil swiss table implementation which uses a common
	// find routine for Get, Put, and Delete, we have to manually inline the
	// find routine for performance.
	seq := makeProbeSeq(h1(h), b.groupMask)
	for ; ; seq = seq.next() {
		g := b.groups.At(uintptr(seq.offset))
		match := g.ctrls.matchH2(h2(h))

		for match != 0 {
			i := match.first()
			s := g.slots.At(i)
			if key == s.key {
				b.used--
				m.used--
				if m.used == 0 {
					// Reset the hash seed to make it more difficult for attackers to
					// repeatedly trigger hash collisions. See issue
					// https://github.com/golang/go/issues/25237.
					m.seed = uintptr(fastrand64())
				}
				*s = slot[K, V]{}

				// Only a full group can appear in the middle of a probe
				// sequence (a group with at least one empty slot terminates
				// probing). Once a group becomes full, it stays full until
				// rehashing/resizing. So if the group isn't full now, we can
				// simply remove the element. Otherwise, we create a tombstone
				// to mark the slot as deleted.
				if g.ctrls.matchEmpty() != 0 {
					g.ctrls.Set(i, ctrlEmpty)
					b.growthLeft++
				} else {
					g.ctrls.Set(i, ctrlDeleted)
				}
				b.checkInvariants(m)
				return
			}
			match = match.removeFirst()
		}

		match = g.ctrls.matchEmpty()
		if match != 0 {
			b.checkInvariants(m)
			return
		}
	}
}

// Clear deletes all entries from the map resulting in an empty map.
func (m *Map[K, V]) Clear() {
	m.buckets(0, func(b *bucket[K, V]) bool {
		for i := uint32(0); i <= b.groupMask; i++ {
			g := b.groups.At(uintptr(i))
			g.ctrls.SetEmpty()
			for j := uint32(0); j < groupSize; j++ {
				*g.slots.At(j) = slot[K, V]{}
			}
		}

		b.used = 0
		b.resetGrowthLeft()
		return true
	})

	// Reset the hash seed to make it more difficult for attackers to
	// repeatedly trigger hash collisions. See issue
	// https://github.com/golang/go/issues/25237.
	m.seed = uintptr(fastrand64())
	m.used = 0
}

// All calls yield sequentially for each key and value present in the map. If
// yield returns false, range stops the iteration. The map can be mutated
// during iteration, though there is no guarantee that the mutations will be
// visible to the iteration.
//
// TODO(peter): The naming of All and its signature are meant to conform to
// the range-over-function Go proposal. When that proposal is accepted (which
// seems likely), we'll be able to iterate over the map by doing:
//
//	for k, v := range m.All {
//	  fmt.Printf("%v: %v\n", k, v)
//	}
//
// See https://github.com/golang/go/issues/61897.
func (m *Map[K, V]) All(yield func(key K, value V) bool) {
	// Randomize iteration order by starting iteration at a random bucket and
	// within each bucket at a random offset.
	offset := uintptr(fastrand64())
	m.buckets(offset>>32, func(b *bucket[K, V]) bool {
		if b.used == 0 {
			return true
		}

		// Snapshot the groups, and groupMask so that iteration remains valid
		// if the map is resized during iteration.
		groups := b.groups
		groupMask := b.groupMask

		offset32 := uint32(offset)
		for i := uint32(0); i <= groupMask; i++ {
			g := groups.At(uintptr((i + offset32) & groupMask))
			// TODO(peter): Skip over groups that are composed of only empty
			// or deleted slots using matchEmptyOrDeleted() and counting the
			// number of bits set.
			for j := uint32(0); j < groupSize; j++ {
				k := (j + offset32) & (groupSize - 1)
				// Match full entries which have a high-bit of zero.
				if (g.ctrls.Get(k) & ctrlEmpty) != ctrlEmpty {
					slot := g.slots.At(k)
					if !yield(slot.key, slot.value) {
						return false
					}
				}
			}
		}
		return true
	})
}

// GoString implements the fmt.GoStringer interface which is used when
// formatting using the "%#v" format specifier.
func (m *Map[K, V]) GoString() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "used=%d  global-depth=%d  bucket-count=%d\n", m.used, m.globalDepth(), m.bucketCount())
	m.buckets(0, func(b *bucket[K, V]) bool {
		fmt.Fprintf(&buf, "bucket %d (%p): local-depth=%d\n", b.index, b, b.localDepth)
		b.goFormat(&buf)
		return true
	})
	return buf.String()
}

// Len returns the number of entries in the map.
func (m *Map[K, V]) Len() int {
	return m.used
}

// capacity returns the total capacity of all map buckets.
func (m *Map[K, V]) capacity() int {
	var capacity int
	m.buckets(0, func(b *bucket[K, V]) bool {
		capacity += int(b.capacity)
		return true
	})
	return capacity
}

// bucket returns the bucket corresponding to hash value h.
func (m *Map[K, V]) bucket(h uintptr) *bucket[K, V] {
	// NB: It is faster to check for the single bucket case using a
	// conditional than to index into the directory.
	if m.globalShift == 0 {
		return &m.bucket0
	}
	// When shifting by a variable amount the Go compiler inserts overflow
	// checks that the shift is less than the maximum allowed (32 or 64).
	// Masking the shift amount allows overflow checks to be elided.
	return m.dir.At(h >> (m.globalShift & shiftMask))
}

func (m *Map[K, V]) mutableBucket(h uintptr) *bucket[K, V] {
	// NB: It is faster to check for the single bucket case using a
	// conditional than to to index into the directory.
	if m.globalShift == 0 {
		return &m.bucket0
	}
	// When shifting by a variable amount the Go compiler inserts overflow
	// checks that the shift is less than the maximum allowed (32 or 64).
	// Masking the shift amount allows overflow checks to be elided.
	b := m.dir.At(h >> (m.globalShift & shiftMask))
	// The mutable bucket is the one located at m.dir[b.index]. This will
	// usually be either the current bucket b, or the immediately preceding
	// bucket which is usually in the same cache line.
	return m.dir.At(uintptr(b.index))
}

// buckets calls yield sequentially for each bucket in the map. If yield
// returns false, iteration stops. Offset specifies the bucket to start
// iteration at (used to randomize iteration order).
func (m *Map[K, V]) buckets(offset uintptr, yield func(b *bucket[K, V]) bool) {
	b := m.dir.At(offset & uintptr(m.bucketCount()-1))
	// We iterate over the first bucket in a logical group of buckets (i.e.
	// buckets which share bucket.groups). The first bucket has the accurate
	// bucket.used field and those are also the buckets that are stepped
	// through using bucketStep().
	b = m.dir.At(uintptr(b.index))

	// Loop termination is handled by remembering the start bucket index and
	// exiting when it is reached again. Note that the startIndex needs to be
	// adjusted to account for the directory growing during iteration (i.e.
	// due to a mutation), so we remember the starting global depth as well in
	// order to perform that adjustment. Whenever the directory grows by
	// doubling, every existing bucket index will be doubled.
	startIndex := b.index
	startGlobalDepth := m.globalDepth()

	for {
		originalGlobalDepth := m.globalDepth()
		originalLocalDepth := b.localDepth
		originalIndex := b.index

		if !yield(b) {
			break
		}

		// The size of the directory can grow if the yield function mutates
		// the map.  We want to iterate over each bucket once, and if a bucket
		// splits while we're iterating over it we want to skip over all of
		// the buckets newly split from the one we're iterating over. We do
		// this by snapshotting the bucket's local depth and using the
		// snapshotted local depth to compute the bucket step.
		//
		// Note that b.index will also change if the directory grows. Consider
		// the directory below with a globalDepth of 2 containing 4 buckets,
		// each of which has a localDepth of 2.
		//
		//    dir   b.index   b.localDepth
		//	+-----+---------+--------------+
		//	|  00 |       0 |            2 |
		//	+-----+---------+--------------+
		//	|  01 |       1 |            2 |
		//	+-----+---------+--------------+
		//	|  10 |       2 |            2 | <--- iteration point
		//	+-----+---------+--------------+
		//	|  11 |       3 |            2 |
		//	+-----+---------+--------------+
		//
		// If the directory grows during iteration, the index of the bucket
		// we're iterating over will change. If the bucket we're iterating
		// over split, then the local depth will have increased. Notice how
		// the bucket that was previously at index 1 now is at index 2 and is
		// pointed to by 2 directory entries: 010 and 011. The bucket being
		// iterated over which was previously at index 2 is now at index 4.
		// Iteration within a bucket takes a snapshot of the controls and
		// slots to make sure we don't miss keys during iteration or iterate
		// over keys more than once. But we also need to take care of the case
		// where the bucket we're iterating over splits. In this case, we need
		// to skip over the bucket at index 5 which can be done by computing
		// the bucketStep using the bucket's depth prior to calling yield
		// which in this example will be 1<<(3-2)==2.
		//
		//    dir   b.index   b.localDepth
		//	+-----+---------+--------------+
		//	| 000 |       0 |            2 |
		//	+-----+         |              |
		//	| 001 |         |              |
		//	+-----+---------+--------------+
		//	| 010 |       2 |            2 |
		//	+-----+         |              |
		//	| 011 |         |              |
		//	+-----+---------+--------------+
		//	| 100 |       4 |            3 |
		//	+-----+---------+--------------+
		//	| 101 |       5 |            3 |
		//	+-----+---------+--------------+
		//	| 110 |       6 |            2 |
		//	+-----+         |              |
		//	| 111 |         |              |
		//	+-----+---------+--------------+

		// After calling yield, b is no longer valid. We determine the next
		// bucket to iterate over using the b.index we cached before calling
		// yield and adjusting for any directory growth that happened during
		// the yield call.
		i := adjustBucketIndex(originalIndex, m.globalDepth(), originalGlobalDepth)
		i += bucketStep(m.globalDepth(), originalLocalDepth)
		i &= (m.bucketCount() - 1)

		// Similar to the adjustment for b's index, we compute the starting
		// bucket's new index accounting for directory growth.
		adjustedStartIndex := adjustBucketIndex(startIndex, m.globalDepth(), startGlobalDepth)
		if i == adjustedStartIndex {
			break
		}

		b = m.dir.At(uintptr(i))
	}
}

// globalDepth returns the number of bits from the top of the hash to use for
// indexing in the buckets directory.
func (m *Map[K, V]) globalDepth() uint32 {
	if m.globalShift == 0 {
		return 0
	}
	return ptrBits - m.globalShift
}

// bucketCount returns the number of buckets in the buckets directory.
func (m *Map[K, V]) bucketCount() uint32 {
	const shiftMask = 31
	return uint32(1) << (m.globalDepth() & shiftMask)
}

// bucketStep is the number of buckets to step over in the buckets directory
// to reach the next different bucket. A bucket occupies 1 or more contiguous
// entries in the buckets directory specified by the range:
//
//	[b.index:b.index+bucketStep(m.globalDepth(), b.localDepth)]
func bucketStep(globalDepth, localDepth uint32) uint32 {
	const shiftMask = 31
	return uint32(1) << ((globalDepth - localDepth) & shiftMask)
}

// adjustBucketIndex adjusts the index of a bucket to account for the growth
// of the directory where index was captured at originalGlobalDepth and we're
// computing where that index will reside in the directory at
// currentGlobalDepth.
func adjustBucketIndex(index, currentGlobalDepth, originalGlobalDepth uint32) uint32 {
	return index * (1 << (currentGlobalDepth - originalGlobalDepth))
}

// installBucket installs a bucket into the buckets directory, overwriting
// every index in the range of entries the bucket occupies.
func (m *Map[K, V]) installBucket(b *bucket[K, V]) *bucket[K, V] {
	step := bucketStep(m.globalDepth(), b.localDepth)
	for i := uint32(0); i < step; i++ {
		*m.dir.At(uintptr(b.index + i)) = *b
	}
	return m.dir.At(uintptr(b.index))
}

// growDirectory grows the directory slice to 1<<newGlobalDepth buckets. Grow
// directory returns the new index location for the bucket specified by index.
func (m *Map[K, V]) growDirectory(newGlobalDepth, index uint32) (newIndex uint32) {
	if invariants && newGlobalDepth > 32 {
		panic(fmt.Sprintf("invariant failed: expectedly large newGlobalDepth %d->%d",
			m.globalDepth(), newGlobalDepth))
	}

	newDir := makeUnsafeSlice(make([]bucket[K, V], 1<<newGlobalDepth))

	// NB: It would be more natural to use Map.buckets() here, but that
	// routine uses b.index during iteration which we're mutating in the loop
	// below.

	lastIndex := uint32(math.MaxUint32)
	setNewIndex := true
	for i, j, n := uint32(0), uint32(0), m.bucketCount(); i < n; i++ {
		b := m.dir.At(uintptr(i))
		if b.index == lastIndex {
			continue
		}
		lastIndex = b.index

		if b.index == index && setNewIndex {
			newIndex = j
			setNewIndex = false
		}
		b.index = j
		step := bucketStep(newGlobalDepth, b.localDepth)
		for k := uint32(0); k < step; k++ {
			*newDir.At(uintptr(j + k)) = *b
		}
		j += step
	}

	// Zero out bucket0 if we're growing from 1 bucket (which uses bucket0) to
	// more than 1 bucket.
	if m.globalShift == 0 {
		m.bucket0 = bucket[K, V]{}
	}
	m.dir = newDir
	m.globalShift = ptrBits - newGlobalDepth

	m.checkInvariants()
	return newIndex
}

// checkInvariants verifies the internal consistency of the map's structure,
// checking conditions that should always be true for a correctly functioning
// map. If any of these invariants are violated, it panics, indicating a bug
// in the map implementation.
func (m *Map[K, V]) checkInvariants() {
	if invariants {
		if m.globalShift == 0 {
			if m.dir.ptr != unsafe.Pointer(&m.bucket0) {
				panic(fmt.Sprintf("directory (%p) does not point to bucket0 (%p)", m.dir.ptr, &m.bucket0))
			}
			if m.bucket0.localDepth != 0 {
				panic(fmt.Sprintf("expected local-depth=0, but found %d", m.bucket0.localDepth))
			}
		} else {
			for i, n := uint32(0), m.bucketCount(); i < n; i++ {
				b := m.dir.At(uintptr(i))
				if b == nil {
					panic(fmt.Sprintf("dir[%d]: nil bucket", i))
				}
				if b.localDepth > m.globalDepth() {
					panic(fmt.Sprintf("dir[%d]: local-depth=%d is greater than global-depth=%d",
						i, b.localDepth, m.globalDepth()))
				}
				n := uint32(1) << (m.globalDepth() - b.localDepth)
				if i < b.index || i >= b.index+n {
					panic(fmt.Sprintf("dir[%d]: out of expected range [%d,%d)", i, b.index, b.index+n))
				}
			}
		}
	}
}

func (b *bucket[K, V]) close(allocator Allocator[K, V]) {
	if b.capacity > 0 {
		allocator.Free(b.groups.Slice(0, uintptr(b.groupMask+1)))
		b.capacity = 0
		b.used = 0
	}
	b.groups = makeUnsafeSlice([]Group[K, V](nil))
	b.groupMask = 0
}

// tombstones returns the number of deleted (tombstone) entries in the bucket.
// A tombstone is a slot that has been deleted but is still considered
// occupied so as not to violate the probing invariant.
func (b *bucket[K, V]) tombstones() uint32 {
	return (b.capacity*maxAvgGroupLoad)/groupSize - b.used - b.growthLeft
}

// uncheckedPut inserts an entry known not to be in the table. Used by Put
// after it has failed to find an existing entry to overwrite duration
// insertion.
func (b *bucket[K, V]) uncheckedPut(h uintptr, key K, value V) {
	if invariants && b.growthLeft == 0 {
		panic(fmt.Sprintf("invariant failed: growthLeft is unexpectedly 0\n%#v", b))
	}

	// Given key and its hash hash(key), to insert it, we construct a
	// probeSeq, and use it to find the first group with an unoccupied (empty
	// or deleted) slot. We place the key/value into the first such slot in
	// the group and mark it as full with key's H2.
	seq := makeProbeSeq(h1(h), b.groupMask)
	for ; ; seq = seq.next() {
		g := b.groups.At(uintptr(seq.offset))
		match := g.ctrls.matchEmptyOrDeleted()
		if match != 0 {
			i := match.first()
			slot := g.slots.At(i)
			slot.key = key
			slot.value = value
			if g.ctrls.Get(i) == ctrlEmpty {
				b.growthLeft--
			}
			g.ctrls.Set(i, ctrl(h2(h)))
			return
		}
	}
}

func (b *bucket[K, V]) rehash(m *Map[K, V]) {
	// Rehash in place if we can recover >= 1/3 of the capacity. Note that
	// this heuristic differs from Abseil's and was experimentally determined
	// to balance performance on the PutDelete benchmark vs achieving a
	// reasonable load-factor.
	//
	// Abseil notes that in the worst case it takes ~4 Put/Delete pairs to
	// create a single tombstone. Rehashing in place is significantly faster
	// than resizing because the common case is that elements remain in their
	// current location. The performance of rehashInPlace is dominated by
	// recomputing the hash of every key. We know how much space we're going
	// to reclaim because every tombstone will be dropped and we're only
	// called if we've reached the thresold of capacity/8 empty slots. So the
	// number of tomstones is capacity*7/8 - used.
	if b.capacity > groupSize && b.tombstones() >= b.capacity/3 {
		b.rehashInPlace(m)
		return
	}

	// If the newCapacity is larger than the maxBucketCapacity split the
	// bucket instead of resizing. Each of the new buckets will be the same
	// size as the current bucket.
	newCapacity := 2 * b.capacity
	if newCapacity > m.maxBucketCapacity {
		b.split(m)
		return
	}

	b.resize(m, newCapacity)
}

func (b *bucket[K, V]) init(m *Map[K, V], newCapacity uint32) {
	if newCapacity < groupSize {
		newCapacity = groupSize
	}

	if invariants && newCapacity&(newCapacity-1) != 0 {
		panic(fmt.Sprintf("invariant failed: bucket size %d is not a power of 2", newCapacity))
	}

	b.capacity = newCapacity
	b.groupMask = b.capacity/groupSize - 1
	b.groups = makeUnsafeSlice(m.allocator.Alloc(int(b.groupMask + 1)))

	for i := uint32(0); i <= b.groupMask; i++ {
		g := b.groups.At(uintptr(i))
		g.ctrls.SetEmpty()
	}

	b.resetGrowthLeft()
}

// resize the capacity of the table by allocating a bigger array and
// uncheckedPutting each element of the table into the new array (we know that
// no insertion here will Put an already-present value), and discard the old
// backing array.
func (b *bucket[K, V]) resize(m *Map[K, V], newCapacity uint32) {
	if invariants && b != m.dir.At(uintptr(b.index)) {
		panic(fmt.Sprintf("invariant failed: attempt to resize bucket %p, but it is not at Map.dir[%d/%p]",
			b, b.index, m.dir.At(uintptr(b.index))))
	}

	oldGroups := b.groups
	oldGroupMask := b.groupMask
	oldCapacity := b.capacity
	b.init(m, newCapacity)

	if oldCapacity > 0 {
		for i := uint32(0); i <= oldGroupMask; i++ {
			g := oldGroups.At(uintptr(i))
			for j := uint32(0); j < groupSize; j++ {
				if (g.ctrls.Get(j) & ctrlEmpty) == ctrlEmpty {
					continue
				}
				slot := g.slots.At(j)
				h := m.hash(noescape(unsafe.Pointer(&slot.key)), m.seed)
				b.uncheckedPut(h, slot.key, slot.value)
			}
		}

		m.allocator.Free(oldGroups.Slice(0, uintptr(oldGroupMask+1)))
	}

	b = m.installBucket(b)
	b.checkInvariants(m)
}

// split divides the entries in a bucket between the receiver and a new bucket
// of the same size, and then installs the new bucket into the buckets
// directory, growing the buckets directory if necessary.
func (b *bucket[K, V]) split(m *Map[K, V]) {
	if invariants && b != m.dir.At(uintptr(b.index)) {
		panic(fmt.Sprintf("invariant failed: attempt to split bucket %p, but it is not at Map.dir[%d/%p]",
			b, b.index, m.dir.At(uintptr(b.index))))
	}

	// Create the new bucket as a clone of the bucket being split. If we're
	// splitting bucket0 we need to allocate a *bucket[K, V] for scratch
	// space. Otherwise we use bucket0 as the scratch space.
	var newb *bucket[K, V]
	if m.globalShift == 0 {
		newb = &bucket[K, V]{}
	} else {
		newb = &m.bucket0
	}
	*newb = bucket[K, V]{
		localDepth: b.localDepth,
		index:      b.index,
	}
	newb.init(m, b.capacity)

	// Divide the records between the 2 buckets (b and newb). This is done by
	// examining the new bit in the hash that will be added to the bucket
	// index. If that bit is 0 the record stays in bucket b. If that bit is 1
	// the record is moved to bucket newb. We're relying on the bucket b
	// staying earlier in the directory than newb after the directory is
	// grown.
	mask := uintptr(1) << (ptrBits - (b.localDepth + 1))
	for i := uint32(0); i <= b.groupMask; i++ {
		g := b.groups.At(uintptr(i))
		for j := uint32(0); j < groupSize; j++ {
			if (g.ctrls.Get(j) & ctrlEmpty) == ctrlEmpty {
				continue
			}

			s := g.slots.At(j)
			h := m.hash(noescape(unsafe.Pointer(&s.key)), m.seed)
			if (h & mask) == 0 {
				// Nothing to do, the record is staying in b.
				continue
			}

			// Insert the record into newb.
			newb.uncheckedPut(h, s.key, s.value)
			newb.used++

			// Delete the record from b.
			if g.ctrls.matchEmpty() != 0 {
				g.ctrls.Set(j, ctrlEmpty)
				b.growthLeft++
			} else {
				g.ctrls.Set(j, ctrlDeleted)
			}

			*s = slot[K, V]{}
			b.used--
		}
	}

	if newb.used == 0 {
		// We didn't move any records to the new bucket. Either
		// maxBucketCapacity is too small and we got unlucky, or we have a
		// degenerate hash function (e.g. one that returns a constant in the
		// high bits).
		m.maxBucketCapacity = 2 * m.maxBucketCapacity
		newb.close(m.allocator)
		*newb = bucket[K, V]{}
		b.resize(m, 2*b.capacity)
		return
	}

	if b.used == 0 {
		// We moved all of the records to the new bucket (note the two
		// conditions are equivalent and both are present merely for clarity).
		// Similar to the above, bump maxBucketCapacity and resize the bucket
		// rather than splitting. We'll replace the old bucket with the new
		// bucket in the directory.
		m.maxBucketCapacity = 2 * m.maxBucketCapacity
		b.close(m.allocator)
		newb = m.installBucket(newb)
		m.checkInvariants()
		newb.resize(m, 2*newb.capacity)
		return
	}

	// We need to ensure bucket b, which we evacuated records from, has empty
	// slots as we may be inserting into it. We also want to drop any
	// tombstones that may have been left in bucket to ensure lookups for
	// non-existent keys don't have to traverse long probe chains. With a good
	// hash function, 50% of the entries in b should have been moved to newb,
	// so we should be able to drop tombstones corresponding to ~50% of the
	// entries.
	b.rehashInPlace(m)

	// Grow the directory if necessary.
	if b.localDepth >= m.globalDepth() {
		// When the directory grows b will be invalidated. We pass in b's
		// index so that growDirectory will return the new index it resides
		// at.
		i := m.growDirectory(b.localDepth+1, b.index)
		b = m.dir.At(uintptr(i))
	}

	// Complete the split by incrementing the local depth for the 2 buckets
	// and installing the new bucket in the directory.
	b.localDepth++
	m.installBucket(b)
	newb.localDepth = b.localDepth
	newb.index = b.index + bucketStep(m.globalDepth(), b.localDepth)
	m.installBucket(newb)
	*newb = bucket[K, V]{}

	if invariants {
		m.checkInvariants()
		m.buckets(0, func(b *bucket[K, V]) bool {
			b.checkInvariants(m)
			return true
		})
	}
}

func (b *bucket[K, V]) rehashInPlace(m *Map[K, V]) {
	if invariants && b != m.dir.At(uintptr(b.index)) {
		panic(fmt.Sprintf("invariant failed: attempt to rehash bucket %p, but it is not at Map.dir[%d/%p]",
			b, b.index, m.dir.At(uintptr(b.index))))
	}
	if b.capacity == 0 {
		return
	}

	// We want to drop all of the deletes in place. We first walk over the
	// control bytes and mark every DELETED slot as EMPTY and every FULL slot
	// as DELETED. Marking the DELETED slots as EMPTY has effectively dropped
	// the tombstones, but we fouled up the probe invariant. Marking the FULL
	// slots as DELETED gives us a marker to locate the previously FULL slots.

	// Mark all DELETED slots as EMPTY and all FULL slots as DELETED.
	for i := uint32(0); i <= b.groupMask; i++ {
		b.groups.At(uintptr(i)).ctrls.convertNonFullToEmptyAndFullToDeleted()
	}

	// Now we walk over all of the DELETED slots (a.k.a. the previously FULL
	// slots). For each slot we find the first probe group we can place the
	// element in which reestablishes the probe invariant. Note that as this
	// loop proceeds we have the invariant that there are no DELETED slots in
	// the range [0, i). We may move the element at i to the range [0, i) if
	// that is where the first group with an empty slot in its probe chain
	// resides, but we never set a slot in [0, i) to DELETED.
	for i := uint32(0); i <= b.groupMask; i++ {
		g := b.groups.At(uintptr(i))
		for j := uint32(0); j < groupSize; j++ {
			if g.ctrls.Get(j) != ctrlDeleted {
				continue
			}

			s := g.slots.At(j)
			h := m.hash(noescape(unsafe.Pointer(&s.key)), m.seed)
			seq := makeProbeSeq(h1(h), b.groupMask)
			desiredOffset := seq.offset

			var targetGroup *Group[K, V]
			var target uint32
			for ; ; seq = seq.next() {
				targetGroup = b.groups.At(uintptr(seq.offset))
				if match := targetGroup.ctrls.matchEmptyOrDeleted(); match != 0 {
					target = match.first()
					break
				}
			}

			switch {
			case i == desiredOffset:
				// If the target index falls within the first probe group
				// then we don't need to move the element as it already
				// falls in the best probe position.
				g.ctrls.Set(j, ctrl(h2(h)))

			case targetGroup.ctrls.Get(target) == ctrlEmpty:
				// The target slot is empty. Transfer the element to the
				// empty slot and mark the slot at index i as empty.
				targetGroup.ctrls.Set(target, ctrl(h2(h)))
				*targetGroup.slots.At(target) = *s
				*s = slot[K, V]{}
				g.ctrls.Set(j, ctrlEmpty)

			case targetGroup.ctrls.Get(target) == ctrlDeleted:
				// The slot at target has an element (i.e. it was FULL).
				// We're going to swap our current element with that
				// element and then repeat processing of index i which now
				// holds the element which was at target.
				targetGroup.ctrls.Set(target, ctrl(h2(h)))
				t := targetGroup.slots.At(target)
				*s, *t = *t, *s
				// Repeat processing of the j'th slot which now holds a
				// new key/value.
				j--

			default:
				panic(fmt.Sprintf("ctrl at position %d (%02x) should be empty or deleted",
					target, targetGroup.ctrls.Get(target)))
			}
		}
	}

	b.resetGrowthLeft()
	b.growthLeft -= b.used

	b.checkInvariants(m)
}

func (b *bucket[K, V]) resetGrowthLeft() {
	var growthLeft int
	if b.capacity <= groupSize {
		// If the map fits in a single group then we're able to fill all of
		// the slots except 1 (an empty slot is needed to terminate find
		// operations).
		growthLeft = int(b.capacity - 1)
	} else {
		growthLeft = int((b.capacity * maxAvgGroupLoad) / groupSize)
	}
	if growthLeft < 0 {
		growthLeft = 0
	}
	b.growthLeft = uint32(growthLeft)
}

// TODO(peter): Should this be removed? It was useful for debugging a
// performance problem with BenchmarkGetMiss.
func (b *bucket[K, V]) fullGroups() uint32 {
	var full uint32
	for i := uint32(0); i <= b.groupMask; i++ {
		g := b.groups.At(uintptr(i))
		if g.ctrls.matchEmpty() == 0 {
			full++
		}
	}
	return full
}

func (b *bucket[K, V]) checkInvariants(m *Map[K, V]) {
	if invariants {
		// For every non-empty slot, verify we can retrieve the key using Get.
		// Count the number of used and deleted slots.
		var used uint32
		var deleted uint32
		var empty uint32
		for i := uint32(0); i <= b.groupMask; i++ {
			g := b.groups.At(uintptr(i))
			for j := uint32(0); j < groupSize; j++ {
				c := g.ctrls.Get(j)
				switch {
				case c == ctrlDeleted:
					deleted++
				case c == ctrlEmpty:
					empty++
				default:
					slot := g.slots.At(j)
					if _, ok := m.Get(slot.key); !ok {
						h := m.hash(noescape(unsafe.Pointer(&slot.key)), m.seed)
						panic(fmt.Sprintf("invariant failed: slot(%d/%d): %v not found [h2=%02x h1=%07x]\n%#v",
							i, j, slot.key, h2(h), h1(h), b))
					}
					used++
				}
			}
		}

		if used != b.used {
			panic(fmt.Sprintf("invariant failed: found %d used slots, but used count is %d\n%#v",
				used, b.used, b))
		}

		growthLeft := (b.capacity*maxAvgGroupLoad)/groupSize - b.used - deleted
		if growthLeft != b.growthLeft {
			panic(fmt.Sprintf("invariant failed: found %d growthLeft, but expected %d\n%#v",
				b.growthLeft, growthLeft, b))
		}
		if deleted != b.tombstones() {
			panic(fmt.Sprintf("invariant failed: found %d tombstones, but expected %d\n%#v",
				deleted, b.tombstones(), b))
		}

		if empty == 0 {
			panic(fmt.Sprintf("invariant failed: found no empty slots (violates probe invariant)\n%#v", b))
		}
	}
}

// GoString implements the fmt.GoStringer interface which is used when
// formatting using the "%#v" format specifier.
func (b *bucket[K, V]) GoString() string {
	var buf strings.Builder
	b.goFormat(&buf)
	return buf.String()
}

func (b *bucket[K, V]) goFormat(w io.Writer) {
	fmt.Fprintf(w, "capacity=%d  used=%d  growth-left=%d\n", b.capacity, b.used, b.growthLeft)
	for i := uint32(0); i <= b.groupMask; i++ {
		g := b.groups.At(uintptr(i))
		fmt.Fprintf(w, "  group %d\n", i)
		for j := uint32(0); j < groupSize; j++ {
			switch c := g.ctrls.Get(j); c {
			case ctrlEmpty:
				fmt.Fprintf(w, "    %d: %02x [empty]\n", j, c)
			case ctrlDeleted:
				fmt.Fprintf(w, "    %d: %02x [deleted]\n", j, c)
			default:
				slot := g.slots.At(j)
				fmt.Fprintf(w, "    %d: %02x [%v:%v]\n", j, c, slot.key, slot.value)
			}
		}
	}
}

// bitset represents a set of slots within a group.
//
// The underlying representation uses one byte per slot, where each byte is
// either 0x80 if the slot is part of the set or 0x00 otherwise. This makes it
// convenient to calculate for an entire group at once (e.g. see matchEmpty).
type bitset uint64

// first assumes that only the MSB of each control byte can be set (e.g. bitset
// is the result of matchEmpty or similar) and returns the relative index of the
// first control byte in the group that has the MSB set.
//
// Returns 8 if the bitset is 0.
// Returns groupSize if the bitset is empty.
func (b bitset) first() uint32 {
	return uint32(bits.TrailingZeros64(uint64(b))) >> 3
}

// removeFirst removes the first set bit (that is, resets the least significant set bit to 0).
func (b bitset) removeFirst() bitset {
	return b & (b - 1)
}

func (b bitset) String() string {
	var buf strings.Builder
	buf.Grow(groupSize)
	for i := 0; i < groupSize; i++ {
		if (b & (bitset(0x80) << (i << 3))) != 0 {
			buf.WriteString("1")
		} else {
			buf.WriteString("0")
		}
	}
	return buf.String()
}

// Each slot in the hash table has a control byte which can have one of three
// states: empty, deleted, and full. They have the following bit patterns:
//
//	  empty: 1 0 0 0 0 0 0 0
//	deleted: 1 1 1 1 1 1 1 0
//	   full: 0 h h h h h h h  // h represents the H1 hash bits
type ctrl uint8

// ctrlGroup is a fixed size array of groupSize control bytes stored in a
// uint64.
//
// See Get() and Set() methods implemented in endian_*.go.
type ctrlGroup uint64

// SetEmpty sets all the control bytes to empty.
func (g *ctrlGroup) SetEmpty() {
	*g = ctrlGroup(bitsetEmpty)
}

// matchH2 returns the set of slots which are full and for which the 7-bit hash
// matches the given value. May return false positives.
func (g *ctrlGroup) matchH2(h uintptr) bitset {
	// NB: This generic matching routine produces false positive matches when
	// h is 2^N and the control bytes have a seq of 2^N followed by 2^N+1. For
	// example: if ctrls==0x0302 and h=02, we'll compute v as 0x0100. When we
	// subtract off 0x0101 the first 2 bytes we'll become 0xffff and both be
	// considered matches of h. The false positive matches are not a problem,
	// just a rare inefficiency. Note that they only occur if there is a real
	// match and never occur on ctrlEmpty, or ctrlDeleted. The subsequent key
	// comparisons ensure that there is no correctness issue.
	v := uint64(*g) ^ (bitsetLSB * uint64(h))
	return bitset(((v - bitsetLSB) &^ v) & bitsetMSB)
}

// matchEmpty returns the set of slots in the group that are empty.
func (g *ctrlGroup) matchEmpty() bitset {
	// An empty slot is   1000 0000
	// A deleted slot is  1111 1110
	// A full slot is     0??? ????
	//
	// A slot is empty iff bit 7 is set and bit 1 is not. We could select any
	// of the other bits here (e.g. v << 1 would also work).
	v := uint64(*g)
	return bitset((v &^ (v << 6)) & bitsetMSB)
}

// matchEmptyOrDeleted returns the set of slots in the group that are empty or
// deleted.
func (g *ctrlGroup) matchEmptyOrDeleted() bitset {
	// An empty slot is  1000 0000
	// A deleted slot is 1111 1110
	// A full slot is    0??? ????
	//
	// A slot is empty or deleted iff bit 7 is set and bit 0 is not.
	v := uint64(*g)
	return bitset((v &^ (v << 7)) & bitsetMSB)
}

// convertNonFullToEmptyAndFullToDeleted converts deleted control bytes in a
// group to empty control bytes, and control bytes indicating full slots to
// deleted control bytes.
func (g *ctrlGroup) convertNonFullToEmptyAndFullToDeleted() {
	// An empty slot is     1000 0000
	// A deleted slot is    1111 1110
	// A full slot is       0??? ????
	//
	// We select the MSB, invert, add 1 if the MSB was set and zero out the low
	// bit.
	//
	//  - if the MSB was set (i.e. slot was empty, or deleted):
	//     v:             1000 0000
	//     ^v:            0111 1111
	//     ^v + (v >> 7): 1000 0000
	//     &^ bitsetLSB:  1000 0000 = empty slot.
	//
	// - if the MSB was not set (i.e. full slot):
	//     v:             0000 0000
	//     ^v:            1111 1111
	//     ^v + (v >> 7): 1111 1111
	//     &^ bitsetLSB:  1111 1110 = deleted slot.
	//
	v := uint64(*g) & bitsetMSB
	*g = ctrlGroup((^v + (v >> 7)) &^ bitsetLSB)
}

func (g *ctrlGroup) String() string {
	var buf strings.Builder
	buf.Grow(groupSize)

	for i := uint32(0); i < groupSize; i++ {
		fmt.Fprintf(&buf, "%02x ", g.Get(i))
	}
	return buf.String()
}

// slotGroup is a fixed size array of groupSize slots.
//
// The keys and values are stored interleaved in slots with a memory layout
// that looks like K/V/K/V/K/V/K/V.../K/V. An alternate layout would be have
// parallel arrays of keys and values: K/K/K/K.../K/V/V/V/V.../V. The latter
// has better space utilization if you have something like uint64 keys and
// bool values, but it is measurably slower at large map sizes. Below shows
// this perf hit with a Map[uint64,uint64]. The problem is due to the key and
// value being stored in separate cache lines.
//
// MapGetHit/swissMap/Int64/262144-10    9.31ns ± 3%  10.04ns ± 3%   +7.81%  (p=0.008 n=5+5)
// MapGetHit/swissMap/Int64/524288-10    16.7ns ± 1%   18.2ns ± 3%   +8.98%  (p=0.008 n=5+5)
// MapGetHit/swissMap/Int64/1048576-10   24.7ns ± 2%   27.6ns ± 0%  +11.74%  (p=0.008 n=5+5)
// MapGetHit/swissMap/Int64/2097152-10   33.3ns ± 1%   37.7ns ± 1%  +13.12%  (p=0.008 n=5+5)
// MapGetHit/swissMap/Int64/4194304-10   36.6ns ± 0%   43.0ns ± 1%  +17.37%  (p=0.008 n=5+5)
type slotGroup[K comparable, V any] struct {
	slots [groupSize]slot[K, V]
}

func (g *slotGroup[K, V]) At(i uint32) *slot[K, V] {
	return (*slot[K, V])(unsafe.Add(unsafe.Pointer(&g.slots[0]), uintptr(i)*unsafe.Sizeof(g.slots[0])))
}

// emptyCtrls is a singleton for a single empty groupSize set of controls.
var emptyCtrls = func() []ctrl {
	var v [groupSize]ctrl
	for i := uint32(0); i < groupSize; i++ {
		v[i] = ctrlEmpty
	}
	return v[:]
}()

// probeSeq maintains the state for a probe sequence that iterates through the
// groups in a bucket. The sequence is a triangular progression of the form
//
//	p(i) := (i^2 + i)/2 + hash (mod mask+1)
//
// The sequence effectively outputs the indexes of *groups*. The group
// machinery allows us to check an entire group with minimal branching.
//
// It turns out that this probe sequence visits every group exactly once if
// the number of groups is a power of two, since (i^2+i)/2 is a bijection in
// Z/(2^m). See https://en.wikipedia.org/wiki/Quadratic_probing
type probeSeq struct {
	mask   uint32
	offset uint32
	index  uint32
}

func makeProbeSeq(hash uintptr, mask uint32) probeSeq {
	return probeSeq{
		mask:   mask,
		offset: uint32(hash) & mask,
		index:  0,
	}
}

func (s probeSeq) next() probeSeq {
	s.index++
	s.offset = (s.offset + s.index) & s.mask
	return s
}

func (s probeSeq) String() string {
	return fmt.Sprintf("mask=%d offset=%d index=%d", s.mask, s.offset, s.index)
}

// Extracts the H1 portion of a hash: the 57 upper bits.
func h1(h uintptr) uintptr {
	return h >> 7
}

// Extracts the H2 portion of a hash: the 7 bits not used for h1.
//
// These are used as an occupied control byte.
func h2(h uintptr) uintptr {
	return h & 0x7f
}

// noescape hides a pointer from escape analysis.  noescape is
// the identity function but escape analysis doesn't think the
// output depends on the input.  noescape is inlined and currently
// compiles down to zero instructions.
// USE CAREFULLY!
//
//go:nosplit
//go:nocheckptr
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

// unsafeSlice provides semi-ergonomic limited slice-like functionality
// without bounds checking for fixed sized slices.
type unsafeSlice[T any] struct {
	ptr unsafe.Pointer
}

func makeUnsafeSlice[T any](s []T) unsafeSlice[T] {
	return unsafeSlice[T]{ptr: unsafe.Pointer(unsafe.SliceData(s))}
}

// At returns a pointer to the element at index i.
//
// The go:nocheckptr declaration is need to silence the runtime check in race
// builds that the memory for the returned *T is entirely contained within a
// single memory allocation. We are "safely" violating this requirement when
// access Groups.ctrls for the empty group. See unsafeConvertSlice for
// additional commentary.
//
//go:nocheckptr
func (s unsafeSlice[T]) At(i uintptr) *T {
	var t T
	return (*T)(unsafe.Add(s.ptr, unsafe.Sizeof(t)*i))
}

// Slice returns a Go slice akin to slice[start:end] for a Go builtin slice.
func (s unsafeSlice[T]) Slice(start, end uintptr) []T {
	return unsafe.Slice((*T)(s.ptr), end)[start:end]
}

// unsafeConvertSlice (unsafely) casts a []Src to a []Dest. The go:nocheckptr
// declaration is needed to silence the runtime check in race builds that the
// memory for the []Dest is entirely contained within a single memory
// allocation. We are "safely" violating this requirement when casting
// emptyCtrls (a []ctrl) to an empty group ([]*Group[K, V]). The reason this
// is safe is that we're never accessing Group.slots because the controls are
// all marked as empty.
//
//go:nocheckptr
func unsafeConvertSlice[Dest any, Src any](s []Src) []Dest {
	return unsafe.Slice((*Dest)(unsafe.Pointer(unsafe.SliceData(s))), len(s))
}

// noCopy may be added to structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
