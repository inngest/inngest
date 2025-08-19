package cron

import (
	"context"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type CronOp int

const (
	CronOpNew CronOp = iota
	CronOpUpdate
	CronOpPause
	CronOpUnpause
)

// CronManager represents the handling of cron
type CronManager interface {
	ScheduleNext(ctx context.Context) error
	// UpdateSchedule handles the updating of the next scheduled item.
	//
	// Scenarios:
	//
	// ## New schedule
	// Creates a new schedule
	//
	// ## Update schedule
	// Updates the schedule when the following conditions are met
	// - function version is larger
	// - queue item ID is not identical (this should be an no-op when a retry happens for the system queue)
	//
	// ## Function pause
	// Deletes the existing schedule
	//
	// ## Function unpause
	// Creates a schedule, pretty much similar to new
	//
	UpdateSchedule(ctx context.Context, ci CronItem) error
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
	Op              CronOp    `json:"op"`
}
