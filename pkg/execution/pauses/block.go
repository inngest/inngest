package pauses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"gocloud.dev/blob"
)

const (
	pkgName = "pauses.execution.inngest"
	// DefaultPausesPerBlock is the number of pauses to store in a single block.
	// A pause equates to roughly ~0.75-1KB of data, so this is a good default.
	DefaultPausesPerBlock = 10_000

	// DefaultCompactionGarbageRatio is the ratio of deletions to block size that triggers compaction.
	DefaultCompactionGarbageRatio = 0.2

	// DefaultCompactionSample gives us a 10% chance of running compactions after
	// a delete.
	DefaultCompactionSample = 0.1

	// DefaultCompactionLeaseRenewInterval is the lease renewal period for compaction.
	DefaultCompactionLeaseRenewInterval = 15 * time.Second

	// DefaultFlushLeaseRenewInterval is the lease renewal period for flushing.
	DefaultFlushLeaseRenewInterval = 10 * time.Second

	// DefaultFetchMargin provides a safety buffer when pre-fetching pause IDs.
	// Used with the block size to ensure enough ordered results are returned,
	// even if some pauses were deleted in the meantime as we can’t rely on
	// an unordered scan for flushing blocks.
	DefaultFetchMargin = DefaultPausesPerBlock / 4

	PauseBlockIndexTombstone         = "-"
	PauseBlockIndexTombstoneDuration = 15 * time.Minute
)

// Block represents a block of pauses.
type Block struct {
	// ID is the block ID.  The timestamp encodes the timestamp of the latest
	// pause in the block at the time of block creation.
	ID ulid.ULID
	// Index is the index for this block, eg. the workspace and event name.
	Index Index
	// Pauses is the slice of pauses in this block, in order of earliest -> latest.
	Pauses []*state.Pause
}

// BlockstoreOpts creates a new BlockStore with dependencies injected.
type BlockstoreOpts struct {
	// PauseClient is the Redis client used to manage block indexes.
	PauseClient *redis_state.PauseClient
	// Bufferer is the bufferer which allows us to read from indexes.
	Bufferer Bufferer
	// Bucket is the backing blobstore for reading and writing blocks.
	Bucket *blob.Bucket
	// Leaser manages leases for a given index.
	Leaser BlockLeaser
	// BlockSize is the number of pauses to store in a single block.
	BlockSize int
	// FetchMargin is the number of additional pauses to pre-fetch ids for when block flushing.
	FetchMargin int
	// FlushLeaseRenewInterval is the interval for flush lease renewals.
	FlushLeaseRenewInterval time.Duration

	// CompactionGarbageRatio is the ratio of deletions to block size that triggers compaction.
	CompactionGarbageRatio float64
	// CompactionSample is the chance of compaction, from 0-1.0
	CompactionSample float64
	// CompactionLeaser manages compaction leases for a given index.
	CompactionLeaser BlockLeaser
	// compactionLeaseRenewInterval is the interval for compaction lease renewals.
	CompactionLeaseRenewInterval time.Duration

	// DeleteAfterFlush is a callback that returns whether we delete from the backing buffer,
	// or if deletes are ignored for the current workspace.
	DeleteAfterFlush FeatureCallback

	// EnableBlockCompaction is a callback that returns whether block compaction is enabled
	// for the current workspace.
	EnableBlockCompaction FeatureCallback
}

func NewBlockstore(opts BlockstoreOpts) (BlockStore, error) {
	if opts.PauseClient == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	if opts.Bufferer == nil {
		return nil, fmt.Errorf("bufferer is required")
	}
	if opts.Leaser == nil {
		return nil, fmt.Errorf("leaser is required")
	}
	if opts.CompactionLeaser == nil {
		return nil, fmt.Errorf("compaction leaser is required")
	}
	if opts.DeleteAfterFlush == nil {
		opts.DeleteAfterFlush = func(ctx context.Context, workspaceID uuid.UUID) bool {
			return false
		}
	}
	if opts.EnableBlockCompaction == nil {
		opts.EnableBlockCompaction = func(ctx context.Context, workspaceID uuid.UUID) bool {
			return false
		}
	}
	if opts.BlockSize == 0 {
		opts.BlockSize = DefaultPausesPerBlock
	}
	if opts.CompactionGarbageRatio == 0 {
		opts.CompactionGarbageRatio = DefaultCompactionGarbageRatio
	}
	if opts.CompactionSample == 0 {
		opts.CompactionSample = DefaultCompactionSample
	}
	if opts.FetchMargin == 0 {
		opts.FetchMargin = DefaultFetchMargin
	}

	if opts.CompactionLeaseRenewInterval.Nanoseconds() == 0 {
		opts.CompactionLeaseRenewInterval = DefaultCompactionLeaseRenewInterval
	}

	if opts.FlushLeaseRenewInterval.Nanoseconds() == 0 {
		opts.FlushLeaseRenewInterval = DefaultFlushLeaseRenewInterval
	}

	return &blockstore{
		pc:                           opts.PauseClient,
		blocksize:                    opts.BlockSize,
		fetchMargin:                  opts.FetchMargin,
		flushLeaseRenewInterval:      opts.FlushLeaseRenewInterval,
		compactionGarbageRatio:       opts.CompactionGarbageRatio,
		compactionSample:             opts.CompactionSample,
		compactionLeaser:             opts.CompactionLeaser,
		compactionLeaseRenewInterval: opts.CompactionLeaseRenewInterval,
		buf:                          opts.Bufferer,
		bucket:                       opts.Bucket,
		leaser:                       opts.Leaser,
		deleteAfterFlush:             opts.DeleteAfterFlush,
		enableBlockCompaction:        opts.EnableBlockCompaction,
	}, nil
}

