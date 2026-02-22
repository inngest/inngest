package batch

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

const (
	pkgName = "batch"

	// DefaultMaxBufferDuration is the default max time to buffer events in-memory
	// before flushing.  must be less than the event ack deadline
	DefaultMaxBufferDuration = 500 * time.Millisecond

	// DefaultMaxBufferSize is the default max events per buffer key before flush
	DefaultMaxBufferSize = 50
)

// appendBuffer manages in-memory buffering for batch appends across varying
// functions and batch pointers.
type appendBuffer struct {
	maxDuration       time.Duration
	maxSize           int
	buffers           map[bufferKey]*batchBuffer
	mu                sync.Mutex
	closed            chan struct{} // signals shutdown to unblock waiting appends
	log               logger.Logger
	totalPendingItems atomic.Int64 // tracks total items across all buffers
}

// bufferKey identifies a unique buffer based on function and batch pointer,
// used to isolate in-mem batches
type bufferKey struct {
	FunctionID   uuid.UUID
	BatchPointer string
}

// pendingItem tracks an item and its waiter channel in a buffer
type pendingItem struct {
	item BatchItem
	fn   inngest.Function
	// pending is shared between original and duplicate callers waiting for the
	// same event to be flushed
	pending *pendingResult
}

// pendingResult is shared between original and duplicate callers for the same event.
// Multiple callers can wait on the done channel, which is closed when the result is ready.
type pendingResult struct {
	done   chan struct{} // closed when result is ready
	result *BatchAppendResult
	err    error
}

// batchBuffer holds pending items for a specific buffer key
type batchBuffer struct {
	mu             sync.Mutex
	key            bufferKey
	items          []pendingItem
	pendingResults map[string]*pendingResult // Local dedup + result sharing
	timer          *time.Timer
	fn             inngest.Function // Function config for batch settings
	createdAt      time.Time        // set when first item appended, reset in reset()
}

// newAppendBuffer creates a new appendBuffer with the given configuration.
func newAppendBuffer(maxDuration time.Duration, maxSize int, log logger.Logger) *appendBuffer {
	// Clamp maxDuration to 5s max due to pub/sub ACK deadline
	if maxDuration > 5*time.Second {
		maxDuration = 5 * time.Second
	}
	if maxDuration <= 0 {
		maxDuration = DefaultMaxBufferDuration
	}
	if maxSize <= 0 {
		maxSize = DefaultMaxBufferSize
	}

	return &appendBuffer{
		maxDuration: maxDuration,
		maxSize:     maxSize,
		buffers:     make(map[bufferKey]*batchBuffer),
		closed:      make(chan struct{}),
		log:         log,
	}
}

