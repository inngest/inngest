package queue

import (
	"context"
	"errors"
	"sync"
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

// alwaysShard returns a selector that always resolves to the given shard.
func alwaysShard(s QueueShard) shardSelector {
	return func(context.Context, uuid.UUID, *string) (QueueShard, error) {
		return s, nil
	}
}

func mustShardRegistry(t *testing.T, shards map[string]QueueShard, opts ...ShardRegistryOpt) ShardRegistryController {
	t.Helper()
	r, err := NewShardRegistry(shards, opts...)
	require.NoError(t, err)
	return r
}

func mustSingleShardRegistry(t *testing.T, shard QueueShard) ShardRegistryController {
	t.Helper()
	r, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)
	return r
}

func TestNewShardRegistry_RejectsEmptyTopology(t *testing.T) {
	_, err := NewShardRegistry(nil)
	require.Error(t, err)
}

func TestNewShardRegistry_RequiresSelector(t *testing.T) {
	a := newTestShard("a", "")
	_, err := NewShardRegistry(map[string]QueueShard{"a": a})
	require.Error(t, err)
}

func TestNewShardRegistry_RejectsNilSelector(t *testing.T) {
	a := newTestShard("a", "")
	_, err := NewShardRegistry(
		map[string]QueueShard{"a": a},
		WithShardSelector(nil),
	)
	require.Error(t, err)
}

func TestNewShardRegistry_PrimaryMustBeInTopology(t *testing.T) {
	other := newTestShard("other", "")
	primary := newTestShard("primary", "")
	_, err := NewShardRegistry(
		map[string]QueueShard{"other": other},
		WithShardSelector(alwaysShard(other)),
		WithPrimary(primary),
	)
	require.Error(t, err)
}

func TestShardRegistry_Primary(t *testing.T) {
	s := newTestShard("primary", "")
	r := mustShardRegistry(t,
		map[string]QueueShard{"primary": s},
		WithShardSelector(alwaysShard(s)),
		WithPrimary(s),
	)
	require.Equal(t, QueueShard(s), r.Primary())
}

func TestShardRegistry_Primary_Unset(t *testing.T) {
	s := newTestShard("x", "")
	r := mustShardRegistry(t,
		map[string]QueueShard{"x": s},
		WithShardSelector(alwaysShard(s)),
	)
	require.Nil(t, r.Primary())
}

func TestShardRegistry_ByName(t *testing.T) {
	a := newTestShard("a", "")
	r := mustShardRegistry(t,
		map[string]QueueShard{"a": a},
		WithShardSelector(alwaysShard(a)),
	)

	got, err := r.ByName("a")
	require.NoError(t, err)
	require.Equal(t, QueueShard(a), got)

	_, err = r.ByName("missing")
	require.ErrorIs(t, err, ErrQueueShardNotFound)
}

func TestShardRegistry_ByGroup(t *testing.T) {
	a := newTestShard("a", "g1")
	b := newTestShard("b", "g1")
	c := newTestShard("c", "g2")
	r := mustShardRegistry(t,
		map[string]QueueShard{"a": a, "b": b, "c": c},
		WithShardSelector(alwaysShard(a)),
	)

	g1 := r.ByGroup("g1")
	require.Len(t, g1, 2)
	require.ElementsMatch(t, []QueueShard{a, b}, g1)

	g2 := r.ByGroup("g2")
	require.Equal(t, []QueueShard{c}, g2)

	require.Nil(t, r.ByGroup("missing"))
}

func TestShardRegistry_ForEach(t *testing.T) {
	a := newTestShard("a", "")
	b := newTestShard("b", "")
	r := mustShardRegistry(t,
		map[string]QueueShard{"a": a, "b": b},
		WithShardSelector(alwaysShard(a)),
	)

	visited := map[string]bool{}
	var mu sync.Mutex
	err := r.ForEach(context.Background(), func(_ context.Context, s QueueShard) error {
		mu.Lock()
		visited[s.Name()] = true
		mu.Unlock()
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, map[string]bool{"a": true, "b": true}, visited)
}

func TestShardRegistry_ForEach_PropagatesError(t *testing.T) {
	a := newTestShard("a", "")
	r := mustShardRegistry(t,
		map[string]QueueShard{"a": a},
		WithShardSelector(alwaysShard(a)),
	)
	sentinel := errors.New("boom")
	err := r.ForEach(context.Background(), func(context.Context, QueueShard) error {
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)
}

func TestShardRegistry_Resolve_DelegatesToSelector(t *testing.T) {
	a := newTestShard("a", "")
	b := newTestShard("b", "")
	called := false
	sel := func(_ context.Context, _ uuid.UUID, _ *string) (QueueShard, error) {
		called = true
		return b, nil
	}
	r := mustShardRegistry(t,
		map[string]QueueShard{"a": a, "b": b},
		WithShardSelector(sel),
	)

	got, err := r.Resolve(context.Background(), uuid.New(), nil)
	require.NoError(t, err)
	require.Equal(t, QueueShard(b), got)
	require.True(t, called)
}

func TestShardRegistry_SetPrimary(t *testing.T) {
	seed := newTestShard("seed", "")
	r := mustShardRegistry(t,
		map[string]QueueShard{"seed": seed},
		WithShardSelector(alwaysShard(seed)),
	)
	a := newTestShard("a", "")

	require.NoError(t, r.Add(a))
	require.NoError(t, r.SetPrimary(context.Background(), a))
	require.Equal(t, QueueShard(a), r.Primary())

	require.NoError(t, r.SetPrimary(context.Background(), nil))
	require.Nil(t, r.Primary())
}

func TestShardRegistry_SetPrimary_RejectsUnknownShard(t *testing.T) {
	seed := newTestShard("seed", "")
	r := mustShardRegistry(t,
		map[string]QueueShard{"seed": seed},
		WithShardSelector(alwaysShard(seed)),
	)
	require.Error(t, r.SetPrimary(context.Background(), newTestShard("a", "")))
}

func TestShardRegistry_Add(t *testing.T) {
	seed := newTestShard("seed", "")
	r := mustShardRegistry(t,
		map[string]QueueShard{"seed": seed},
		WithShardSelector(alwaysShard(seed)),
	)
	a := newTestShard("a", "")
	require.NoError(t, r.Add(a))

	got, err := r.ByName("a")
	require.NoError(t, err)
	require.Equal(t, QueueShard(a), got)

	require.Error(t, r.Add(nil))
}

func TestNewSingleShardRegistry(t *testing.T) {
	t.Run("seeds topology and primary from one shard", func(t *testing.T) {
		s := newTestShard("only", "g")
		r := mustSingleShardRegistry(t, s)

		require.Equal(t, QueueShard(s), r.Primary())

		got, err := r.ByName("only")
		require.NoError(t, err)
		require.Equal(t, QueueShard(s), got)

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
		require.Equal(t, QueueShard(s), got)

		qn := "ignored"
		got, err = r.Resolve(context.Background(), uuid.Nil, &qn)
		require.NoError(t, err)
		require.Equal(t, QueueShard(s), got)
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

	t.Run("nil shard returns an error", func(t *testing.T) {
		r, err := NewSingleShardRegistry(nil)
		require.Error(t, err)
		require.Nil(t, r)
	})
}