type blockstore struct {
	// size is the size of blocks when writing
	blocksize int

	// fetchMargin is the number of additional pauses to pre-fetch ids for when block flushing.
	fetchMargin int

	// flushLeaseRenewInterval is the interval for flush lease renewals.
	flushLeaseRenewInterval time.Duration

	// compactionGarbageRatio is the ratio of deletions to block size that triggers compaction.
	compactionGarbageRatio float64
	// CompactionSample is the chance of compaction, from 0-100
	compactionSample float64
	// compactionLeaser manages compaction leases for a given index.
	compactionLeaser BlockLeaser
	// compactionLeaseRenewInterval is the interval for compaction lease renewals.
	compactionLeaseRenewInterval time.Duration

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

	// pc is the Redis client used to manage block indexes.
	pc *redis_state.PauseClient

	// deleteAfterFlush, if it returns false, prevents deleting items from the backing buffer when flushed.
	deleteAfterFlush FeatureCallback

	// enableBlockCompaction, if it returns false, prevents block compaction for the workspace.
	enableBlockCompaction FeatureCallback
}

func (b blockstore) BlockSize() int {
	return b.blocksize
}

// FlushIndexBlock processes a given index, fetching pauses from the backing buffer
// and writing to a block.
func (b blockstore) FlushIndexBlock(ctx context.Context, index Index) error {
	if b.buf == nil || b.bucket == nil || b.blocksize == 0 {
		logger.StdlibLogger(ctx).Warn("skipping block flush", "block_size", b.blocksize, "buf_set", b.buf != nil, "bucket_set", b.bucket != nil)
		return nil
	}

	return util.Lease(
		ctx,
		"flush index block",
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
			return b.flushIndexBlock(ctx, index)
		},
		b.flushLeaseRenewInterval,
	)
}

