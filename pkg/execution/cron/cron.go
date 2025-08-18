package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	cron "github.com/robfig/cron/v3"
)

const (
	pkgName = "cron.execution.inngest"
)

var (
	// parser is a global cron expression parser that supports minute-level precision
	// and includes descriptive names (e.g., @hourly, @daily)
	parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
)

var (
	errNextScheduleNotFound = fmt.Errorf("next schedule not found")
)

// Next returns the next scheduled time for the cron expression based on the time providedk
func Next(expr string, from time.Time) (time.Time, error) {
	schedule, err := parser.Parse(expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing cron expression: %w", err)
	}
	return schedule.Next(from), nil
}

type CronSyncer interface {
	// EnqueueSync handles the enqueueing of cron schedule sync jobs
	Sync(ctx context.Context, ci CronItem) error
}

// CronManager represents the handling of cron
type CronManager interface {
	CronSyncer

	// ScheduleNext handles the scheduling of the next cron job
	ScheduleNext(ctx context.Context, ci CronItem) (*CronItem, error)
	// CanRun checks if the cron item can be scheduled for execution
	CanRun(ctx context.Context, ci CronItem) (bool, error)
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
	// NextScheduledItemForFunction retrieves the next cron item for the function
	NextScheduledItemForFunction(ctx context.Context, fnID uuid.UUID) (*CronItem, error)
}

// CronItem represent an item that can be scheduled via the cron expression
type CronItem struct {
	// ID embeds the time this job needs to run if it's a process type
	// ULID was chosen as a convinent way to provide a unique identifier with a timestamp
	// embedded in it
	ID ulid.ULID `json:"id"`
	// Tenant
	AccountID       uuid.UUID `json:"acctID"`
	WorkspaceID     uuid.UUID `json:"wsID"`
	AppID           uuid.UUID `json:"appID"`
	FunctionID      uuid.UUID `json:"fnID"`
	FunctionVersion int       `json:"fnV"`
	// Expression is the actual cron expression being used
	Expression string `jaon:"expr"`
	// JobID stores queue item ID that's supposed to be handling this cron item.
	// This is only available if it's a process type.
	//
	// NOTE
	// This is based on the assumption that the ID field is always used for JobID assignments when enqueueing for idempotency handling reasons.
	JobID string `json:"prevJobID,omitempty"`
	// Op indicates what type of cron operation this item is for.
	Op enums.CronOp `json:"op"`
}

// Equal checks if the cron item is identical
// NOTE this just do a dump field check right now, there might be better ways of handling equation checks
func (i CronItem) Equal(ci CronItem) bool {
	return i.ID == ci.ID &&
		i.AccountID == ci.AccountID &&
		i.WorkspaceID == ci.WorkspaceID &&
		i.AppID == ci.AppID &&
		i.FunctionID == ci.FunctionID &&
		i.FunctionVersion == ci.FunctionVersion &&
		i.Expression == ci.Expression &&
		i.JobID == ci.JobID &&
		i.Op == ci.Op
}

// SyncID is used for the jobID when enqueueing non processing types
func (i CronItem) SyncID() string {
	return fmt.Sprintf("%s:sync", i.ID)
}

// ProcessID is used for the jobID when enqueueing processing types
func (i CronItem) ProcessID() string {
	return fmt.Sprintf("%s:process", i.ID)
}
