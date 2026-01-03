package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	mathRand "math/rand"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
)

func (q *queue) ActiveCheck(ctx context.Context) (int, error) {
	l := logger.StdlibLogger(ctx).With("scope", "active-check")

	client := q.RedisClient.Client()
	kg := q.RedisClient.KeyGenerator()

	// Check account entrypoint
	if mathRand.Intn(100) <= q.ActiveCheckAccountProbability {
		accountIDs, err := q.AccountActiveCheckPeek(ctx, q.ActiveCheckAccountConcurrency)
		if err != nil {
			return 0, fmt.Errorf("could not peek accounts for active checker: %w", err)
		}

		eg := errgroup.Group{}
		for _, accountID := range accountIDs {
			accountID := accountID
			eg.Go(func() error {
				checkID, err := ulid.New(ulid.Timestamp(q.Clock.Now()), rand.Reader)
				if err != nil {
					return fmt.Errorf("could not create checkID: %w", err)
				}

				l := l.With("account_id", accountID.String(), "check_id", checkID)

				l.Debug("attempting to active check account")

				readOnly := true
				if q.ReadOnlySpotChecks != nil && !q.ReadOnlySpotChecks(ctx, accountID) {
					readOnly = false
				}

				err = q.accountActiveCheck(logger.WithStdlib(ctx, l.With("check-scope", "account-check")), accountID, client, kg, readOnly)
				if err != nil {
					return fmt.Errorf("could not check account active items: %w", err)
				}

				err = q.activeCheckRemove(
					ctx,
					kg.AccountActiveCheckSet(),
					kg.AccountActiveCheckCooldown(accountID.String()),
					accountID.String(),
					osqueue.AccountActiveCheckCooldownDuration,
				)
				if err != nil {
					l.Error("could not remove backlog from active check set", "err", err)
				}

				return nil
			})
		}

		err = eg.Wait()
		if err != nil {
			return 0, fmt.Errorf("could not active check accounts: %w", err)
		}

		// We also always want to check backlogs, do not return yet
	}

	// Peek backlogs for active checks
	backlogs, err := q.BacklogActiveCheckPeek(ctx, q.ActiveCheckBacklogConcurrency)
	if err != nil {
		return 0, fmt.Errorf("could not peek backlogs for active checker: %w", err)
	}

	var checked int64

	eg := errgroup.Group{}

	// Process backlogs in parallel
	for _, backlog := range backlogs {
		backlog := backlog
		eg.Go(func() error {
			checkID, err := ulid.New(ulid.Timestamp(q.Clock.Now()), rand.Reader)
			if err != nil {
				return fmt.Errorf("could not create checkID: %w", err)
			}

			l := l.With("backlog", backlog, "check_id", checkID)

			l.Debug("attempting to active check backlog")

			cleanup, err := q.backlogActiveCheck(logger.WithStdlib(ctx, l), backlog, kg)
			if cleanup {
				cerr := q.activeCheckRemove(
					ctx,
					kg.BacklogActiveCheckSet(),
					kg.BacklogActiveCheckCooldown(backlog.BacklogID),
					backlog.BacklogID,
					osqueue.BacklogActiveCheckCooldownDuration,
				)
				if cerr != nil {
					l.Error("could not remove backlog from active check set", "err", cerr)
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

func (q *queue) backlogActiveCheck(ctx context.Context, b *osqueue.QueueBacklog, kg QueueKeyGenerator) (bool, error) {
	accountID := uuid.Nil

	start := q.Clock.Now()
	defer func() {
		dur := q.Clock.Now().Sub(start)

		metrics.HistogramQueueActiveCheckDuration(ctx, dur, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard": q.Name(),
				"type":        "backlog",
				"account_id":  accountID,
			},
		})
	}()

	l := logger.StdlibLogger(ctx)
	client := q.RedisClient.Client()

	var sp osqueue.QueueShadowPartition

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

	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	readOnly := true
	if q.ReadOnlySpotChecks != nil && !q.ReadOnlySpotChecks(ctx, accountID) {
		readOnly = false
	}

	l = l.With(
		"backlog_id", b.BacklogID,
		"partition_id", sp.PartitionID,
		"account_id", accountID,
	)

	l.Debug("starting active check for partition")

	// Check account
	_, accountSpotCheckProbability := q.ActiveSpotCheckProbability(ctx, accountID)
	if accountID != uuid.Nil && mathRand.Intn(100) <= accountSpotCheckProbability {
		err := q.accountActiveCheck(logger.WithStdlib(ctx, l.With("check-scope", "account-check")), accountID, client, kg, readOnly)
		if err != nil {
			return false, fmt.Errorf("could not check account active items: %w", err)
		}
	}

	// Check partition
	err := q.partitionActiveCheck(logger.WithStdlib(ctx, l.With("check-scope", "partition-check")), &sp, accountID, client, kg, readOnly)
	if err != nil {
		return false, fmt.Errorf("could not check account for invalid active items: %w", err)
	}

	// Check custom concurrency keys
	for _, key := range b.ConcurrencyKeys {
		err := q.customConcurrencyActiveCheck(logger.WithStdlib(ctx, l.With("check-scope", "backlog-check")), &sp, accountID, key, client, kg, readOnly)
		if err != nil {
			return false, fmt.Errorf("could not check custom concurrency key: %w", err)
		}
	}

	l.Debug("completed backlog check for invalid active keys")

	return true, nil
}

func (q *queue) accountActiveCheck(
	ctx context.Context,
	accountID uuid.UUID,
	client rueidis.Client,
	kg QueueKeyGenerator,
	readOnly bool,
) error {
	start := q.Clock.Now()
	defer func() {
		dur := q.Clock.Now().Sub(start)

		metrics.HistogramQueueActiveCheckDuration(ctx, dur, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard": q.Name(),
				"type":        "account",
				"account_id":  accountID.String(),
			},
		})
	}()

	l := logger.StdlibLogger(ctx)

	// Compare the account active key
	keyActive := kg.ActiveSet("account", accountID.String())

	// To the account in progress key
	keyInProgress := kg.Concurrency("account", accountID.String())

	l.Debug("checking account for invalid or missing active keys", "account_id", accountID, "key", keyActive, "in_progress", keyInProgress)

	var cursor int64

	for {
		chunkID, err := ulid.New(ulid.Timestamp(q.Clock.Now()), rand.Reader)
		if err != nil {
			return fmt.Errorf("could not create checkID: %w", err)
		}

		l := l.With("chunk_id", chunkID)

		res, err := q.activeCheckScan(ctx, keyActive, keyInProgress, cursor, q.ActiveCheckScanBatchSize)
		if err != nil {
			return fmt.Errorf("could not scan account: %w", err)
		}

		l.Debug("scanned account", "res", res)

		invalidItems := make([]string, 0)

		if len(res.MissingItems) > 0 {
			metrics.IncrQueueActiveCheckInvalidItemsFoundCounter(ctx, int64(len(res.MissingItems)), metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.Name(),
					"account_id":  accountID.String(),
					"check":       "account",
					"reason":      "missing-item",
				},
			})
			invalidItems = append(invalidItems, res.MissingItems...)
		}

		if len(res.StaleItems) > 0 {
			for _, item := range res.StaleItems {
				metrics.IncrQueueActiveCheckInvalidItemsFoundCounter(ctx, 1, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"queue_shard": q.Name(),
						"account_id":  accountID.String(),
						"check":       "account",
						"reason":      "missing-in-targets",
					},
				})
				invalidItems = append(invalidItems, item.ID)
			}
		}

		if len(invalidItems) > 0 {
			l.Debug(
				"removing invalid items from account active key",
				"mode", "account",
				"job_id", invalidItems,
				"active", keyActive,
				"in_progress", keyInProgress,
				"readonly", readOnly,
			)

			if !readOnly {
				metrics.IncrQueueActiveCheckInvalidItemsRemovedCounter(ctx, int64(len(invalidItems)), metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"queue_shard": q.Name(),
						"account_id":  accountID.String(),
						"check":       "account",
					},
				})

				cmd := client.B().Srem().Key(keyActive).Member(invalidItems...).Build()
				err := client.Do(ctx, cmd).Error()
				if err != nil {
					return fmt.Errorf("could not remove invalid items from active set: %w", err)
				}
			}
		}

		if res.NextCursor == 0 {
			break
		}

		cursor = res.NextCursor

		<-time.After(100 * time.Millisecond)
	}

	metrics.IncrQueueActiveCheckAccountScannedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.Name(),
			"account_id":  accountID.String(),
		},
	})

	return nil
}

