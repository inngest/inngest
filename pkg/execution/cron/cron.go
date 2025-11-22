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

var (
	// parser is a global cron expression parser that supports minute-level precision
	// and includes descriptive names (e.g., @hourly, @daily)
	parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
)

// Next returns the next scheduled time for the cron expression based on the time provided
func Next(expr string, from time.Time) (time.Time, error) {
	schedule, err := parser.Parse(expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing cron expression: %w", err)
	}
	return schedule.Next(from), nil
}

type CronSyncer interface {
	// Sync handles the enqueueing of cron schedule sync jobs
	Sync(ctx context.Context, ci CronItem) error
}

type CronHealthChecker interface {
	// HealthCheck checks if a "cron" queue item exists in the system queue for the next expected schedule time
	HealthCheck(ctx context.Context, functionID uuid.UUID, expr string, fnVersion int) (CronHealthCheckStatus, error)

	// Enqueues the next periodic global cron-health-check system job
	EnqueueNextHealthCheck(ctx context.Context, cur time.Time) error

	// Enqueues an ad-hoc cron-health-check system job *now*
	// This could be for a specific function, a specific account or globally for all crons.
	EnqueueHealthCheck(ctx context.Context, ci CronItem) error
}

// CronManager represents the handling of cron
type CronManager interface {
	CronSyncer

	CronHealthChecker

	CronProcessJobID(schedule time.Time, expr string, fnID uuid.UUID, fnVersion int) string

	// ScheduleNext handles the scheduling of the next cron job
	ScheduleNext(ctx context.Context, ci CronItem) (*CronItem, error)
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
	Expression string `json:"expr"`
	// JobID stores queue item ID that's supposed to be handling this cron item.
	// This is only available if it's a process type.
	//
	// NOTE
	// This is based on the assumption that the ID field is always used for JobID assignments when enqueueing for idempotency handling reasons.
	JobID string `json:"prevJobID,omitempty"`
	// Op indicates what type of cron operation this item is for.
	Op enums.CronOp `json:"op"`
}

// SyncID is used for the jobID when enqueueing non processing types
func (i CronItem) SyncID() string {
	return fmt.Sprintf("%s:sync", i.ID)
}

type CronHealthCheckStatus struct {
	// next expected cron schedule time
	Next time.Time `json:"next"`
	// JobID is the "cron" system queue item's jobID for a given fnID, fnVersion, cronExpr combination for the next expected schedule time
	JobID string `json:"jobID"`
	// Scheduled indicates whether a queue item with the above jobID exists in the system queue
	Scheduled bool `json:"scheduled"`
}
