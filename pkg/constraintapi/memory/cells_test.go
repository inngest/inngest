package memory

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSemaphoreCellTakeNeverExceedsCapacity(t *testing.T) {
	const capacity = int64(100)
	var c semaphoreCell
	var outstanding atomic.Int64
	var overshoot atomic.Int64
	var deadSeen atomic.Int64

	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			w := int64(g%3 + 1)
			held := 0
			for i := 0; i < 2000; i++ {
				fit, ok := c.take(capacity, w, 4)
				if !ok {
					deadSeen.Add(1)
					return
				}
				held += fit
				if outstanding.Add(w*int64(fit)) > capacity {
					overshoot.Add(1)
				}
				if held > 0 && i%3 == 0 {
					// decrement the tracker first so it never lags below
					// the counter and reports a false overshoot
					outstanding.Add(-w)
					c.give(w)
					held--
				}
			}
			for ; held > 0; held-- {
				outstanding.Add(-w)
				c.give(w)
			}
		}(g)
	}
	wg.Wait()

	require.Zero(t, deadSeen.Load(), "a live cell never reads dead")
	require.Zero(t, overshoot.Load(), "confirmed grants exceeded capacity")
	v, ok := c.load()
	require.True(t, ok)
	require.Equal(t, int64(0), v, "every grant was given back")
}

func TestSemaphoreCellTakeRollbackIsExact(t *testing.T) {
	var c semaphoreCell

	fit, ok := c.take(5, 2, 4)
	require.True(t, ok)
	require.Equal(t, 2, fit, "two units of weight two fit under five")
	v, _ := c.load()
	require.Equal(t, int64(4), v)

	fit, ok = c.take(5, 2, 1)
	require.True(t, ok)
	require.Equal(t, 0, fit, "one unit left is below the weight")
	v, _ = c.load()
	require.Equal(t, int64(4), v, "a zero fit adds nothing")

	fit, _ = c.take(5, 1, 3)
	require.Equal(t, 1, fit)
	v, _ = c.load()
	require.Equal(t, int64(5), v)

	fit, _ = c.take(5, 1, 0)
	require.Equal(t, 0, fit)

	fit, _ = c.take(3, 1, 1)
	require.Equal(t, 0, fit, "capacity below usage grants nothing")
	v, _ = c.load()
	require.Equal(t, int64(5), v)
}

func TestSemaphoreCellGiveClampsAtZero(t *testing.T) {
	var c semaphoreCell
	require.True(t, c.set(1))

	v, ok := c.give(5)
	require.True(t, ok)
	require.Equal(t, int64(0), v)

	require.True(t, c.set(10))
	var negative atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				v, _ := c.give(1)
				if v < 0 {
					negative.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	require.Zero(t, negative.Load())
	v, _ = c.load()
	require.Equal(t, int64(0), v)
}

func TestSemaphoreCellAdjustClampsAtZero(t *testing.T) {
	var c semaphoreCell
	require.True(t, c.set(5))

	v, ok := c.adjust(3)
	require.True(t, ok)
	require.Equal(t, int64(8), v)

	v, ok = c.adjust(-10)
	require.True(t, ok)
	require.Equal(t, int64(0), v)

	v, _ = c.load()
	require.Equal(t, int64(0), v)
}

func TestSemaphoreCellDead(t *testing.T) {
	var c semaphoreCell
	require.True(t, c.set(5))
	require.False(t, c.kill(), "a non zero cell is not killed")

	require.True(t, c.set(0))
	require.True(t, c.kill())
	require.False(t, c.kill(), "kill is not repeated")

	_, ok := c.load()
	require.False(t, ok)
	fit, ok := c.take(10, 1, 1)
	require.False(t, ok)
	require.Zero(t, fit)
	_, ok = c.give(1)
	require.False(t, ok)
	require.False(t, c.set(3))
	_, ok = c.adjust(3)
	require.False(t, ok)

	_, ok = c.load()
	require.False(t, ok, "no operation revives a dead cell")
}

func TestGCRACellUpdateUnderContention(t *testing.T) {
	var c gcraCell
	const now = int64(1_000)

	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				c.update(now, func(tat float64, present bool) (float64, bool, int64) {
					if !present {
						tat = 0
					}
					return tat + 1, true, 60
				})
			}
		}()
	}
	wg.Wait()

	tat, present := c.load(now)
	require.True(t, present)
	require.Equal(t, 1600.0, tat, "every update lands exactly once")
}

func TestGCRACellExpiry(t *testing.T) {
	var c gcraCell

	_, present := c.load(0)
	require.False(t, present, "a fresh cell is absent")

	var sawPresent bool
	c.update(0, func(tat float64, present bool) (float64, bool, int64) {
		sawPresent = present
		return 12.5, true, 1
	})
	require.False(t, sawPresent)

	tat, present := c.load(999)
	require.True(t, present)
	require.Equal(t, 12.5, tat)

	_, present = c.load(1_000)
	require.False(t, present, "expiry is exclusive like Redis EX")

	c.update(500, func(tat float64, present bool) (float64, bool, int64) {
		return 99, false, 1
	})
	tat, present = c.load(500)
	require.True(t, present)
	require.Equal(t, 12.5, tat, "store false writes nothing")

	c.update(2_000, func(tat float64, present bool) (float64, bool, int64) {
		require.False(t, present, "an expired cell reads as absent inside update")
		return 7, true, 1
	})
	tat, present = c.load(2_500)
	require.True(t, present)
	require.Equal(t, 7.0, tat)
}
