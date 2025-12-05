package pauses

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/expressions/expragg"
	"github.com/oklog/ulid/v2"
)

var ErrNotInBuffer = fmt.Errorf("pause not in buffer")

// SkipFlushing returns whether we should skip flushing for any of the pauses
// defined in the slice, or for given indexes.
//
// This allows us to keep specific pauses for a given index in the buffer.
func SkipFlushing(index Index, pauses []*state.Pause) bool {
	if index.EventName == "" {
		// Signals aren't events, but are pauses:  these require O(1) lookups, so ignore
		// flushing without events.
		return true
	}
	if index.EventName == consts.FnFinishedName {
		// If the index is for the FnFinishedName, treat this as an invoke and never
		// flush this block for fast O(1) lookups.
		return true
	}

	return slices.ContainsFunc(pauses, func(s *state.Pause) bool {
		// NOTE: If this pause has a correlation ID, ignore the block flushing for this index.
		// CorrelationIDs must NEVER be flushed, as we use the buffer for an O(1) lookup to
		// retrieve pauses with low latency to resolve invokes and step.waitForCallback.
		return s.InvokeCorrelationID != nil
	})
}

// Index represents the index for a specific pause.  A pause is a signal
// for a given workspace/event combination and expression;  its always bound
// by this index.
type Index struct {
	WorkspaceID uuid.UUID `json:"workspaceID"`
	EventName   string    `json:"eventName"`
}

// BlockInfo contains information about a single block.
type BlockInfo struct {
	ID             string    `json:"id"`
	Length         int       `json:"length"`
	FirstTimestamp time.Time `json:"firstTimestamp"`
	LastTimestamp  time.Time `json:"lastTimestamp"`
	DeleteCount    int64     `json:"deleteCount"`
}

// IndexStats contains statistics about a pause index.
type IndexStats struct {
	WorkspaceID  uuid.UUID    `json:"workspaceID"`
	EventName    string       `json:"eventName"`
	BufferLength int64        `json:"bufferLength"`
	Blocks       []*BlockInfo `json:"blocks"`
}

// PauseIndex returns an index for a given pause.
func PauseIndex(p state.Pause) Index {
	idx := Index{WorkspaceID: p.WorkspaceID}
	if p.Event != nil {
		idx.EventName = *p.Event
	}
	return idx
}

// Manager implements a buffer and a block reader/flusher, writing pauses to a buffer
// then flushing them to blocks when the buffer fills.
type Manager interface {
	// EvaluableLoader allows the Manager to be used within aggregate expression engines.
	expragg.EvaluableLoader

	// Bufferer is the core interface used to interact with pauses;  as an end user
	// of this package you need to only write and read pauses since a given date.
	Bufferer

	// Aggregated returns whether the index should be aggregated.  This is a quick lookup
	// to see if we have flushed pauses to blocks, or if the buffer length is greater than
	// the given number
	Aggregated(ctx context.Context, index Index, minLen int64) (bool, error)

	// PausesSince loads pauses in the buffer for a given index, since a given time.
	// If the time is ZeroTime, this must return all indexes in the buffer.  This time is
	// inclusive, ie. it will include pauses created from the current since timestamp in
	// seconds.
	//
	// Note that this does not return blocks, as this only reads from the BufferIndexer.
	PausesSince(ctx context.Context, index Index, since time.Time) (state.PauseIterator, error)

	// PauseByID fetches a pause for a given ID.  It may return the pause from the buffer
	// or from block storage, depending on the pause
	PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error)

	// Write writes one or more pauses to the backing store.  Note that the index
	// for each pause must be the same.
	//
	// This returns the total number of pauses in the buffer.
	Write(ctx context.Context, index Index, pauses ...*state.Pause) (int, error)

	// IndexExists returns whether the given index has pauses.  This returns true if there
	// are items in the buffer, or if there are any blocks written to the backing block store.
	IndexExists(ctx context.Context, i Index) (bool, error)

	// ConsumePause consumes a pause.  This must be idempotent and first-write-wins:
	// only one request to consume a pause can succeed, which requires locking pauses.
	//
	// Note that this may return state.ErrPauseNotFound if the current pause ID has already
	// been consumed by another parallel process or because of a race condition.
	//
	// NOTE: This consumes a pause in the buffer, then calls m.Delete to ensure the pause is
	// deleted from the backing block store.
	ConsumePause(ctx context.Context, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error)

	// Delete deletes a pause from either the block index or the buffer, depending on
	// where the pause is stored.
	Delete(ctx context.Context, index Index, pause state.Pause, opts ...state.DeletePauseOpt) error

	// FlushIndexBlock flushes a new pauses block for the specified index.
	FlushIndexBlock(ctx context.Context, index Index) error

	// IndexStats returns statistics about an index including block information.
	// Used for debugging pause storage and block compaction status.
	IndexStats(ctx context.Context, index Index) (*IndexStats, error)

	// GetBlockPauseIDs returns all pause IDs from a specific block.
	// Used for debugging block contents. Returns IDs, total count, and error.
	GetBlockPauseIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error)

	// GetBlockDeletedIDs returns all deleted pause IDs for a specific block.
	// Used for debugging block deletion tracking. Returns IDs, total count, and error.
	GetBlockDeletedIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error)

	// DeletePauseByID deletes a pause by its ID, handling both buffer and block storage
	DeletePauseByID(ctx context.Context, pauseID uuid.UUID, workspaceID uuid.UUID) error
}

