package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrackingSemaphoreAvailable(t *testing.T) {
	ctx := context.Background()
	sem := NewTrackingSemaphore(2)

	require.Equal(t, int64(0), sem.Count())
	require.Equal(t, int64(2), sem.Available())

	require.NoError(t, sem.Acquire(ctx, 1))
	require.Equal(t, int64(1), sem.Count())
	require.Equal(t, int64(1), sem.Available())

	require.True(t, sem.TryAcquire(1))
	require.Equal(t, int64(2), sem.Count())
	require.Equal(t, int64(0), sem.Available())

	require.False(t, sem.TryAcquire(1))
	require.Equal(t, int64(2), sem.Count())
	require.Equal(t, int64(0), sem.Available())

	sem.Release(2)
	require.Equal(t, int64(0), sem.Count())
	require.Equal(t, int64(2), sem.Available())
}
