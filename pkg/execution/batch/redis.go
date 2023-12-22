package batch

import (
	"context"

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
	return nil
}
