package cron

import (
	"context"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// CronManager represents the handling of cron
type CronManager interface {
	ScheduleNext(ctx context.Context) error
	UpsertSchedule(ctx context.Context, fnID uuid.UUID) error
}

// CronItem represent an item that can be scheduled via the cron expression
type CronItem struct {
	ID              ulid.ULID `json:"id"`
	AccountID       uuid.UUID `json:"acctID"`
	WorkspaceID     uuid.UUID `json:"wsID"`
	AppID           uuid.UUID `json:"appID"`
	FunctionID      uuid.UUID `json:"fnID"`
	FunctionVersion int       `json:"fnV"`
	Expression      string    `jaon:"expr"`
}
