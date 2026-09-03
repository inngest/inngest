package memory

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlabAllocAndGet(t *testing.T) {
	var s slab
	rs := &requestState{}

	seq1, sl1 := s.allocIn(0, 5_000, rs)
	require.Equal(t, uint64(1), seq1, "seq starts at one so a zero ULID never matches")
	require.NotNil(t, sl1)
	require.Equal(t, int64(5_000), sl1.expiresAtMS)
	require.Same(t, rs, sl1.req.Load())

	seq2, _ := s.allocIn(0, 6_000, rs)
	require.Equal(t, uint64(2), seq2)

	seq3, sl3 := s.allocIn(3, 7_000, rs)
	require.Equal(t, uint64(3)<<shardShift|1, seq3, "the shard is in the top byte")
	require.Same(t, sl3, s.get(seq3))
	require.Equal(t, 2, s.pageCount(), "each shard fills its own pages")

	require.Same(t, sl1, s.get(seq1))
	require.Nil(t, s.get(0))
	require.Nil(t, s.get(3), "not yet allocated")
	require.Nil(t, s.get(1<<40), "no page")

	// alloc picks a shard at random
	seq4, sl4 := s.alloc(0, 8_000, rs)
	require.NotZero(t, seq4)
	require.Same(t, sl4, s.get(seq4))
}

func TestSlotTakeExactlyOnce(t *testing.T) {
	var s slab
	seq, _ := s.alloc(0, 1_000, &requestState{})

	var winners atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if sl := s.get(seq); sl != nil && sl.take() {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, int64(1), winners.Load())
	require.Nil(t, s.get(seq), "a taken slot is not returned")
}

func TestSlabFreePages(t *testing.T) {
	var s slab

	first, firstSlot := s.allocIn(0, 1_000, &requestState{})
	// fill the rest of page zero of shard zero and step into its page one
	for i := 1; i < pageSize; i++ {
		_, sl := s.allocIn(0, 1_000, &requestState{})
		require.True(t, sl.take())
	}
	current, _ := s.allocIn(0, 1_000, &requestState{})
	require.Equal(t, uint64(pageSize+1), current)
	require.Equal(t, 2, s.pageCount())

	require.Equal(t, 0, s.freePages(), "page zero still has a live slot")
	require.NotNil(t, s.get(first))

	require.True(t, firstSlot.take())
	require.Equal(t, 1, s.freePages(), "every slot taken frees the page")
	require.Equal(t, 1, s.pageCount())
	require.Nil(t, s.page(first))
	require.Nil(t, s.get(first), "a freed page reads as no lease")

	// the shard's allocation page is never freed, even when every slot in it
	// so far is taken
	require.True(t, s.get(current).take())
	require.Equal(t, 0, s.freePages())
	require.Equal(t, 1, s.pageCount())

	// a slot claimed but not yet published keeps its page
	var other slab
	for i := 1; i < pageSize; i++ {
		_, sl := other.allocIn(0, 1_000, &requestState{})
		require.True(t, sl.take())
	}
	next, _ := other.allocIn(0, 1_000, &requestState{})
	require.Equal(t, uint64(pageSize), next, "the first seq of page one")
	require.Equal(t, 2, other.pageCount())
	other.page(1).slots[5].state.Store(slotEmpty)
	require.Equal(t, 0, other.freePages(), "an empty slot means an alloc is in flight")
	other.page(1).slots[5].state.Store(slotTaken)
	require.Equal(t, 1, other.freePages())
	require.Nil(t, other.page(1))
}