func (b blockstore) flushIndexBlock(ctx context.Context, index Index) error {
	if SkipFlushing(index, nil) {
		// Don't bother.
		return nil
	}

	start := time.Now()

	// Firstly, we need to find the last block written for the current buffer.
	// This lets us know where to read from, so that we can ignore any previous
	// buffer flushes that may not have had corresponding deletes (as deletes)
	// happen in goroutines best-effort.
	var since time.Time
	lastBlock, err := b.LastBlockMetadata(ctx, index)
	if err == nil && lastBlock != nil {
		since = lastBlock.LastTimestamp()
	}

	l := logger.StdlibLogger(ctx).With("workspace_id", index.WorkspaceID, "event_name", index.EventName, "since", since.UnixMilli())
	l.Debug("flushing block index")

	iter, err := b.buf.PausesSinceWithCreatedAt(ctx, index, since, int64(b.blocksize+b.fetchMargin))
	if err != nil {
		return fmt.Errorf("failed to load pauses from buffer: %w", err)
	}

	block := &Block{
		Index:  index,
		Pauses: make([]*state.Pause, b.blocksize),
	}

	n := 0
	deleted := 0
	for iter.Next(ctx) {
		item := iter.Val(ctx)
		if item == nil {
			deleted = deleted + 1
			continue
		}

		if !item.CreatedAt.After(since) {
			// Since iterator query uses second precision but pause timestamps have millisecond precision,
			// we may retrieve pauses that occurred slightly before the last block boundary.
			// Skipping these prevents breaking the assumption that blocks are contiguous.
			// Pauses before the boundary will remain in buffer indefinitely.
			l.Warn("skipping pause before block boundary",
				"pause_created_at", item.CreatedAt,
				"block_boundary", since)
			continue
		}

		if item.CreatedAt.IsZero() {
			// Old pauses don't have the time embedded in the pause item but the iterator should have
			// injected it from the pause index when prefetching IDs/scores.

			// We cannot allow this because we lose the creation timestamp as soon as the pause
			// is deleted after block flushing.
			l.ReportError(
				errors.New("pause without creation time"),
				"encountered pause without creation time when flushing, this should never happen",
				logger.WithErrorReportTags(map[string]string{
					"pause_id":     item.ID.String(),
					"workspace_id": item.WorkspaceID.String(),
				}),
			)
			continue
		}

		block.Pauses[n] = item

		n++
		if n >= b.blocksize {
			// We've hit our block size. Quit iterating
			break
		}
	}

	// A cancelled context in iterator while the parent is still
	// not done just means that we are done scanning...
	if iter.Error() != nil && iter.Error() != context.Canceled && ctx.Err() == nil {
		return fmt.Errorf("error iterating over buffered pauses: %w", iter.Error())
	}

	// Trim any pauses that are nil.
	block.Pauses = block.Pauses[:n]

	skipFlush := SkipFlushing(index, block.Pauses)
	if n < b.blocksize || skipFlush {
		var cause string

		if skipFlush {
			cause = "skipped"
		} else if iter.Count() < b.blocksize {
			// Pauses in buffer were deleted in the timeframe of enqueuing the flush job
			// and starting it
			cause = "pauses_deleted_normal"

		} else if deleted >= b.fetchMargin {
			// We didn't find enough non-nil pauses to fill the block.  Log a warning
			// and return.  This shouldn't happen, as we shouldn't return nil pauses
			// from iterators often;  this only happens in a race where the iterator
			// has pauses in-memory which are then deleted while iterating, leading to
			// a race condition. We do have fetchMargin that we additionnaly prefetch
			// which should help but not in all cases.
			cause = "pauses_deleted_race"
		} else {
			// XXX: Temporary situation until full rollout:
			// This can also occur during the gradual rollout of the feature when pauses are not
			// deleted from the buffer after being flushed to a block. The buffer stays near the
			// threshold and keeps triggering flush jobs, but we still can’t find enough new pauses
			// since the last block.
			cause = "rollout"
		}

		metrics.IncrPausesBlockFlushExpectedFail(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"cause": cause}})
		l.Warn("could not find enough pauses to flush into buffer", "len", len(block.Pauses), "cause", cause)

		return nil
	}

	// We have enough pauses at this point we can now sort them based on msec precision,
	// the Redis iterator uses ZRANGE which guarantees order but at seconds precision.
	slices.SortFunc(block.Pauses, func(a, b *state.Pause) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	metadata, err := b.blockMetadata(ctx, index, block)
	if err != nil {
		return fmt.Errorf("failed to generate block metadata: %w", err)
	}

	// Generate a deterministic ULID for the block ID based off of the last pause
	// timestamp and ID in our block.
	block.ID = blockID(block, metadata)

	// Marshal the block.  We currently use JSON encoding everywhere, but
	// we can amend Serialize to use protobuf if we desire via a new tag.
	byt, err := Serialize(block, encodingJSON, 0x00)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}

	// Now that we have our block, write it to the backing store.
	key := b.BlockKey(index, block.ID)
	if err := b.bucket.WriteAll(ctx, key, byt, nil); err != nil {
		return fmt.Errorf("failed to write block: %w", err)
	}

	// Write block index to our zset.
	if err := b.addBlockIndex(ctx, index, block, metadata); err != nil {
		return fmt.Errorf("failed to write block index: %w", err)
	}

	// Remove len(block.Pauses) from the buffer, as they've been flushed.
	// NOTE: This can happen in the background as we pick flushing up from the
	// last block written.
	go func() {
		// Otherwise the job will end and we won't be able to finish deleting
		ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		defer cancel()

		if b.deleteAfterFlush(ctx, index.WorkspaceID) {
			start := time.Now()
			var deleted int64

			for _, p := range block.Pauses {
				err := b.buf.Delete(ctx, index, *p, state.WithWriteBlockIndex(block.ID.String(), index.EventName))
				switch {
				case err == nil:
					deleted = deleted + 1
				case errors.Is(err, state.ErrPauseNotInBuffer):
					if err := b.Delete(ctx, index, *p); err != nil {
						logger.StdlibLogger(ctx).Error("error marking pause deleted in block", "error", err, "pause_id", p.ID)
					} else {
						deleted = deleted + 1
					}
				default:
					logger.StdlibLogger(ctx).Error("error deleting pause from buffer after flushing block", "error", err)
				}
				time.Sleep(5 * time.Millisecond)
			}

			l.Debug("deleted pauses after flush", "len", deleted)

			metrics.HistogramPauseDeleteLatencyAfterBlockFlush(ctx, time.Since(start), metrics.HistogramOpt{
				PkgName: pkgName,
				Tags:    map[string]any{},
			})

			metrics.IncrPausesDeletedAfterBlockFlush(ctx, deleted, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{},
			})

		}
		// XXX: We should add an N% chance of loading all pauses from 0 -> wm.Epoch
		// in case any deletions in a previous flush failed.
	}()

	l.Debug("flushed block index", "duration", time.Since(start).Milliseconds(), "len", len(block.Pauses), "block_key", key)

	metrics.HistogramPauseBlockFlushLatency(ctx, time.Since(start), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags:    map[string]any{},
	})

	metrics.IncrPausesFlushedToBlocks(ctx, int64(len(block.Pauses)), metrics.CounterOpt{
		PkgName: pkgName,
		Tags:    map[string]any{},
	})

	metrics.IncrPausesBlocksCreated(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags:    map[string]any{},
	})

	return nil
}

// BlocksSince returns all block IDs that have been written for a given index,
// since a given time.
//
// If the time is ZeroTime, all blocks for the index must be returned.
//
// NOTE: This is NOT INCLUSIVE of since, ie. the range is (since, now].
func (b blockstore) BlocksSince(ctx context.Context, index Index, since time.Time) ([]ulid.ULID, error) {
	// Read from backing KV (redis/valkey/memorydb/fdb) indexes.
	ms := since.UnixMilli()
	score := "(" + strconv.Itoa(int(ms))
	if since.IsZero() {
		score = "-inf"
	}

	ids, err := b.pc.Client().Do(
		ctx,
		b.pc.Client().B().Zrangebyscore().Key(blockIndexKey(index)).Min(score).Max("+inf").Build(),
	).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error querying block index: since %s: %w", score, err)
	}

	ulids := make([]ulid.ULID, len(ids))
	for i, id := range ids {
		ulids[i], err = ulid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("error parsing block ULID '%s': %w", id, err)
		}
	}
	return ulids, nil
}