// append adds an item to a buffer. This method BLOCKS until the event is committed
// to Redis, ensuring events are not ACK'd until persisted.
func (ab *appendBuffer) append(ctx context.Context, bi BatchItem, fn inngest.Function, mgr *redisBatchManager) (*BatchAppendResult, error) {
	batchPointer, err := mgr.batchPointer(ctx, fn, bi.Event)
	if err != nil {
		return nil, err
	}
	key := bufferKey{FunctionID: fn.ID, BatchPointer: batchPointer}

	buf := ab.getOrCreateBuffer(key, fn)
	buf.mu.Lock()

	eventIDStr := bi.EventID.String()
	if existing, seen := buf.pendingResults[eventIDStr]; seen {
		// this event is already buffered but not yet flushed.  wait for
		// the original flush to complete so we don't ACK the event before flushing
		buf.mu.Unlock()

		metrics.IncrBatchBufferDedupCounter(ctx, metrics.CounterOpt{PkgName: pkgName})

		select {
		case <-existing.done:
			if existing.err != nil {
				return nil, existing.err
			}
			return &BatchAppendResult{
				Status:          enums.BatchItemExists,
				BatchPointerKey: batchPointer,
			}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ab.closed:
			return nil, context.Canceled
		}
	}

	// Create a shared pending result for this event
	pr := &pendingResult{done: make(chan struct{})}

	// Add to buffer with pending result
	buf.items = append(buf.items, pendingItem{
		item:    bi,
		fn:      fn,
		pending: pr,
	})
	buf.pendingResults[eventIDStr] = pr

	// Set createdAt on first item
	if buf.createdAt.IsZero() {
		buf.createdAt = time.Now()
	}

	// Track pending items gauge
	pending := ab.totalPendingItems.Add(1)
	metrics.GaugeBatchBufferItemsPending(ctx, pending, metrics.GaugeOpt{PkgName: pkgName})

	// Check if we should flush based on function's batch config or buffer's global max
	batchMaxSize := ab.maxSize
	if fn.EventBatch != nil && fn.EventBatch.MaxSize > 0 {
		batchMaxSize = fn.EventBatch.MaxSize
	}
	shouldFlush := len(buf.items) >= batchMaxSize

	// If we're about to flush manually, stop the timer to prevent a concurrent
	// timer-triggered flush racing with our manual flush.
	if shouldFlush && buf.timer != nil {
		buf.timer.Stop()
		buf.timer = nil
	}

	// Start timer if not running (first item in buffer)
	if len(buf.items) == 1 && !shouldFlush {
		flushDuration := ab.flushDuration(fn)
		buf.timer = time.AfterFunc(flushDuration, func() {
			ab.flush(buf, mgr, "timer")
		})
	}

	buf.mu.Unlock()

	// Trigger immediate flush if buffer is full
	if shouldFlush {
		ab.flush(buf, mgr, "size")
	}

	// Block until result is available
	select {
	case <-pr.done:
		return pr.result, pr.err
	case <-ctx.Done():
		metrics.IncrBatchBufferErrorsCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"error_type": "context_cancelled"},
		})
		return nil, ctx.Err()
	case <-ab.closed:
		metrics.IncrBatchBufferErrorsCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"error_type": "context_cancelled"},
		})
		return nil, context.Canceled
	}
}

// flushDuration returns the duration to wait before flushing, clamped to the
// function's batch timeout to avoid buffering longer than the batch window.
func (ab *appendBuffer) flushDuration(fn inngest.Function) time.Duration {
	if fn.EventBatch == nil || fn.EventBatch.Timeout == "" {
		return ab.maxDuration
	}

	batchTimeout, err := time.ParseDuration(fn.EventBatch.Timeout)
	if err != nil || batchTimeout <= 0 || batchTimeout >= ab.maxDuration {
		return ab.maxDuration
	}

	return batchTimeout
}

// getOrCreateBuffer returns the buffer for the given key, creating it if needed.
func (ab *appendBuffer) getOrCreateBuffer(key bufferKey, fn inngest.Function) *batchBuffer {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	if buf, exists := ab.buffers[key]; exists {
		return buf
	}

	buf := &batchBuffer{
		key:            key,
		items:          make([]pendingItem, 0),
		pendingResults: make(map[string]*pendingResult),
		fn:             fn,
	}
	ab.buffers[key] = buf

	metrics.GaugeBatchBufferKeysActive(context.Background(), int64(len(ab.buffers)), metrics.GaugeOpt{PkgName: pkgName})

	return buf
}

