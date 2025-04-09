package pauses

import (
	"context"
	"fmt"
	"time"

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

func (m manager) FlushIndexBlock(ctx context.Context, index Index) error {
	// Ensure we delay writing the block.  This prevents clock skew on non-precision
	// clocks from impacting out-of-order pauses;  we want pauses to be stored in-order
	// and pause blocks to contain ordered pauses.
	//
	// flushDelay is the amount of clock skew we mitigate.
	time.Sleep(m.flushDelay)
	return m.flusher.FlushIndexBlock(ctx, index)
}

// redisAdapter transforms a state.Manager into a state.Buffer, changing the interfaces slightly
// according to this package.
type redisAdapter struct {
	// rsm represents the redis state manager in redis_state.
	rsm state.Manager
}

// Write writes one or more pauses to the backing store.  Note that the index
// for each pause must be the same.
//
// This returns the total number of pauses in the buffer.
func (r redisAdapter) Write(ctx context.Context, index Index, pauses ...*state.Pause) (int, error) {
	var total int
	for _, p := range pauses {
		n, err := r.rsm.SavePause(ctx, *p)
		if err != nil {
			return 0, err
		}
		total = int(n)

	}
	return total, nil
}

// PausesSince loads pauses in the bfufer for a given index, since a given time.
// If the time is ZeroTime, this must return all indexes in the buffer.
//
// Note that this does not return blocks, as this only reads from the backing redis index.
func (r redisAdapter) PausesSince(ctx context.Context, index Index, since time.Time) (state.PauseIterator, error) {
	return r.rsm.PausesByEventSince(ctx, index.WorkspaceID, index.EventName, since)
}
