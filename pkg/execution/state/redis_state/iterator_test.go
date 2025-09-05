package redis_state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state/peek"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestIterateSortedSet(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	// kg := defaultShard.RedisClient.kg

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	testcases := []struct {
		name             string
		num              int
		interval         time.Duration
		from             time.Time
		until            time.Time
		expectedItems    int
		keyQueuesEnabled bool
		batchSize        int
	}{
		{
			name:          "multiple batches, same score",
			num:           100,
			from:          time.Now().Truncate(time.Minute),
			interval:      0,
			until:         time.Now().Truncate(time.Minute).Add(30 * time.Second),
			expectedItems: 100,
			batchSize:     10,
		},
		{
			name:          "simple iteration",
			num:           100,
			from:          time.Now().Truncate(time.Minute),
			interval:      0,
			until:         time.Now().Truncate(time.Minute).Add(30 * time.Second),
			expectedItems: 100,
			batchSize:     10,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

			q := NewQueue(
				defaultShard,
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
					return tc.keyQueuesEnabled
				}),
				WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
					return false
				}),
				WithClock(clock),
			)

			for i := range tc.num {
				at := clock.Now()
				if !tc.from.IsZero() {
					at = tc.from
				}
				at = at.Add(time.Duration(i) * tc.interval)

				item := osqueue.QueueItem{
					ID:          fmt.Sprintf("test%d", i),
					FunctionID:  fnID,
					WorkspaceID: wsID,
					Data: osqueue.Item{
						WorkspaceID: wsID,
						Kind:        osqueue.KindEdge,
						Identifier: state.Identifier{
							AccountID:       acctId,
							WorkspaceID:     wsID,
							WorkflowID:      fnID,
							WorkflowVersion: 1,
						},
					},
				}

				_, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			peeker := peek.NewPeeker(
				func() *osqueue.QueueItem {
					return &osqueue.QueueItem{}
				},
				peek.WithPeekerClient(defaultShard.RedisClient.Client()),
				peek.WithPeekerHandleMissingItems(func(ctx context.Context, pointers []string) error {
					// Ignore them for now, regular execution will clean up
					return nil
				}),
				peek.WithPeekerMaxPeekSize(tc.batchSize+1), // allow 1 extra element for duplicate score pagination
				peek.WithPeekerMetadataHashKey(defaultShard.RedisClient.kg.QueueItem()),
				peek.WithPeekerMillisecondPrecision(true),
				peek.WithPeekerOpName("iteratePartition"),
			)

			items := q.iterateSortedSetQueue(ctx, defaultShard, queueSortedSetIterationOptions{
				keySortedSet: defaultShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""),
				partitionID:  fnID.String(),
				from:         tc.from,
				until:        tc.until,
				pageSize:     tc.batchSize,
				peeker:       peeker,
			})

			uniqMap := map[string]struct{}{}
			for item := range items {
				if _, seen := uniqMap[item.ID]; seen {
					require.Fail(t, "found duplicate item", item)
				}
				uniqMap[item.ID] = struct{}{}
			}

			require.Equal(t, tc.expectedItems, len(uniqMap), r.Dump())
		})
	}
}

