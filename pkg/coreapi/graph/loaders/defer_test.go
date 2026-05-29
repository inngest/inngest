package loader

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/graph-gophers/dataloader"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// loadByRunID is the shared batch helper used by the run-defer, deferred-from,
// and trace-run-by-id loaders. Its contract is load-bearing for every defer
// linkage query, so the result-slot bookkeeping is tested here directly.

func TestLoadByRunID(t *testing.T) {
	ctx := context.Background()

	a := ulid.Make()
	b := ulid.Make()
	c := ulid.Make()

	t.Run("hits, miss, and parse error preserve slot order", func(t *testing.T) {
		var calls atomic.Int32
		fetch := func(_ context.Context, ids []ulid.ULID) (map[ulid.ULID]int, error) {
			calls.Add(1)
			// b is intentionally absent from the result map.
			return map[ulid.ULID]int{a: 1, c: 3}, nil
		}

		keys := dataloader.NewKeysFromStrings([]string{
			a.String(),
			"not-a-ulid",
			b.String(),
			c.String(),
		})
		results := loadByRunID(ctx, keys, fetch)
		require.Len(t, results, 4)

		// Hit
		require.NoError(t, results[0].Error)
		require.Equal(t, 1, results[0].Data)

		// Parse error survives the post-fetch fill (which only writes nil slots)
		require.Error(t, results[1].Error)
		require.Nil(t, results[1].Data)

		// Miss is mapped to (nil, nil): callers decide what absence means.
		require.NoError(t, results[2].Error)
		require.Nil(t, results[2].Data)

		// Hit
		require.NoError(t, results[3].Error)
		require.Equal(t, 3, results[3].Data)

		// Fetch is batched into a single call.
		require.Equal(t, int32(1), calls.Load())
	})

	t.Run("fetch error fans out to unfilled slots but preserves parse errors", func(t *testing.T) {
		boom := errors.New("boom")
		fetch := func(_ context.Context, _ []ulid.ULID) (map[ulid.ULID]int, error) {
			return nil, boom
		}

		keys := dataloader.NewKeysFromStrings([]string{
			a.String(),
			"not-a-ulid",
			b.String(),
		})
		results := loadByRunID(ctx, keys, fetch)
		require.Len(t, results, 3)

		require.ErrorIs(t, results[0].Error, boom)
		// Parse error is NOT overwritten by the fetch error — the slot was
		// already populated, and the post-fetch loop skips non-nil slots.
		require.Error(t, results[1].Error)
		require.NotErrorIs(t, results[1].Error, boom)
		require.ErrorIs(t, results[2].Error, boom)
	})
}
