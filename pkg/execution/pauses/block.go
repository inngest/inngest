package pauses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
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

	// DefaultCompactionLimit is the number of pauses that have to be deleted from
	// a block to compact it.  This prevents us from rewriting pauses on every
	// deletion - a waste of ops.
	DefaultCompactionLimit = (DefaultPausesPerBlock / 5)

	// DefaultCompactionSample gives us a 10% chance of running compactions after
	// a delete.
	DefaultCompactionSample = 0.1
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
	// RC is the Redis client used to manage block indexes.
	RC rueidis.Client
	// Bufferer is the bufferer which allows us to read from indexes.
	Bufferer Bufferer
	// Bucket is the backing blobstore for reading and writing blocks.
	Bucket *blob.Bucket
	// Leaser manages leases for a given index.
	Leaser BlockLeaser
	// BlockSize is the number of pauses to store in a single block.
	BlockSize int
	// CompactionLimit is the total number of pauses that should trigger a compaction.
	// Note that this doesnt always trigger a compaction;
	CompactionLimit int
	// CompactionSample is the chance of compaction, from 0-100
	CompactionSample float64
	// DeleteAfterFlush is a callback that returns whether we delete from the backing buffer,
	// or if deletes are ignored for the current workspace.
	DeleteAfterFlush FeatureCallback
}

func NewBlockstore(opts BlockstoreOpts) (BlockStore, error) {
	if opts.RC == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	if opts.Bucket == nil {
		return nil, fmt.Errorf("bucket is required")
	}
	if opts.Bufferer == nil {
		return nil, fmt.Errorf("bufferer is required")
	}
	if opts.Leaser == nil {
		return nil, fmt.Errorf("leaser is required")
	}
	if opts.DeleteAfterFlush == nil {
		opts.DeleteAfterFlush = func(ctx context.Context, workspaceID uuid.UUID) bool {
			return false
		}
	}
	if opts.BlockSize == 0 {
		opts.BlockSize = DefaultPausesPerBlock
	}
	if opts.CompactionLimit == 0 {
		opts.CompactionLimit = DefaultCompactionLimit
	}
	if opts.CompactionSample == 0 {
		opts.CompactionSample = DefaultCompactionSample
	}

	return &blockstore{
		rc:               opts.RC,
		blocksize:        opts.BlockSize,
		compactionLimit:  opts.CompactionLimit,
		compactionSample: opts.CompactionSample,
		buf:              opts.Bufferer,
		bucket:           opts.Bucket,
		leaser:           opts.Leaser,
		deleteAfterFlush: opts.DeleteAfterFlush,
	}, nil
}

type blockstore struct {
	// size is the size of blocks when writing
	blocksize int

	// compactionLimit is the number of pauses that have to be deleted from
	// a block to compact it.  This prevents us from rewriting pauses on every
	// deletion - a waste of ops.
	compactionLimit int
	// CompactionSample is the chance of compaction, from 0-100
	compactionSample float64

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

	// rc is the Redis client used to manage block indexes.
	rc rueidis.Client

	// deleteAfterFlush, if it returns false, prevents deleting items from the backing buffer when flushed.
	deleteAfterFlush FeatureCallback
}

func (b blockstore) BlockSize() int {
	return b.blocksize
}

