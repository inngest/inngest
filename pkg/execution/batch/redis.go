package batch

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/expressions"
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

func (b redisBatchManager) batchKey(ctx context.Context, evt event.Event, fn inngest.Function) (string, error) {
	if fn.EventBatch.Key == nil {
		return fn.ID.String(), nil
	}

	out, _, err := expressions.Evaluate(ctx, *fn.EventBatch.Key, map[string]any{"event": evt.Map()})
	if err != nil {
		log.From(ctx).Error().Err(err).
			Str("expression", *fn.EventBatch.Key).
			Interface("event", evt.Map()).
			Msg("error evaluating batch key expression")
		return fn.ID.String(), fmt.Errorf("invalid expression: %w", err)
	}
	if str, ok := out.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", out), nil
}

func (b redisBatchManager) batchPointer(ctx context.Context, fn inngest.Function, evt event.Event) (string, error) {
	batchPointer := b.k.BatchPointer(ctx, fn.ID)

	if fn.EventBatch.Key != nil {
		batchKey, err := b.batchKey(ctx, evt, fn)
		if err != nil {
			return "", fmt.Errorf("could not retrieve batch key: %w", err)
		}

		hashedBatchKey := sha256.Sum256([]byte(batchKey))
		encodedBatchKey := base64.StdEncoding.EncodeToString(hashedBatchKey[:])

		batchPointer = b.k.BatchPointerWithKey(ctx, fn.ID, encodedBatchKey)
	}

	return batchPointer, nil
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
	config := fn.EventBatch
	if config == nil {
		// TODO: this should not happen, report this to sentry or logs
		return nil, fmt.Errorf("no batch config found for for function: %s", fn.Slug)
	}

	batchPointer, err := b.batchPointer(ctx, fn, bi.Event)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve batch pointer: %w", err)
	}

	// script keys
	keys := []string{
		batchPointer,
	}

	// script args
	newULID := ulid.MustNew(uint64(time.Now().UnixMilli()), rand.Reader)
	args, err := redis_state.StrSlice([]any{
		config.MaxSize,
		bi,
		newULID,
		b.k.QueuePrefix(),
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

	items := []BatchItem{}
	for _, str := range itemStrList {
		item := &BatchItem{}
		if err := json.Unmarshal([]byte(str), &item); err != nil {
			return empty, fmt.Errorf("failed to decode item for batch '%s': %v", batchID, err)
		}
		items = append(items, *item)
	}

	return items, nil
}

// StartExecution sets the status to `started`
// If it has already started, don't do anything
func (b redisBatchManager) StartExecutionWithBatchPointer(ctx context.Context, batchID ulid.ULID, batchPointer string) (string, error) {
	keys := []string{
		b.k.BatchMetadata(ctx, batchID),
		batchPointer,
	}
	args := []string{
		enums.BatchStatusStarted.String(),
		ulid.Make().String(),
	}

	status, err := scripts["start"].Exec(
		ctx,
		b.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return "", fmt.Errorf("failed to start batch execution: %w", err)
	}

	switch status {
	case 0: // haven't started, so mark mark it started
		return enums.BatchStatusReady.String(), nil

	case 1: // Already started
		return enums.BatchStatusStarted.String(), nil

	default:
		return "", fmt.Errorf("invalid status for start batch ops: %d", status)
	}
}

// StartExecution sets the status to `started`
// If it has already started, don't do anything
func (b redisBatchManager) StartExecution(ctx context.Context, fnID uuid.UUID, batchID ulid.ULID) (string, error) {
	keys := []string{
		b.k.BatchMetadata(ctx, batchID),
		b.k.BatchPointer(ctx, fnID),
	}
	args := []string{
		enums.BatchStatusStarted.String(),
		ulid.Make().String(),
	}

	status, err := scripts["start"].Exec(
		ctx,
		b.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return "", fmt.Errorf("failed to start batch execution: %w", err)
	}

	switch status {
	case 0: // haven't started, so mark mark it started
		return enums.BatchStatusReady.String(), nil

	case 1: // Already started
		return enums.BatchStatusStarted.String(), nil

	default:
		return "", fmt.Errorf("invalid status for start batch ops: %d", status)
	}
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
			Key:             fmt.Sprintf("batchschedule:%s", opts.BatchID),
			AccountID:       opts.AccountID,
			WorkspaceID:     opts.WorkspaceID,
			AppID:           opts.AppID,
		},
		Attempt:     0,
		MaxAttempts: &maxAttempts,
		Payload:     opts.ScheduleBatchPayload,
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
