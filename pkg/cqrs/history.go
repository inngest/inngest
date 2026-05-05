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

	// GetRunDefers returns the structured list of defers for a parent run,
	// parsed from the run's history opcodes and joined to any child runs
	// triggered by the deterministic deferred.schedule events.
	//
	// Defers are returned in the order their DeferAdd opcodes first
	// appeared. A DeferAdd whose hashed step ID was later cancelled by a
	// DeferCancel folds into a single entry with status ABORTED; otherwise
	// the entry is SCHEDULED. DeferCancel opcodes without a matching
	// DeferAdd are ignored.
	GetRunDefers(ctx context.Context, runID ulid.ULID) ([]RunDefer, error)

	// GetRunDeferredFrom returns the parent-run linkage for a deferred run, or
	// nil if the run was not triggered by an inngest/deferred.schedule event.
	GetRunDeferredFrom(ctx context.Context, runID ulid.ULID) (*RunDeferredFrom, error)
}
