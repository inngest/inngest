package pauses

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
)

// StateBufferer transforms a state.Manager into a state.Bufferer
func StateBufferer(rsm state.PauseManager) Bufferer {
	return &redisAdapter{rsm}
}

// redisAdapter transforms a state.Manager into a state.Buffer, changing the interfaces slightly
// according to this package.
type redisAdapter struct {
	// rsm represents the redis state manager in redis_state.
	rsm state.PauseManager
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
func (r redisAdapter) Delete(ctx context.Context, index Index, pause state.Pause, opts ...state.DeletePauseOpt) error {
	return r.rsm.DeletePause(ctx, pause, opts...)
}

// PauseByID loads pauses by ID.
//
// This is only used for legacy pause timeout jobs enqueued without events or pauses;  in this case,
// we need to load pauses by only their ID.  To do this, we keep a record of pause ID -> block ID
// when flushing pauses to block storage.
//
// This can be deleted in Sept 2025.
func (r redisAdapter) PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error) {
	return r.rsm.PauseByID(ctx, pauseID)
}

// PauseTimestamp returns the created at timestamp for a pause.
func (r redisAdapter) PauseTimestamp(ctx context.Context, index Index, pause state.Pause) (time.Time, error) {
	// Fetch timestamp from index.
	return r.rsm.PauseCreatedAt(ctx, index.WorkspaceID, index.EventName, pause.ID)
}

func (r redisAdapter) ConsumePause(ctx context.Context, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error) {
	return r.rsm.ConsumePause(ctx, pause, opts)
}

func (r redisAdapter) PauseByInvokeCorrelationID(ctx context.Context, workspaceID uuid.UUID, correlationID string) (*state.Pause, error) {
	return r.rsm.PauseByInvokeCorrelationID(ctx, workspaceID, correlationID)
}

func (r redisAdapter) PauseBySignalID(ctx context.Context, workspaceID uuid.UUID, signal string) (*state.Pause, error) {
	return r.rsm.PauseBySignalID(ctx, workspaceID, signal)
}

// IndexExists returns whether the buffer has any pauses for the index.
func (r redisAdapter) IndexExists(ctx context.Context, i Index) (bool, error) {
	return r.rsm.EventHasPauses(ctx, i.WorkspaceID, i.EventName)
}

func (r redisAdapter) BufferLen(ctx context.Context, i Index) (int64, error) {
	return r.rsm.PauseLen(ctx, i.WorkspaceID, i.EventName)
}

// PausesSinceWithCreatedAt loads up to limit pauses for a given index since a given time,
// ordered by creation time, with createdAt populated from Redis sorted set scores.
func (r redisAdapter) PausesSinceWithCreatedAt(ctx context.Context, index Index, since time.Time, limit int64) (state.PauseIterator, error) {
	return r.rsm.PausesByEventSinceWithCreatedAt(ctx, index.WorkspaceID, index.EventName, since, limit)
}

func (r redisAdapter) DeletePauseByID(ctx context.Context, pauseID uuid.UUID, workspaceID uuid.UUID) error {
	return r.rsm.DeletePauseByID(ctx, pauseID, workspaceID)
}
