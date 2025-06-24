package pauses

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
)

var BlockFlushQueueName = "block-flush"

var defaultFlushDelay = 10 * time.Second

type ManagerOpt func(m Manager)

func WithFlushDelay(delay time.Duration) ManagerOpt {
	return func(m Manager) {
		if mgr, ok := m.(*manager); ok {
			mgr.flushDelay = delay
		}
	}
}

// NewManager returns a new pause writer, writing pauses to a Valkey/Redis/MemoryDB
// compatible buffer
//
// Blocks are flushed from the buffer in background jobs enqueued to the given queue.
// This prevents eg. executors and new-runs from retaining blocks in-memory.
func NewManager(buf Bufferer, bs BlockStore, flusher BlockFlushEnqueuer, opts ...ManagerOpt) Manager {
	mgr := &manager{
		buf:        buf,
		bs:         bs,
		flusher:    flusher,
		flushDelay: defaultFlushDelay,
	}

	for _, o := range opts {
		o(mgr)
	}

	return mgr
}

// NewRedisOnlyManager is a manager that only uses Redis as a buffer, without block flushing.
func NewRedisOnlyManager(rsm state.PauseManager) Manager {
	return NewManager(
		StateBufferer(rsm),
		nil,
		nil,
	)
}

type manager struct {
	buf        Bufferer
	bs         BlockStore
	flusher    BlockFlushEnqueuer
	flushDelay time.Duration
}

// PauseTimestamp returns the created at timestamp for a pause.
func (m manager) PauseTimestamp(ctx context.Context, index Index, pause state.Pause) (time.Time, error) {
	return m.buf.PauseTimestamp(ctx, index, pause)
}

func (m manager) PauseByInvokeCorrelationID(ctx context.Context, workspaceID uuid.UUID, correlationID string) (*state.Pause, error) {
	return m.buf.PauseByInvokeCorrelationID(ctx, workspaceID, correlationID)
}

func (m manager) PauseBySignalID(ctx context.Context, workspaceID uuid.UUID, signal string) (*state.Pause, error) {
	return m.buf.PauseBySignalID(ctx, workspaceID, signal)
}

func (m manager) BufferLen(ctx context.Context, idx Index) (int64, error) {
	return m.buf.BufferLen(ctx, idx)
}

func (m manager) Aggregated(ctx context.Context, idx Index, minLen int64) (bool, error) {
	// Check the buffer length by default.
	n, err := m.buf.BufferLen(ctx, idx)
	if err != nil {
		return true, err
	}
	if n > minLen {
		return true, nil
	}
	if m.bs == nil {
		return false, nil
	}
	// If we've written a blob, aggregate, assuming there are always many pauses for this index.
	return m.bs.IndexExists(ctx, idx)
}

func (m manager) IndexExists(ctx context.Context, i Index) (bool, error) {
	ok, err := m.buf.IndexExists(ctx, i)
	if err != nil || ok || m.bs == nil {
		// It exists in the buffer, so no need to check blobstore.
		return ok, err
	}

	return m.bs.IndexExists(ctx, i)
}

func (m manager) ConsumePause(ctx context.Context, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error) {
	// NOTE: There is a race condition when flushing blocks:  we may copy a pause
	// into a block, then while writing the block to disk delete/consume a pause
	// that is being written.  In this case the metadata for a block
	// isn't yet in the index. EG:
	//
	// 1. We read the buffer and add to a block
	// 2. And while uploading the block
	// 3. In parallel, we may delete/consume one of the buffer’s pauses
	//
	// Unfortunately, we only write the block to indexes after uploads complete.
	// This means that a pause may exist in a block but have been consumed.
	//
	// This is fine technically speaking:  consuming pauses is idempotent and leases
	// each pause.
	//
	// However, in order to eventually compact we need to handle the “pause not found”
	// case when consuming, and always re-delete the pause.  that’s no big deal, but
	// not the best.
	//
	// In the future, we could add two block indexes:  pending and flushed.  this is a
	// pain, though, because we may die when uploading pending blocks, and that requires
	// a bit of thought to work around, so we’ll just go with double deletes for now,
	// assuming this won’t happen a ton.  this can be improved later.
	res, cleanup, err := m.buf.ConsumePause(ctx, pause, opts)
	// Is this an ErrDuplicateResponse?  If so, we've already consumed this pause,
	// so delete it.  Similarly, if the error is nil we just consumed, so go ahead
	// and delete the pause then continue
	if err != nil {
		return res, cleanup, err
	}

	idx := Index{WorkspaceID: pause.WorkspaceID}
	if pause.Event != nil {
		idx.EventName = *pause.Event
	}

	// Note that we cannot consume pauses from the blobstore with no event or backing
	// blob.
	if SkipFlushing(idx, []*state.Pause{&pause}) {
		// This only exists in the buffer.  Return the buffer results.
		return res, cleanup, err
	}

	// override the cleanup with idx deletion
	cleanup = func() error {
		err := m.Delete(ctx, idx, pause)
		if err != nil {
			// We only log here if the delete fails. Consuming is idempotent and is the
			// action that updates state.
			logger.StdlibLogger(ctx).Error(
				"error deleting pause once consumed",
				"error", err,
				"pause", pause,
				"index", idx,
			)
		}
		return err
	}

	return res, cleanup, nil
}