func (b blockstore) ReadBlock(ctx context.Context, index Index, blockID ulid.ULID) (*Block, error) {
	if b.bucket == nil {
		return nil, fmt.Errorf("error bucket is not setup")
	}
	key := b.BlockKey(index, blockID)
	logger.StdlibLogger(ctx).Debug("reading block", "block_key", key)
	byt, err := b.bucket.ReadAll(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("error reading block from index '%s': id '%s': %w", index, blockID, err)
	}

	// Unmarshal the block, using the first byte to figure out encoding.
	return Deserialize(byt)
}

// Delete deletes a pause from the backing blob.  Note that blobs are immutable;  we cannot
// yeet the pause out of a blob as-is.  Instead, we track which blocks have deleted pauses
// via indexes, and eventually compact blocks.
func (b blockstore) Delete(ctx context.Context, index Index, pause state.Pause, opts ...state.DeletePauseOpt) error {
	var blockIDs []ulid.ULID
	var err error

	if pause.CreatedAt.IsZero() {
		// Legacy pauses without timestamps: add to all blocks in the index
		blockIDs, err = b.BlocksSince(ctx, index, time.Time{})
		if err != nil {
			return fmt.Errorf("error fetching all blocks for legacy pause: %w", err)
		}
		metrics.IncrPausesLegacyDeletionCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
	} else {
		// Normal pauses with timestamps: find specific blocks that may contain the pause
		blockIDs, err = b.blockIDsForTimestamp(ctx, index, pause.CreatedAt)
		if err != nil {
			return fmt.Errorf("error fetching blocks for timestamp: %w", err)
		}
		logger.StdlibLogger(ctx).Debug("deleting pause", "pause", pause.ID, "createdAt", pause.CreatedAt.UnixMilli(), "blks", blockIDs)
	}

	if len(blockIDs) == 0 {
		return nil
	}

	blockIndexKey := b.pc.KeyGenerator().PauseBlockIndex(ctx, pause.ID)
	err = b.pc.Client().Do(ctx, b.pc.Client().B().Set().Key(blockIndexKey).Value(PauseBlockIndexTombstone).Ex(PauseBlockIndexTombstoneDuration).Build()).Error()
	if err != nil {
		return fmt.Errorf("error deleting block index while deleting pause: %w", err)
	}

	// Track deletion in each relevant block.
	// This is typically 1 operation, except for:
	// - Pauses on block boundaries (2 operations)
	// - Legacy pauses without timestamps (all blocks)
	for _, blockID := range blockIDs {
		err = b.pc.Client().Do(ctx, b.pc.Client().B().Sadd().Key(blockDeleteKey(index, blockID)).Member(pause.ID.String()).Build()).Error()
		if err != nil {
			return fmt.Errorf("error tracking pause delete in block %s: %w", blockID, err)
		}
	}

	// As an optimization, check delete counts across all blocks and trigger compaction
	// if any block exceeds the compaction limit.
	// Legacy pauses get added to all blocks, so reduce compaction frequency to limit Redis overhead.
	compactionSample := b.compactionSample
	if pause.CreatedAt.IsZero() {
		compactionSample = b.compactionSample * 0.1 // 10x lower chance for legacy pauses
	}

	if rand.IntN(100) <= int(compactionSample*100) {
		go func() {
			var maxDeletes int64
			for _, blockID := range blockIDs {
				size, err := b.pc.Client().Do(ctx, b.pc.Client().B().Scard().Key(blockDeleteKey(index, blockID)).Build()).AsInt64()
				if err != nil {
					logger.StdlibLogger(ctx).Warn("error fetching block delete length", "error", err, "block_id", blockID)
					continue
				}
				maxDeletes = max(maxDeletes, size)
			}

			// Trigger a new compaction.
			if maxDeletes >= int64(float64(b.blocksize)*b.compactionGarbageRatio) {
				logger.StdlibLogger(ctx).Debug("compacting block deletes", "max_deletes", maxDeletes, "index", index)
				b.Compact(ctx, index)
			}
		}()
	}

	return nil
}

// DeleteByID deletes a pause from a block by marking it as deleted in the block's delete tracking set.
// This method must be called before the pause is deleted from the buffer, otherwise the block index
// lookup will fail and we won't know which block contains the pause.
// Note: This method does not trigger compaction as it can be called by any service that only has 
// access to the buffer and not necessarily the block store.
func (b blockstore) DeleteByID(ctx context.Context, pauseID uuid.UUID, workspaceID uuid.UUID) error {
	blockIndexKey := b.pc.KeyGenerator().PauseBlockIndex(ctx, pauseID)

	blockIDStr, err := b.pc.Client().Do(ctx, b.pc.Client().B().Getdel().Key(blockIndexKey).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil
		}
		return fmt.Errorf("error reading block index for pause %s: %w", pauseID, err)
	}

	if blockIDStr == PauseBlockIndexTombstone {
		return nil
	}

	var blockIndex state.BlockIndex
	if err := json.Unmarshal([]byte(blockIDStr), &blockIndex); err != nil {
		return fmt.Errorf("error parsing block index JSON '%s': %w", blockIDStr, err)
	}

	blockID, err := ulid.Parse(blockIndex.BlockID)
	if err != nil {
		return fmt.Errorf("error parsing block ID '%s': %w", blockIndex.BlockID, err)
	}

	index := Index{WorkspaceID: workspaceID, EventName: blockIndex.EventName}

	return b.pc.Client().Do(ctx, b.pc.Client().B().Sadd().Key(blockDeleteKey(index, blockID)).Member(pauseID.String()).Build()).Error()
}