func TestIterateSortedSetSimple(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	// kg := defaultShard.RedisClient.kg

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q := NewQueue(
		defaultShard,
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		}),
		WithClock(clock),
	)

	from := clock.Now()
	until := from.Add(10 * time.Second)
	count := 5
	for i := range count {
		at := from

		item := osqueue.QueueItem{
			ID:          fmt.Sprintf("test%d", i),
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					AccountID:       acctId,
					WorkspaceID:     wsID,
					WorkflowID:      fnID,
					WorkflowVersion: 1,
				},
			},
		}

		_, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}

	peeker := peek.NewPeeker(
		func() *osqueue.QueueItem {
			return &osqueue.QueueItem{}
		},
		peek.WithPeekerClient(defaultShard.RedisClient.Client()),
		peek.WithPeekerHandleMissingItems(func(ctx context.Context, pointers []string) error {
			// Ignore them for now, regular execution will clean up
			return nil
		}),
		peek.WithPeekerMaxPeekSize(3), // allow 1 extra element for duplicate score pagination
		peek.WithPeekerMetadataHashKey(defaultShard.RedisClient.kg.QueueItem()),
		peek.WithPeekerMillisecondPrecision(true),
		peek.WithPeekerOpName("iteratePartition"),
	)

	items := q.iterateSortedSetQueue(ctx, defaultShard, queueSortedSetIterationOptions{
		keySortedSet: defaultShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""),
		partitionID:  fnID.String(),
		from:         from,
		until:        until,
		pageSize:     2,
		peeker:       peeker,
	})

	uniqMap := map[string]struct{}{}
	for item := range items {
		if _, seen := uniqMap[item.ID]; seen {
			require.Fail(t, "found duplicate item", item)
		}
		uniqMap[item.ID] = struct{}{}
	}

	expected := count
	require.Equal(t, expected, len(uniqMap), r.Dump())
}

func TestIterateSortedSetOffsetBasedPagination(t *testing.T) {
	_, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	// kg := defaultShard.RedisClient.kg

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q := NewQueue(
		defaultShard,
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		}),
		WithClock(clock),
	)

	from := clock.Now().Truncate(time.Second)
	until := from.Add(10 * time.Second)

	windowSize := 3

	itemScores := []time.Duration{
		0,
		time.Second,
		time.Second,
		// first window
		time.Second,
		2 * time.Second,
		2 * time.Second,
		// second window
		3 * time.Second,
	}
	count := len(itemScores)
	expected := count

	for i, dur := range itemScores {
		at := from
		if dur > 0 {
			at = at.Add(dur)
		}

		item := osqueue.QueueItem{
			ID:          fmt.Sprintf("test%d", i),
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					AccountID:       acctId,
					WorkspaceID:     wsID,
					WorkflowID:      fnID,
					WorkflowVersion: 1,
				},
			},
		}

		_, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}

	peeker := peek.NewPeeker(
		func() *osqueue.QueueItem {
			return &osqueue.QueueItem{}
		},
		peek.WithPeekerClient(defaultShard.RedisClient.Client()),
		peek.WithPeekerHandleMissingItems(func(ctx context.Context, pointers []string) error {
			// Ignore them for now, regular execution will clean up
			return nil
		}),
		peek.WithPeekerMaxPeekSize(windowSize+1), // allow 1 extra element for duplicate score pagination
		peek.WithPeekerMetadataHashKey(defaultShard.RedisClient.kg.QueueItem()),
		peek.WithPeekerMillisecondPrecision(true),
		peek.WithPeekerOpName("iteratePartition"),
	)

	items := q.iterateSortedSetQueue(ctx, defaultShard, queueSortedSetIterationOptions{
		keySortedSet: defaultShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), ""),
		partitionID:  fnID.String(),
		from:         from,
		until:        until,
		pageSize:     windowSize,
		peeker:       peeker,
	})

	durMap := map[string]time.Duration{}
	durSlice := []time.Duration{}
	for item := range items {
		if _, seen := durMap[item.ID]; seen {
			require.Fail(t, "found duplicate item", item)
		}
		dur := time.UnixMilli(item.AtMS).Sub(from)
		t.Log("got item", item.ID, "at", dur)
		durMap[item.ID] = dur
		durSlice = append(durSlice, dur)
	}

	for _, d := range itemScores {
		var hasDuration bool
		for _, d2 := range durSlice {
			if d2 == d {
				hasDuration = true
				break
			}
		}
		if !hasDuration {
			require.Failf(t, "missing duration", "missing duration %s", d)
		}

	}
	require.Equal(t, expected, len(durMap), "", durMap, durSlice)
}