// flush commits all pending items in a buffer to Redis atomically.
// trigger indicates why the flush occurred: "timer", "size", or "close".
func (ab *appendBuffer) flush(buf *batchBuffer, mgr BatchManager, trigger string) {
	buf.mu.Lock()

	// nothing to flush.  buffer may have been appended to after timer started
	// which hit max cap.
	if len(buf.items) == 0 {
		buf.mu.Unlock()
		return
	}

	var (
		// snapshot before resetting
		pending    = buf.items
		fn         = buf.fn
		createdAt  = buf.createdAt
		flushCount = int64(len(buf.items))
	)
	buf.reset()
	buf.mu.Unlock()

	ctx := context.Background()
	triggerTags := map[string]any{"trigger": trigger}

	// Decrement pending items and record gauge
	newPending := ab.totalPendingItems.Add(-flushCount)
	metrics.GaugeBatchBufferItemsPending(ctx, newPending, metrics.GaugeOpt{PkgName: pkgName})

	// Record wait duration if createdAt was set
	var waitDurationMs int64
	if !createdAt.IsZero() {
		waitDurationMs = time.Since(createdAt).Milliseconds()
	}

	// extract BatchItems for the bulk call
	items := make([]BatchItem, len(pending))
	for i, p := range pending {
		items[i] = p.item
	}

	// call BulkAppend - this commits all items atomically
	redisStart := time.Now()
	bulkResult, err := mgr.BulkAppend(ctx, items, fn)
	redisDurationMs := time.Since(redisStart).Milliseconds()

	metrics.HistogramBatchBufferRedisFlushDuration(ctx, redisDurationMs, metrics.HistogramOpt{PkgName: pkgName})

	if err != nil {
		metrics.IncrBatchBufferErrorsCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"error_type": "bulk_append"},
		})
	}

	if err == nil && bulkResult != nil {
		ab.handleScheduling(bulkResult, fn, items[0], mgr)

		go func() {
			metrics.IncrBatchBufferFlushCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: triggerTags})
			metrics.IncrBatchBufferItemsFlushedCounter(ctx, flushCount, metrics.CounterOpt{PkgName: pkgName, Tags: triggerTags})
			metrics.HistogramBatchBufferFlushSize(ctx, flushCount, metrics.HistogramOpt{PkgName: pkgName})
			if waitDurationMs > 0 {
				metrics.HistogramBatchBufferWaitDuration(ctx, waitDurationMs, metrics.HistogramOpt{PkgName: pkgName})
			}
			metrics.IncrBatchBufferBulkAppendCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"status": bulkResult.Status},
			})
			if bulkResult.Committed > 0 {
				metrics.IncrBatchBufferItemsCommittedCounter(ctx, int64(bulkResult.Committed), metrics.CounterOpt{PkgName: pkgName})
			}
			if bulkResult.Duplicates > 0 {
				metrics.IncrBatchBufferItemsDuplicatedCounter(ctx, int64(bulkResult.Duplicates), metrics.CounterOpt{PkgName: pkgName})
			}
		}()

	}

	ab.log.Trace("flushed in-memory buffer", "len_pending", len(pending), "len_items", len(items), "result", bulkResult)

	// Send results to all waiters
	for i, p := range pending {
		if err != nil {
			p.pending.err = err
		} else {
			status := ab.mapBulkStatus(bulkResult.Status, i)
			p.pending.result = &BatchAppendResult{
				Status:          status,
				BatchID:         bulkResult.BatchID,
				BatchPointerKey: bulkResult.BatchPointer,
			}
		}
		close(p.pending.done)
	}

	// clean up empty buffer to prevent unbounded map growth.
	ab.mu.Lock()
	buf.mu.Lock()
	if len(buf.items) == 0 {
		delete(ab.buffers, buf.key)
	}
	activeKeys := int64(len(ab.buffers))
	buf.mu.Unlock()
	ab.mu.Unlock()

	metrics.GaugeBatchBufferKeysActive(ctx, activeKeys, metrics.GaugeOpt{PkgName: pkgName})
}

// mapBulkStatus maps a bulk append status to an individual item status.
// Note: The buffer's handleScheduling handles all scheduling, so we return
// BatchAppend for most statuses to prevent the executor from interfering.
func (ab *appendBuffer) mapBulkStatus(bulkStatus string, itemIndex int) enums.Batch {
	switch bulkStatus {
	case "itemexists":
		return enums.BatchItemExists
	default:
		// Buffer's handleScheduling handles all scheduling for new, full, maxsize, overflow.
		// Return Append so executor doesn't try to schedule.
		return enums.BatchAppend
	}
}

