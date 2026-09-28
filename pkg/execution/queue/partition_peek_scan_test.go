package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type partitionPeekScanShard struct {
	*mockShardForIterator
	mu     sync.Mutex
	limits []int64
}

func (s *partitionPeekScanShard) PartitionPeek(_ context.Context, _ bool, _ time.Time, limit int64) ([]*QueuePartition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limits = append(s.limits, limit)
	return nil, nil
}

func (s *partitionPeekScanShard) PeekAccountPartitions(ctx context.Context, _ uuid.UUID, limit int64, until time.Time, sequential bool) ([]*QueuePartition, error) {
	return s.PartitionPeek(ctx, sequential, until, limit)
}

func TestPartitionPeekScanDynamicLimit(t *testing.T) {
	for _, accountScan := range []bool{false, true} {
		name := "global"
		mode := QueueRunMode{Partition: true}
		if accountScan {
			name = "accounts"
			mode = QueueRunMode{Account: true, ExclusiveAccounts: []uuid.UUID{uuid.New(), uuid.New()}}
		}
		t.Run(name, func(t *testing.T) {
			shard := &partitionPeekScanShard{mockShardForIterator: &mockShardForIterator{name: "ss3"}}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)
			limit := int64(750)
			calls := 0
			q, err := New(context.Background(), "test", registry, WithRunMode(mode), WithPartitionPeekMaxGetter(func(_ context.Context, name string) int64 {
				require.Equal(t, "ss3", name)
				calls++
				return limit
			}))
			require.NoError(t, err)
			for _, value := range []int64{750, 500, 1} {
				limit = value
				shard.limits = nil
				require.NoError(t, q.scan(context.Background(), nil))
				effectiveLimit := max(value, PartitionSelectionMax)
				want := []int64{effectiveLimit}
				if accountScan {
					want = []int64{max(effectiveLimit/2, 1), max(effectiveLimit/2, 1)}
				}
				require.Equal(t, want, shard.limits)
			}
			require.Equal(t, 3, calls, "evaluate once per scan")
		})
	}
}
