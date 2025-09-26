package batch

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

const (
	defaultBatchSizeLimit = 10 * 1024 * 1024 // 10MiB
)

type RedisBatchManagerOpt func(m *redisBatchManager)

func WithRedisBatchSizeLimit(l int) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		m.sizeLimit = l
	}
}

func WithLogger(l logger.Logger) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		m.log = l
	}
}

func NewRedisBatchManager(b *redis_state.BatchClient, q redis_state.QueueManager, opts ...RedisBatchManagerOpt) BatchManager {
	manager := redisBatchManager{
		b:         b,
		q:         q,
		sizeLimit: defaultBatchSizeLimit,
		log:       logger.StdlibLogger(context.Background()),
	}

	for _, apply := range opts {
		apply(&manager)
	}

	return manager
}

type redisBatchManager struct {
	b *redis_state.BatchClient
	q redis_state.QueueManager

	// sizeLimit is the size limit that a batch can have
	sizeLimit int
	log       logger.Logger
}

func (b redisBatchManager) batchKey(ctx context.Context, evt event.Event, fn inngest.Function) (string, error) {
	if fn.EventBatch.Key == nil {
		return "default", nil
	}

	out, _, err := expressions.Evaluate(ctx, *fn.EventBatch.Key, map[string]any{"event": evt.Map()})
	if err != nil {
		b.log.Error("error evaluating batch key expression",
			"error", err,
			"expression", *fn.EventBatch.Key,
			"event", evt.Map(),
		)
		return fn.ID.String(), fmt.Errorf("invalid expression: %w", err)
	}
	if str, ok := out.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", out), nil
}

func (b redisBatchManager) batchPointer(ctx context.Context, fn inngest.Function, evt event.Event) (string, error) {
	batchPointer := b.b.KeyGenerator().BatchPointer(ctx, fn.ID)

	if fn.EventBatch.Key != nil {
		batchKey, err := b.batchKey(ctx, evt, fn)
		if err != nil {
			return "", fmt.Errorf("could not retrieve batch key: %w", err)
		}

		hashedBatchKey := sha256.Sum256([]byte(batchKey))
		encodedBatchKey := base64.StdEncoding.EncodeToString(hashedBatchKey[:])

		batchPointer = b.b.KeyGenerator().BatchPointerWithKey(ctx, fn.ID, encodedBatchKey)
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
		bi.EventID.String(),
		bi,
		newULID,
		// This is used within the Lua script to create the batch metadata key
		b.b.KeyGenerator().QueuePrefix(ctx, bi.FunctionID),
		enums.BatchStatusPending,
		enums.BatchStatusStarted,
		b.sizeLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("error preparing batch: %w", err)
	}

	resp, err := retriableScripts["append"].Exec(
		ctx,
		b.b.Client(),
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
func (b redisBatchManager) RetrieveItems(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID) ([]BatchItem, error) {
	empty := make([]BatchItem, 0)

	itemStrList, err := retriableScripts["retrieve"].Exec(
		ctx,
		b.b.Client(),
		[]string{b.b.KeyGenerator().Batch(ctx, functionId, batchID)},
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
func (b redisBatchManager) StartExecution(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID, batchPointer string) (string, error) {
	keys := []string{
		b.b.KeyGenerator().BatchMetadata(ctx, functionId, batchID),
		batchPointer,
	}
	args := []string{
		enums.BatchStatusStarted.String(),
		ulid.Make().String(),
	}

	status, err := retriableScripts["start"].Exec(
		ctx,
		b.b.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return "", fmt.Errorf("failed to start batch execution: %w", err)
	}

	switch status {
	case -1: // the batch status is gone somehow
		return enums.BatchStatusAbsent.String(), nil

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
	jobID := opts.JobID()
	maxAttempts := consts.MaxRetries + 1

	queueName := queue.KindScheduleBatch
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
		QueueName:   &queueName,
	}, opts.At, queue.EnqueueOpts{})
	if err == redis_state.ErrQueueItemExists {
		b.log.Debug("queue item already exists for scheduled batch", "job_id", jobID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("error enqueueing batch scheduler: %v", err)
	}

	return nil
}

// DeleteKeys drops keys related to the provided batchID.
func (b redisBatchManager) DeleteKeys(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID) error {
	keys := []string{
		b.b.KeyGenerator().Batch(ctx, functionId, batchID),
		b.b.KeyGenerator().BatchMetadata(ctx, functionId, batchID),
		b.b.KeyGenerator().BatchIdempotenceKey(ctx, functionId, batchID),
	}

	args, err := redis_state.StrSlice([]any{})
	if err != nil {
		return fmt.Errorf("error constructing batch deletion: %w", err)
	}

	if _, err = retriableScripts["drop_keys"].Exec(
		ctx,
		b.b.Client(),
		keys,
		args,
	).AsInt64(); err != nil {
		return fmt.Errorf("failed to delete batch '%s' related keys: %v", batchID, err)
	}

	return nil
}
