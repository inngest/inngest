package pauses

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"gocloud.dev/blob"
)

const (
	// DefaultPausesPerBlock is the number of pauses to store in a single block.
	// A pause equates to roughly ~0.75-1KB of data, so this is a good default
	// of roughly 25mb blocks.
	DefaultPausesPerBlock = 25_000
)

// Block represents a block of pauses.
type Block struct {
	// TODO: Maybe metadata in the header.

	// ID is the block ID.
	ID ulid.ULID
	// Index is the index for this block, eg. the workspac and event name.
	Index Index
	// Pauses is the slice of pauses in this block.
	Pauses []*state.Pause
}

// BlockstoreOpts creates a new BlockStore with dependencies injected.
type BlockstoreOpts struct {
	// Bufferer is the bufferer which allows us to read from indexes.
	Bufferer Bufferer
	// Bucket is the backing blobstore for reading and writing blocks.
	Bucket *blob.Bucket
	// Leaser manages leases for a given index.
	Leaser BlockLeaser
	// BlockSize is the number of pauses to store in a single block.
	BlockSize int
}

func NewBlockstore(opts BlockstoreOpts) (BlockStore, error) {
	if opts.Bucket == nil {
		return nil, fmt.Errorf("bucket is required")
	}
	if opts.Bufferer == nil {
		return nil, fmt.Errorf("bufferer is required")
	}
	if opts.Leaser == nil {
		return nil, fmt.Errorf("leaser is required")
	}
	if opts.BlockSize == 0 {
		opts.BlockSize = DefaultPausesPerBlock
	}

	return &blockstore{
		size:   opts.BlockSize,
		buf:    opts.Bufferer,
		bucket: opts.Bucket,
		leaser: opts.Leaser,
	}, nil
}

type blockstore struct {
	// size is the size of blocks when writing
	size int

	// buf is the backing buffer that we process blocks from when flushing.
	//
	// We call `PausesSince` on this buffer to get all pauses from zeroTime to
	// now.  Note that this is not optimal;  this may load more pauses (in batches,
	// not in entirety) than we need.  In the future we may want to add a method
	// to fetch the first N pauses from the buffer.
	//
	// Right now, the backing implementation
	buf Bufferer

	// bucket is the backing blobstore for reading and writing blocks.
	bucket *blob.Bucket

	// leaser manages leases for a given index.
	leaser BlockLeaser
}

func (b blockstore) BlockSize() int {
	return b.size
}

// FlushIndexBlock processes a given index, fetching pauses from the backing buffer
// and writing to a block.
func (b blockstore) FlushIndexBlock(ctx context.Context, index Index) error {
	if b.buf == nil || b.bucket == nil || b.size == 0 {
		return nil
	}

	return util.Lease(
		ctx,
		// NOTE: Lease, Renew, and Revoke are closures because they need
		// access to the Index field.  This makes util.Lease simple and
		// minimal.
		func(ctx context.Context) (ulid.ULID, error) {
			return b.leaser.Lease(ctx, index)
		},
		func(ctx context.Context, leaseID ulid.ULID) (ulid.ULID, error) {
			return b.leaser.Renew(ctx, index, leaseID)
		},
		func(ctx context.Context, leaseID ulid.ULID) error {
			return b.leaser.Revoke(ctx, index, leaseID)
		},
		func(ctx context.Context) error {
			// Call this function and block, renewing leases in the background
			// until this function is done.
			return b.FlushIndexBlock(ctx, index)
		},
		10*time.Second,
	)
}

func (b blockstore) flushIndexBlock(ctx context.Context, index Index) error {
	iter, err := b.buf.PausesSince(ctx, index, time.Time{})
	if err != nil {
		return fmt.Errorf("failed to load pauses from buffer: %w", err)
	}

	block := &Block{
		Index:  index,
		Pauses: make([]*state.Pause, b.size),
	}

	n := 0
	for iter.Next(ctx) {
		item := iter.Val(ctx)
		if item == nil {
			continue
		}

		block.Pauses[n] = item

		n++
		if n >= b.size {
			// We've hit our block size. Quit iterating
			break
		}
	}

	if iter.Error() != nil {
		return fmt.Errorf("error iterating over buffered pauses: %w", iter.Error())
	}

	if n < b.size {
		// We didn't find enough non-nil pauses to fill the block.  Log a warning
		// and return.  This shouldn't happen, as we shouldn't return nil pauses
		// from iterators often;  this only happens in a race where the iterator
		// has pauses in-memory which are then deleted while iterating, leading to
		// a race condition.
		return nil
	}

	// Trim any pauses that are nil.
	block.Pauses = block.Pauses[:n]

	// TODO: Use the last pause ID as the block ID;  this ensures the block ID
	// encodes the last pause's timestamp, which is useful for ordering.
	//
	// NOTE: Pause IDs are UUIDs, and before April 9 were NOT v7 UUIDs...  they
	//       were random.
	//
	//       We generate deterministic ULIDs from pauses here based off of the
	//       pause timestamp and the UUID.  If the UUIDs are NOT v7, we attempt
	//       to get the "added at" index for the pause (zset score).  If THAT fails,
	//       we use a hard-coded timestamp in the past (as all new pauses are V7).

	// TODO: Marshal the block.  Parquet/Protobuf/etc.

	// Now that we have our block, write it to the backing store.
	key := b.BlockKey(index, block.ID)
	if err := b.bucket.WriteAll(ctx, key, []byte{}, nil); err != nil {
		return fmt.Errorf("failed to write block: %w", err)
	}

	// TODO: Write block metadata

	// TODO: Remove len(block.Pauses) from the buffer, as they've been flushed.
	//       We can't use the standard DeletePause item from the state store as
	//       this will add delete indexes for compaction.

	return fmt.Errorf("not implemented")
}

func (b blockstore) BlocksSince(ctx context.Context, index Index, since time.Time) ([]ulid.ULID, error) {
	// TODO: Read from backing KV (redis/valkey/memorydb/fdb) indexes.
	return nil, fmt.Errorf("not implemented")
}

func (b blockstore) ReadBlock(ctx context.Context, index Index, blockID ulid.ULID) (*Block, error) {
	key := b.BlockKey(index, blockID)
	byt, err := b.bucket.ReadAll(ctx, key)
	if err != nil {
		return nil, err
	}

	// TODO: Unmarshal the block
	_ = byt

	return nil, fmt.Errorf("not implemented")
}

// GenerateKey generates a key for a given block ID.
func (b blockstore) BlockKey(idx Index, blockID ulid.ULID) string {
	return fmt.Sprintf("pauses/%s/%s/blk_%s", idx.WorkspaceID, idx.EventName, blockID)
}
