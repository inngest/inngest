package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// StaleRunInfo contains the identifiers for a run detected as stale (RUNNING with
// no outstanding queue items past the threshold). This is used by the stale run
// scavenger to pass run information to the cancellation handler.
type StaleRunInfo struct {
	RunID       ulid.ULID
	FunctionID  uuid.UUID
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       uuid.UUID
}

// StaleRunScavenger is an optional interface that queue shards can implement
// to support stale run detection. The stale run scavenger detects RUNNING runs
// that have no outstanding queue items and have been active longer than the
// given threshold, which indicates they were orphaned during a rolling deployment.
type StaleRunScavenger interface {
	// ScavengeStaleRuns finds runs tracked in the active runs index that have
	// no outstanding queue items and were started before (now - threshold).
	// It returns their identifiers for cancellation by the caller.
	// Runs that are no longer in the active runs index (already finalized)
	// are cleaned up automatically.
	ScavengeStaleRuns(ctx context.Context, threshold time.Duration) ([]StaleRunInfo, error)

	// TrackActiveRun adds a run to the active runs index, scored by startTime.
	// This should be called when a run starts (KindStart item is enqueued).
	TrackActiveRun(ctx context.Context, info StaleRunInfo, startTime time.Time) error

	// RemoveActiveRun removes a run from the active runs index.
	// This is called during scavenging cleanup or when a run is finalized.
	RemoveActiveRun(ctx context.Context, info StaleRunInfo) error
}
