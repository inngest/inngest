package pauses

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

const (
	// DefaultConcurrentBlockFetches represents the number of block fetches we run
	// concurrently.  Note that this will consume _at least_ N * DefaultPausesPerBlock
	// KB of memory.
	//
	// With a default of 20, this is approx 500MB of memory.
	DefaultConcurrentBlockFetches = 20
)

func newDualIter(idx Index, bufferedIter state.PauseIterator, rdr BlockReader, blockIDs []ulid.ULID) *dualIter {
	// NOTE: This is just an estimate, as pauses may have been compacted / had deletions.
	count := bufferedIter.Count() + (len(blockIDs) * DefaultPausesPerBlock)

	return &dualIter{
		idx:             idx,
		count:           count,
		bufferIter:      bufferedIter,
		blockReader:     rdr,
		unfetchedBlocks: blockIDs,
		inflightBlocks:  map[ulid.ULID]struct{}{},
		l:               &sync.Mutex{},
		start:           time.Now(),
	}
}

// dualIter represents an iterator that reads from blocks as well as buffers,
// downloading blocks in parallel to maximize throughput.
type dualIter struct {
	idx Index

	// index is the current iterator index.
	index int64

	// count is an esitmate of the max pauses in the iterator.
	count int

	// usingBuffer indicates whether we're using the buffer for the next
	// .Val() call.  This is set to true to begin with, while the buffer iter
	// is being used.
	usingBuffer bool

	// bufferIter is the buffered iterator.
	bufferIter state.PauseIterator

	// blockReader is the block reader to fetch metadata and blocks.
	blockReader BlockReader

	// unfetchedBlocks represents the set of blocks that haven't been fetched
	// from the backing store yet.
	unfetchedBlocks []ulid.ULID

	// inflightBlocks represents blocks that are currently being fetched from
	// the backing store.
	inflightBlocks map[ulid.ULID]struct{}

	// pauses represents the current pauses that have been fetched from downloaded
	// blocks.
	pauses []*state.Pause

	err error

	// l represents a lock held when mutating block slices or pauses.
	l *sync.Mutex

	// start represents the creation time of this iterator, it is used
	// to measure how long it took for iterate through all pauses. (buffered + in blocks)
	start time.Time
}

// Count returns the count of the pause iteration at the time of querying.
//
// Due to concurrent processing, the total number of iterated fields may not match
// this count;  the count is a snapshot in time.
func (d *dualIter) Count() int {
	return d.count
}

// Next advances the iterator, returning an erorr or context.Canceled if the iteration
// is complete.
//
// Next should be called prior to any call to the iterator's Val method, after
// the iterator has been created.
//
// The order of the iterator is unspecified.
func (d *dualIter) Next(ctx context.Context) bool {
	// Always attempt to fetch blocks if there's space.  This runs in the
	// background.  We do this before calling bufferIter to ensure that we
	// have older pauses loaded when the buffer iteration has finished.
	//
	// Iteration does NOT need to happen in-order.
	d.fetchNextBlocks()

	// Check on the buffer iterator and advance on this.
	if d.bufferIter.Next(ctx) {
		d.usingBuffer = true
		return true
	}

	d.usingBuffer = false

	// NOTE: We must release the lock as soon as possible such that the fetchBlock
	// background thread can grab the lock to adjust pauses.
	d.l.Lock()
	quit := len(d.pauses) == 0 && len(d.inflightBlocks) == 0
	d.l.Unlock()

	if quit {
		// We are done!  There are no pauses downloaded or inflight.
		d.err = context.Canceled

		metrics.HistogramAggregatePausesLoadDuration(ctx, time.Since(d.start).Milliseconds(), metrics.HistogramOpt{
			PkgName: pkgName,
			// TODO: tag workspace ID eventually??
			Tags: map[string]any{
				"iterator": "dual",
			},
		})
		return false
	}

	// Complex case:  we may be waiting for in flight blocks to download, but only
	// if we've handled all already downloaded pauses:
	//
	// eg:  0 blocks downloaded / 10 being downloaded.  spin until len(d.pauses) > 0.
	d.l.Lock()
	spin := len(d.inflightBlocks) > 0 && len(d.pauses) == 0
	d.l.Unlock()

	for spin {
		// Wait 100ms for the block to download and try again.
		time.Sleep(100 * time.Millisecond)

		d.l.Lock()
		if d.err != nil {
			d.l.Unlock()
			// Skip as we've errored.
			return false
		}

		spin = len(d.inflightBlocks) > 0 && len(d.pauses) == 0
		d.l.Unlock()
	}

	d.l.Lock()
	lenPauses := len(d.pauses)
	d.l.Unlock()

	if lenPauses > 0 {
		// Simple case:  pauses are downloaded, so yeah we have a next value.
		return true
	}

	// If we're here and we've already processed all in-flight blocks, attempt
	// to redownload blocks.
	return d.Next(ctx)
}

// Error returns the error returned during iteration, if any.  Use this to check
// for errors during iteration when Next() returns false.
func (d *dualIter) Error() error {
	return d.err
}

// Val returns the current Pause from the iterator.
func (d *dualIter) Val(ctx context.Context) *state.Pause {
	atomic.AddInt64(&d.index, 1)

	if d.usingBuffer {
		return d.bufferIter.Val(ctx)
	}

	d.l.Lock()
	defer d.l.Unlock()

	if len(d.pauses) == 0 {
		return nil
	}

	pause := d.pauses[0]
	d.pauses = d.pauses[1:]

	return pause
}

// Index shows how far the iterator has progressed
func (d *dualIter) Index() int64 {
	return d.index
}

// fetchNextBlocks fetches blocks, returning whether we're fetching anything
// in the background.
func (d *dualIter) fetchNextBlocks() bool {
	d.l.Lock()
	defer d.l.Unlock()

	maxFetch := DefaultConcurrentBlockFetches - len(d.inflightBlocks)
	if maxFetch > len(d.unfetchedBlocks) {
		maxFetch = len(d.unfetchedBlocks)
	}

	if maxFetch == 0 {
		return false
	}

	blockIDs := d.unfetchedBlocks[0:maxFetch]

	// Move the blocks to in-flight.
	for _, blockID := range blockIDs {
		d.inflightBlocks[blockID] = struct{}{}
	}

	// And remove from unfetched
	d.unfetchedBlocks = d.unfetchedBlocks[maxFetch:]

	for _, bid := range blockIDs {
		// Fetch each block, without capturing the ID.
		id := bid
		go d.fetchBlock(context.Background(), id)
	}

	return true
}

func (d *dualIter) fetchBlock(ctx context.Context, id ulid.ULID) {
	if d.err != nil {
		// Don't bother to fetch, as we already have an error
		return
	}
	start := time.Now()

	block, err := d.blockReader.ReadBlock(ctx, d.idx, id)
	// TODO: Maybe we should retry if it's a retriable error
	if err != nil && d.err == nil {
		d.l.Lock()
		d.err = err
		d.l.Unlock()

		metrics.HistogramPauseBlockFetchLatency(ctx, time.Since(start), metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"success": false},
		})
		return
	}

	// TODO:  Here we can optionally filter out deleted pauses from
	// the delete buffer.

	d.l.Lock()
	defer d.l.Unlock()
	// Remove this from in-flight stuff.
	delete(d.inflightBlocks, id)
	// And, of course, add our pauses so that we can iterate through them.
	d.pauses = append(d.pauses, block.Pauses...)

	metrics.HistogramPauseBlockFetchLatency(ctx, time.Since(start), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags:    map[string]any{"success": true},
	})
}
