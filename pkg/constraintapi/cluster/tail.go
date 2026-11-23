package cluster

import (
	"sort"
	"sync"
	"time"
)

// chunkOverhead is what one chunk costs beside its bytes.  it counts toward
// the byte limit so a tail of many small chunks stays inside it.
const chunkOverhead = 64

// tailChunk is one run of encoded records.  data is never changed, so a
// message can share it.
type tailChunk struct {
	first uint64
	count int
	at    time.Time
	data  []byte
}

// tail is the newest records of one shard's log, encoded, in order.  a
// primary resumes a follower from it and a new primary pulls what it lacks
// from it.  it drops whole chunks from the front when it is over maxBytes or
// when a chunk is older than window.
type tail struct {
	mu       sync.Mutex
	maxBytes int
	window   time.Duration
	// next is the LSN after the last record.  with no chunks it is the only
	// LSN the tail covers.
	next uint64
	// chunks[head:] are the chunks the tail has.
	chunks []tailChunk
	head   int
	bytes  int
}

func newTail(maxBytes int, window time.Duration) *tail {
	return &tail{
		maxBytes: maxBytes,
		window:   window,
		next:     1,
	}
}

// reset empties the tail.  the next record is base + 1.
func (t *tail) reset(base uint64) {
	t.mu.Lock()
	t.resetLocked(base + 1)
	t.mu.Unlock()
}

func (t *tail) resetLocked(next uint64) {
	t.chunks = nil
	t.head = 0
	t.bytes = 0
	t.next = next
}

// append adds count records encoded in data that start at first.  the tail
// keeps data, so the caller must not change it afterwards.
func (t *tail) append(first uint64, count int, data []byte, now time.Time) {
	if count == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if first != t.next {
		t.resetLocked(first)
	}
	t.chunks = append(t.chunks, tailChunk{
		first: first,
		count: count,
		at:    now,
		data:  data,
	})
	t.bytes += len(data) + chunkOverhead
	t.next = first + uint64(count)
	t.evictLocked(now)
}

// evict drops what the limits no longer allow.  an idle shard gets no
// append, so the node calls this on every report.
func (t *tail) evict(now time.Time) {
	t.mu.Lock()
	t.evictLocked(now)
	t.mu.Unlock()
}

func (t *tail) evictLocked(now time.Time) {
	for t.head < len(t.chunks) {
		c := &t.chunks[t.head]
		if t.bytes <= t.maxBytes && now.Sub(c.at) <= t.window {
			break
		}
		t.bytes -= len(c.data) + chunkOverhead
		*c = tailChunk{}
		t.head++
	}
	switch {
	case t.head == len(t.chunks):
		t.chunks = t.chunks[:0]
		t.head = 0
	case t.head > 64 && t.head > len(t.chunks)/2:
		n := copy(t.chunks, t.chunks[t.head:])
		clear(t.chunks[n:])
		t.chunks = t.chunks[:n]
		t.head = 0
	}
}

// firstLocked is the lowest LSN the tail can serve.
func (t *tail) firstLocked() uint64 {
	if t.head < len(t.chunks) {
		return t.chunks[t.head].first
	}
	return t.next
}

// covers reports whether the tail has every record from the LSN from on.
// the LSN after the last record is covered, also when the tail is empty.
func (t *tail) covers(from uint64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return from >= t.firstLocked() && from <= t.next
}

// read returns encoded records that go on from the LSN from.  ok is false
// when the tail no longer has from or has never reached it.  first is the
// LSN of the first record in data.  it is below from when from is inside a
// chunk, because chunks are not cut.  the caller skips what it has.  read
// joins whole chunks while they fit in limit bytes and always returns at
// least one chunk.  limit 0 means no limit.  count is 0 when from is the
// LSN after the last record.  the caller must not change data.
func (t *tail) read(from uint64, limit int) (first uint64, count int, data []byte, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if from < t.firstLocked() || from > t.next {
		return 0, 0, nil, false
	}
	if from == t.next {
		return from, 0, nil, true
	}
	live := t.chunks[t.head:]
	// the chunk that has from is the last one that starts at or below it.
	i := sort.Search(len(live), func(i int) bool { return live[i].first > from }) - 1
	end := i + 1
	size := len(live[i].data)
	count = live[i].count
	for end < len(live) && (limit <= 0 || size+len(live[end].data) <= limit) {
		size += len(live[end].data)
		count += live[end].count
		end++
	}
	if end == i+1 {
		return live[i].first, count, live[i].data, true
	}
	data = make([]byte, 0, size)
	for _, c := range live[i:end] {
		data = append(data, c.data...)
	}
	return live[i].first, count, data, true
}

// size returns the bytes the tail counts against its limit and how many
// records it has.
func (t *tail) size() (bytes, records int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, c := range t.chunks[t.head:] {
		records += c.count
	}
	return t.bytes, records
}