func (b *blockstore) IndexExists(ctx context.Context, i Index) (bool, error) {
	md, err := b.LastBlockMetadata(ctx, i)
	if err != nil {
		return false, err
	}
	// the index exists if we have metadata.
	return md != nil, nil
}

func (b *blockstore) LastBlockMetadata(ctx context.Context, index Index) (*blockMetadata, error) {
	cmd := b.pc.Client().B().
		Zrevrangebyscore().
		Key(blockIndexKey(index)).
		Max("+inf").
		Min("-inf").
		Limit(0, 1).
		Build()

	ids, err := b.pc.Client().Do(ctx, cmd).AsStrSlice()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// Doesn't exist.
			return nil, nil
		}
		return nil, fmt.Errorf("error looking up last block metadata write: %w", err)
	}

	if len(ids) == 0 {
		// No blocks exist for this index.
		return nil, nil
	}

	id := ids[0]

	cmd = b.pc.Client().B().Hget().Key(blockMetadataKey(index)).Field(id).Build()

	md := &blockMetadata{}
	if err := b.pc.Client().Do(ctx, cmd).DecodeJSON(md); err != nil {
		return nil, fmt.Errorf("error loading last block metadata: %w", err)
	}
	return md, nil
}

func (b *blockstore) PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error) {
	// First, look up the pause ID -> block ID mapping from the pause block index
	indexKey := b.pc.KeyGenerator().PauseBlockIndex(ctx, pauseID)
	blockIDStr, err := b.pc.Client().Do(ctx, b.pc.Client().B().Get().Key(indexKey).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, state.ErrPauseNotFound
		}
		return nil, fmt.Errorf("error looking up pause block index: %w", err)
	}

	if blockIDStr == PauseBlockIndexTombstone {
		return nil, state.ErrPauseNotFound
	}

	// Parse the block index JSON to extract the block ID
	var blockIndex state.BlockIndex
	if err := json.Unmarshal([]byte(blockIDStr), &blockIndex); err != nil {
		return nil, fmt.Errorf("error parsing block index JSON '%s': %w", blockIDStr, err)
	}

	blockID, err := ulid.Parse(blockIndex.BlockID)
	if err != nil {
		return nil, fmt.Errorf("error parsing block ID '%s': %w", blockIndex.BlockID, err)
	}

	// Pause timeouts don't have the event name in the queue item and will call this PauseByID
	// without it.
	if index.EventName == "" {
		index = Index{WorkspaceID: index.WorkspaceID, EventName: blockIndex.EventName}
	}

	// Read the block from storage
	block, err := b.ReadBlock(ctx, index, blockID)
	if err != nil {
		return nil, fmt.Errorf("error reading block '%s': %w", blockID, err)
	}

	// Search for the pause in the block
	for _, pause := range block.Pauses {
		if pause.ID == pauseID {
			// Check if this pause has been marked for deletion
			deleteKey := blockDeleteKey(index, blockID)
			isDeleted, err := b.pc.Client().Do(ctx, b.pc.Client().B().Sismember().Key(deleteKey).Member(pauseID.String()).Build()).AsBool()
			if err != nil {
				return nil, fmt.Errorf("error checking if pause is deleted: %w", err)
			}

			if isDeleted {
				return nil, state.ErrPauseNotFound
			}

			return pause, nil
		}
	}

	return nil, state.ErrPauseNotFound
}

// Compact reads all indexed deletes from block for an index, then compacts any blocks over a given threshold
// by removing pauses and rewriting blocks.
func (b *blockstore) Compact(ctx context.Context, index Index) {
	_ = util.Lease(
		ctx,
		"compact index blocks",
		// NOTE: Lease, Renew, and Revoke are closures because they need
		// access to the Index field.  This makes util.Lease simple and
		// minimal.
		func(ctx context.Context) (ulid.ULID, error) {
			return b.compactionLeaser.Lease(ctx, index)
		},
		func(ctx context.Context, leaseID ulid.ULID) (ulid.ULID, error) {
			return b.compactionLeaser.Renew(ctx, index, leaseID)
		},
		func(ctx context.Context, leaseID ulid.ULID) error {
			return b.compactionLeaser.Revoke(ctx, index, leaseID)
		},
		func(ctx context.Context) error {
			// Call this function and block, renewing leases in the background
			// until this function is done.
			return b.compact(ctx, index)
		},
		b.compactionLeaseRenewInterval,
	)
}