// handleScheduling schedules batch execution based on the bulk append result.
func (ab *appendBuffer) handleScheduling(result *BulkAppendResult, fn inngest.Function, firstItem BatchItem, mgr BatchManager) {
	timeout, err := time.ParseDuration(fn.EventBatch.Timeout)
	if err != nil {
		ab.log.Error("failed to parse batch timeout", "error", err, "timeout", fn.EventBatch.Timeout)
		timeout = 60 * time.Second // fallback
	}

	ctx := context.Background()

	// For new batches, schedule an execution after the batch timeout.
	//
	// If there were duplicate events in the buffered batch, also schedule an execution after the batch timeout.
	// This is necessary for cases where the first event in a new batch fails due to transient issues like i/o timeouts writing to redis,
	// we might still write the event to a redis batch and return an error, which leads to not scheduling the batch for execution ever.
	// This results in stuck batches.
	//
	// To avoid that, we always schedule the batch for execution when any of the events are duplicates.
	// While this scheduling attempt is only required if the retried event was the first event in a new batch, it is hard to distinguish
	// that case because we bulk append. So we just schedule a job every time there are _any_ duplicate elements in a batch.
	// This is safe because batcher.ScheduleExecution is idempotent for a given batchID, so if a job already exists, the schedule call is a no-op.
	if result.Status == "new" || result.Duplicates > 0 {
		if err := ab.scheduleBatchExecution(ctx, mgr, result.BatchID, result, firstItem, fn, time.Now().Add(timeout), "new"); err != nil {
			return
		}
	}

	// Schedule immediate execution for the full batch
	if result.Status == "full" || result.Status == "maxsize" {
		if err := ab.scheduleBatchExecution(ctx, mgr, result.BatchID, result, firstItem, fn, time.Now(), result.Status); err != nil {
			return
		}
	}

	if result.Status == "overflow" {
		// Schedule immediate execution for the current full batch
		if err := ab.scheduleBatchExecution(ctx, mgr, result.BatchID, result, firstItem, fn, time.Now(), "overflow_full"); err != nil {
			return
		}

		// Schedule execution after timeout for the overflow batch
		if result.NextBatchID != "" {
			if err := ab.scheduleBatchExecution(ctx, mgr, result.NextBatchID, result, firstItem, fn, time.Now().Add(timeout), "overflow_next"); err != nil {
				return
			}
		}
	}

	// For "append" where no duplicates are present, no action is needed. The batch was already scheduled from when it was created
}

// scheduleBatchExecution parses a batch ID, schedules execution, and emits metrics.
// Returns a non-nil error only if parsing the batch ID fails.
func (ab *appendBuffer) scheduleBatchExecution(ctx context.Context, mgr BatchManager, rawBatchID string, result *BulkAppendResult, firstItem BatchItem, fn inngest.Function, at time.Time, scheduleType string) error {
	batchID, err := ulid.Parse(rawBatchID)
	if err != nil {
		ab.log.Error("failed to parse batch ID", "error", err, "batchID", rawBatchID)
		metrics.IncrBatchBufferErrorsCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"error_type": "parse_batch_id"},
		})
		return err
	}

	scheduleErr := mgr.ScheduleExecution(ctx, ScheduleBatchOpts{
		ScheduleBatchPayload: ScheduleBatchPayload{
			BatchID:         batchID,
			BatchPointer:    result.BatchPointer,
			AccountID:       firstItem.AccountID,
			WorkspaceID:     firstItem.WorkspaceID,
			AppID:           firstItem.AppID,
			FunctionID:      fn.ID,
			FunctionVersion: firstItem.FunctionVersion,
		},
		At: at,
	})
	if scheduleErr != nil {
		ab.log.Error("failed to schedule batch execution", "error", scheduleErr)
		metrics.IncrBatchBufferScheduleCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"schedule_type": scheduleType, "status": "error"},
		})
		metrics.IncrBatchBufferErrorsCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"error_type": "schedule"},
		})
	} else {
		metrics.IncrBatchBufferScheduleCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"schedule_type": scheduleType, "status": "success"},
		})
	}
	return nil
}

// close shuts down the appendBuffer, flushing all pending buffers.
func (ab *appendBuffer) close(mgr *redisBatchManager) error {
	// Flush all remaining buffers before closing the channel
	// so that pending waiters receive their results
	ab.mu.Lock()
	buffersToFlush := make([]*batchBuffer, 0, len(ab.buffers))
	for _, buf := range ab.buffers {
		buffersToFlush = append(buffersToFlush, buf)
	}
	ab.mu.Unlock()

	for _, buf := range buffersToFlush {
		ab.flush(buf, mgr, "close")
	}

	// Close channel to unblock any remaining waiters
	close(ab.closed)

	return nil
}

// reset resets a batch buffer
func (buf *batchBuffer) reset() {
	buf.items = make([]pendingItem, 0)
	buf.pendingResults = make(map[string]*pendingResult)
	buf.createdAt = time.Time{}
	if buf.timer != nil {
		buf.timer.Stop()
		buf.timer = nil
	}
}
