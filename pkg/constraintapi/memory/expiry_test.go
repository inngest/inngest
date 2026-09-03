package memory

import (
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func collect(fn func(fn func(seq uint64))) []uint64 {
	var out []uint64
	fn(func(seq uint64) { out = append(out, seq) })
	return out
}

func TestExpiryIndexDrainWholeSeconds(t *testing.T) {
	e := newExpiryIndex(10_000)
	require.Equal(t, int64(9), e.swept.Load())

	e.add(12_500, 1)
	e.add(11_000, 2)
	e.add(11_999, 3)
	require.Equal(t, 2, e.bucketCount())

	require.Empty(t, collect(func(fn func(uint64)) { e.drain(11_500, fn) }), "the current second is not drained")
	require.Equal(t, int64(10), e.swept.Load())

	got := collect(func(fn func(uint64)) { e.drain(13_000, fn) })
	require.Equal(t, []uint64{2, 3, 1}, got, "buckets drain in second order")
	require.Equal(t, int64(12), e.swept.Load())
	require.Equal(t, 0, e.bucketCount())

	require.Empty(t, collect(func(fn func(uint64)) { e.drain(13_000, fn) }))
	e.drain(11_000, func(uint64) { t.Fatal("no bucket to drain") })
	require.Equal(t, int64(12), e.swept.Load(), "swept never moves backwards")
}

func TestExpiryIndexScanLeavesEntries(t *testing.T) {
	e := newExpiryIndex(12_000)
	e.add(12_300, 5)
	e.add(12_900, 6)
	e.add(13_100, 7)

	got := collect(func(fn func(uint64)) { e.scan(12_500, fn) })
	require.Equal(t, []uint64{5, 6}, got)
	require.Equal(t, 2, e.bucketCount(), "scan removes nothing")

	got = collect(func(fn func(uint64)) { e.drain(14_000, fn) })
	require.Equal(t, []uint64{5, 6, 7}, got, "scanned seqs drain later, the caller skips taken slots")
}

func TestExpiryIndexAddIntoSweptSecondMovesForward(t *testing.T) {
	e := newExpiryIndex(10_000)
	e.drain(13_000, func(uint64) {})
	require.Equal(t, int64(12), e.swept.Load())

	e.add(11_000, 7)
	require.Equal(t, 1, e.bucketCount())
	require.Empty(t, collect(func(fn func(uint64)) { e.drain(13_000, fn) }), "it landed in the current second")
	got := collect(func(fn func(uint64)) { e.scan(13_000, fn) })
	require.Equal(t, []uint64{7}, got)

	got = collect(func(fn func(uint64)) { e.drain(15_000, fn) })
	require.Equal(t, []uint64{7}, got)
}

// TestExpiryIndexNoLostEntriesUnderDrain adds seqs into the second being
// drained from many goroutines.  every seq must come out of a drain exactly
// once.  a lost seq is a lease the sweeper never reclaims.
func TestExpiryIndexNoLostEntriesUnderDrain(t *testing.T) {
	e := newExpiryIndex(0)
	const perG = 500

	var mu sync.Mutex
	var drained []uint64
	drainFn := func(seq uint64) {
		mu.Lock()
		drained = append(drained, seq)
		mu.Unlock()
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		now := int64(2_000)
		for {
			select {
			case <-stop:
				return
			default:
			}
			e.drain(now, drainFn)
			now += 1_000
			// a real sweeper moves one second per second.  a tight loop here
			// would race ahead of every add and turn the test into a chase.
			// one millisecond per second still drains the first seconds
			// while the adders are filling them.
			time.Sleep(time.Millisecond)
		}
	}()

	var adders sync.WaitGroup
	for g := 0; g < 8; g++ {
		adders.Add(1)
		go func(g int) {
			defer adders.Done()
			for i := 0; i < perG; i++ {
				seq := uint64(g*perG + i + 1)
				// expiry seconds race with the drainer's advancing now
				e.add(int64(i)*10, seq)
			}
		}(g)
	}
	adders.Wait()
	close(stop)
	wg.Wait()

	e.drain(1<<40, drainFn)

	sort.Slice(drained, func(i, j int) bool { return drained[i] < drained[j] })
	require.Len(t, drained, 8*perG)
	for i, seq := range drained {
		require.Equal(t, uint64(i+1), seq, "seq %d missing or duplicated", i+1)
	}
}
