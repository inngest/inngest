package batch

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

const (
	// DefaultMaxBufferDuration is the default max time to buffer events in-memory
	// before flushing.  must be less than the event ack deadline
	DefaultMaxBufferDuration = 500 * time.Millisecond

	// DefaultMaxBufferSize is the default max events per buffer key before flush
	DefaultMaxBufferSize = 100
)

// appendBuffer manages in-memory buffering for batch appends across varying
// functions and batch pointers.
type appendBuffer struct {
	maxDuration time.Duration
	maxSize     int
	buffers     map[bufferKey]*batchBuffer
	mu          sync.Mutex
	closed      chan struct{} // signals shutdown to unblock waiting appends
	log         logger.Logger
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
			ab.flush(buf, mgr)
		})
	}

	buf.mu.Unlock()

	// Trigger immediate flush if buffer is full
	if shouldFlush {
		ab.flush(buf, mgr)
	}

	// Block until result is available
	select {
	case <-pr.done:
		return pr.result, pr.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ab.closed:
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
	return buf
}

// flush commits all pending items in a buffer to Redis atomically.
func (ab *appendBuffer) flush(buf *batchBuffer, mgr BatchManager) {
	buf.mu.Lock()

	// nothing to flush.  buffer may have been appended to after timer started
	// which hit max cap.
	if len(buf.items) == 0 {
		buf.mu.Unlock()
		return
	}

	var (
		// snapshot before resetting
		pending = buf.items
		fn      = buf.fn
	)
	buf.reset()
	buf.mu.Unlock()

	// extract BatchItems for the bulk call
	items := make([]BatchItem, len(pending))
	for i, p := range pending {
		items[i] = p.item
	}

	// call BulkAppend - this commits all items atomically
	bulkResult, err := mgr.BulkAppend(context.Background(), items, fn)

	if err == nil && bulkResult != nil {
		ab.handleScheduling(bulkResult, fn, items[0], mgr)
	}

	ab.log.Debug("flushed in-memory buffer", "len_pending", len(pending), "len_items", len(items), "result", bulkResult)

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

	switch result.Status {
	case "new":
		// Schedule batch timeout for the new batch
		batchID, err := ulid.Parse(result.BatchID)
		if err != nil {
			ab.log.Error("failed to parse batch ID", "error", err, "batchID", result.BatchID)
			return
		}

		scheduleErr := mgr.ScheduleExecution(context.Background(), ScheduleBatchOpts{
			ScheduleBatchPayload: ScheduleBatchPayload{
				BatchID:         batchID,
				BatchPointer:    result.BatchPointer,
				AccountID:       firstItem.AccountID,
				WorkspaceID:     firstItem.WorkspaceID,
				AppID:           firstItem.AppID,
				FunctionID:      fn.ID,
				FunctionVersion: firstItem.FunctionVersion,
			},
			At: time.Now().Add(timeout),
		})
		if scheduleErr != nil {
			ab.log.Error("failed to schedule batch execution", "error", scheduleErr)
		}

	case "full", "maxsize":
		// Batch is full - schedule immediate execution
		batchID, err := ulid.Parse(result.BatchID)
		if err != nil {
			ab.log.Error("failed to parse batch ID", "error", err, "batchID", result.BatchID)
			return
		}

		scheduleErr := mgr.ScheduleExecution(context.Background(), ScheduleBatchOpts{
			ScheduleBatchPayload: ScheduleBatchPayload{
				BatchID:         batchID,
				BatchPointer:    result.BatchPointer,
				AccountID:       firstItem.AccountID,
				WorkspaceID:     firstItem.WorkspaceID,
				AppID:           firstItem.AppID,
				FunctionID:      fn.ID,
				FunctionVersion: firstItem.FunctionVersion,
			},
			At: time.Now(), // Immediate execution
		})
		if scheduleErr != nil {
			ab.log.Error("failed to schedule full batch execution", "error", scheduleErr)
		}

	case "overflow":
		// Batch overflowed - schedule immediate execution for the full batch,
		// and schedule timeout for the overflow batch
		batchID, err := ulid.Parse(result.BatchID)
		if err != nil {
			ab.log.Error("failed to parse batch ID", "error", err, "batchID", result.BatchID)
			return
		}

		// Schedule immediate execution for the full batch
		scheduleErr := mgr.ScheduleExecution(context.Background(), ScheduleBatchOpts{
			ScheduleBatchPayload: ScheduleBatchPayload{
				BatchID:         batchID,
				BatchPointer:    result.BatchPointer,
				AccountID:       firstItem.AccountID,
				WorkspaceID:     firstItem.WorkspaceID,
				AppID:           firstItem.AppID,
				FunctionID:      fn.ID,
				FunctionVersion: firstItem.FunctionVersion,
			},
			At: time.Now(), // Immediate execution
		})
		if scheduleErr != nil {
			ab.log.Error("failed to schedule full batch execution", "error", scheduleErr)
		}

		// Schedule timeout for the overflow batch
		if result.NextBatchID != "" {
			nextBatchID, err := ulid.Parse(result.NextBatchID)
			if err != nil {
				ab.log.Error("failed to parse next batch ID", "error", err, "batchID", result.NextBatchID)
				return
			}

			scheduleErr := mgr.ScheduleExecution(context.Background(), ScheduleBatchOpts{
				ScheduleBatchPayload: ScheduleBatchPayload{
					BatchID:         nextBatchID,
					BatchPointer:    result.BatchPointer,
					AccountID:       firstItem.AccountID,
					WorkspaceID:     firstItem.WorkspaceID,
					AppID:           firstItem.AppID,
					FunctionID:      fn.ID,
					FunctionVersion: firstItem.FunctionVersion,
				},
				At: time.Now().Add(timeout),
			})
			if scheduleErr != nil {
				ab.log.Error("failed to schedule overflow batch execution", "error", scheduleErr)
			}
		}
	}
	// For "append", "itemexists" - no scheduling needed
	// The batch is already scheduled from when it was created
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
		ab.flush(buf, mgr)
	}

	// Close channel to unblock any remaining waiters
	close(ab.closed)

	return nil
}

// reset resets a batch buffer
func (buf *batchBuffer) reset() {
	buf.items = make([]pendingItem, 0)
	buf.pendingResults = make(map[string]*pendingResult)
	if buf.timer != nil {
		buf.timer.Stop()
		buf.timer = nil
	}
}
