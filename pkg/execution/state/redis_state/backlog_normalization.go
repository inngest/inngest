package redis_state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
)

// iterateNormalizationPartition scans and iterate through the global normalization partition to process backlogs needing to be normalized
func (q *queue) iterateNormalizationPartition(ctx context.Context, bc chan *QueueBacklog) error {
	l := logger.StdlibLogger(ctx)

	// TODO: check capacity

	// TODO: handle account peek

	// TODO: scan for backlogs to be normalized
	partitionKey := q.primaryQueueShard.RedisClient.kg.GlobalAccountNormalizeSet()

	backlogs, err := duration(ctx, q.primaryQueueShard.Name, "normalize_peek", q.clock.Now(), func(ctx context.Context) ([]*QueueBacklog, error) {
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

	// TODO: counter metric for scanned backlogs in normalization partition

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
