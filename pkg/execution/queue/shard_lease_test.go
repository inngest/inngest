package queue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type shardLeaseTestShard struct {
	*registryTestShard
	err   error
	calls atomic.Int32
}

func (s *shardLeaseTestShard) ShardAssignmentConfig() ShardAssignmentConfig {
	config := s.registryTestShard.ShardAssignmentConfig()
	config.NumExecutors = 1
	return config
}

func (s *shardLeaseTestShard) ShardLease(context.Context, string, time.Duration, int, ...*ulid.ULID) (*ulid.ULID, error) {
	s.calls.Add(1)
	return nil, s.err
}

func TestTryClaimShardLease_ContinuesPastShardErrors(t *testing.T) {
	firstErr := errors.New("first unavailable")
	secondErr := errors.New("second unavailable")
	first := &shardLeaseTestShard{registryTestShard: newTestShard("first", "group"), err: firstErr}
	second := &shardLeaseTestShard{registryTestShard: newTestShard("second", "group"), err: secondErr}
	registry := mustShardRegistry(t,
		map[string]QueueShard{first.Name(): first, second.Name(): second},
		WithShardSelector(alwaysShard(first)),
	)
	q, err := New(context.Background(), "test", registry, WithRunMode(QueueRunMode{ShardGroup: "group"}))
	require.NoError(t, err)

	claimed, err := q.tryClaimShardLease(context.Background(), []QueueShard{first, second})
	require.NoError(t, err)
	require.False(t, claimed)
	require.Equal(t, int32(1), first.calls.Load())
	require.Equal(t, int32(1), second.calls.Load())
}

func TestTryClaimShardLease_ReturnsCallerCancellation(t *testing.T) {
	shard := &shardLeaseTestShard{registryTestShard: newTestShard("shard", "group"), err: context.Canceled}
	registry := mustShardRegistry(t,
		map[string]QueueShard{shard.Name(): shard},
		WithShardSelector(alwaysShard(shard)),
	)
	q, err := New(context.Background(), "test", registry, WithRunMode(QueueRunMode{ShardGroup: "group"}))
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	claimed, err := q.tryClaimShardLease(ctx, []QueueShard{shard})
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, claimed)
}
