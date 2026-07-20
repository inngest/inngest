package apiv2

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
)

// App represents a synced app within an Inngest environment, alongside
// derived display data such as its function count.
type App struct {
	// ID is the user-facing app ID used in v2 API paths.
	ID string
	// InternalID is the internal surrogate key representing this app.
	InternalID uuid.UUID
	// Name is the app name.
	Name string
	// Method is how the app communicates with Inngest (serve, connect, api).
	Method enums.AppMethod
	// AppVersion is the user-defined app version, if set.
	AppVersion string
	// CreatedAt is when the app was first synced.
	CreatedAt time.Time
	// ArchivedAt, if non-zero, indicates that the app is archived as of the given time.
	ArchivedAt time.Time
	// FunctionCount is the number of functions in the app.
	FunctionCount int
	// LatestSync contains data reported by the latest app sync, if available.
	LatestSync *AppSync
}

type AppSync struct {
	Status      string
	SyncedAt    time.Time
	SdkLanguage string
	SdkVersion  string
	Framework   string
	URL         string
	Error       string
	AppVersion  string
}

type AppProvider interface {
	// GetApp returns an app given its external ID OR internal UUID.
	GetApp(ctx context.Context, identifier string) (App, error)
}

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

type EventProvider interface {
	ReplayEvent(ctx context.Context, eventID ulid.ULID, opts ReplayEventOpts) (*ReplayEventResult, error)
}

type ReplayEventMode string

const (
	ReplayEventModeForce    ReplayEventMode = "force"
	ReplayEventModeIfNoRuns ReplayEventMode = "if_no_runs"
)

const ReplayEventSkipReasonEventHasRuns = "event_has_runs"

type ReplayEventOpts struct {
	Mode ReplayEventMode
}

type ReplayEventResult struct {
	EventID       ulid.ULID
	Replayed      bool
	SkippedReason string
}

type GetRunOpts struct {
	IncludeOutput bool
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

type RunProvider interface {
	GetRun(ctx context.Context, runID ulid.ULID, opts GetRunOpts) (*cqrs.FunctionRun, error)
	GetRuns(ctx context.Context, opts GetRunsOpts) (*GetRunsResult, error)
	Rerun(ctx context.Context, runID ulid.ULID, opts RerunOpts) (ulid.ULID, error)
}

type RerunOpts struct {
	FromStep *RerunFromStep
}

type RerunFromStep struct {
	StepID string
	Input  json.RawMessage
}

type FunctionTraceReader interface {
	GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error)
	GetSpanOutput(ctx context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error)
	GetStepSpanByStepID(ctx context.Context, runID ulid.ULID, stepID string, accountID, workspaceID uuid.UUID) (*cqrs.OtelSpan, error)
}

type ScoreExperimentInput struct {
	ExperimentName string
	Variant        string
}

// ScoreInput describes a single named score recorded against a run, or against
// a specific step when StepID is set.
type ScoreInput struct {
	StepID     *string
	Experiment *ScoreExperimentInput
	Name       string
	// Value is a finite float64 or a bool.
	Value    any
	Metadata []metadata.Update
}

// CreateScoresParams describes one or more scores recorded against a run.
type CreateScoresParams struct {
	RunID  ulid.ULID
	Scores []ScoreInput
}

type ScoreProvider interface {
	// CreateScores records one or more scores for a run or step. Score writes
	// are best-effort; targets are not validated before writing, and scores for
	// missing runs or steps may not surface in queries. Implementations return
	// ErrScoresNotEnabled when the account cannot submit scores.
	CreateScores(ctx context.Context, params CreateScoresParams) error
}
