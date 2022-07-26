package inmemory

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/state"
)

type pauseIterator struct {
	n      int
	pauses []*state.Pause
}

// Next advances the iterator and returns whether the next call to Val will
// return a non-nil pause.
//
// Next should be called prior to any call to the iterator's Val method, after
// the iterator has been created.
//
// The order of the iterator is unspecified.
func (p *pauseIterator) Next(ctx context.Context) bool {
	p.n++
	return len(p.pauses) > 0 && p.n <= len(p.pauses)
}

// Val returns the current Pause from the iterator.
func (p *pauseIterator) Val(context.Context) *state.Pause {
	if p.n > len(p.pauses) {
		return nil
	}
	return p.pauses[p.n-1]
}

// Len returns the total number of items being iterated over.
func (p *pauseIterator) Len() int {
	return len(p.pauses)
}
