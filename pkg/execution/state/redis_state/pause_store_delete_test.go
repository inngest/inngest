package redis_state

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// Deleting the pauses of a run that never waited on anything costs a single
// read; a run with pauses still has its index removed.
func TestPauseStoreDeletePausesForRunSkipsEmptyIndex(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{mr.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	unsharded := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	store := NewPauseStore(unsharded)
	kg := unsharded.Pauses().kg

	t.Run("run without pauses", func(t *testing.T) {
		before := mr.CommandCount()
		require.NoError(t, store.DeletePausesForRun(ctx, ulid.Make(), uuid.New()))
		require.Equal(t, 1, mr.CommandCount()-before, "only the index read")
	})

	t.Run("run with a pause", func(t *testing.T) {
		runID := ulid.Make()
		indexKey := kg.RunPauses(ctx, runID)
		_, err := mr.SAdd(indexKey, uuid.New().String())
		require.NoError(t, err)

		require.NoError(t, store.DeletePausesForRun(ctx, runID, uuid.New()))
		require.False(t, mr.Exists(indexKey), "index removed")
	})
}
