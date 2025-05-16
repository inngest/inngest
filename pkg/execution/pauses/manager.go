package pauses

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
)

var BlockFlushQueueName = "block-flush"

var defaultFlushDelay = 10 * time.Second

// StateBufferer transforms a state.Manager into a state.Bufferer
func StateBufferer(rsm state.Manager) Bufferer {
	return &redisAdapter{rsm}
}

// NewManager returns a new pause writer, writing pauses to a Valkey/Redis/MemoryDB
// compatible buffer
//
// Blocks are flushed from the buffer in background jobs enqueued to the given queue.
// This prevents eg. executors and new-runs from retaining blocks in-memory.
func NewManager(buf Bufferer, bs BlockStore, flusher BlockFlushEnqueuer) *manager {
	return &manager{
		buf:        buf,
		bs:         bs,
		flusher:    flusher,
		flushDelay: defaultFlushDelay,
	}
}

type manager struct {
	buf        Bufferer
	bs         BlockStore
	flusher    BlockFlushEnqueuer
	flushDelay time.Duration
}

func (m manager) ConsumePause(ctx context.Context, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, error) {
	if pause.Event == nil {
		// A Pause must always have an event for this manager, else we cannot build the
		// Index struct for deleting pauses.  It's also no longer possible to have pauses without
		// events, so this should never happen.
		return state.ConsumePauseResult{}, fmt.Errorf("pause has no event")
	}

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
	// In the future, we could add two block indexes:  pending, and stored.  this is a
	// pain, though, because we may die when uploading pending blocks, and that requires
	// a bit of thought to work around, so we’ll just go with double deletes for now,
	// assuming this won’t happen a ton.  this can be improved later.

	res, err := m.buf.ConsumePause(ctx, pause, opts)
	// Is this an ErrDuplicateResponse?  If so, we've already consumed this pause,
	// so delete it.  Similarly, if the error is nil we just consumed, so go ahead
	// and delete the pause then continue
	if err != nil {
		return res, err
	}

	idx := Index{
		pause.WorkspaceID,
		*pause.Event,
	}
	if err := m.Delete(ctx, idx, pause); err != nil {
		// We only log here if the delete fails. Consuming is idempotent and is the
		// action that updates state.
		logger.StdlibLogger(ctx).Error(
			"error deleting pause once consumed",
			"error", err,
			"pause", pause,
			"index", idx,
		)
	}
	return res, nil
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

	// If this is larger than the max buffer len, schedule a new block write.  We only
	// enqueue this job once per index ID, using queue singletons to handle these.
	if m.bs != nil && n >= m.bs.BlockSize() {
		if err := m.flusher.Enqueue(ctx, index); err != nil {
			logger.StdlibLogger(ctx).Error("error attempting to flush block", "error", err)
		}
	}

	return n, nil
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

// Delete deletes a pause from from block storage or the buffer.
func (m manager) Delete(ctx context.Context, index Index, pause state.Pause) error {
	// XXX: Potential future optimization:  cache the last written block for an index
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
	// Always also delegate to the flusher, just in case a block was written whilst
	// we issued the delete request.
	return m.bs.Delete(ctx, index, pause)
}

func (m manager) FlushIndexBlock(ctx context.Context, index Index) error {
	// Ensure we delay writing the block.  This prevents clock skew on non-precision
	// clocks from impacting out-of-order pauses;  we want pauses to be stored in-order
	// and pause blocks to contain ordered pauses.
	//
	// flushDelay is the amount of clock skew we mitigate.
	time.Sleep(m.flushDelay)
	return m.bs.FlushIndexBlock(ctx, index)
}
