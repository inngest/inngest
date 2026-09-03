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

	seq1, sl1 := s.alloc(10, 5_000, rs)
	require.Equal(t, uint64(1), seq1, "seq starts at one so a zero ULID never matches")
	require.NotNil(t, sl1)
	require.Equal(t, int64(5_000), sl1.expiresAtMS)
	require.Same(t, rs, sl1.req.Load())

	seq2, _ := s.alloc(10, 6_000, rs)
	require.Equal(t, uint64(2), seq2)

	require.Same(t, sl1, s.get(seq1))
	require.Nil(t, s.get(0))
	require.Nil(t, s.get(3), "not yet allocated")
	require.Nil(t, s.get(1<<40), "no page")

	require.Equal(t, 1, s.pageCount())
	p := s.page(seq1)
	require.NotNil(t, p)
	require.Equal(t, int64(2), p.live.Load())
	require.Equal(t, int64(10), p.createdAtMS)
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
	const grace = int64(5_000)

	first, firstSlot := s.alloc(0, 1_000, &requestState{})
	// fill the rest of page zero and step into page one
	for i := 1; i < pageSize; i++ {
		seq, sl := s.alloc(0, 1_000, &requestState{})
		require.True(t, sl.take())
		s.page(seq).live.Add(-1)
	}
	current, _ := s.alloc(0, 1_000, &requestState{})
	require.Equal(t, uint64(pageSize+1), current)
	require.Equal(t, 2, s.pageCount())

	require.Equal(t, 0, s.freePages(10_000, grace), "page zero still has a live slot")
	require.NotNil(t, s.get(first))

	require.True(t, firstSlot.take())
	s.page(first).live.Add(-1)
	require.Equal(t, 0, s.freePages(20_000, grace), "the first empty sighting only stamps the page")
	require.Equal(t, 0, s.freePages(20_000+grace, grace), "not empty for long enough yet")

	// a slot going live again resets the stamp
	s.page(first).live.Add(1)
	require.Equal(t, 0, s.freePages(20_000+grace+1, grace))
	s.page(first).live.Add(-1)
	require.Equal(t, 0, s.freePages(30_000, grace), "stamped again")
	require.Equal(t, 1, s.freePages(30_000+grace+1, grace))
	require.Equal(t, 1, s.pageCount())
	require.Nil(t, s.page(first))
	require.Nil(t, s.get(first), "a freed page reads as no lease")

	// the current allocation page is never freed, even when idle and empty
	cur := s.page(current)
	require.True(t, s.get(current).take())
	cur.live.Add(-1)
	require.Equal(t, 0, s.freePages(1<<40, grace))
	require.Equal(t, 0, s.freePages(1<<41, grace))
	require.Equal(t, 1, s.pageCount())
}