// Write writes one or more pauses to the backing store.  Note that the index
// for each pause must be the same.
//
// This returns the total number of pauses in the buffer.
func (m manager) Write(ctx context.Context, index Index, pauses ...*state.Pause) (int, error) {
	n, err := m.buf.Write(ctx, index, pauses...)
	if err != nil {
		return n, err
	}

	if m.bs == nil || SkipFlushing(index, pauses) {
		// Don't bother flushing, as this needs to be kept in the buffer.
		return n, nil
	}

	// If this is larger than the max buffer len, schedule a new block write.  We only
	// enqueue this job once per index ID, using queue singletons to handle these.
	if n >= m.bs.BlockSize() {
		if err := m.flusher.Enqueue(ctx, index); err != nil {
			logger.StdlibLogger(ctx).Error("error attempting to flush block", "error", err)
		}
	}

	return n, nil
}

func (m manager) PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error) {
	// NOTE: This is only used to look up pauses when they time out.  As of this PR, timeout jobs
	// embed each pause, prevent the need to do lookups.
	//
	// First, attempt to load this pause from the buffer.  Some pauses will definitely be here:
	//
	// - There aren't enough to flush to blocks, or we havent flushed yet.
	// - We always keep pauses by ID for `step.invoke` and `step.waitForSignal` for fast O(1)
	//   lookups to resolve these quickly
	//
	// If the pause isn't in the buffer, we check if the [env, event] index has been flushed before,
	// and if so we attempt to load from the blobstore.
	//
	//
	// # Loading from blobstores
	//
	// Loading pauses from the blobstore is hard. Pauses have V4 UUIDs as IDs:  they are random.
	// This means there's no way of knowing which block/blob a pause belongs to without an index
	// lookup of [pause ID] -> "created at".

	pause, err := m.buf.PauseByID(ctx, index, pauseID)
	if pause != nil && err == nil {
		return pause, err
	}

	if m.bs != nil {
		// We couldn't load from the buffer, so fall back.
		return m.bs.PauseByID(ctx, index, pauseID)
	}

	// without a block store we should fall back to returning the error from the buffer.
	return nil, err
}

// PausesSince loads pauses in the bfufer for a given index, since a given time.
// If the time is ZeroTime, this must return all indexes in the buffer.
//
// NOTE: On a manager, this reads from a buffer and the backing block reader to read
// all pauses for an Index, on both blobs and the buffer.
func (m manager) PausesSince(ctx context.Context, index Index, since time.Time) (state.PauseIterator, error) {
	bufIter, err := m.buf.PausesSince(ctx, index, since)
	if err != nil {
		return nil, err
	}

	if m.bs == nil {
		return bufIter, nil
	}

	blocks, err := m.bs.BlocksSince(ctx, index, since)
	if err != nil {
		return nil, err
	}

	// Read from block stores and the buffer, creating an iterator that does all.
	return newDualIter(
		index,
		bufIter,
		m.bs,
		blocks,
	), nil
}

// LoadEvaluablesSince calls PausesSince and implements the aggregate expression interface implementation
// for grouping many pauses together.
func (m manager) LoadEvaluablesSince(ctx context.Context, workspaceID uuid.UUID, eventName string, since time.Time, do func(context.Context, expr.Evaluable) error) error {
	iter, err := m.PausesSince(ctx, Index{WorkspaceID: workspaceID, EventName: eventName}, since)
	if err != nil {
		return err
	}

	for iter.Next(ctx) {
		pause := iter.Val(ctx)
		if pause == nil {
			continue
		}
		if err := do(ctx, pause); err != nil {
			return err
		}
	}

	if iter.Error() != context.Canceled && (iter.Error() != nil && iter.Error().Error() != "scan done") {
		return iter.Error()
	}

	return nil
}

// Delete deletes a pause from from block storage or the buffer.
func (m manager) Delete(ctx context.Context, index Index, pause state.Pause) error {
	// Potential future optimization:  cache the last written block for an index
	// in-memory so we can fast lookup here:
	//
	// if blockID.ts > pause.ts, skip deleting from the buffer as the pause is in a block.
	//
	// This lets us skip deleting from the buffer, as this is a longer and more complex
	// transaction than a single lookup.
	err := m.buf.Delete(ctx, index, pause)
	if err != nil && !errors.Is(err, ErrNotInBuffer) {
		return err
	}

	if m.bs == nil {
		return nil
	}

	// Always also delegate to the flusher, just in case a block was written whilst
	// we issued the delete request.
	return m.bs.Delete(ctx, index, pause)
}

func (m manager) FlushIndexBlock(ctx context.Context, index Index) error {
	if m.bs == nil {
		return nil
	}

	// Ensure we delay writing the block.  This prevents clock skew on non-precision
	// clocks from impacting out-of-order pauses;  we want pauses to be stored in-order
	// and pause blocks to contain ordered pauses.
	//
	// flushDelay is the amount of clock skew we mitigate.
	time.Sleep(m.flushDelay)
	return m.bs.FlushIndexBlock(ctx, index)
}