func (q *queue) partitionActiveCheck(
	ctx context.Context,
	sp *osqueue.QueueShadowPartition,
	accountID uuid.UUID,
	client rueidis.Client,
	kg QueueKeyGenerator,
	readOnly bool,
) error {
	l := logger.StdlibLogger(ctx)

	keyActive := shadowPartitionActiveKey(*sp, kg)
	keyInProgress := shadowPartitionInProgressKey(*sp, kg)
	keyReady := shadowPartitionReadyQueueKey(*sp, kg)

	var cursor int64

	for {
		chunkID, err := ulid.New(ulid.Timestamp(q.Clock.Now()), rand.Reader)
		if err != nil {
			return fmt.Errorf("could not create checkID: %w", err)
		}

		l := l.With("chunk_id", chunkID)
		l.Debug("scanning partition",
			"cursor", cursor,
			"active", keyActive,
			"in_progress", keyInProgress,
			"ready", keyActive,
		)

		res, err := q.activeCheckScan(ctx, keyActive, keyInProgress, cursor, q.ActiveCheckScanBatchSize)
		if err != nil {
			return fmt.Errorf("could not scan partition: %w", err)
		}

		l.Debug("scanned partition", "res", res)

		invalidItems := make([]string, 0)

		if len(res.MissingItems) > 0 {
			metrics.IncrQueueActiveCheckInvalidItemsFoundCounter(ctx, int64(len(res.MissingItems)), metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.Name(),
					"account_id":  accountID.String(),
					"check":       "partition",
					"reason":      "missing-item",
				},
			})
			invalidItems = append(invalidItems, res.MissingItems...)
		}

		if len(res.StaleItems) > 0 {
			for _, item := range res.StaleItems {
				metrics.IncrQueueActiveCheckInvalidItemsFoundCounter(ctx, 1, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"queue_shard": q.Name(),
						"account_id":  accountID.String(),
						"check":       "partition",
						"reason":      "missing-in-targets",
					},
				})
				invalidItems = append(invalidItems, item.ID)
			}
		}

		if len(invalidItems) > 0 {
			l.Debug(
				"removing invalid items from active key",
				"mode", "partition",
				"job_id", invalidItems,
				"partition_id", sp.PartitionID,
				"active", keyActive,
				"ready", keyReady,
				"in_progress", keyInProgress,
				"readonly", readOnly,
			)

			if !readOnly {
				metrics.IncrQueueActiveCheckInvalidItemsRemovedCounter(ctx, int64(len(invalidItems)), metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"queue_shard": q.Name(),
						"account_id":  accountID.String(),
						"check":       "partition",
					},
				})

				cmd := client.B().Srem().Key(keyActive).Member(invalidItems...).Build()
				err := client.Do(ctx, cmd).Error()
				if err != nil {
					return fmt.Errorf("could not remove invalid items from active set: %w", err)
				}
			}
		}

		if res.NextCursor == 0 {
			break
		}

		cursor = res.NextCursor

		<-time.After(100 * time.Millisecond)
	}

	return nil
}

