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
	TriggerType     string
	EventID         ulid.ULID
	BatchID         ulid.ULID
	OriginalRunID   ulid.ULID
}

type FunctionRunManager interface {
	FunctionRunWriter
	FunctionRunReader
}

type FunctionRunWriter interface {
	InsertFunctionRun(ctx context.Context, run FunctionRun) error
}

type FunctionRunReader interface {
	GetFunctionRunsTimebound(ctx context.Context, t Timebound, limit int) ([]*FunctionRun, error)
}
