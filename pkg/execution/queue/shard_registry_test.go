package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// registryTestShard is a QueueShard with a configurable group, for ByGroup
// tests. It reuses mockShardForIterator (defined in processor_iterator_test.go)
// to satisfy the rest of the QueueShard surface.
type registryTestShard struct {
	mockShardForIterator
	group string
}

func (r *registryTestShard) ShardAssignmentConfig() ShardAssignmentConfig {
	return ShardAssignmentConfig{ShardGroup: r.group}
}

func newTestShard(name, group string) *registryTestShard {
	return &registryTestShard{
		mockShardForIterator: mockShardForIterator{name: name},
		group:                group,
	}
}

func mustSingleShardRegistry(t *testing.T, shard QueueShard) ShardRegistryController {
	t.Helper()
	r, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)
	return r
}

func TestNewSingleShardRegistry(t *testing.T) {
	t.Run("seeds topology and primary from one shard", func(t *testing.T) {
		s := newTestShard("only", "g")
		r := mustSingleShardRegistry(t, s)

		require.Equal(t, s, r.Primary())

		got, err := r.ByName("only")
		require.NoError(t, err)
		require.Equal(t, s, got)

		require.ElementsMatch(t, []QueueShard{s}, r.ByGroup("g"))
	})

	t.Run("ByName returns ErrQueueShardNotFound for unknown name", func(t *testing.T) {
		r := mustSingleShardRegistry(t, newTestShard("only", "g"))

		_, err := r.ByName("missing")
		require.ErrorIs(t, err, ErrQueueShardNotFound)
	})

	t.Run("ByGroup returns empty for unknown group", func(t *testing.T) {
		r := mustSingleShardRegistry(t, newTestShard("only", "g"))

		require.Empty(t, r.ByGroup("missing"))
	})

	t.Run("Resolve returns the shard regardless of account or queue name", func(t *testing.T) {
		s := newTestShard("only", "g")
		r := mustSingleShardRegistry(t, s)

		got, err := r.Resolve(context.Background(), uuid.New(), nil)
		require.NoError(t, err)
		require.Equal(t, s, got)

		got, err = r.Resolve(context.Background(), uuid.Nil, nil)
		require.NoError(t, err)
		require.Equal(t, s, got)

		qn := "ignored"
		got, err = r.Resolve(context.Background(), uuid.New(), &qn)
		require.NoError(t, err)
		require.Equal(t, s, got)

		got, err = r.Resolve(context.Background(), uuid.Nil, &qn)
		require.NoError(t, err)
		require.Equal(t, s, got)
	})

	t.Run("ForEach visits the single shard", func(t *testing.T) {
		r := mustSingleShardRegistry(t, newTestShard("only", ""))

		var visited []string
		err := r.ForEach(context.Background(), func(_ context.Context, qs QueueShard) error {
			visited = append(visited, qs.Name())
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, []string{"only"}, visited)
	})

	t.Run("ForEach surfaces fn error", func(t *testing.T) {
		r := mustSingleShardRegistry(t, newTestShard("only", ""))

		want := errors.New("nope")
		err := r.ForEach(context.Background(), func(context.Context, QueueShard) error {
			return want
		})
		require.ErrorIs(t, err, want)
	})

	t.Run("SetPrimary succeeds when the shard matches the registered one", func(t *testing.T) {
		s := newTestShard("only", "")
		r := mustSingleShardRegistry(t, s)

		require.NoError(t, r.SetPrimary(context.Background(), s))
		// Same name, different instance — still considered known.
		require.NoError(t, r.SetPrimary(context.Background(), newTestShard("only", "")))
	})

	t.Run("SetPrimary errors on unknown shard", func(t *testing.T) {
		s := newTestShard("only", "")
		r := mustSingleShardRegistry(t, s)

		require.ErrorIs(t, r.SetPrimary(context.Background(), newTestShard("other", "")), ErrQueueShardNotFound)
		require.ErrorIs(t, r.SetPrimary(context.Background(), nil), ErrQueueShardNotFound)
		// Primary unchanged.
		require.Equal(t, s, r.Primary())
	})

	t.Run("Add errors — single shard registry is fixed", func(t *testing.T) {
		s := newTestShard("only", "")
		r := mustSingleShardRegistry(t, s)

		require.Error(t, r.Add(newTestShard("other", "")))
		_, err := r.ByName("other")
		require.ErrorIs(t, err, ErrQueueShardNotFound)
		// Primary unchanged.
		require.Equal(t, s, r.Primary())
	})

	t.Run("nil shard returns an error", func(t *testing.T) {
		r, err := NewSingleShardRegistry(nil)
		require.Error(t, err)
		require.Nil(t, r)
	})
}
