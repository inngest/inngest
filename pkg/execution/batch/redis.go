package batch

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
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
	config := fn.EventBatch
	if config == nil {
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
			RunID:           ulid.Make(),
			Key:             fmt.Sprintf("batchschedule:%s", opts.BatchID),
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

// AppendAndSchedule appends a new batch item. If a new batch is created, it will be scheduled to run
// after the batch timeout. If the item finalizes the batch, a function run is immediately scheduled.
func (b redisBatchManager) AppendAndSchedule(ctx context.Context, executor execution.Executor, fn inngest.Function, bi BatchItem) error {
	result, err := b.Append(ctx, bi, fn)
	if err != nil {
		return err
	}

	switch result.Status {
	case enums.BatchAppend:
		// noop
	case enums.BatchNew:
		dur, err := time.ParseDuration(fn.EventBatch.Timeout)
		if err != nil {
			return err
		}
		at := time.Now().Add(dur)

		if err := b.ScheduleExecution(ctx, ScheduleBatchOpts{
			ScheduleBatchPayload: ScheduleBatchPayload{
				BatchID:         ulid.MustParse(result.BatchID),
				AccountID:       bi.AccountID,
				WorkspaceID:     bi.WorkspaceID,
				AppID:           bi.AppID,
				FunctionID:      bi.FunctionID,
				FunctionVersion: bi.FunctionVersion,
			},
			At: at,
		}); err != nil {
			return err
		}
	case enums.BatchFull:
		// start execution immediately
		batchID := ulid.MustParse(result.BatchID)
		if err := b.RetrieveAndSchedule(ctx, executor, batchID, fn, bi.AccountID, bi.WorkspaceID, bi.AppID); err != nil {
			return fmt.Errorf("could not retrieve and schedule batch items: %w", err)
		}
	default:
		return fmt.Errorf("invalid status of batch append ops: %d", result.Status)
	}

	return nil
}

// RetrieveAndSchedule retrieves all items from a started batch and schedules a function run
func (b redisBatchManager) RetrieveAndSchedule(ctx context.Context, executor execution.Executor, batchID ulid.ULID, fn inngest.Function, accountId, workspaceId, appId uuid.UUID) error {
	evtList, err := b.RetrieveItems(ctx, batchID)
	if err != nil {
		return err
	}

	events := make([]event.TrackedEvent, len(evtList))
	for i, e := range evtList {
		events[i] = e
	}

	ctx, span := telemetry.NewSpan(ctx,
		telemetry.WithScope(consts.OtelScopeBatch),
		telemetry.WithName(consts.OtelSpanBatch),
		telemetry.WithSpanAttributes(
			attribute.String(consts.OtelSysAccountID, accountId.String()),
			attribute.String(consts.OtelSysWorkspaceID, workspaceId.String()),
			attribute.String(consts.OtelSysAppID, appId.String()),
			attribute.String(consts.OtelSysFunctionID, fn.ID.String()),
			attribute.String(consts.OtelSysBatchID, batchID.String()),
			attribute.Bool(consts.OtelSysBatchFull, true),
		))
	defer span.End()

	key := fmt.Sprintf("%s-%s", fn.ID, batchID)
	_, err = executor.Schedule(ctx, execution.ScheduleRequest{
		AccountID:      accountId,
		WorkspaceID:    workspaceId,
		AppID:          appId,
		Function:       fn,
		Events:         events,
		BatchID:        &batchID,
		IdempotencyKey: &key,
	})
	if err != nil {
		return err
	}

	if err := b.ExpireKeys(ctx, batchID); err != nil {
		return err
	}

	return nil
}