func (b *blockstore) compact(ctx context.Context, index Index) error {
	start := time.Now()
	dryRun := !b.enableBlockCompaction(ctx, index.WorkspaceID)

	l := logger.StdlibLogger(ctx).With("workspace_id", index.WorkspaceID, "event_name", index.EventName, "dry_run", dryRun)
	l.Debug("compacting block index")

	blockMetadataList, err := b.readAllBlockMetadata(ctx, index)
	if err != nil {
		return err
	}

	// Identify blocks that need compaction
	var blocksToCompact []ulid.ULID
	for blockIDStr := range blockMetadataList {
		blockID, err := ulid.Parse(blockIDStr)
		if err != nil {
			l.ReportError(
				err,
				"invalid block ID found in metadata, this should never happen",
				logger.WithErrorReportTags(map[string]string{
					"block_id":     blockIDStr,
					"workspace_id": index.WorkspaceID.String(),
					"event_name":   index.EventName,
				}),
			)
			continue
		}

		deleteKey := blockDeleteKey(index, blockID)
		deleteCount, err := b.pc.Client().Do(ctx, b.pc.Client().B().Scard().Key(deleteKey).Build()).AsInt64()
		if err != nil {
			l.Error("error getting delete count for block", "block_id", blockIDStr, "error", err)
			continue
		}

		blockMD := blockMetadataList[blockIDStr]
		if blockMD == nil {
			l.Error("no metadata found for block", "block_id", blockIDStr)
			continue
		}

		ratioThreshold := int64(float64(blockMD.Len) * b.compactionGarbageRatio)
		if deleteCount >= ratioThreshold {
			blocksToCompact = append(blocksToCompact, blockID)
		}
	}

	if len(blocksToCompact) > 0 {
		l.Debug("blocks planned for compaction", "count", len(blocksToCompact), "blocks", blocksToCompact)
	} else {
		l.Debug("no blocks need compaction")
		return nil
	}

	if !dryRun {
		// Compact the identified blocks
		for _, blockID := range blocksToCompact {
			b.compactBlock(ctx, l, index, blockID)
		}
	}

	l.Debug("blocks compaction finished", "count", len(blocksToCompact), "duration", time.Since(start).Milliseconds())

	return nil
}

// compactBlock compacts a single block
func (b *blockstore) compactBlock(ctx context.Context, l logger.Logger, index Index, blockID ulid.ULID) {
	startTime := time.Now()
	status := "fail"

	defer func() {
		duration := time.Since(startTime)
		metrics.HistogramPauseBlockCompactionDuration(ctx, duration, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"status": status,
			},
		})
	}()

	blockIDTags := logger.WithErrorReportTags(map[string]string{"block_id": blockID.String()})

	// Fetch the block from storage
	block, err := b.ReadBlock(ctx, index, blockID)
	if err != nil {
		l.ReportError(err, "error reading block for compaction", blockIDTags)
		return
	}

	// Filter out deleted pauses from the block
	deleteKey := blockDeleteKey(index, blockID)
	deletedPauseIDs, err := b.pc.Client().Do(ctx, b.pc.Client().B().Smembers().Key(deleteKey).Build()).AsStrSlice()
	if err != nil {
		l.ReportError(err, "error getting deleted pause IDs for block", blockIDTags)
		return
	}

	deletedSet := make(map[string]bool)
	for _, pauseID := range deletedPauseIDs {
		deletedSet[pauseID] = true
	}

	// Filter out deleted pauses
	var remainingPauses []*state.Pause
	for _, pause := range block.Pauses {
		if !deletedSet[pause.ID.String()] {
			remainingPauses = append(remainingPauses, pause)
		}
	}

	originalCount := len(block.Pauses)
	l.Debug("filtered pauses for compaction", "block_id", blockID, "original_count", len(block.Pauses), "remaining_count", len(remainingPauses), "deleted_count", len(deletedPauseIDs))

	// Handle the case where all pauses were deleted
	if len(remainingPauses) == 0 {
		l.Debug("all pauses deleted, removing block completely", "block_id", blockID)
		if err := b.deleteBlock(ctx, index, blockID); err != nil {
			l.ReportError(err, "error deleting empty block", blockIDTags)
			return
		}
		status = "success"
		return
	}

	block.Pauses = remainingPauses

	byt, err := Serialize(block, encodingJSON, 0x00)
	if err != nil {
		l.ReportError(err, "error serializing compacted block", blockIDTags)
		return
	}

	// Generate metadata before overwriting the block, it could break the boundary checks,
	// same timestamp for boundaries.
	md, err := b.blockMetadata(ctx, index, block)
	if err != nil {
		status = "boundary_fail"
		l.ReportError(err, "error generating block metadata after compaction", blockIDTags)
		return
	}

	key := b.BlockKey(index, blockID)
	if err := b.bucket.WriteAll(ctx, key, byt, nil); err != nil {
		l.ReportError(err, "error writing compacted block to storage", blockIDTags)
		return
	}

	// XXX: At this point a block can be read with unmatching metadata but that should be fine

	// addBlockIndex uses block.ID from the struct, which preserves the original stable block ID
	if err := b.addBlockIndex(ctx, index, block, md); err != nil {
		l.ReportError(err, "error updating block metadata after compaction", blockIDTags)
		return
	}

	// Cleanup deleted pauses from the block deletion key, it can't be done in one OP because
	// deletions could have happened while compaction is running.
	batchSize := 100
	for i := 0; i < len(deletedPauseIDs); i += batchSize {
		end := min(i+batchSize, len(deletedPauseIDs))
		batch := deletedPauseIDs[i:end]

		cmd := b.pc.Client().B().Srem().Key(deleteKey).Member(batch...).Build()
		err = b.pc.Client().Do(ctx, cmd).Error()
		if err != nil {
			l.ReportError(err, "error cleaning up delete tracking batch for compacted block", blockIDTags)
			return
		}
	}

	l.Debug("compaction successful", "block_id", "duration",
		time.Since(startTime).Milliseconds(), blockID,
		"original_count", originalCount, "cleaned_pause_ids",
		len(deletedPauseIDs), "remaining_count", len(remainingPauses))

	status = "success"
}

