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

	// DefaultMaxBufferByteSize is the default max cumulative byte size of buffered
	// events before flushing
	DefaultMaxBufferByteSize = 4 * 1024 * 1024
)

// appendBuffer manages in-memory buffering for batch appends across varying
// functions and batch pointers.
type appendBuffer struct {
	maxDuration       time.Duration
	maxSize           int
	maxByteSize       int
	buffers           map[bufferKey]*batchBuffer
	mu                sync.Mutex
	closed            chan struct{} // signals shutdown to unblock waiting appends
	log               logger.Logger
	totalPendingItems atomic.Int64 // tracks total items across all buffers
	totalPendingBytes atomic.Int64 // tracks event payload bytes across all buffers
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
	byteSize       int                       // cumulative byte size of buffered events
	pendingResults map[string]*pendingResult // Local dedup + result sharing
	timer          *time.Timer
	fn             inngest.Function // Function config for batch settings
	createdAt      time.Time        // set when first item appended, reset in reset()
}

// newAppendBuffer creates a new appendBuffer with the given configuration.
// maxByteSize of 0 uses DefaultMaxBufferByteSize.
func newAppendBuffer(maxDuration time.Duration, maxSize int, maxByteSize int, log logger.Logger) *appendBuffer {
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
	if maxByteSize <= 0 {
		maxByteSize = DefaultMaxBufferByteSize
	}

	return &appendBuffer{
		maxDuration: maxDuration,
		maxSize:     maxSize,
		maxByteSize: maxByteSize,
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
			batchID := ""
			if existing.result != nil {
				batchID = existing.result.BatchID
			}
			return &BatchAppendResult{
				Status:          enums.BatchItemExists,
				BatchID:         batchID,
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
	eventBytes := int64(bi.Event.Size())
	buf.byteSize += int(eventBytes)

	// Set createdAt on first item
	if buf.createdAt.IsZero() {
		buf.createdAt = time.Now()
	}

	// Track pending items gauge
	pending := ab.totalPendingItems.Add(1)
	metrics.GaugeBatchBufferItemsPending(ctx, pending, metrics.GaugeOpt{PkgName: pkgName})
	pendingBytes := ab.totalPendingBytes.Add(eventBytes)
	metrics.GaugeBatchBufferBytesPending(ctx, pendingBytes, metrics.GaugeOpt{PkgName: pkgName})
	metrics.IncrBatchBufferBytesAddedCounter(ctx, eventBytes, metrics.CounterOpt{PkgName: pkgName})

	// Check if we should flush based on function's batch config or buffer's global max
	batchMaxSize := ab.maxSize
	if fn.EventBatch != nil && fn.EventBatch.MaxSize > 0 {
		batchMaxSize = fn.EventBatch.MaxSize
	}
	shouldFlush := len(buf.items) >= batchMaxSize || buf.byteSize >= ab.maxByteSize
	flushTrigger := "size"
	if buf.byteSize >= ab.maxByteSize {
		flushTrigger = "bytesize"
	}

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
		ab.flush(buf, mgr, flushTrigger)
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

// batchBufferManager exposes the optional metric-aware bulk append path used by
// managers in this package. Other BatchManager implementations can still use
// appendBuffer through the public BulkAppend contract.
type batchBufferManager interface {
	BatchManager
	accountScopedMetricTags(context.Context, uuid.UUID, uuid.UUID) map[string]any
	bulkAppend(context.Context, []BatchItem, inngest.Function, map[string]any) (*BulkAppendResult, error)
}

var _ batchBufferManager = (*redisBatchManager)(nil)

func withBatchMetricTags(base, extra map[string]any) map[string]any {
	tags := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		tags[key] = value
	}
	for key, value := range extra {
		tags[key] = value
	}
	return tags
}

// flush commits all pending items in a buffer to storage atomically.
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
		flushBytes = int64(buf.byteSize)
	)
	buf.reset()
	buf.mu.Unlock()

	ctx := context.Background()
	triggerTags := map[string]any{"trigger": trigger}

	// Decrement pending items and record gauge
	newPending := ab.totalPendingItems.Add(-flushCount)
	metrics.GaugeBatchBufferItemsPending(ctx, newPending, metrics.GaugeOpt{PkgName: pkgName})
	newPendingBytes := ab.totalPendingBytes.Add(-flushBytes)
	metrics.GaugeBatchBufferBytesPending(ctx, newPendingBytes, metrics.GaugeOpt{PkgName: pkgName})
	if flushBytes > 0 {
		metrics.IncrBatchBufferBytesRemovedCounter(ctx, flushBytes, metrics.CounterOpt{PkgName: pkgName, Tags: triggerTags})
	}

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

	batchMaxSize := ab.maxSize
	if fn.EventBatch != nil && fn.EventBatch.MaxSize > 0 {
		batchMaxSize = fn.EventBatch.MaxSize
	}
	// Evaluate account-scoped instrumentation once per flush and before the
	// storage latency measurement. Managers without the optional metric-aware
	// path keep existing buffer metrics untagged.
	var metricTags map[string]any
	metricMgr, hasMetricManager := mgr.(batchBufferManager)
	if hasMetricManager {
		metricTags = metricMgr.accountScopedMetricTags(ctx, items[0].AccountID, items[0].WorkspaceID)
	}

	// Record per-flush metrics once for the entire flush, independent of how
	// many BulkAppend calls are made below.
	go func() {
		metrics.IncrBatchBufferFlushCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: triggerTags})
		metrics.IncrBatchBufferItemsFlushedCounter(ctx, flushCount, metrics.CounterOpt{PkgName: pkgName, Tags: triggerTags})
		metrics.HistogramBatchBufferFlushSize(ctx, flushCount, metrics.HistogramOpt{PkgName: pkgName})
		if waitDurationMs > 0 {
			metrics.HistogramBatchBufferWaitDuration(ctx, waitDurationMs, metrics.HistogramOpt{PkgName: pkgName})
		}
	}()

	// During append(), if we detect that the buffer contains `batchMaxSize` items,
	// we trigger a flush. append() releases the mutex lock on the shared buffer
	// before calling flush() and flush() get a mutex lock on the same buffer to
	// get all the items out of the buffer.
	// There is an inherent race condition here. After append releases the mutex,
	// even if the buffer is full, other appends can claim the mutex lock and
	// continue to add to the full buffer before flush can get the lock and flush
	// the buffer. This can lead to a scenario where we flush more than `batchMaxSize`
	// items in a single flush.
	//
	// Since bulk-append just appends all overflow items into a single batch,
	// this can result in a batch that is larger than the configured `batchMaxSize`.
	// To avoid that, we make multiple BulkAppend calls with chunks of `batchMaxSize`
	// items until we flush all the items in the buffer.
	//
	// Split items into chunks of batchMaxSize and call BulkAppend once per chunk.
	// Each chunk is sized to fill at most one Redis batch, so subsequent chunks
	// naturally flow into freshly-created batches after the previous one is filled
	// and its pointer rotated — without relying on Lua-level overflow handling.
	for start := 0; start < len(items); start += batchMaxSize {
		end := min(start+batchMaxSize, len(items))
		chunk := items[start:end]
		chunkPending := pending[start:end]
		storageStart := time.Now()
		var bulkResult *BulkAppendResult
		var err error
		if hasMetricManager {
			bulkResult, err = metricMgr.bulkAppend(ctx, chunk, fn, metricTags)
		} else {
			bulkResult, err = mgr.BulkAppend(ctx, chunk, fn)
		}
		storageDurationMs := time.Since(storageStart).Milliseconds()
		metrics.HistogramBatchBufferRedisFlushDuration(ctx, storageDurationMs, metrics.HistogramOpt{PkgName: pkgName, Tags: metricTags})

		if err != nil {
			ab.log.Error("error bulk-appending events to batch ", "chunk_size", len(chunk), "first_event", chunk[0].EventID, "function_id", fn.ID, "error", err)

			metrics.IncrBatchBufferErrorsCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    withBatchMetricTags(metricTags, map[string]any{"error_type": "bulk_append"}),
			})
			for _, p := range chunkPending {
				p.pending.err = err
				close(p.pending.done)
			}
			continue
		}

		if bulkResult == nil {
			continue
		}

		if err := ab.handleScheduling(bulkResult, fn, chunk[0], mgr); err != nil {
			for _, p := range chunkPending {
				p.pending.err = err
				close(p.pending.done)
			}
			continue
		}

		go func() {
			metrics.IncrBatchBufferBulkAppendCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    withBatchMetricTags(metricTags, map[string]any{"status": bulkResult.Status}),
			})
			if bulkResult.Committed > 0 {
				metrics.IncrBatchBufferItemsCommittedCounter(ctx, int64(bulkResult.Committed), metrics.CounterOpt{PkgName: pkgName, Tags: metricTags})
			}
			if bulkResult.Duplicates > 0 {
				metrics.IncrBatchBufferItemsDuplicatedCounter(ctx, int64(bulkResult.Duplicates), metrics.CounterOpt{PkgName: pkgName, Tags: metricTags})
			}
		}()

		for i, p := range chunkPending {
			p.pending.result = &BatchAppendResult{
				Status:          ab.mapBulkStatus(bulkResult.Status, i),
				BatchID:         bulkResult.BatchID,
				BatchPointerKey: bulkResult.BatchPointer,
			}
			close(p.pending.done)
		}
	}

	ab.log.Trace("flushed in-memory buffer", "len_pending", len(pending), "len_items", len(items))

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
func (ab *appendBuffer) handleScheduling(result *BulkAppendResult, fn inngest.Function, firstItem BatchItem, mgr BatchManager) error {
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
			return err
		}
	}

	// Schedule immediate execution for the full batch
	if result.Status == "full" || result.Status == "maxsize" {
		if err := ab.scheduleBatchExecution(ctx, mgr, result.BatchID, result, firstItem, fn, time.Now(), result.Status); err != nil {
			return err
		}
	}

	if result.Status == "overflow" {
		// Schedule immediate execution for the current full batch
		if err := ab.scheduleBatchExecution(ctx, mgr, result.BatchID, result, firstItem, fn, time.Now(), "overflow_full"); err != nil {
			return err
		}

		// Schedule execution after timeout for the overflow batch
		if result.NextBatchID != "" {
			if err := ab.scheduleBatchExecution(ctx, mgr, result.NextBatchID, result, firstItem, fn, time.Now().Add(timeout), "overflow_next"); err != nil {
				return err
			}
		}
	}

	// For "append" where no duplicates are present, no action is needed. The batch was already scheduled from when it was created
	return nil
}

// scheduleBatchExecution parses a batch ID, schedules execution, and emits metrics.
// Returns any batch ID parsing or execution scheduling error.
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
		ab.log.Error(
			"failed to schedule batch execution",
			"error", scheduleErr,
			"batchID", batchID,
			"function_id", fn.ID,
			"schedule_type", scheduleType,
			"scheduled_at", at,
		)
		metrics.IncrBatchBufferScheduleCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"schedule_type": scheduleType, "status": "error"},
		})
		metrics.IncrBatchBufferErrorsCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"error_type": "schedule"},
		})
		return scheduleErr
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
	buf.byteSize = 0
	buf.pendingResults = make(map[string]*pendingResult)
	buf.createdAt = time.Time{}
	if buf.timer != nil {
		buf.timer.Stop()
		buf.timer = nil
	}
}
