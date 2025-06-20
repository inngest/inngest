package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
	mathRand "math/rand"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	PartitionActiveCheckPeekMax          = 10
	BacklogActiveCheckPeekMax            = 10
	PartitionActiveCheckBacklogPeekMax   = 20
	PartitionActiveCheckCooldownDuration = 5 * time.Minute
	BacklogActiveCheckCooldownDuration   = 5 * time.Minute
)

func (q *queue) ActiveCheck(ctx context.Context) (int, error) {
	// Peek shadow partitions for active checks
	backlogs, err := q.BacklogActiveCheckPeek(ctx, BacklogActiveCheckPeekMax)
	if err != nil {
		return 0, fmt.Errorf("could not peek backlogs for active checker: %w", err)
	}

	l := q.log.With("scope", "active-check")

	shard := q.primaryQueueShard
	client := shard.RedisClient.Client()
	kg := shard.RedisClient.KeyGenerator()

	var checked int64

	eg := errgroup.Group{}

	// Process backlogs in parallel
	for _, backlog := range backlogs {
		backlog := backlog
		eg.Go(func() error {
			checkID, err := ulid.New(ulid.Timestamp(q.clock.Now()), rand.Reader)
			if err != nil {
				return fmt.Errorf("could not create checkID: %w", err)
			}

			l = l.With("backlog", backlog, "check_id", checkID)

			l.Debug("attempting to active check backlog")

			cleanup, err := q.backlogActiveCheck(ctx, backlog, client, kg, l)
			if cleanup {
				status, cerr := scripts["queue/activeCheckRemoveBacklog"].Exec(
					ctx,
					client,
					[]string{
						kg.BacklogActiveCheckSet(),
						kg.BacklogActiveCheckCooldown(backlog.BacklogID),
					},
					[]string{
						backlog.BacklogID,
						strconv.Itoa(int(q.clock.Now().UnixMilli())),
						strconv.Itoa(int(BacklogActiveCheckCooldownDuration.Seconds())),
					},
				).ToInt64()
				if cerr != nil {
					l.Error("could not mark backlog active check cooldown", "err", cerr)
				}

				if status != 0 {
					l.Error("invalid status received from active check removal", "status", status)
				}
			}

			if err != nil {
				return fmt.Errorf("could not check backlog for active keys: %w", err)
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

func (q *queue) backlogActiveCheck(ctx context.Context, b *QueueBacklog, client rueidis.Client, kg QueueKeyGenerator, l logger.Logger) (bool, error) {
	var sp QueueShadowPartition

	{
		str, err := client.Do(ctx, client.B().Hget().Key(kg.ShadowPartitionMeta()).Field(b.ShadowPartitionID).Build()).ToString()
		if err != nil {
			if rueidis.IsRedisNil(err) {
				l.Debug("shadow partition meta hash not found, exiting")
				return true, nil
			}

			return false, fmt.Errorf("could not get shadow partition: %w", err)
		}

		// If shadow partition is missing, clean up
		if str == "" {
			l.Debug("shadow partition not found for backlog, exiting")
			return true, nil
		}

		if err := json.Unmarshal([]byte(str), &sp); err != nil {
			l.Error("failed to unmarshal shadow partition", "err", err, "str", str)
			return true, nil
		}
	}

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	readOnly := true
	if q.readOnlySpotChecks != nil && !q.readOnlySpotChecks(ctx, accountID) {
		readOnly = false
	}

	l = l.With("partition_id", sp.PartitionID, "account_id", accountID)

	l.Debug("starting active check for partition")

	// Check account
	if accountID != uuid.Nil && mathRand.Intn(100) <= q.runMode.ActiveCheckAccountCheckProbability {
		err := q.accountActiveCheck(ctx, &sp, client, kg, l.With("check-scope", "account-check"), readOnly)
		if err != nil {
			return false, fmt.Errorf("could not check account active items: %w", err)
		}
	}

	// Check partition
	err := q.partitionActiveCheck(ctx, &sp, client, kg, l.With("check-scope", "partition-check"), readOnly)
	if err != nil {
		return false, fmt.Errorf("could not check account for invalid active items: %w", err)
	}

	// Check custom concurrency keys
	for _, key := range b.ConcurrencyKeys {
		err := q.customConcurrencyActiveCheck(ctx, &sp, key, client, kg, l.With("check-scope", "backlog-check"), readOnly)
		if err != nil {
			return false, fmt.Errorf("could not check custom concurrency key: %w", err)
		}
	}

	l.Debug("checked partition for invalid active keys", "partition_id", sp.PartitionID)

	return true, nil
}

func (q *queue) accountActiveCheck(
	ctx context.Context,
	sp *QueueShadowPartition,
	client rueidis.Client,
	kg QueueKeyGenerator,
	l logger.Logger,
	readOnly bool,
) error {
	// Compare the account active key
	keyActive := sp.accountActiveKey(kg)

	// To the account in progress key
	keyInProgress := sp.accountInProgressKey(kg)

	invalidItems := make([]string, 0)

	l.Debug("checking account for invalid or missing active keys", "account_id", sp.AccountID, "key", keyActive)

	err := q.findMissingItemsWithDynamicTargets(ctx, client, kg, keyActive, l, func(chunk []*osqueue.QueueItem) map[string][]string {
		res := make(map[string][]string)

		chunkIDs := make([]string, len(chunk))
		for i, d := range chunk {
			chunkIDs[i] = d.ID
		}

		// Always check in progress
		res[keyInProgress] = chunkIDs

		// Generate ready queue keys for item functions
		for _, item := range chunk {
			keyReady := kg.PartitionQueueSet(enums.PartitionTypeDefault, item.FunctionID.String(), "")
			if item.FunctionID == uuid.Nil && item.QueueName != nil {
				keyReady = kg.PartitionQueueSet(enums.PartitionTypeDefault, *item.QueueName, "")
			}

			res[keyReady] = append(res[keyReady], item.ID)
		}

		return res
	}, func(pointer string) {
		invalidItems = append(invalidItems, pointer)
	})
	if err != nil {
		return fmt.Errorf("could not check account for missing active items: %w", err)
	}

	if len(invalidItems) > 0 {
		l.Debug("removing invalid items from account active key", "mode", "account", "invalid", invalidItems, "partition_id", sp.PartitionID, "active", keyActive, "in_progress", keyInProgress, "readonly", readOnly)

		if !readOnly {
			cmd := client.B().Zrem().Key(keyActive).Member(invalidItems...).Build()
			err := client.Do(ctx, cmd).Error()
			if err != nil {
				return fmt.Errorf("could not remove invalid items from active set: %w", err)
			}
		}
	}

	return nil
}

func (q *queue) partitionActiveCheck(
	ctx context.Context,
	sp *QueueShadowPartition,
	client rueidis.Client,
	kg QueueKeyGenerator,
	l logger.Logger,
	readOnly bool,
) error {
	keyActive := sp.activeKey(kg)
	keyInProgress := sp.inProgressKey(kg)
	keyReady := sp.readyQueueKey(kg)

	invalidItems := make([]string, 0)

	err := q.findMissingItemsWithStaticTargets(ctx, client, keyActive, []string{keyInProgress, keyReady}, func(pointer string) {
		invalidItems = append(invalidItems, pointer)
	})
	if err != nil {
		return fmt.Errorf("could not check partition for missing active items: %w", err)
	}

	if len(invalidItems) > 0 {
		l.Debug("removing invalid items from active key", "mode", "partition", "invalid", invalidItems, "partition_id", sp.PartitionID, "active", keyActive, "ready", keyReady, "in_progress", keyInProgress, "readonly", readOnly)

		if !readOnly {
			cmd := client.B().Zrem().Key(keyActive).Member(invalidItems...).Build()
			err := client.Do(ctx, cmd).Error()
			if err != nil {
				return fmt.Errorf("could not remove invalid items from active set: %w", err)
			}
		}
	}

	return nil
}

func (q *queue) customConcurrencyActiveCheck(ctx context.Context, sp *QueueShadowPartition, bcc BacklogConcurrencyKey, client rueidis.Client, kg QueueKeyGenerator, l logger.Logger, readOnly bool) error {
	keyActive := bcc.activeKey(kg)
	keyInProgress := bcc.concurrencyKey(kg)

	// Can use the partition ready queue as it includes _all_ concurrency keys' items
	keyReady := sp.readyQueueKey(kg)

	invalidItems := make([]string, 0)

	err := q.findMissingItemsWithStaticTargets(ctx, client, keyActive, []string{keyInProgress, keyReady}, func(pointer string) {
		invalidItems = append(invalidItems, pointer)
	})
	if err != nil {
		return fmt.Errorf("could not check custom concurrency key for missing active items: %w", err)
	}

	if len(invalidItems) > 0 {
		l.Debug("removing invalid items from active key", "invalid", "mode", "custom_concurrency", "bcc", bcc, invalidItems, "partition_id", sp.PartitionID, "active", keyActive, "ready", keyReady, "in_progress", keyInProgress, "readonly", readOnly)

		if !readOnly {
			cmd := client.B().Zrem().Key(keyActive).Member(invalidItems...).Build()
			err := client.Do(ctx, cmd).Error()
			if err != nil {
				return fmt.Errorf("could not remove invalid items from active set: %w", err)
			}
		}
	}

	return nil
}

// findMissingItemsWithStaticTargets attempts to find all items in sourceKey, which are not present in any of the targetKeys.
//
// In constrast to findMissingItemsWithDynamicTargets, this function does not assume any particular data type and only operates
// on item pointers.
//
// The missing items will then be bubbled up via onMissing.
func (q *queue) findMissingItemsWithStaticTargets(ctx context.Context, client rueidis.Client, sourceSetKey string, targetKeys []string, onMissing func(pointer string)) error {
	var cursor uint64
	var count int64 = 20

	for {
		cmd := client.B().Sscan().Key(sourceSetKey).Cursor(cursor).Count(count).Build()
		entry, err := client.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			if rueidis.IsRedisNil(err) {
				return nil
			}
			return fmt.Errorf("could not iterate source key for missing items: %w", err)
		}

		if len(entry.Elements) == 0 {
			return nil
		}

		// Entries are returned as [item ID, score] tuples, so we want to skip all scores
		entryIDs := make([]string, 0, len(entry.Elements)/2)
		for i := 0; i < len(entry.Elements); i += 2 {
			entryIDs = append(entryIDs, entry.Elements[i])
		}

		entriesFound := make(map[string]struct{})

		for _, targetKey := range targetKeys {
			resp, err := client.Do(ctx, client.B().Zmscore().Key(targetKey).Member(entryIDs...).Build()).ToAny()
			if err != nil && !rueidis.IsRedisNil(err) {
				return fmt.Errorf("could not check key 2 for missing items: %w", err)
			}

			scores, ok := resp.([]interface{})
			if !ok {
				return nil
			}

			for i, score := range scores {
				if score != nil {
					entriesFound[entryIDs[i]] = struct{}{}
				}
			}
		}

		for _, element := range entryIDs {
			if _, has := entriesFound[element]; !has {
				onMissing(element)
			}
		}

		if entry.Cursor == 0 {
			return nil
		}

		cursor = entry.Cursor

		<-time.After(100 * time.Millisecond)
	}
}

// findMissingItemsWithDynamicTargets attempts to find all items in sourceSetKey, which are not present in any of the targetKeys.
//
// In constrast to findMissingItemsWithStaticTargets, this function strictly operates on queue items and will pass chunks of items
// to a transformation function to retrieve targets and pointers to check for each target.
//
// The missing items will then be bubbled up via onMissing.
func (q *queue) findMissingItemsWithDynamicTargets(
	ctx context.Context,
	client rueidis.Client,
	kg QueueKeyGenerator,
	sourceSetKey string,
	l logger.Logger,
	targetKeys func(chunk []*osqueue.QueueItem) map[string][]string,
	onMissing func(pointer string),
) error {
	var cursor uint64
	var count int64 = 20

	for {
		// Load chunk
		cmd := client.B().Sscan().Key(sourceSetKey).Cursor(cursor).Count(count).Build()
		entry, err := client.Do(ctx, cmd).AsScanEntry()

		l.Debug("scanned source", "key", sourceSetKey, "returned", len(entry.Elements), "cursor", entry.Cursor)

		if err != nil {
			if rueidis.IsRedisNil(err) {
				return nil
			}
			return fmt.Errorf("could not iterate key 1 for missing items: %w", err)
		}

		if len(entry.Elements) == 0 {
			return nil
		}

		// Entries are returned as [item ID, score] tuples, so we want to skip all scores
		entryIDs := make([]string, 0, len(entry.Elements)/2)
		for i := 0; i < len(entry.Elements); i += 2 {
			entryIDs = append(entryIDs, entry.Elements[i])
		}

		// Retrieve item data
		items := make([]*osqueue.QueueItem, 0, len(entryIDs))
		itemData, err := client.Do(ctx, client.B().Hmget().Key(kg.QueueItem()).Field(entryIDs...).Build()).AsStrSlice()
		if err != nil && !rueidis.IsRedisNil(err) {
			return fmt.Errorf("could not get queue items: %w", err)
		}

		l.Debug("retrieved item chunk", "key", sourceSetKey, "item_ids", entryIDs)

		for i, itemStr := range itemData {
			if itemStr == "" {
				onMissing(entryIDs[i])
				continue
			}

			qi := osqueue.QueueItem{}
			err := json.Unmarshal([]byte(itemStr), &qi)
			if err != nil {
				return fmt.Errorf("could not unmarshal queue item: %w", err)
			}

			// Item is definitely in progress if actively leased
			if qi.IsLeased(q.clock.Now()) {
				continue
			}

			items = append(items, &qi)
		}

		entriesFound := make(map[string]struct{})

		// Retrieve keys to check against (need to check individual items but want to run batched operations)
		// Worst case, this transform 20 queue items in the chunk to 20 target keys, but usually this will
		// be more efficient as items may belong to the same workflows.
		for targetKey, items := range targetKeys(items) {
			l.Debug("comparing against target", "source", sourceSetKey, "target", targetKey, "items", items)

			if len(items) == 0 {
				continue
			}

			// Batch check scores for items
			resp, err := client.Do(ctx, client.B().Zmscore().Key(targetKey).Member(items...).Build()).ToAny()
			if err != nil && !rueidis.IsRedisNil(err) {
				return fmt.Errorf("could not check target key for missing items: %w", err)
			}

			scores, ok := resp.([]interface{})
			if !ok {
				return nil
			}

			for i, score := range scores {
				if score != nil {
					entriesFound[items[i]] = struct{}{}
				}
			}
		}

		for _, element := range entryIDs {
			if _, has := entriesFound[element]; !has {
				onMissing(element)
			}
		}

		if entry.Cursor == 0 {
			return nil
		}

		cursor = entry.Cursor

		<-time.After(100 * time.Millisecond)
	}
}

func (q *queue) BacklogActiveCheckPeek(ctx context.Context, peekSize int64) ([]*QueueBacklog, error) {
	shard := q.primaryQueueShard
	client := shard.RedisClient.Client()
	kg := shard.RedisClient.KeyGenerator()

	key := kg.BacklogActiveCheckSet()

	peeker := peeker[QueueBacklog]{
		q:                      q,
		max:                    PartitionActiveCheckPeekMax,
		opName:                 "peekPartitionActiveCheck",
		isMillisecondPrecision: true,
		handleMissingItems:     CleanupMissingPointers(ctx, key, client, q.log),
		maker: func() *QueueBacklog {
			return &QueueBacklog{}
		},
		keyMetadataHash: kg.BacklogMeta(),
	}

	// Pick random partitions within bounds
	isSequential := false

	res, err := peeker.peek(ctx, key, isSequential, q.clock.Now(), peekSize)
	if err != nil {
		return nil, fmt.Errorf("could not peek active check partitions: %w", err)
	}

	return res.Items, nil
}

func (q *queue) AddBacklogToActiveCheck(ctx context.Context, shard QueueShard, accountID uuid.UUID, backlogID string) error {
	kg := shard.RedisClient.KeyGenerator()
	client := shard.RedisClient.Client()

	status, err := scripts["queue/activeCheckAddBacklog"].Exec(ctx, client, []string{
		kg.BacklogActiveCheckSet(),
		kg.BacklogActiveCheckCooldown(backlogID),
	},
		[]string{
			backlogID,
			strconv.Itoa(int(q.clock.Now().UnixMilli())),
		}).ToInt64()
	if err != nil {
		return fmt.Errorf("could not add backlog to active check: %w", err)
	}

	switch status {
	case 0:
		return nil
	default:
		return fmt.Errorf("invalid status code %v returned by add to active check", status)
	}
}
