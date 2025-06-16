package redis_state

import (
	"context"
	"fmt"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/util"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
	"sync/atomic"
	"time"
)

const (
	PartitionActiveCheckPeekMax        = 10
	PartitionActiveCheckBacklogPeekMax = 20
)

func (q *queue) ActiveCheck(ctx context.Context) (int, error) {
	// Peek partitions for active checks
	parts, err := q.PartitionActiveCheckPeek(ctx, PartitionActiveCheckPeekMax)
	if err != nil {
		return 0, fmt.Errorf("could not peek partitions for active checker: %w", err)
	}

	shard := q.primaryQueueShard
	client := shard.RedisClient.Client()
	kg := shard.RedisClient.KeyGenerator()

	var checked int64

	eg := errgroup.Group{}

	// Process partitions in parallel
	for _, part := range parts {
		part := part
		eg.Go(func() error {
			err := q.shadowPartitionActiveCheck(ctx, part, client, kg)
			if err != nil {
				return fmt.Errorf("could not check partition for active keys: %w", err)
			}

			atomic.AddInt64(&checked, 1)

			return nil
		})
	}

	err = eg.Wait()
	if err != nil {
		return int(atomic.LoadInt64(&checked)), fmt.Errorf("could not perform active check: %w", err)
	}

	return int(atomic.LoadInt64(&checked)), nil
}

func (q *queue) shadowPartitionActiveCheck(ctx context.Context, sp *QueueShadowPartition, client rueidis.Client, kg QueueKeyGenerator) error {
	// Check account

	// Check partition
	err := q.partitionActiveCheck(ctx, sp, client, kg)
	if err != nil {
		return fmt.Errorf("could not check account for invalid active items: %w", err)
	}

	// Check custom concurrency keys
	sequentialBacklogs := false
	backlogs, _, err := q.ShadowPartitionPeek(ctx, sp, sequentialBacklogs, q.clock.Now(), PartitionActiveCheckBacklogPeekMax)
	if err != nil {
		return fmt.Errorf("could not peek backlogs for shadow partition active check: %w", err)
	}

	for _, bidx := range util.RandPerm(len(backlogs)) {
		backlog := backlogs[bidx]

		for _, key := range backlog.ConcurrencyKeys {
			err := q.customConcurrencyActiveCheck(ctx, sp, key, client, kg)
			if err != nil {
				return fmt.Errorf("could not check custom concurrency key: %w", err)
			}
		}
	}

	return nil
}

func (q *queue) partitionActiveCheck(ctx context.Context, sp *QueueShadowPartition, client rueidis.Client, kg QueueKeyGenerator) error {
	keyActive := sp.activeKey(kg)
	keyInProgress := sp.inProgressKey(kg)
	keyReady := sp.readyQueueKey(kg)

	invalidItems := make([]string, 0)

	err := q.checkForMissingItems(ctx, client, keyActive, []string{keyInProgress, keyReady}, func(pointer string) {
		invalidItems = append(invalidItems, pointer)
	})
	if err != nil {
		return fmt.Errorf("could not check partition for missing active items: %w", err)
	}

	if len(invalidItems) > 0 {
		q.log.Debug("removing invalid items from active key", "mode", "partition", "invalid", invalidItems, "partition", sp.PartitionID, "active", keyActive, "ready", keyReady, "in_progress", keyInProgress)

		cmd := client.B().Zrem().Key(keyActive).Member(invalidItems...).Build()
		err := client.Do(ctx, cmd).Error()
		if err != nil {
			return fmt.Errorf("could not remove invalid items from active set: %w", err)
		}
	}

	return nil
}

func (q *queue) customConcurrencyActiveCheck(ctx context.Context, sp *QueueShadowPartition, bcc BacklogConcurrencyKey, client rueidis.Client, kg QueueKeyGenerator) error {
	keyActive := bcc.activeKey(kg)
	keyInProgress := bcc.concurrencyKey(kg)

	// Can use the partition ready queue as it includes _all_ concurrency keys' items
	keyReady := sp.readyQueueKey(kg)

	invalidItems := make([]string, 0)

	err := q.checkForMissingItems(ctx, client, keyActive, []string{keyInProgress, keyReady}, func(pointer string) {
		invalidItems = append(invalidItems, pointer)
	})
	if err != nil {
		return fmt.Errorf("could not check custom concurrency key for missing active items: %w", err)
	}

	if len(invalidItems) > 0 {
		q.log.Debug("removing invalid items from active key", "invalid", "mode", "custom_concurrency", "bcc", bcc, invalidItems, "partition", sp.PartitionID, "active", keyActive, "ready", keyReady, "in_progress", keyInProgress)

		cmd := client.B().Zrem().Key(keyActive).Member(invalidItems...).Build()
		err := client.Do(ctx, cmd).Error()
		if err != nil {
			return fmt.Errorf("could not remove invalid items from active set: %w", err)
		}
	}

	return nil
}