func (q *queue) customConcurrencyActiveCheck(ctx context.Context, sp *osqueue.QueueShadowPartition, accountID uuid.UUID, bcc osqueue.BacklogConcurrencyKey, client rueidis.Client, kg QueueKeyGenerator, readOnly bool) error {
	l := logger.StdlibLogger(ctx)

	keyActive := backlogConcurrencyKeyActiveKey(bcc, kg)
	keyInProgress := backlogConcurrencyKey(bcc, kg)

	var cursor int64

	for {
		chunkID, err := ulid.New(ulid.Timestamp(q.Clock.Now()), rand.Reader)
		if err != nil {
			return fmt.Errorf("could not create checkID: %w", err)
		}

		l := l.With("chunk_id", chunkID)
		l.Debug("scanning custom concurrency key",
			"cursor", cursor,
			"active", keyActive,
			"in_progress", keyInProgress,
		)

		res, err := q.activeCheckScan(ctx, keyActive, keyInProgress, cursor, q.ActiveCheckScanBatchSize)
		if err != nil {
			return fmt.Errorf("could not scan custom concurrency key: %w", err)
		}

		l.Debug("scanned partition", "res", res)

		invalidItems := make([]string, 0)

		if len(res.MissingItems) > 0 {
			metrics.IncrQueueActiveCheckInvalidItemsFoundCounter(ctx, int64(len(res.MissingItems)), metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.Name(),
					"account_id":  accountID.String(),
					"check":       "custom-concurrency",
					"reason":      "missing-item",
				},
			})
			invalidItems = append(invalidItems, res.MissingItems...)
		}

		if len(res.StaleItems) > 0 {
			for _, item := range res.StaleItems {
				metrics.IncrQueueActiveCheckInvalidItemsFoundCounter(ctx, 1, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"queue_shard": q.Name(),
						"account_id":  accountID.String(),
						"check":       "custom-concurrency",
						"reason":      "missing-in-targets",
					},
				})
				invalidItems = append(invalidItems, item.ID)
			}
		}

		if len(invalidItems) > 0 {
			l.Debug(
				"removing invalid items from active key",
				"job_id", invalidItems,
				"mode", "custom_concurrency",
				"bcc", bcc,
				"partition_id", sp.PartitionID,
				"active", keyActive,
				"in_progress", keyInProgress,
				"readonly", readOnly,
			)

			if !readOnly {
				metrics.IncrQueueActiveCheckInvalidItemsRemovedCounter(ctx, int64(len(invalidItems)), metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"queue_shard": q.Name(),
						"account_id":  accountID.String(),
						"check":       "custom-concurrency",
					},
				})

				cmd := client.B().Srem().Key(keyActive).Member(invalidItems...).Build()
				err := client.Do(ctx, cmd).Error()
				if err != nil {
					return fmt.Errorf("could not remove invalid items from active set: %w", err)
				}
			}
		}

		if res.NextCursor == 0 {
			break
		}

		cursor = res.NextCursor

		<-time.After(100 * time.Millisecond)
	}

	return nil
}

