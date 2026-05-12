package cqrs

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/oklog/ulid/v2"
)

type HistoryManager interface {
	HistoryWriter
	HistoryReader
}

type HistoryWriter interface {
	InsertHistory(ctx context.Context, h history.History) error
}

type HistoryReader interface {
	// GetFunctionRunHistory must return history for the given function run,
	// ordered from oldest to newest.
	GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*history.History, error)

	// GetRunDefers returns the defers for each parent run, keyed by parent run
	// ID, in the order their DeferAdd opcodes first appeared. A DeferAdd later
	// aborted by a matching DeferAbort folds into a single entry with status
	// ABORTED; otherwise the entry is SCHEDULED. Orphan DeferAbort opcodes are
	// ignored. Parents with no defers are omitted from the map.
	GetRunDefers(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]RunDefer, error)

	// GetRunDeferredFrom returns the parent-run linkage for each deferred run,
	// keyed by child run ID. Runs not triggered by an inngest/deferred.schedule
	// event are omitted from the map.
	GetRunDeferredFrom(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID]*RunDeferredFrom, error)
}
