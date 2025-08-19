package cron

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	cron "github.com/robfig/cron/v3"
)

type CronOp int

const (
	CronOpNew CronOp = iota
	CronOpUpdate
	CronOpPause
	CronOpUnpause
	CronOpProcess
)

var (
	// parser is a global cron expression parser that supports minute-level precision
	// and includes descriptive names (e.g., @hourly, @daily)
	parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	allowedVariant = 50 * time.Second
)

// Parser returns the global cron parser instance
// NOTE comment this out for now until its needed
// func Parser() cron.Parser {
// 	return parser
// }

// Parse parses a cron expression string and returns a schedule
func Parse(str string) (cron.Schedule, error) {
	return parser.Parse(str)
}

// IsAt checks if the given time falls within a window of when the cron schedule
// should execute. This provides tolerance for timing variations in cron execution.
func IsAt(cs cron.Schedule, t time.Time) bool {
	next := cs.Next(t.Add(-50 * time.Second))
	diff := t.Sub(next).Seconds()
	return diff >= 0 && diff <= float64(allowedVariant)
}

// Validate checks if a cron expression string is syntactically valid
func Validate(str string) error {
	_, err := parser.Parse(str)
	return err
}

// CronManager represents the handling of cron
type CronManager interface {
	// Next returns the next schedule time based on the cron item attributes with a jitter
	Next(ctx context.Context, ci CronItem) (time.Time, error)
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