func (q *queue) BacklogActiveCheckPeek(ctx context.Context, peekSize int64) ([]*osqueue.QueueBacklog, error) {
	l := logger.StdlibLogger(ctx)
	client := q.RedisClient.Client()
	kg := q.RedisClient.KeyGenerator()

	key := kg.BacklogActiveCheckSet()

	peeker := peeker[osqueue.QueueBacklog]{
		q:                      q,
		max:                    q.ActiveCheckBacklogConcurrency,
		opName:                 "peekBacklogActiveCheck",
		isMillisecondPrecision: true,
		handleMissingItems:     CleanupMissingPointers(ctx, key, client, l),
		maker: func() *osqueue.QueueBacklog {
			return &osqueue.QueueBacklog{}
		},
		keyMetadataHash: kg.BacklogMeta(),
	}

	// Pick random backlogs within bounds
	isSequential := false

	res, err := peeker.peek(ctx, key, isSequential, q.Clock.Now(), peekSize)
	if err != nil {
		return nil, fmt.Errorf("could not peek active check backlogs: %w", err)
	}

	return res.Items, nil
}

func (q *queue) AccountActiveCheckPeek(ctx context.Context, peekSize int64) ([]uuid.UUID, error) {
	l := logger.StdlibLogger(ctx)
	client := q.RedisClient.Client()
	kg := q.RedisClient.KeyGenerator()

	key := kg.AccountActiveCheckSet()

	peeker := peeker[osqueue.QueueBacklog]{
		q:                      q,
		max:                    q.ActiveCheckAccountConcurrency,
		opName:                 "peekAccountActiveCheck",
		isMillisecondPrecision: true,
		handleMissingItems:     CleanupMissingPointers(ctx, key, client, l),
		keyMetadataHash:        kg.BacklogMeta(),
	}

	// Pick random account IDs within bounds
	isSequential := false

	accountIDs, err := peeker.peekUUIDPointer(ctx, key, isSequential, q.Clock.Now(), peekSize)
	if err != nil {
		return nil, fmt.Errorf("could not peek active check accounts: %w", err)
	}

	return accountIDs, nil
}

