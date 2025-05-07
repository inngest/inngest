package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"math"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
)

// iterateNormalizationPartition scans and iterate through the global normalization partition to process backlogs needing to be normalized
func (q *queue) iterateNormalizationPartition(ctx context.Context, until time.Time, bc chan *QueueBacklog) error {
	l := logger.StdlibLogger(ctx)

	// TODO: check capacity

	// TODO introduce weight probability to blend account/global scanning
	peekedAccounts, err := q.peekGlobalNormalizeAccounts(ctx, until, NormalizeAccountPeekMax)
	if err != nil {
		return fmt.Errorf("could not peek global normalize accounts: %w", err)
	}

	if len(peekedAccounts) == 0 {
		return nil
	}

	// Reduce number of peeked partitions as we're processing multiple accounts in parallel
	// Note: This is not optimal as some accounts may have fewer partitions than others and
	// we're leaving capacity on the table. We'll need to find a better way to determine the
	// optimal peek size in this case.
	accountShadowPartitionPeekMax := int64(math.Round(float64(ShadowPartitionPeekMax / int64(len(peekedAccounts)))))

	// Scan and process account shadow partitions in parallel
	eg := errgroup.Group{}
	for _, account := range peekedAccounts {
		account := account

		eg.Go(func() error {
			partitionKey := q.primaryQueueShard.RedisClient.kg.AccountNormalizeSet(account)

			return q.iterateNormalizationShadowPartition(ctx, partitionKey, accountShadowPartitionPeekMax, until, bc, l)
		})
	}

	err = eg.Wait()
	if err != nil {
		return fmt.Errorf("failed to scan and normalize backlogs for accounts: %w", err)
	}

	// TODO: counter metric for scanned backlogs in normalization partition

	return nil
}

func (q *queue) iterateNormalizationShadowPartition(ctx context.Context, shadowPartitionIndexKey string, peekLimit int64, until time.Time, bc chan *QueueBacklog, l *slog.Logger) error {
	// Find partitions in account or globally with backlogs to normalize
	shadowPartitions, err := q.peekShadowPartitions(ctx, shadowPartitionIndexKey, peekLimit, until)
	if err != nil {
		return fmt.Errorf("could not peek shadow partitions to normalize: %w", err)
	}

	kg := q.primaryQueueShard.RedisClient.kg

	// For each partition, attempt to normalize backlogs
	for _, partition := range shadowPartitions {
		partitionKey := kg.PartitionNormalizeSet(partition.PartitionID)

		backlogs, err := duration(ctx, q.primaryQueueShard.Name, "normalize_peek", until, func(ctx context.Context) ([]*QueueBacklog, error) {
			return q.normalizePartitionPeek(ctx, partitionKey, NormalizePartitionPeekMax)
		})
		if err != nil {
			return err
		}

		for _, bl := range backlogs {
			// lease the backlog
			if err := q.leaseBacklogForNormalization(ctx, bl); err != nil {
				l.Error("error leasing backlog for normalization", "error", err, "backlog", bl)
				continue
			}

			// dump it into the channel for the workers to do their thing
			bc <- bl
		}
	}

	return nil
}

func (q *queue) normalizePartitionPeek(ctx context.Context, partitionKey string, limit int64) ([]*QueueBacklog, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "normalize_partition_peek"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for normalizePartitionPeek: %s", q.primaryQueueShard.Kind)
	}

	if limit <= 0 || limit > NormalizePartitionPeekMax {
		limit = NormalizePartitionPeekMax
	}

	keys := []string{
		partitionKey,
		q.primaryQueueShard.RedisClient.kg.BacklogMeta(),
	}

	args, err := StrSlice([]any{
		limit,
	})
	if err != nil {
		return nil, err
	}

	byt, err := scripts["queue/normalizePartitionPeek"].Exec(
		redis_telemetry.WithScriptName(ctx, "normalizePartitionPeek"),
		q.primaryQueueShard.RedisClient.Client(),
		keys,
		args,
	).AsBytes()
	if err != nil {
		return nil, err
	}

	type peekResult struct {
		Count    int64           `json:"count"`
		Backlogs []*QueueBacklog `json:"backlogs"`
		IDs      []string        `json:"ids"`
	}

	var res peekResult
	if err := json.Unmarshal(byt, &res); err != nil {
		return nil, fmt.Errorf("error parsing normalizePartitionPeek result: %w", err)
	}

	// TODO: do some clean up work

	return res.Backlogs, nil
}

func (q *queue) leaseBacklogForNormalization(ctx context.Context, bl *QueueBacklog) error {
	return fmt.Errorf("not implemented")
}

// normalizeBacklog must be called with exclusive access to the shadow partition
// NOTE: ideally this is one transaction in a lua script but enqueue_to_backlog is way too much work to
// utilize
func (q *queue) normalizeBacklog(ctx context.Context, backlog *QueueBacklog) error {
	rc := q.primaryQueueShard.RedisClient

	l := logger.StdlibLogger(ctx).With("backlog", backlog)

	// TODO: extend the lease

	shard := q.primaryQueueShard
	var processed int64
	for {
		res, err := q.BacklogNormalizePeek(ctx, backlog, NormalizePartitionPeekMax)
		if err != nil {
			return fmt.Errorf("could not peek backlog items for normalization: %w", err)
		}

		for _, item := range res.Items {
			if _, err := q.EnqueueItem(ctx, shard, *item, time.UnixMilli(item.AtMS), osqueue.EnqueueOpts{
				PassthroughJobId:       true,
				NormalizeFromBacklogID: backlog.BacklogID,
			}); err != nil {
				return fmt.Errorf("could not re-enqueue backlog item: %w", err)
			}

			processed += 1
		}

		l.Info("processed normalization for backlog",
			"processed", processed,
			"removed", res.RemovedCount,
		)
	}
}