// readAllBlockMetadata reads all block metadata for a given index from Redis
func (b *blockstore) GetBlockMetadata(ctx context.Context, index Index) (map[string]*blockMetadata, error) {
	return b.readAllBlockMetadata(ctx, index)
}

func (b *blockstore) GetBlockDeleteCount(ctx context.Context, index Index, blockID ulid.ULID) (int64, error) {
	deleteKey := blockDeleteKey(index, blockID)
	return b.pc.Client().Do(ctx, b.pc.Client().B().Scard().Key(deleteKey).Build()).AsInt64()
}

func (b *blockstore) GetBlockPauseIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error) {
	// Read the full block from blob storage
	block, err := b.ReadBlock(ctx, index, blockID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read block: %w", err)
	}

	totalCount := int64(len(block.Pauses))
	pauseIDs := make([]string, len(block.Pauses))

	for i, pause := range block.Pauses {
		pauseIDs[i] = pause.ID.String()
	}

	return pauseIDs, totalCount, nil
}

func (b *blockstore) GetBlockDeletedIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error) {
	deleteKey := blockDeleteKey(index, blockID)

	// Get total count first
	totalCount, err := b.pc.Client().Do(ctx, b.pc.Client().B().Scard().Key(deleteKey).Build()).AsInt64()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get delete count: %w", err)
	}

	// Get all deleted IDs
	deletedIDs, err := b.pc.Client().Do(ctx, b.pc.Client().B().Smembers().Key(deleteKey).Build()).AsStrSlice()
	if err != nil {
		return nil, totalCount, fmt.Errorf("failed to get deleted IDs: %w", err)
	}

	return deletedIDs, totalCount, nil
}

func (b *blockstore) readAllBlockMetadata(ctx context.Context, index Index) (map[string]*blockMetadata, error) {
	metadataKey := blockMetadataKey(index)
	metadataMap, err := b.pc.Client().Do(ctx, b.pc.Client().B().Hgetall().Key(metadataKey).Build()).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("error reading block metadata for index %s: %w", index, err)
	}

	blockMetadataList := make(map[string]*blockMetadata)
	for blockIDStr, metadataJSON := range metadataMap {
		md := &blockMetadata{}
		if err := json.Unmarshal([]byte(metadataJSON), md); err != nil {
			return nil, fmt.Errorf("error unmarshaling metadata for block %s: %w", blockIDStr, err)
		}
		blockMetadataList[blockIDStr] = md
	}

	return blockMetadataList, nil
}

