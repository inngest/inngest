package queue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestConcurrencyRequeueExtensionOption(t *testing.T) {
	ctx := context.Background()
	p := PartitionIdentifier{AccountID: uuid.New(), FunctionID: uuid.New()}
	require.Equal(t, 5*time.Second, NewQueueOptions().ConcurrencyRequeueExtension(ctx, "selected", p))
	for _, d := range []time.Duration{-time.Second, 0, time.Second, 2*time.Second - time.Nanosecond, 2 * time.Second, 7 * time.Second} {
		t.Run(d.String(), func(t *testing.T) {
			opts := NewQueueOptions(WithConcurrencyRequeueExtension(func(_ context.Context, shard string, got PartitionIdentifier) time.Duration {
				require.Equal(t, p, got)
				if shard == "selected" {
					return d
				}
				return 0
			}))
			want := d
			if d < 2*time.Second {
				want = 5 * time.Second
			}
			require.Equal(t, want, opts.ConcurrencyRequeueExtension(ctx, "selected", p))
			require.Equal(t, 5*time.Second, opts.ConcurrencyRequeueExtension(ctx, "other", p))
		})
	}
	name := "system"
	opts := NewQueueOptions(WithConcurrencyRequeueExtension(func(context.Context, string, PartitionIdentifier) time.Duration {
		t.Fatal("system queues must not call the override")
		return 2 * time.Second
	}))
	p.SystemQueueName = &name
	require.Equal(t, 5*time.Second, opts.ConcurrencyRequeueExtension(ctx, "selected", p))
	t.Run("account opt-in", func(t *testing.T) {
		selected := uuid.New()
		opts := NewQueueOptions(WithConcurrencyRequeueExtension(func(_ context.Context, shard string, p PartitionIdentifier) time.Duration {
			if shard == "selected" && p.AccountID == selected {
				return 2 * time.Second
			}
			return 0
		}))
		require.Equal(t, 2*time.Second, opts.ConcurrencyRequeueExtension(ctx, "selected", PartitionIdentifier{AccountID: selected}))
		require.Equal(t, 5*time.Second, opts.ConcurrencyRequeueExtension(ctx, "selected", PartitionIdentifier{AccountID: uuid.New()}))
		require.Equal(t, 5*time.Second, opts.ConcurrencyRequeueExtension(ctx, "other", PartitionIdentifier{AccountID: selected}))
	})
}

type concurrencyRequeueShard struct {
	mockShardForIterator
	item  *QueueItem
	limit error
}

func (s *concurrencyRequeueShard) Peek(context.Context, *QueuePartition, time.Time, int64) ([]*QueueItem, error) {
	return []*QueueItem{s.item}, nil
}

func (s *concurrencyRequeueShard) PeekRandom(context.Context, *QueuePartition, time.Time, int64) ([]*QueueItem, error) {
	return []*QueueItem{s.item}, nil
}

func (s *concurrencyRequeueShard) Lease(context.Context, QueueItem, time.Duration, time.Time, ...LeaseOptionFn) (*ulid.ULID, error) {
	return nil, s.limit
}

func TestProcessPartitionConcurrencyRequeueExtension(t *testing.T) {
	for _, tc := range []struct {
		name     string
		limit    error
		optIn    bool
		parallel bool
		want     time.Duration
		calls    int32
	}{
		{"default", ErrAccountConcurrencyLimit, false, false, 5 * time.Second, 1},
		{"account", ErrAccountConcurrencyLimit, true, false, 2 * time.Second, 1},
		{"function", ErrPartitionConcurrencyLimit, true, false, 2 * time.Second, 1},
		{"custom key", ErrConcurrencyLimitCustomKey, true, false, 2 * time.Second, 1},
		{"random offset", ErrConcurrencyLimitCustomKey, true, true, 2 * time.Second, 2},
		{"throttle unchanged", ErrQueueItemThrottled, true, false, PartitionThrottleLimitRequeueExtension, 1},
		{"semaphore unchanged", ErrSemaphoreLimit, true, false, PartitionSemaphoreLimitRequeueExtension, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			clock := clockwork.NewFakeClockAt(time.Unix(100, 750_000_000))
			accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
			shard := &concurrencyRequeueShard{
				mockShardForIterator: mockShardForIterator{name: "selected"},
				item:                 semaphoreTestItem(accountID, envID, fnID, clock.Now()),
				limit:                tc.limit,
			}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)
			opts := []QueueOpt{WithClock(clock)}
			if tc.optIn {
				opts = append(opts, WithConcurrencyRequeueExtension(func(_ context.Context, name string, p PartitionIdentifier) time.Duration {
					require.NotEqual(t, ErrQueueItemThrottled, tc.limit)
					require.NotEqual(t, ErrSemaphoreLimit, tc.limit)
					require.Equal(t, "selected", name)
					require.Equal(t, accountID, p.AccountID)
					return 2 * time.Second
				}))
			}
			if tc.parallel {
				opts = append(opts, WithDisableFifoForFunctions(map[string]struct{}{fnID.String(): {}}))
			}
			q, err := New(ctx, "test", registry, opts...)
			require.NoError(t, err)
			err = q.ProcessPartition(ctx, &QueuePartition{ID: fnID.String(), FunctionID: &fnID, EnvID: &envID, AccountID: accountID}, 0, false, func(context.Context, ProcessItem) (DispatchedItem, error) {
				t.Error("constrained work must not dispatch")
				return nil, nil
			})
			require.NoError(t, err)
			require.Equal(t, tc.calls, atomic.LoadInt32(&shard.partitionRequeueCount))
			require.True(t, shard.partitionRequeueForceAt)
			require.Equal(t, clock.Now().Truncate(time.Second).Add(tc.want), shard.partitionRequeueAt)
		})
	}
}