func (q *queue) AddBacklogToActiveCheck(ctx context.Context, accountID uuid.UUID, backlogID string) error {
	kg := q.RedisClient.KeyGenerator()
	client := q.RedisClient.Client()

	status, err := scripts["queue/activeCheckAddBacklog"].Exec(ctx, client, []string{
		kg.BacklogActiveCheckSet(),
		kg.BacklogActiveCheckCooldown(backlogID),
	},
		[]string{
			backlogID,
			strconv.Itoa(int(q.Clock.Now().UnixMilli())),
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

type activeCheckScanResult struct {
	NextCursor   int64
	MissingItems []string
	LeasedItems  []string
	StaleItems   []osqueue.QueueItem
}

func (q *queue) activeCheckScan(ctx context.Context, keyActive, keyInProgress string, cursor, count int64) (*activeCheckScanResult, error) {
	kg := q.RedisClient.KeyGenerator()
	client := q.RedisClient.Client()

	res, err := osqueue.Duration(
		ctx,
		q.Name(),
		"active_check_scan",
		q.Clock.Now(),
		func(ctx context.Context) (any, error) {
			res, err := scripts["queue/activeCheckScan"].Exec(ctx, client, []string{
				keyActive,
				keyInProgress,
				kg.QueueItem(),
			},
				[]string{
					strconv.Itoa(int(cursor)),
					strconv.Itoa(int(count)),
					strconv.Itoa(int(q.Clock.Now().UnixMilli())),
					kg.QueuePrefix(),
				}).ToAny()
			return res, err
		})
	if err != nil {
		return nil, fmt.Errorf("could not scan for active check: %w", err)
	}

	return parseScanResult(res)
}

func parseScanResult(res any) (*activeCheckScanResult, error) {
	returnedSet, ok := res.([]any)
	if !ok {
		return nil, fmt.Errorf("expected to receive one or more set items")
	}

	if len(returnedSet) != 4 {
		return nil, fmt.Errorf("expected 4 items to be returned")
	}

	nextCursor, ok := returnedSet[0].(string)
	if !ok {
		return nil, fmt.Errorf("missing next cursor")
	}

	parsedCursor, err := strconv.Atoi(nextCursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor returned")
	}

	missingItems, ok := returnedSet[1].([]any)
	if !ok {
		return nil, fmt.Errorf("missing missing items")
	}

	missing := make([]string, len(missingItems))
	for i, item := range missingItems {
		if itemID, ok := item.(string); ok {
			missing[i] = itemID
		}
	}

	leasedItems, ok := returnedSet[2].([]any)
	if !ok {
		return nil, fmt.Errorf("missing leased items")
	}

	leased := make([]string, len(leasedItems))
	for i, item := range leasedItems {
		if itemID, ok := item.(string); ok {
			leased[i] = itemID
		}
	}

	staleItems, ok := returnedSet[3].([]any)
	if !ok {
		return nil, fmt.Errorf("missing stale items")
	}

	stale := make([]osqueue.QueueItem, len(staleItems))
	for i, item := range staleItems {
		if itemData, ok := item.(string); ok {
			err := json.Unmarshal([]byte(itemData), &stale[i])
			if err != nil {
				return nil, fmt.Errorf("invalid queue item: %w", err)
			}
		}
	}

	return &activeCheckScanResult{
		NextCursor:   int64(parsedCursor),
		MissingItems: missing,
		LeasedItems:  leased,
		StaleItems:   stale,
	}, nil
}

func (q *queue) activeCheckRemove(ctx context.Context, keyActiveCheckSet, keyActiveCheckCooldown, pointer string, cooldown time.Duration) error {
	client := q.RedisClient.Client()

	status, err := scripts["queue/activeCheckRemove"].Exec(
		ctx,
		client,
		[]string{
			keyActiveCheckSet,
			keyActiveCheckCooldown,
		},
		[]string{
			pointer,
			strconv.Itoa(int(q.Clock.Now().UnixMilli())),
			strconv.Itoa(int(cooldown.Seconds())),
		},
	).ToInt64()
	if err != nil {
		return fmt.Errorf("could not mark active check cooldown: %w", err)
	}

	if status != 0 {
		return fmt.Errorf("invalid status received from active check removal: %v", status)
	}

	return nil
}
