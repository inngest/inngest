package pauses

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
)

var (
	ErrNotInBuffer = fmt.Errorf("pause not in buffer")
)

// Index represents the index for a specific pause.  A pause is a signal
// for a given workspace/event combination and expression;  its always bound
// by this index.
type Index struct {
	WorkspaceID uuid.UUID `json:"workspaceID"`
	EventName   string    `json:"eventName"`
}

// Manager implements a buffer and a block reader/flusher, writing pauses to a buffer
// then flushing them to blocks when the buffer fills.
type Manager interface {
	// Bufferer is the core interface used to interact with pauses;  as an end user
	// of this package you need to only write and read pauses since a given date.
	Bufferer

	// ConsumePause consumes a pause.  This must be idempotent and first-write-wins:
	// only one request to consume a pause can succeed, which requires locking pauses.
	//
	// Note that this may return state.ErrPauseNotFound if the current pause ID has already
	// been consumed by another parallel process or because of a race condition.
	ConsumePause(ctx context.Context, id uuid.UUID, data any) (state.ConsumePauseResult, error)
}

// Bufferer represents a datastore which accepts all writes for pauses.
// The buffer writes them to a datastore before being periodically flushed
// to blocks on disk.
type Bufferer interface {
	// Write writes one or more pauses to the backing store.  Note that the index
	// for each pause must be the same.
	//
	// This returns the total number of pauses in the buffer.
	Write(ctx context.Context, index Index, pauses ...*state.Pause) (int, error)

	// PausesSince loads pauses in the buffer for a given index, since a given time.
	// If the time is ZeroTime, this must return all indexes in the buffer.
	//
	// Note that this does not return blocks, as this only reads from the BufferIndexer.
	//
	// NOTE: This is NOT INCLUSIVE of since, ie. the range is (since, now].
	PausesSince(ctx context.Context, index Index, since time.Time) (state.PauseIterator, error)

	// Delete deletes a pause from the buffer, or returns ErrNotInBuffer if the pause is not in
	// the buffer.
	Delete(ctx context.Context, index Index, pause state.Pause) error

	// PauseTimestamp returns the created at timestamp for a pause.
	PauseTimestamp(ctx context.Context, pause state.Pause) (time.Time, error)
}

// BlockStore is an implementation that reads and writes blocks.
type BlockStore interface {
	BlockFlusher
	BlockReader
}

// BlockFlusher is an interface which writes blocks to a backing store, and deletes pauses from
// a backing store.  Deleting pauses may write first to a buffer before compacting blocks.
type BlockFlusher interface {
	// FlushIndexBlock processes a given index, fetching pauses from the backing buffer
	// and writing to a block.
	FlushIndexBlock(ctx context.Context, index Index) error

	// BlockSize returns the number of pauses saved in each block.
	BlockSize() int

	// Delete deletes a pause from from block storage.
	Delete(ctx context.Context, index Index, pause state.Pause) error
}

// BlockReader reads blocks for a given index.
type BlockReader interface {
	// BlocksSince returns all block IDs that have been written for a given index,
	// since a given time.
	//
	// If the time is ZeroTime, all blocks for the index must be returned.
	//
	// NOTE: This is NOT INCLUSIVE of since, ie. the range is (since, now].
	BlocksSince(ctx context.Context, index Index, since time.Time) ([]ulid.ULID, error)

	// ReadBlock reads a single block given an index and block ID.
	ReadBlock(ctx context.Context, index Index, blockID ulid.ULID) (*Block, error)
}

// BlockLeaser manages leases when flushing blocks.  This is a separate interface
// and is supplied to a BlockStore when initializing.
type BlockLeaser interface {
	// Lease leases a given index, ensuring that only one worker can
	// flush an index at a time.
	Lease(ctx context.Context, index Index) (leaseID ulid.ULID, err error)

	// Renew renews a lease while we are flushing an index.
	Renew(ctx context.Context, index Index, leaseID ulid.ULID) (newLeaseID ulid.ULID, err error)

	// Revoke drops a lease, allowing any other worker to flush an index.
	Revoke(ctx context.Context, index Index, leaseID ulid.ULID) (err error)
}

type BlockKeyGenerator interface {
	// GenerateKey generates a key for a given block ID.
	BlockKey(idx Index, blockID ulid.ULID) string
}
