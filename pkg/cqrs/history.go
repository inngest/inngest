package cqrs

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/history"
)

type HistoryManager interface {
	HistoryWriter
}

type HistoryWriter interface {
	InsertHistory(ctx context.Context, h history.History) error
}
