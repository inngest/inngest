package pauses

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
)

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

// Delete deletes a pause from the buffer, or returns ErrNotInBuffer if the pause is not in
// the buffer.
func (r redisAdapter) Delete(ctx context.Context, index Index, pause state.Pause) error {
	// Check if pause is in buffer;  if not, return ErrNotInBuffer so that we can default
	// to deleting the pause from the backing store.
	if r.rsm.PauseExists(ctx, pause.ID) == state.ErrPauseNotFound {
		return ErrNotInBuffer
	}
	return r.rsm.DeletePause(ctx, pause)
}

// PauseTimestamp returns the created at timestamp for a pause.
func (r redisAdapter) PauseTimestamp(ctx context.Context, index Index, pause state.Pause) (time.Time, error) {
	// Fetch timestamp from index.
	return r.rsm.PauseCreatedAt(ctx, index.WorkspaceID, index.EventName, pause.ID)
}

func (r redisAdapter) ConsumePause(ctx context.Context, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error) {
	return r.rsm.ConsumePause(ctx, pause, opts)
}
