package batch

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

func NewRedisBatchManager(r rueidis.Client, k redis_state.BatchKeyGenerator, q redis_state.QueueManager) BatchManager {
	return redisBatchManager{
		r: r,
		k: k,
		q: q,
	}
}

type redisBatchManager struct {
	r rueidis.Client
	k redis_state.BatchKeyGenerator
	q redis_state.QueueManager
}

// Append add an item to a batch, and handle things slightly differently based on the batch sitation after
// the item is appended.
//
//  1. First item in the batch
//     Schedule a job to start the batch execution after the provided `timeout`. The scheduled job actually
//     executes or not depends on the batch state at the time.
//
//  2. Batch is full
//     Starts the batch job immediately and update the status.
//
//  3. Neither #1 or #2
//     No-op
func (b redisBatchManager) Append(ctx context.Context, bi BatchItem, fn inngest.Function) (*BatchAppendResult, error) {
	batchConfig := fn.EventBatch
	if batchConfig == nil {
		// TODO: this should not happen, report this to sentry or logs
		return nil, fmt.Errorf("no batch config found for for function: %s", fn.Slug)
	}

	// script keys
	keys := []string{
		b.k.BatchPointer(ctx, bi.FunctionID),
	}

	// script args
	newULID := ulid.MustNew(uint64(time.Now().UnixMilli()), rand.Reader)
	args, err := redis_state.StrSlice([]any{
		batchConfig.MaxSize,
		bi.GetEvent(), // NOTE
		newULID,
		// TODO: queue prefix
		enums.BatchStatusPending,
		enums.BatchStatusStarted,
	})
	if err != nil {
		return nil, fmt.Errorf("error preparing batch: %w", err)
	}

	resp, err := scripts["append"].Exec(
		ctx,
		b.r,
		keys,
		args,
	).AsBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to append event: '%s' to a batch: %v", bi.EventID, err)
	}

	result := &BatchAppendResult{}
	if err := json.Unmarshal(resp, result); err != nil {
		return nil, fmt.Errorf("failed to decode append result: %v", err)
	}

	return result, nil
}

// RetrieveItems retrieve the data associated with the specified batch.
func (b redisBatchManager) RetrieveItems(ctx context.Context, batchID ulid.ULID) ([]BatchItem, error) {
	empty := make([]BatchItem, 0)

	itemStrList, err := scripts["retrieve"].Exec(
		ctx,
		b.r,
		[]string{b.k.Batch(ctx, batchID)},
		[]string{},
	).AsStrSlice()
	if err != nil {
		return empty, fmt.Errorf("failed to retrieve list of events for batch '%s': %v", batchID, err)
	}

	items := make([]BatchItem, 0)
	for i, str := range itemStrList {
		item := &BatchItem{}
		byt := []byte(str)
		if err := json.Unmarshal(byt, &item); err != nil {
			return empty, fmt.Errorf("failed to decode item '%s' from batch '%s': %v", item.GetInternalID(), batchID, err)
		}
		items[i] = *item
	}

	return items, nil
}

// ScheduleExecution enqueues a job to run the batch job after the specified duration.
func (b redisBatchManager) ScheduleExecution(ctx context.Context, opts ScheduleBatchOpts) error {
	jobID := fmt.Sprintf("%s:%s", opts.WorkspaceID, opts.BatchID)
	maxAttempts := 20

	err := b.q.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		GroupID:     uuid.New().String(),
		WorkspaceID: opts.WorkspaceID,
		Kind:        queue.KindScheduleBatch,
		Identifier: state.Identifier{
			WorkflowID:      opts.FunctionID,
			WorkflowVersion: opts.FunctionVersion,
			RunID:           ulid.Make(),
			Key:             fmt.Sprintf("batchschedule:%s", opts.BatchID),
		},
		Attempt:     0,
		MaxAttempts: &maxAttempts,
		Payload: ScheduleBatchPayload{
			AccountID:       opts.AccountID,
			WorkspaceID:     opts.WorkspaceID,
			FunctionID:      opts.FunctionID,
			FunctionVersion: opts.FunctionVersion,
			BatchID:         opts.BatchID,
		},
	}, opts.At)
	if err == redis_state.ErrQueueItemExists {
		log.From(ctx).
			Debug().
			Interface("job_id", jobID).
			Msg("queue item already exists for scheduled batch")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error enqueueing batch scheduler: %v", err)
	}

	return nil
}

// ExpireKeys sets the TTL for the keys related to the provided batchID.
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
