package pauses

import (
	"context"
	"fmt"
	"sync"

	"github.com/inngest/inngest/pkg/execution/state"
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
	// NOTE: This is just an estimate
	count := bufferedIter.Count() + (len(blockIDs) * DefaultPausesPerBlock)

	return &dualIter{
		idx:             idx,
		count:           count,
		bufferIter:      bufferedIter,
		blockReader:     rdr,
		unfetchedBlocks: blockIDs,
		l:               &sync.Mutex{},
	}
}

// dualIter represents an iterator that reads from blocks as well as buffers,
// downloading blocks in parallel to maximize throughput.
type dualIter struct {
	idx Index

	// count is an esitmate of the max pauses in the iterator.
	count int

	// bufferIter is the buffered iterator.
	bufferIter state.PauseIterator

	// blockReader is the block reader to fetch metadata and blocks.
	blockReader BlockReader

	// unfetchedBlocks represents the set of blocks that haven't been fetched
	// from the backing store yet.
	unfetchedBlocks []ulid.ULID

	// inflightBlocks represents blocks that are currently being fetched from
	// the backing store.
	inflightBlocks []ulid.ULID

	// pauses represents the current pauses that have been fetched from downloaded
	// blocks.
	pauses []*state.Pause

	err error

	// l represents a lock held when mutating block slices or pauses.
	l *sync.Mutex
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
	// TODO: Block on this if len(unfetchedBlocks) > 0 and len(pauses) == 0
	d.fetchNextBlocks()

	return false
}

// Error returns the error returned during iteration, if any.  Use this to check
// for errors during iteration when Next() returns false.
func (d *dualIter) Error() error {
	return fmt.Errorf("not implemented")
}

// Val returns the current Pause from the iterator.
func (d *dualIter) Val(context.Context) *state.Pause {
	return nil
}

// Index shows how far the iterator has progressed
func (d *dualIter) Index() int64 {
	return 0
}

func (d *dualIter) fetchNextBlocks() {
	d.l.Lock()
	defer d.l.Unlock()

	maxFetch := DefaultConcurrentBlockFetches - len(d.inflightBlocks)
	if maxFetch > len(d.unfetchedBlocks) {
		maxFetch = len(d.unfetchedBlocks)
	}

	if maxFetch == 0 {
		return
	}

	blockIDs := d.unfetchedBlocks[0:maxFetch]

	// Move the blocks to in-flight.
	d.inflightBlocks = append(d.inflightBlocks, blockIDs...)

	// And remove from unfetched
	d.unfetchedBlocks = d.unfetchedBlocks[maxFetch:]

	for _, bid := range blockIDs {
		// Fetch each block, without capturing the ID.
		id := bid
		go d.fetchBlock(context.Background(), id)
	}
}

func (d *dualIter) fetchBlock(ctx context.Context, id ulid.ULID) {
	if d.err != nil {
		// Don't bother to fetch, as we already have an error
		return
	}

	block, err := d.blockReader.ReadBlock(ctx, d.idx, id)
	if err != nil && d.err != nil {
		d.err = err
		return
	}

	d.l.Lock()
	defer d.l.Unlock()
	d.pauses = append(d.pauses, block.Pauses...)
}
