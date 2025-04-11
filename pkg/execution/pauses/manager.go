package pauses

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
)

// StateBufferer transforms a state.Manager into a state.Bufferer.
func StateBufferer(rsm state.Manager) Bufferer {
	return &redisAdapter{rsm}
}

// NewManager returns a new pause writer, writing pauses to a Valkey/Redis/MemoryDB
// compatible buffer
func NewManager(buf Bufferer, flusher BlockFlusher) *manager {
	return nil
}

type manager struct {
	buf        Bufferer
	flusher    BlockFlusher
	flushDelay time.Duration
}

func (m manager) ConsumePause(ctx context.Context, id uuid.UUID, data any) (state.ConsumePauseResult, error) {
	// NOTE: There is a race condition when flushing blocks:  we may copy a pause
	// into a block, then while writing the block to disk delete/consume a pause
	// that is being written.  Unfortunately, in this case the metadata for a block
	// isn't yet in the index. EG:
	//
	// 1. We read the buffer and add to a block
	// 2. And while uploading the block
	// 3. Kn parallel, we may delete one of the buffer’s pauses
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
	// assuming this won’t happen a ton.  this can eb optimized later
	return state.ConsumePauseResult{}, fmt.Errorf("not implemented")
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

	// If this is larger than the max buffer len, schedule a new block write.
	if m.flusher != nil && n >= m.flusher.BlockSize() {
		go m.FlushIndexBlock(ctx, index)
	}

	return n, nil
}

// PausesSince loads pauses in the bfufer for a given index, since a given time.
// If the time is ZeroTime, this must return all indexes in the buffer.
//
// NOTE: On a manager, this reads from a buffer and the backing block reader to read
// all pauses for an Index, on both blobs and the buffer.
func (m manager) PausesSince(ctx context.Context, index Index, since time.Time) (state.PauseIterator, error) {
	// TODO: Read from block stores and the buffer, creating an iterator that does all.
	return nil, fmt.Errorf("not implemented")
}

// Delete deletes a pause from from block storage or the buffer.
func (m manager) Delete(ctx context.Context, index Index, pause state.Pause) error {
	// XXX: Future optimization:  cache the last written block for an index in-memory so
	// we can fast lookup here: blockID.ts > pause.ts, if so always delete from flusher.
	err := m.buf.Delete(ctx, index, pause)
	if err == nil {
		return nil
	}
	if err == ErrNotInBuffer {
		return m.flusher.Delete(ctx, index, pause)
	}
	return fmt.Errorf("error deleting pause from buffer: %w", err)
}

func (m manager) FlushIndexBlock(ctx context.Context, index Index) error {
	// Ensure we delay writing the block.  This prevents clock skew on non-precision
	// clocks from impacting out-of-order pauses;  we want pauses to be stored in-order
	// and pause blocks to contain ordered pauses.
	//
	// flushDelay is the amount of clock skew we mitigate.
	time.Sleep(m.flushDelay)
	return m.flusher.FlushIndexBlock(ctx, index)
}