// FlushIndexBlock processes a given index, fetching pauses from the backing buffer
// and writing to a block.
func (b blockstore) FlushIndexBlock(ctx context.Context, index Index) error {
	if b.buf == nil || b.bucket == nil || b.blocksize == 0 {
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
		10*time.Second,
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
	if err != nil {
		return fmt.Errorf("could not last block metadata: %w", err)
	} else if lastBlock != nil {
		since = lastBlock.LastTimestamp()
	} else {
		// Hack so that PausesSince never uses a key scan to get the pauses
		// as that would result in unordered pauses that cannot be sorted
		// by score because that data isn't available in the pause item.
		// When <since> isn't zeroed PausesSince uses a ZRANGE BYSCORE which
		// will return pauses correctly sorted by score.

		// Perf wise it should be fine, we only need <blocksize> amount of
		// pauses anyways.
		since = time.Unix(0, 1)
	}

	l := logger.StdlibLogger(ctx).With("workspace_id", index.WorkspaceID, "event_name", index.EventName, "since", since.UnixMilli())
	l.Debug("flushing block index")

	iter, err := b.buf.PausesSince(ctx, index, since)
	if err != nil {
		return fmt.Errorf("failed to load pauses from buffer: %w", err)
	}

	block := &Block{
		Index:  index,
		Pauses: make([]*state.Pause, b.blocksize),
	}

	n := 0
	for iter.Next(ctx) {
		item := iter.Val(ctx)
		if item == nil {
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

	if n < b.blocksize || SkipFlushing(index, block.Pauses) {
		// We didn't find enough non-nil pauses to fill the block.  Log a warning
		// and return.  This shouldn't happen, as we shouldn't return nil pauses
		// from iterators often;  this only happens in a race where the iterator
		// has pauses in-memory which are then deleted while iterating, leading to
		// a race condition.
		// Another case where this could happen is during rollout when we are not
		// deleting pauses from the buffer after flushing to a block which will
		// make it that we trigger flush jobs because the buffer is always at the
		// threshold but we can't find enough pauses since the last block.
		l.Warn("could not find enough pauses to flush into buffer", "len", len(block.Pauses))
		return nil
	}

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
				if err := b.buf.Delete(ctx, index, *p); err != nil {
					logger.StdlibLogger(ctx).Warn("error deleting pause from buffer after flushing block", "error", err)
				} else {
					deleted = deleted + 1
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

	// TODO: remove this
	for i := 0; i < len(block.Pauses); i++ {
		l.Debug("pause", "i", i, "pause", *block.Pauses[i])
	}

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

	ids, err := b.rc.Do(
		ctx,
		b.rc.B().Zrangebyscore().Key(blockIndexKey(index)).Min(score).Max("+inf").Build(),
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
func (b blockstore) Delete(ctx context.Context, index Index, pause state.Pause) error {
	// Check which blocks this pause exists in, then delete from the block index.
	ts, err := b.buf.PauseTimestamp(ctx, index, pause)
	if err != nil {
		return fmt.Errorf("unable to get timestamp for pause when processing block deletion: %w", err)
	}

	blockID, err := b.blockIDForTimestamp(ctx, index, ts)
	if err != nil {
		return fmt.Errorf("error fetching block for timestamp: %w", err)
	}
	if blockID == nil {
		return nil
	}

	err = b.rc.Do(ctx, b.rc.B().Sadd().Key(blockDeleteKey(index)).Member(pause.ID.String()).Build()).Error()
	if err != nil {
		return fmt.Errorf("error tracking pause delete in block index: %w", err)
	}

	// As an optimization, check how many deletes this index has and trigger compaction if over the
	// compaction limit.
	if rand.IntN(100) <= int(b.compactionSample*100) {
		go func() {
			size, err := b.rc.Do(ctx, b.rc.B().Scard().Key(blockDeleteKey(index)).Build()).AsInt64()
			if err != nil {
				logger.StdlibLogger(ctx).Warn("error fetching block delete length", "error", err)
				return
			}
			if size < int64(b.compactionLimit) {
				return
			}

			// Trigger a new compaction.
			logger.StdlibLogger(ctx).Debug("compacting block deletes", "len", size, "index", index)
			b.Compact(ctx, index)
		}()
	}

	return nil
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
	cmd := b.rc.B().
		Zrevrangebyscore().
		Key(blockIndexKey(index)).
		Max("+inf").
		Min("-inf").
		Limit(0, 1).
		Build()

	ids, err := b.rc.Do(ctx, cmd).AsStrSlice()
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

	cmd = b.rc.B().Hget().Key(blockMetadataKey(index)).Field(id).Build()

	md := &blockMetadata{}
	if err := b.rc.Do(ctx, cmd).DecodeJSON(md); err != nil {
		return nil, fmt.Errorf("error loading last block metadata: %w", err)
	}
	return md, nil
}

func (b *blockstore) PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error) {
	// TODO: Check if the index has an empty event.  If so, look up the pause ID -> block ID map.
	return nil, fmt.Errorf("not implemented")
}

// Compact reads all indexed deletes from block for an index, then compacts any blocks over a given threshold
// by removing pauses and rewriting blocks.
func (b *blockstore) Compact(ctx context.Context, idx Index) {
	// Implement the following:
	// TODO: Lease compaction for the index.
	// TODO: Read all block metadata for the index
	// TODO: Read all blockDeleteKey entries for the index
	// TODO: For each deleted entry, record which block the delete is for.
	// TODO: If len(block_deletes) > block_compact_ratio, rewrite the block by:
	// 1. fetching the block
	// 2. filtering deleted pauses from the block
	// 3. rewriting the block
}

// blockIDForTimestamp returns the block ID that contains pauses for the given timestamp.
func (b *blockstore) blockIDForTimestamp(ctx context.Context, idx Index, ts time.Time) (*ulid.ULID, error) {
	score := strconv.Itoa(int(ts.UnixMilli()))
	ids, err := b.rc.Do(
		ctx,
		b.rc.B().Zrange().Key(blockIndexKey(idx)).Min("("+score).Max("+inf").Byscore().Limit(0, 1).Build(),
	).AsStrSlice()
	if len(ids) == 1 {
		id, err := ulid.Parse(ids[0])
		return &id, err
	}
	if err == nil || rueidis.IsRedisNil(err) {
		return nil, nil
	}
	return nil, err
}

// blockMetadata loads metadata for the givne block.  It reads timestamps for the first and last
// pauses (as these aren't embedded in the pause) to create the correct start and end timestamps.
func (b *blockstore) blockMetadata(ctx context.Context, idx Index, block *Block) (*blockMetadata, error) {
	earliest, err := b.buf.PauseTimestamp(ctx, idx, *block.Pauses[0])
	if err != nil {
		return nil, fmt.Errorf("error fetching earliest pause time: %w", err)
	}
	latest, err := b.buf.PauseTimestamp(ctx, idx, *block.Pauses[len(block.Pauses)-1])
	if err != nil {
		return nil, fmt.Errorf("error fetching latest pause time: %w", err)
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

	cmd := b.rc.B().
		Zadd().
		Key(blockIndexKey(idx)).
		ScoreMember().
		ScoreMember(
			float64(ulid.Time(block.ID.Time()).UnixMilli()),
			block.ID.String(),
		).
		Build()
	if err := b.rc.Do(ctx, cmd).Error(); err != nil {
		return err
	}

	return b.rc.Do(
		ctx,
		b.rc.B().
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

// blockDeleteKey tracks all deletes for a given index.
// note that block
func blockDeleteKey(idx Index) string {
	return fmt.Sprintf("{estate}:blk:dels:%s:%s", idx.WorkspaceID, util.XXHash(idx.EventName))
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
