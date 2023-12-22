package batch

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

func NewRedisBatchManager(r rueidis.Client, k redis_state.BatchKeyGenerator) BatchManager {
	return redisBatchManager{
		r: r,
		k: k,
	}
}

type redisBatchManager struct {
	r rueidis.Client
	k redis_state.BatchKeyGenerator
}

func (b redisBatchManager) Append(ctx context.Context, bi BatchItem) (*BatchAppendResult, error) {
	return nil, nil
}

func (b redisBatchManager) RetrieveItems(ctx context.Context, batchID ulid.ULID) ([]BatchItem, error) {
	items := make([]BatchItem, 0)
	return items, nil
}

func (b redisBatchManager) ScheduleExecution(ctx context.Context, opts ScheduleBatchOpts) error {
	return nil
}

func (b redisBatchManager) ExpireKeys(ctx context.Context, batchID ulid.ULID) error {
	keys := []string{
		b.k.Batch(ctx, batchID),
		b.k.BatchMetadata(ctx, batchID),
	}

	timeout := consts.MaxBatchTTL.Seconds()

	args, err := redis_state.StrSlice([]any{timeout})
	if err != nil {
		return fmt.Errorf("error constructing batch expiration: %w", err)
	}

	if _, err = scripts["expire"].Exec(
		ctx,
		b.r,
		keys,
		args,
	).AsInt64(); err != nil {
		return fmt.Errorf("failed to expire batch '%s' related keys: %v", batchID, err)
	}

	return nil
}