func (q *queue) checkForMissingItems(ctx context.Context, client rueidis.Client, sourceKey string, targetKeys []string, onMissing func(pointer string)) error {
	var cursor uint64
	var count int64 = 20

	for {
		cmd := client.B().Zscan().Key(sourceKey).Cursor(cursor).Count(count).Build()
		entry, err := client.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			if rueidis.IsRedisNil(err) {
				return nil
			}
			return fmt.Errorf("could not iterate key 1 for missing items: %w", err)
		}

		if entry.Cursor == 0 {
			return nil
		}

		cursor = entry.Cursor

		entriesFound := make(map[string]struct{})

		for _, targetKey := range targetKeys {
			resp, err := client.Do(ctx, client.B().Zmscore().Key(targetKey).Member(entry.Elements...).Build()).ToAny()
			if err != nil && !rueidis.IsRedisNil(err) {
				return fmt.Errorf("could not check key 2 for missing items: %w", err)
			}

			scores, ok := resp.([]interface{})
			if !ok {
				return nil
			}

			for i, score := range scores {
				if score != nil {
					entriesFound[entry.Elements[i]] = struct{}{}
				}
			}
		}

		for _, element := range entry.Elements {
			if _, has := entriesFound[element]; !has {
				onMissing(element)
			}
		}

		<-time.After(100 * time.Millisecond)
	}
}

func (q *queue) iterateSource(
	ctx context.Context,
	client rueidis.Client,
	kg QueueKeyGenerator,
	sourceKey string,
	targetKeys func(chunk []*osqueue.QueueItem) []string,
	onMissing func(pointer *osqueue.QueueItem),
) error {
	var cursor uint64
	var count int64 = 20

	p := peeker[osqueue.QueueItem]{
		q:                      q,
		max:                    count,
		opName:                 "peekIterateSource",
		ignoreUntil:            true,
		isMillisecondPrecision: true,
		handleMissingItems:     CleanupMissingPointers(ctx, sourceKey, client, q.log),
		maker: func() *osqueue.QueueItem {
			return &osqueue.QueueItem{}
		},
		keyMetadataHash: kg.QueueItem(),
	}

	for {
		cmd := client.B().Zscan().Key(sourceKey).Cursor(cursor).Count(count).Build()
		entry, err := client.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			if rueidis.IsRedisNil(err) {
				return nil
			}
			return fmt.Errorf("could not iterate key 1 for missing items: %w", err)
		}

		if entry.Cursor == 0 {
			return nil
		}

		cursor = entry.Cursor

		entriesFound := make(map[string]struct{})

		for _, targetKey := range targetKeys {
			resp, err := client.Do(ctx, client.B().Zmscore().Key(targetKey).Member(entry.Elements...).Build()).ToAny()
			if err != nil && !rueidis.IsRedisNil(err) {
				return fmt.Errorf("could not check key 2 for missing items: %w", err)
			}

			scores, ok := resp.([]interface{})
			if !ok {
				return nil
			}

			for i, score := range scores {
				if score != nil {
					entriesFound[entry.Elements[i]] = struct{}{}
				}
			}
		}

		for _, element := range entry.Elements {
			if _, has := entriesFound[element]; !has {
				onMissing(element)
			}
		}

		<-time.After(100 * time.Millisecond)
	}
}

func (q *queue) partitionActiveCheck(ctx context.Context, sp *QueueShadowPartition, client rueidis.Client, kg QueueKeyGenerator) error {
	keyActive := sp.activeKey(kg)
	keyInProgress := sp.inProgressKey(kg)
	keyReady := sp.readyQueueKey(kg)

	invalidItems := make([]string, 0)

	err := q.checkForMissingItems(ctx, client, keyActive, []string{keyInProgress, keyReady}, func(pointer string) {
		invalidItems = append(invalidItems, pointer)
	})
	if err != nil {
		return fmt.Errorf("could not check partition for missing active items: %w", err)
	}

	if len(invalidItems) > 0 {
		q.log.Debug("removing invalid items from active key", "mode", "partition", "invalid", invalidItems, "partition", sp.PartitionID, "active", keyActive, "ready", keyReady, "in_progress", keyInProgress)

		cmd := client.B().Zrem().Key(keyActive).Member(invalidItems...).Build()
		err := client.Do(ctx, cmd).Error()
		if err != nil {
			return fmt.Errorf("could not remove invalid items from active set: %w", err)
		}
	}

	return nil
}

func (q *queue) PartitionActiveCheckPeek(ctx context.Context, peekSize int64) ([]*QueueShadowPartition, error) {
	shard := q.primaryQueueShard
	client := shard.RedisClient.Client()
	kg := shard.RedisClient.KeyGenerator()

	key := kg.PartitionActiveCheckSet()

	peeker := peeker[QueueShadowPartition]{
		q:                      q,
		max:                    PartitionActiveCheckPeekMax,
		opName:                 "peekPartitionActiveCheck",
		isMillisecondPrecision: true,
		handleMissingItems:     CleanupMissingPointers(ctx, key, client, q.log),
		maker: func() *QueueShadowPartition {
			return &QueueShadowPartition{}
		},
		keyMetadataHash: kg.ShadowPartitionMeta(),
	}

	// Pick random partitions within bounds
	isSequential := false

	res, err := peeker.peek(ctx, key, isSequential, q.clock.Now(), peekSize)
	if err != nil {
		return nil, fmt.Errorf("could not peek active check partitions: %w", err)
	}

	return res.Items, nil
}
