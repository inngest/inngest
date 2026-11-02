package peg

// maxArenaChunk caps how many elements one arena chunk holds. Chunks double
// from the evaluator's initial size up to this, so a small parse still gets
// one small chunk and a large one doesn't overshoot by more than a chunk.
const maxArenaChunk = 16 << 10

// arena bump-allocates T values out of a list of fixed chunks. A pointer or
// sub-slice handed out stays valid until reset: a full chunk is never
// copied or rewritten, the arena just starts a new one. Growing one slice
// with append instead would copy it on every regrowth, and since the copies
// it leaves behind are still referenced (by memoized nodes and their
// parents) they stay live too, which at append's ~1.25x growth for large
// slices is ~5x the memory the elements themselves need.
type arena[T any] struct {
	chunk  []T   // the chunk being filled: len is what's handed out, cap its size
	chunks [][]T // every chunk allocated, in order, kept for reuse after reset
	next   int   // index in chunks of the chunk after the current one
	full   int   // elements handed out from chunks before the current one
}

func newArena[T any](size int) arena[T] {
	c := make([]T, 0, max(size, 64))
	return arena[T]{chunk: c, chunks: [][]T{c}, next: 1}
}

func (a *arena[T]) alloc(v T) *T {
	if len(a.chunk) == cap(a.chunk) {
		a.grow(1)
	}
	a.chunk = append(a.chunk, v)
	return &a.chunk[len(a.chunk)-1]
}

// allocSlice copies vs into the arena, returning a cap-limited view over the
// copy (so an errant append elsewhere reallocates instead of clobbering
// whatever the arena hands out next).
func (a *arena[T]) allocSlice(vs ...T) []T {
	if cap(a.chunk)-len(a.chunk) < len(vs) {
		a.grow(len(vs))
	}
	start := len(a.chunk)
	a.chunk = append(a.chunk, vs...)
	end := len(a.chunk)
	return a.chunk[start:end:end]
}

// grow moves on to a chunk with room for need more elements, reusing one a
// previous parse allocated if it's big enough.
func (a *arena[T]) grow(need int) {
	a.full += len(a.chunk)
	if a.next < len(a.chunks) && cap(a.chunks[a.next]) >= need {
		a.chunk = a.chunks[a.next][:0]
		a.next++
		return
	}
	c := make([]T, 0, max(min(cap(a.chunk)*2, maxArenaChunk), need))
	a.chunks = append(a.chunks[:a.next], c)
	a.next = len(a.chunks)
	a.chunk = c
}

// len reports how many elements were handed out since the last reset.
func (a *arena[T]) len() int { return a.full + len(a.chunk) }

// size reports the total capacity of every chunk the arena holds.
func (a *arena[T]) size() int {
	n := 0
	for _, c := range a.chunks {
		n += cap(c)
	}
	return n
}

// reset makes every chunk available again, invalidating everything handed
// out so far.
func (a *arena[T]) reset() {
	a.chunk = a.chunks[0][:0]
	a.next = 1
	a.full = 0
}
