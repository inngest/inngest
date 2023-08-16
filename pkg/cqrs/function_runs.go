package cqrs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type FunctionRun struct {
	RunID           ulid.ULID
	RunStartedAt    time.Time
	FunctionID      uuid.UUID
	FunctionVersion int64
	EventID         ulid.ULID
	BatchID         ulid.ULID
	OriginalRunID   ulid.ULID
}

type FunctionRunManager interface {
	FunctionRunWriter
}

type FunctionRunWriter interface {
	InsertFunctionRun(ctx context.Context, run FunctionRun) error
}
