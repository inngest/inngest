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
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

const (
	defaultBatchSizeLimit                = 10 * 1024 * 1024 // 10MiB
	defaultEventIdempotenceCleanupCutOff = 120              // 120 seconds
	defaultEventIdempotenceSetTTL        = 1800             // 30 minutes
)

type RedisBatchManagerOpt func(m *redisBatchManager)

func WithRedisBatchSizeLimit(l int) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		m.sizeLimit = l
	}
}

func WithRedisBatchIdempotenceSetCleanupCutoff(l int64) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		m.idempotenceSetCleanupCutoffSeconds = l
	}
}

func WithRedisBatchIdempotenceSetTTL(ttl int64) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		m.idempotenceSetTTL = ttl
	}
}

func WithLogger(l logger.Logger) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		m.log = l
	}
}

// WithBufferSettings changes in-memory buffering before committing.  events are buffered
// for up to maxDuration or until maxSize events, whichever comes first.  when buffering is enabled,
// Append() blocks until the batch is committed to Redis.
func WithBufferSettings(maxDuration time.Duration, maxSize int) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		m.buffer = newAppendBuffer(maxDuration, maxSize, 0, m.log)
	}
}

// WithBufferMaxByteSize sets the maximum cumulative byte size of buffered events
// before a flush is triggered. 0 uses DefaultMaxBufferByteSize.
func WithBufferMaxByteSize(maxByteSize int) RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		if m.buffer != nil {
			m.buffer.maxByteSize = maxByteSize
			if maxByteSize <= 0 {
				m.buffer.maxByteSize = DefaultMaxBufferByteSize
			}
		}
	}
}

// WithoutBuffer removes buffering from in-memory items
func WithoutBuffer() RedisBatchManagerOpt {
	return func(m *redisBatchManager) {
		m.buffer = nil
	}
}

// NewRedisBatchManager creates a new redis batch manager, using Redis as the backing manager.
//
// Note that this buffers in-memory using the defaults via [DefaultMaxBufferDuration] and
// [DefaultMaxBufferSize].  to change these or disable buffering, use [WithBufferSettings]
// or [WithoutBuffer].
func NewRedisBatchManager(b *redis_state.BatchClient, q queue.QueueManager, opts ...RedisBatchManagerOpt) BatchManager {
	manager := &redisBatchManager{
		b:                                  b,
		q:                                  q,
		sizeLimit:                          defaultBatchSizeLimit,
		idempotenceSetCleanupCutoffSeconds: defaultEventIdempotenceCleanupCutOff,
		idempotenceSetTTL:                  defaultEventIdempotenceSetTTL,
		log:                                logger.StdlibLogger(context.Background()),
	}

	// add default buffer
	manager.buffer = newAppendBuffer(DefaultMaxBufferDuration, DefaultMaxBufferSize, 0, manager.log)

	for _, apply := range opts {
		apply(manager)
	}

	return manager
}

type redisBatchManager struct {
	b *redis_state.BatchClient
	q queue.QueueManager

	// sizeLimit is the size limit that a batch can have
	sizeLimit int
	// All event IDs appended to a batch are tracked in a set to ensure idempotence to guard against processsing of duplicate eventIDs.
	// This cutoff denotes the last X seconds of eventsIDs to keep active in the idempotence set.
	idempotenceSetCleanupCutoffSeconds int64
	// Every append call sets the TTL to this value to ensure that after this amount of inactivity, this set gets cleared.
	idempotenceSetTTL int64
	log               logger.Logger

	// buffer is optional in-memory buffering for batch appends.
	// When nil, appends go directly to Redis. When set, appends are buffered
	// and flushed periodically or when the buffer is full.
	buffer *appendBuffer
}

