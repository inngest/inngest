package ttlupsert

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	ID string
}

func TestUpsertSkipsSuccessfulKeyWithinTTL(t *testing.T) {
	ctx := context.Background()
	u := NewWithKey(func(i testItem) string { return i.ID })

	var calls int
	ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.True(t, ran)

	ran, err = u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.False(t, ran)
	require.Equal(t, 1, calls)
}

func TestUpsertFailureDoesNotPopulateCache(t *testing.T) {
	ctx := context.Background()
	u := NewWithKey(func(i testItem) string { return i.ID })
	expectedErr := errors.New("db unavailable")

	var calls int
	ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return expectedErr
	})
	require.ErrorIs(t, err, expectedErr)
	require.True(t, ran)

	ran, err = u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.True(t, ran)
	require.Equal(t, 2, calls)
}

func TestUpsertRunsAfterTTLExpires(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	u := NewWithKey(
		func(i testItem) string { return i.ID },
		WithClock(clock),
		WithTTL(10*time.Second),
	)

	var calls int
	ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.True(t, ran)

	clock.Advance(10 * time.Second)
	ran, err = u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.True(t, ran)
	require.Equal(t, 2, calls)
}

func TestUpsertHitRefreshesExpiry(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	u := NewWithKey(
		func(i testItem) string { return i.ID },
		WithClock(clock),
		WithTTL(10*time.Second),
	)

	var calls int
	ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.True(t, ran)

	clock.Advance(9 * time.Second)
	ran, err = u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.False(t, ran)

	clock.Advance(9 * time.Second)
	ran, err = u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.False(t, ran)
	require.Equal(t, 1, calls)
}

func TestUpsertSeparatesDifferentKeys(t *testing.T) {
	ctx := context.Background()
	u := NewWithKey(func(i testItem) string { return i.ID })

	var calls int
	for _, id := range []string{"a", "b"} {
		ran, err := u.Upsert(ctx, testItem{ID: id}, func(context.Context) error {
			calls++
			return nil
		})
		require.NoError(t, err)
		require.True(t, ran)
	}

	require.Equal(t, 2, calls)
}

func TestUpsertUsesCustomKey(t *testing.T) {
	ctx := context.Background()
	u := NewWithKey(func(i testItem) string {
		return "same-" + i.ID
	})

	var calls int
	ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.True(t, ran)

	ran, err = u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.False(t, ran)
	require.Equal(t, 1, calls)
}

func TestUpsertValidationErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("empty key", func(t *testing.T) {
		u := NewWithKey(func(testItem) string { return "" })

		var calls int
		ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
			calls++
			return nil
		})
		require.ErrorIs(t, err, ErrEmptyKey)
		require.False(t, ran)
		require.Equal(t, 0, calls)
	})

	t.Run("nil transaction", func(t *testing.T) {
		u := NewWithKey(func(i testItem) string { return i.ID })

		ran, err := u.Upsert(ctx, testItem{ID: "a"}, nil)
		require.ErrorIs(t, err, ErrNilTx)
		require.False(t, ran)
	})
}

func TestUpsertCapacityEvictsLeastRecentlyUsed(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	u := NewWithKey(
		func(i testItem) string { return i.ID },
		WithClock(clock),
		WithTTL(time.Minute),
		WithCapacity(2),
	)

	calls := map[string]int{}
	upsert := func(id string) bool {
		ran, err := u.Upsert(ctx, testItem{ID: id}, func(context.Context) error {
			calls[id]++
			return nil
		})
		require.NoError(t, err)
		return ran
	}

	require.True(t, upsert("a"))
	clock.Advance(time.Second)
	require.True(t, upsert("b"))
	clock.Advance(time.Second)
	require.False(t, upsert("a"))
	clock.Advance(time.Second)
	require.True(t, upsert("c"))

	require.False(t, upsert("a"))
	require.True(t, upsert("b"))
	require.Equal(t, 1, calls["a"])
	require.Equal(t, 2, calls["b"])
	require.Equal(t, 1, calls["c"])
}

func TestUpsertNonPositiveTTLOrCapacityDisablesCache(t *testing.T) {
	ctx := context.Background()

	for _, u := range []Upserter[testItem]{
		NewWithKey(func(i testItem) string { return i.ID }, WithTTL(0)),
		NewWithKey(func(i testItem) string { return i.ID }, WithCapacity(0)),
	} {
		var calls int
		for i := 0; i < 2; i++ {
			ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
				calls++
				return nil
			})
			require.NoError(t, err)
			require.True(t, ran)
		}
		require.Equal(t, 2, calls)
	}
}

func TestUpsertCoalescesConcurrentSuccessfulCalls(t *testing.T) {
	ctx := context.Background()
	u := NewWithKey(func(i testItem) string { return i.ID })

	start := make(chan struct{})
	release := make(chan struct{})
	var txCalls atomic.Int32
	var wg sync.WaitGroup
	const goroutines = 20

	results := make(chan bool, goroutines)
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
				if txCalls.Add(1) == 1 {
					close(release)
				}
				<-release
				return nil
			})
			results <- ran
			errs <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	var ranCount int
	for ran := range results {
		if ran {
			ranCount++
		}
	}
	require.Equal(t, int32(1), txCalls.Load())
	require.Equal(t, 1, ranCount)
}

func TestUpsertCoalescesConcurrentFailure(t *testing.T) {
	ctx := context.Background()
	u := NewWithKey(func(i testItem) string { return i.ID })
	expectedErr := errors.New("db unavailable")

	started := make(chan struct{})
	release := make(chan struct{})
	var txCalls atomic.Int32
	leaderDone := make(chan struct{})

	go func() {
		defer close(leaderDone)
		ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
			txCalls.Add(1)
			close(started)
			<-release
			return expectedErr
		})
		require.ErrorIs(t, err, expectedErr)
		require.True(t, ran)
	}()

	<-started
	waiterDone := make(chan struct{})
	go func() {
		defer close(waiterDone)
		ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
			txCalls.Add(1)
			return nil
		})
		require.ErrorIs(t, err, expectedErr)
		require.False(t, ran)
	}()

	require.Never(t, func() bool {
		select {
		case <-waiterDone:
			return true
		default:
			return false
		}
	}, 10*time.Millisecond, time.Millisecond)

	close(release)
	<-leaderDone
	<-waiterDone

	require.Equal(t, int32(1), txCalls.Load())

	ran, err := u.Upsert(ctx, testItem{ID: "a"}, func(context.Context) error {
		txCalls.Add(1)
		return nil
	})
	require.NoError(t, err)
	require.True(t, ran)
	require.Equal(t, int32(2), txCalls.Load())
}
