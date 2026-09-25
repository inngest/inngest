package queue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// normalizeScanShard records the calls the normalization scanner makes.
type normalizeScanShard struct {
	*mockShardForIterator

	accounts   []uuid.UUID
	partitions map[uuid.UUID][]*QueueShadowPartition
	backlogs   map[string][]*QueueBacklog

	accountPartitionPeeks atomic.Int32
	shadowPartitionPeeks  atomic.Int32
	backlogPeeks          atomic.Int32
}

func (s *normalizeScanShard) PeekGlobalNormalizeAccounts(ctx context.Context, until time.Time, limit int64) ([]uuid.UUID, error) {
	return s.accounts, nil
}

func (s *normalizeScanShard) PeekAccountNormalizePartitions(ctx context.Context, accountID uuid.UUID, until time.Time, limit int64) ([]*QueueShadowPartition, error) {
	s.accountPartitionPeeks.Add(1)
	return s.partitions[accountID], nil
}

func (s *normalizeScanShard) PeekShadowPartitions(ctx context.Context, accountID *uuid.UUID, sequential bool, peekLimit int64, until time.Time) ([]*QueueShadowPartition, error) {
	s.shadowPartitionPeeks.Add(1)
	return nil, nil
}

func (s *normalizeScanShard) ShadowPartitionPeekNormalizeBacklogs(ctx context.Context, sp *QueueShadowPartition, limit int64) ([]*QueueBacklog, error) {
	s.backlogPeeks.Add(1)
	return s.backlogs[sp.PartitionID], nil
}

func TestIterateNormalizationPartition(t *testing.T) {
	ctx := context.Background()

	accountA, accountB := uuid.New(), uuid.New()
	spA := &QueueShadowPartition{PartitionID: uuid.NewString(), AccountID: &accountA}
	spB := &QueueShadowPartition{PartitionID: uuid.NewString(), AccountID: &accountB}
	blA := &QueueBacklog{BacklogID: "fn:" + spA.PartitionID, ShadowPartitionID: spA.PartitionID}
	blB := &QueueBacklog{BacklogID: "fn:" + spB.PartitionID, ShadowPartitionID: spB.PartitionID}

	tests := []struct {
		name              string
		accounts          []uuid.UUID
		partitions        map[uuid.UUID][]*QueueShadowPartition
		backlogs          map[string][]*QueueBacklog
		wantAccountPeeks  int32
		wantBacklogPeeks  int32
		wantDispatchedIDs []string
	}{
		{
			name:     "no accounts to normalize",
			accounts: nil,
		},
		{
			name:     "dispatches backlogs from each account's normalize partitions",
			accounts: []uuid.UUID{accountA, accountB},
			partitions: map[uuid.UUID][]*QueueShadowPartition{
				accountA: {spA},
				accountB: {spB},
			},
			backlogs: map[string][]*QueueBacklog{
				spA.PartitionID: {blA},
				spB.PartitionID: {blB},
			},
			wantAccountPeeks:  2,
			wantBacklogPeeks:  2,
			wantDispatchedIDs: []string{blA.BacklogID, blB.BacklogID},
		},
		{
			name:             "account without normalize partitions does no backlog peeks",
			accounts:         []uuid.UUID{accountA},
			partitions:       map[uuid.UUID][]*QueueShadowPartition{},
			wantAccountPeeks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shard := &normalizeScanShard{
				mockShardForIterator: &mockShardForIterator{name: "test-shard"},
				accounts:             tt.accounts,
				partitions:           tt.partitions,
				backlogs:             tt.backlogs,
			}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)
			q, err := New(ctx, "test", registry)
			require.NoError(t, err)

			bc := make(chan normalizeWorkerChanMsg, 10)
			require.NoError(t, q.iterateNormalizationPartition(ctx, time.Now(), bc))
			close(bc)

			var dispatched []string
			for msg := range bc {
				dispatched = append(dispatched, msg.b.BacklogID)
			}

			require.ElementsMatch(t, tt.wantDispatchedIDs, dispatched)
			require.Equal(t, tt.wantAccountPeeks, shard.accountPartitionPeeks.Load())
			require.Equal(t, tt.wantBacklogPeeks, shard.backlogPeeks.Load())
			// The scanner must never walk the account's active shadow partitions.
			require.Zero(t, shard.shadowPartitionPeeks.Load())
		})
	}
}