func (b *redisBatchManager) batchKey(ctx context.Context, evt event.Event, fn inngest.Function) (string, error) {
	if fn.EventBatch.Key == nil {
		return "default", nil
	}

	out, err := expressions.Evaluate(ctx, *fn.EventBatch.Key, map[string]any{"event": evt.Map()})
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

func (b *redisBatchManager) batchPointer(ctx context.Context, fn inngest.Function, evt event.Event) (string, error) {
	batchPointer := b.b.KeyGenerator().BatchPointer(ctx, fn.ID)

	if fn.EventBatch.Key != nil {
		batchKey, err := b.batchKey(ctx, evt, fn)
		if err != nil {
			return "", fmt.Errorf("could not retrieve batch key: %w", err)
		}

		encodedBatchKey := HashBatchKey(batchKey)

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
//
// When buffering is enabled, this method blocks until the batch is committed to Redis.
func (b *redisBatchManager) Append(ctx context.Context, bi BatchItem, fn inngest.Function) (*BatchAppendResult, error) {
	// If buffering is enabled, delegate to buffer
	if b.buffer != nil {
		return b.buffer.append(ctx, bi, fn, b)
	}

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

	nowUnixSeconds := time.Now().Unix()
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
		nowUnixSeconds,
		b.idempotenceSetTTL,
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
func (b *redisBatchManager) RetrieveItems(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID) ([]BatchItem, error) {
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
func (b *redisBatchManager) StartExecution(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID, batchPointer string) (string, error) {
	keys := []string{
		b.b.KeyGenerator().BatchMetadata(ctx, functionId, batchID),
		batchPointer,
	}
	args := []string{
		enums.BatchStatusStarted.String(),
		ulid.Make().String(),
		batchID.String(), // Pass batchID for conditional pointer update
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
func (b *redisBatchManager) ScheduleExecution(ctx context.Context, opts ScheduleBatchOpts) error {
	if b.q == nil {
		// No queue manager configured, skip scheduling (useful for tests)
		return nil
	}

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
	if err == queue.ErrQueueItemExists {
		b.log.Debug("queue item already exists for scheduled batch", "job_id", jobID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("error enqueueing batch scheduler: %v", err)
	}

	return nil
}

// DeleteKeys drops keys related to the provided batchID.
func (b *redisBatchManager) DeleteKeys(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID) error {
	keys := []string{
		b.b.KeyGenerator().Batch(ctx, functionId, batchID),
		b.b.KeyGenerator().BatchMetadata(ctx, functionId, batchID),
	}
	nowUnixSeconds := time.Now().Unix()

	args, err := redis_state.StrSlice([]any{
		b.b.KeyGenerator().BatchIdempotenceKey(ctx, functionId),
		nowUnixSeconds - b.idempotenceSetCleanupCutoffSeconds,
	})
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

// DeleteBatch deletes the current batch for a function and batch key.
func (b *redisBatchManager) DeleteBatch(ctx context.Context, functionID uuid.UUID, batchKey string) (*DeleteBatchResult, error) {
	// First get the batch info to know what we're deleting
	info, err := b.GetBatchInfo(ctx, functionID, batchKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch info: %w", err)
	}

	if info.BatchID == "" {
		// No active batch to delete
		return &DeleteBatchResult{
			Deleted:   false,
			BatchID:   "",
			ItemCount: 0,
		}, nil
	}

	batchID, err := ulid.Parse(info.BatchID)
	if err != nil {
		return nil, fmt.Errorf("invalid batch ID: %w", err)
	}

	// Delete the batch keys
	if err := b.DeleteKeys(ctx, functionID, batchID); err != nil {
		return nil, fmt.Errorf("failed to delete batch keys: %w", err)
	}

	// Delete the batch pointer
	var batchPointerKey string
	if batchKey == "" || batchKey == "default" {
		batchPointerKey = b.b.KeyGenerator().BatchPointer(ctx, functionID)
	} else {
		encodedBatchKey := HashBatchKey(batchKey)
		batchPointerKey = b.b.KeyGenerator().BatchPointerWithKey(ctx, functionID, encodedBatchKey)
	}

	if err := b.b.Client().Do(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Del().Key(batchPointerKey).Build()
	}).Error(); err != nil {
		return nil, fmt.Errorf("failed to delete batch pointer: %w", err)
	}

	return &DeleteBatchResult{
		Deleted:   true,
		BatchID:   info.BatchID,
		ItemCount: len(info.Items),
	}, nil
}

// RunBatch schedules immediate execution of a batch by creating a timeout job that runs in one second.
func (b *redisBatchManager) RunBatch(ctx context.Context, opts RunBatchOpts) (*RunBatchResult, error) {
	// Get the batch info first
	info, err := b.GetBatchInfo(ctx, opts.FunctionID, opts.BatchKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch info: %w", err)
	}

	if info.BatchID == "" {
		// No active batch to run
		return &RunBatchResult{
			Scheduled: false,
			BatchID:   "",
			ItemCount: 0,
		}, nil
	}

	batchID, err := ulid.Parse(info.BatchID)
	if err != nil {
		return nil, fmt.Errorf("invalid batch ID: %w", err)
	}

	// Determine the batch pointer key
	var batchPointerKey string
	if opts.BatchKey == "" || opts.BatchKey == "default" {
		batchPointerKey = b.b.KeyGenerator().BatchPointer(ctx, opts.FunctionID)
	} else {
		encodedBatchKey := HashBatchKey(opts.BatchKey)
		batchPointerKey = b.b.KeyGenerator().BatchPointerWithKey(ctx, opts.FunctionID, encodedBatchKey)
	}

	// Get function version from the first batch item if available
	functionVersion := 0
	if len(info.Items) > 0 {
		functionVersion = info.Items[0].FunctionVersion
	}

	// Schedule execution to run in 1 second
	scheduleOpts := ScheduleBatchOpts{
		ScheduleBatchPayload: ScheduleBatchPayload{
			BatchID:         batchID,
			BatchPointer:    batchPointerKey,
			AccountID:       opts.AccountID,
			WorkspaceID:     opts.WorkspaceID,
			AppID:           opts.AppID,
			FunctionID:      opts.FunctionID,
			FunctionVersion: functionVersion,
		},
		At: time.Now().Add(time.Second),
	}

	if err := b.ScheduleExecution(ctx, scheduleOpts); err != nil {
		return nil, fmt.Errorf("failed to schedule batch execution: %w", err)
	}

	return &RunBatchResult{
		Scheduled: true,
		BatchID:   info.BatchID,
		ItemCount: len(info.Items),
	}, nil
}

// BulkAppendResult represents the result of a bulk append operation.
type BulkAppendResult struct {
	Status        string `json:"status"`                  // "new", "append", "full", "overflow", "maxsize", "itemexists"
	BatchID       string `json:"batchID"`                 // The batch ID that events were added to
	BatchPointer  string `json:"batchPointerKey"`         // The batch pointer key
	Committed     int    `json:"committed"`               // Number of events committed
	Duplicates    int    `json:"duplicates"`              // Number of duplicate events skipped
	NextBatchID   string `json:"nextBatchID,omitempty"`   // If overflow, the new batch ID
	OverflowCount int    `json:"overflowCount,omitempty"` // Number of events in overflow batch
}

// BulkAppend appends multiple items to a batch atomically. If the batch becomes full,
// overflow events are placed into a new batch.
func (b *redisBatchManager) BulkAppend(ctx context.Context, items []BatchItem, fn inngest.Function) (*BulkAppendResult, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to append")
	}

	config := fn.EventBatch
	if config == nil {
		return nil, fmt.Errorf("no batch config found for function: %s", fn.Slug)
	}

	// Use the first item's event to determine the batch pointer
	batchPointer, err := b.batchPointer(ctx, fn, items[0].Event)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve batch pointer: %w", err)
	}

	keys := []string{
		batchPointer,
	}

	nowUnixSeconds := time.Now().Unix()
	newULID := ulid.MustNew(uint64(time.Now().UnixMilli()), rand.Reader)
	overflowULID := ulid.MustNew(uint64(time.Now().UnixMilli())+1, rand.Reader)

	// Build args: batchLimit, batchSizeLimit, prefix, statuses, timestamps, ULIDs, eventCount, then event pairs
	baseArgs := []any{
		config.MaxSize,
		b.sizeLimit,
		b.b.KeyGenerator().QueuePrefix(ctx, items[0].FunctionID),
		enums.BatchStatusPending,
		enums.BatchStatusStarted,
		nowUnixSeconds,
		b.idempotenceSetTTL,
		newULID,
		overflowULID,
		len(items),
	}

	// Add event pairs: eventID1, event1, eventID2, event2, ...
	for _, item := range items {
		baseArgs = append(baseArgs, item.EventID.String(), item)
	}

	args, err := redis_state.StrSlice(baseArgs)
	if err != nil {
		return nil, fmt.Errorf("error preparing bulk batch: %w", err)
	}

	resp, err := retriableScripts["bulk_append"].Exec(
		ctx,
		b.b.Client(),
		keys,
		args,
	).AsBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to bulk append events to batch: %v", err)
	}

	result := &BulkAppendResult{}
	if err := json.Unmarshal(resp, result); err != nil {
		return nil, fmt.Errorf("failed to decode bulk append result: %v", err)
	}

	return result, nil
}

// Close gracefully shuts down the batch manager, flushing any pending buffers.
func (b *redisBatchManager) Close() error {
	if b.buffer != nil {
		return b.buffer.close(b)
	}
	return nil
}

// GetBatchInfo retrieves information about the current batch for a function and batch key.
func (b *redisBatchManager) GetBatchInfo(ctx context.Context, functionID uuid.UUID, batchKey string) (*BatchInfo, error) {
	// Determine the batch pointer key based on the batch key
	// When batchKey is "default" or empty, use BatchPointer (no key suffix)
	// This matches the behavior of Append when fn.EventBatch.Key is nil
	var batchPointerKey string
	if batchKey == "" || batchKey == "default" {
		batchPointerKey = b.b.KeyGenerator().BatchPointer(ctx, functionID)
	} else {
		// Hash the batch key to get the pointer key
		encodedBatchKey := HashBatchKey(batchKey)
		batchPointerKey = b.b.KeyGenerator().BatchPointerWithKey(ctx, functionID, encodedBatchKey)
	}

	// Get the batch ID from the pointer
	batchIDStr, err := b.b.Client().Do(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Get().Key(batchPointerKey).Build()
	}).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// No active batch
			return &BatchInfo{
				BatchID: "",
				Items:   []BatchItem{},
				Status:  "none",
			}, nil
		}
		return nil, fmt.Errorf("failed to get batch pointer: %w", err)
	}

	batchID, err := ulid.Parse(batchIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid batch ID in pointer: %w", err)
	}

	// Retrieve the batch items
	items, err := b.RetrieveItems(ctx, functionID, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve batch items: %w", err)
	}

	// Get the batch metadata (status)
	metadataKey := b.b.KeyGenerator().BatchMetadata(ctx, functionID, batchID)
	status, err := b.b.Client().Do(ctx, func(c rueidis.Client) rueidis.Completed {
		return c.B().Hget().Key(metadataKey).Field("status").Build()
	}).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			status = "pending"
		} else {
			status = "unknown"
		}
	}

	return &BatchInfo{
		BatchID: batchIDStr,
		Items:   items,
		Status:  status,
	}, nil
}