// blockIDsForTimestamp returns the block IDs that may contain pauses for the given timestamp.
// Handles boundary cases where a pause with the same timestamp as a block boundary
// might exist in both the ending and starting blocks.
func (b *blockstore) blockIDsForTimestamp(ctx context.Context, idx Index, ts time.Time) ([]ulid.ULID, error) {
	score := strconv.Itoa(int(ts.UnixMilli()))

	// Get first 2 blocks that could contain this timestamp
	ids, err := b.pc.Client().Do(
		ctx,
		b.pc.Client().B().Zrange().Key(blockIndexKey(idx)).Min(score).Max("+inf").Byscore().Limit(0, 2).Build(),
	).AsStrSlice()
	if err != nil && !rueidis.IsRedisNil(err) {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	// Always include the first block
	result := make([]ulid.ULID, 0, 2)
	firstID, err := ulid.Parse(ids[0])
	if err != nil {
		return nil, fmt.Errorf("error parsing first block ULID '%s': %w", ids[0], err)
	}
	result = append(result, firstID)

	// Check if we need the second block (boundary case)
	if len(ids) == 2 {
		secondID, err := ulid.Parse(ids[1])
		if err != nil {
			return nil, fmt.Errorf("error parsing second block ULID '%s': %w", ids[1], err)
		}

		// If pause timestamp equals the first block's last timestamp,
		// the pause might exist in both blocks due to inclusive boundary
		firstBlockLastTimestamp := ulid.Time(firstID.Time()).UnixMilli()
		if ts.UnixMilli() == firstBlockLastTimestamp {
			result = append(result, secondID)
		}
	}

	return result, nil
}

// blockMetadata creates metadata for the given block. It expects all pauses
// to include valid creation timestamps and uses them to determine the block’s
// start and end times.
func (b *blockstore) blockMetadata(ctx context.Context, idx Index, block *Block) (*blockMetadata, error) {
	earliest := block.Pauses[0].CreatedAt
	if earliest.IsZero() {
		return nil, errors.New("block earliest boundary is not set")
	}

	latest := block.Pauses[len(block.Pauses)-1].CreatedAt
	if latest.IsZero() {
		return nil, errors.New("block latest boundary is not set")
	}

	if earliest.Equal(latest) {
		// This should never normally occur. Since we use Unix seconds for pause index scores,
		// there's an upper limit on how many pauses can be added within a single second.
		// Exceeding that limit (blockSize) could trigger this condition.
		// If this happens in practice, consider increasing the block size to accommodate more pauses per second.
		return nil, errors.New("block boundaries should never be the same, consider increasing the block size")
	}

	// Block indexes are a zset of blocks stored by last pause timestamp,
	// which is embedded into the pause ID.
	//
	// We also have a mapping of block ID -> metadata, storing the timeranges and
	// current block size.  This is used during compaction.
	return &blockMetadata{
		Timeranges: [2]int64{earliest.UnixMilli(), latest.UnixMilli()}, // earliest/latest
		UUIDranges: [2]uuid.UUID{block.Pauses[0].ID, block.Pauses[len(block.Pauses)-1].ID},
		Len:        len(block.Pauses),
	}, nil
}

// addBlockIndex writes the block metadata to the given index, recording the block as flushed.
func (b *blockstore) addBlockIndex(ctx context.Context, idx Index, block *Block, md *blockMetadata) error {
	// Block indexes are a zset of blocks stored by last pause timestamp,
	// which is embedded into the pause ID.
	//
	// We also have a mapping of block ID -> metadata, storing the timeranges and
	// current block size.  This is used during compaction.
	metadata, err := json.Marshal(md)
	if err != nil {
		return err
	}

	cmd := b.pc.Client().B().
		Zadd().
		Key(blockIndexKey(idx)).
		ScoreMember().
		ScoreMember(
			// We use metadata timestamp because the time in the blockID can diverge
			// during compaction, we want a stable blockID even though the boundaries
			// changed.
			float64(md.LastTimestamp().UnixMilli()),
			block.ID.String(),
		).
		Build()
	if err := b.pc.Client().Do(ctx, cmd).Error(); err != nil {
		return err
	}

	return b.pc.Client().Do(
		ctx,
		b.pc.Client().B().
			Hset().
			Key(blockMetadataKey(idx)).
			FieldValue().
			FieldValue(block.ID.String(), string(metadata)).
			Build(),
	).Error()
}

// GenerateKey generates a key for a given block ID.  This is used as the blobstore
// path for writing blocks.
func (b blockstore) BlockKey(idx Index, blockID ulid.ULID) string {
	return fmt.Sprintf("pauses/%s/%s/blk_%s", idx.WorkspaceID, idx.EventName, blockID)
}

// blockIndexKey is internal and stores a list of all blocks for a given index.
//
// This is a zset containing block IDs -> the last pause timestamp.
func blockIndexKey(idx Index) string {
	return fmt.Sprintf("{estate}:blk:idx:%s:%s", idx.WorkspaceID, util.XXHash(idx.EventName))
}

// blockMetadataKey is internal and stores metadata for a given block.
//
// This is an HMAP of block IDs -> metadata.
func blockMetadataKey(idx Index) string {
	return fmt.Sprintf("{estate}:blk:md:%s:%s", idx.WorkspaceID, util.XXHash(idx.EventName))
}

// blockDeleteKey tracks all deletes for a specific block within an index.
func blockDeleteKey(idx Index, blockID ulid.ULID) string {
	return fmt.Sprintf("{estate}:blk:dels:%s:%s:%s", idx.WorkspaceID, util.XXHash(idx.EventName), blockID.String())
}

type blockMetadata struct {
	// Timeranges are the unix millisecond time ranges that this block covers,
	// in ascending order.  This includes the earliest and latest pauses stored
	// in the block AT THE TIME OF BLOCK CREATION.
	Timeranges [2]int64 `json:"tr"`

	// UUIDranges represents the first and last UUID for the pauses in this block
	// AT THE TIME OF BLOCK CREATION.
	//
	// Note that this is only useful for V7 UUIDs, and many pauses may be V4 UUIDs,
	// which means this only stores the first and last pause ID.
	UUIDranges [2]uuid.UUID `json:"ur"`

	// Len is the current number of pauses in the block.  This decreases on compaction -
	// only when a block is compacted and deletes are actually written to a block.
	Len int `json:"len"`
}

func (b blockMetadata) FirstTimestamp() time.Time {
	return time.UnixMilli(b.Timeranges[0])
}

func (b blockMetadata) LastTimestamp() time.Time {
	return time.UnixMilli(b.Timeranges[1])
}

// blockID generates a deterministic ULID based off of this timestamp and
// the last pause ID
func blockID(b *Block, m *blockMetadata) ulid.ULID {
	sum := util.XXHash(b.Pauses[len(b.Pauses)-1].ID.String())
	entropy := ulid.Monotonic(strings.NewReader(sum), 0)
	return ulid.MustNew(uint64(m.Timeranges[1]), entropy)
}

// deleteBlock completely removes a block when all its pauses have been deleted.
// This includes removing the block from Redis indexes, metadata, delete tracking, and blob storage.
func (b *blockstore) deleteBlock(ctx context.Context, index Index, blockID ulid.ULID) error {
	indexKey := blockIndexKey(index)
	err := b.pc.Client().Do(ctx, b.pc.Client().B().Zrem().Key(indexKey).Member(blockID.String()).Build()).Error()
	if err != nil {
		return fmt.Errorf("error removing block from index: %w", err)
	}

	metadataKey := blockMetadataKey(index)
	err = b.pc.Client().Do(ctx, b.pc.Client().B().Hdel().Key(metadataKey).Field(blockID.String()).Build()).Error()
	if err != nil {
		return fmt.Errorf("error removing block metadata: %w", err)
	}

	deleteKey := blockDeleteKey(index, blockID)
	err = b.pc.Client().Do(ctx, b.pc.Client().B().Del().Key(deleteKey).Build()).Error()
	if err != nil {
		return fmt.Errorf("error removing block delete tracking: %w", err)
	}

	// Remove from blob storage
	blobKey := b.BlockKey(index, blockID)
	err = b.bucket.Delete(ctx, blobKey)
	if err != nil {
		return fmt.Errorf("error removing block from blob storage: %w", err)
	}

	return nil
}