// Bufferer represents a datastore which accepts all writes for pauses.
// The buffer writes them to a datastore before being periodically flushed
// to blocks on disk.
type Bufferer interface {
	state.PauseDeleter
	// Write writes one or more pauses to the backing store.  Note that the index
	// for each pause must be the same.
	//
	// This returns the total number of pauses in the buffer.
	Write(ctx context.Context, index Index, pauses ...*state.Pause) (int, error)

	// IndexExists returns whether the given index has pauses.  This returns true if there
	// are items in the buffer, or if there are any blocks written to the backing block store.
	IndexExists(ctx context.Context, i Index) (bool, error)

	// BufferLen returns the number of pauses stored in the buffer for a given index.
	BufferLen(ctx context.Context, i Index) (int64, error)

	// PausesSince loads pauses in the buffer for a given index, since a given time.
	// If the time is ZeroTime, this must return all indexes in the buffer.  This time is
	// inclusive, ie. it will include pauses created from the current since timestamp in
	// seconds.
	//
	// Note that this does not return blocks, as this only reads from the BufferIndexer.
	PausesSince(ctx context.Context, index Index, since time.Time) (state.PauseIterator, error)

	// PausesSinceWithCreatedAt loads up to limit pauses for a given index since a given time,
	// ordered by creation time, with createdAt populated from Redis sorted set scores. The since time is inclusive.
	PausesSinceWithCreatedAt(ctx context.Context, index Index, since time.Time, limit int64) (state.PauseIterator, error)

	// Delete deletes a pause from the buffer, or returns ErrNotInBuffer if the pause is not in
	// the buffer.
	Delete(ctx context.Context, index Index, pause state.Pause, opts ...state.DeletePauseOpt) error

	// PauseTimestamp returns the created at timestamp for a pause.
	PauseTimestamp(ctx context.Context, index Index, pause state.Pause) (time.Time, error)

	// ConsumePause consumes a pause, writing the deleted status to the buffer.
	ConsumePause(ctx context.Context, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error)

	// PauseByID fetches a pause for a given ID.  It may return the pause from the buffer
	// or from block storage, depending on the pause
	PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error)

	// PauseByInvokeCorrelationID returns a given pause by the correlation ID.
	//
	// This must return expired invoke pauses that have not yet been consumed
	// in order to properly handle timeouts.
	//
	// NOTE: The bufferer handles O1 lookups of correlation IDs -> pauses.  These are not removed
	// from the buffer and flushed to blocks
	PauseByInvokeCorrelationID(ctx context.Context, envID uuid.UUID, correlationID string) (*state.Pause, error)

	// PauseBySignalID returns a given pause by the signal ID.
	//
	// NOTE: The bufferer handles O1 lookups of signals -> pauses.  These are not removed
	// from the buffer and flushed to blocks
	PauseBySignalID(ctx context.Context, envID uuid.UUID, signalID string) (*state.Pause, error)
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

	// Delete deletes a pause from block storage.
	Delete(ctx context.Context, index Index, pause state.Pause, opts ...state.DeletePauseOpt) error

	// DeleteByID deletes a pause from a block by its ID.
	DeleteByID(ctx context.Context, pauseID uuid.UUID, workspaceID uuid.UUID) error
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

	// LastBlockMetadata returns metadata on the last block written for the given index.
	// This allows us to check the last timestamp flushed overall.
	LastBlockMetadata(ctx context.Context, index Index) (*blockMetadata, error)

	// GetBlockMetadata returns metadata for all blocks in the given index.
	// Used for debugging block information.
	GetBlockMetadata(ctx context.Context, index Index) (map[string]*blockMetadata, error)

	// GetBlockDeleteCount returns the number of deleted pauses for a specific block.
	// Used for debugging block compaction status.
	GetBlockDeleteCount(ctx context.Context, index Index, blockID ulid.ULID) (int64, error)

	// GetBlockPauseIDs returns all pause IDs from a specific block.
	// Used for debugging block contents. Returns total count and all IDs.
	GetBlockPauseIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error)

	// GetBlockDeletedIDs returns all deleted pause IDs for a specific block.
	// Used for debugging block deletion tracking. Returns total count and all IDs.
	GetBlockDeletedIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error)

	// IndexExists returns whether we've written any blocks for the given index.
	IndexExists(ctx context.Context, i Index) (bool, error)

	// PauseByID returns a pause by a given ID.  Note that an index is required.
	PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error)
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
