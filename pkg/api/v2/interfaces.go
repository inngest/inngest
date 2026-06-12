package apiv2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
)

type FunctionScheduler interface {
	// Schedule initializes a new function run, ensuring that the function will be
	// executed via our async execution engine as quickly as possible.
	//
	// This returns a run ID, metadata for the run, and any errors scheduling.
	//
	// If the run was impacted by flow control (idempotency, rate limiting, debounce, etc.),
	// metadata will be nil.  This will return the original run ID if runs were skipped due
	// to idemptoency.
	Schedule(ctx context.Context, req execution.ScheduleRequest) (*ulid.ULID, *sv2.Metadata, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event event.TrackedEvent) error
}

type GetFunctionRunOpts struct {
	IncludeOutput bool
}

type FunctionRunReader interface {
	GetFunctionRun(ctx context.Context, runID ulid.ULID, opts GetFunctionRunOpts) (*cqrs.FunctionRun, error)
}

type GetRunsOpts struct {
	EventID       ulid.ULID
	Cursor        ulid.ULID
	Limit         int
	IncludeOutput bool
}

type RunListItem struct {
	RunID        ulid.ULID
	RunStartedAt time.Time
	EventID      ulid.ULID
	BatchID      *ulid.ULID
	Cron         *string
	Status       enums.RunStatus
	EndedAt      *time.Time
	Output       json.RawMessage

	FunctionID   string
	FunctionName string
	AppID        string
}

type GetRunsResult struct {
	Runs    []*RunListItem
	HasMore bool
}

type RunsReader interface {
	GetRuns(ctx context.Context, opts GetRunsOpts) (*GetRunsResult, error)
}

type FunctionTraceReader interface {
	GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error)
	GetSpanOutput(ctx context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error)
}
