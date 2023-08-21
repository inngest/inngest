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
}
