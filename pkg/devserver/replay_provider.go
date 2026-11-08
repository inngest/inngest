package devserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

const replayPageSize = 100

type replayDataReader interface {
	GetRuns(context.Context, cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error)
	GetEventsByInternalIDs(context.Context, []ulid.ULID) ([]*cqrs.Event, error)
}

type replayFunctionProvider interface {
	GetFunctionByApp(context.Context, string, string) (inngest.DeployedFunction, error)
}

type replayProvider struct {
	data      replayDataReader
	functions replayFunctionProvider
	scheduler apiv2.FunctionScheduler
	isPaused  func(uuid.UUID) bool
}

// NewReplayProvider starts replay work in memory without a durable replay row.
// Pending work is lost on restart, and replay IDs cannot be listed.
func NewReplayProvider(data replayDataReader, functions replayFunctionProvider, scheduler apiv2.FunctionScheduler, isPaused func(uuid.UUID) bool) apiv2.ReplayProvider {
	return &replayProvider{data: data, functions: functions, scheduler: scheduler, isPaused: isPaused}
}

func (p *replayProvider) CreateReplay(ctx context.Context, opts apiv2.CreateReplayOpts) (apiv2.ReplayResult, error) {
	fn, err := p.functions.GetFunctionByApp(ctx, opts.AppID, opts.FunctionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiv2.ReplayResult{}, apiv2.ErrFunctionNotFound
		}
		return apiv2.ReplayResult{}, err
	}
	if fn.AccountID != consts.DevServerAccountID || fn.EnvironmentID != consts.DevServerEnvID || fn.ID == uuid.Nil {
		return apiv2.ReplayResult{}, apiv2.ErrFunctionNotFound
	}
	if (!fn.PausedAt.IsZero() && fn.PausedAt.Before(time.Now())) || (p.isPaused != nil && p.isPaused(fn.ID)) {
		return apiv2.ReplayResult{}, apiv2.ErrFunctionPaused
	}

	result := apiv2.ReplayResult{ID: uuid.New(), CreatedAt: time.Now().UTC()}
	opts.Statuses = slices.Clone(opts.Statuses)
	// A future upper bound must not cause this operation to replay its own runs.
	if opts.Until.After(result.CreatedAt) {
		opts.Until = result.CreatedAt
	}
	ctx = logger.WithStdlib(context.WithoutCancel(ctx), logger.StdlibLogger(ctx).With("replay_id", result.ID))
	go p.replay(ctx, result.ID, fn, opts)
	return result, nil
}

func (p *replayProvider) replay(ctx context.Context, replayID uuid.UUID, fn inngest.DeployedFunction, opts apiv2.CreateReplayOpts) {
	if opts.From.After(opts.Until) {
		return
	}
	statuses, skipReasons, err := enums.ExpandReplayRunStatuses(opts.Statuses)
	if err != nil {
		logger.StdlibLogger(ctx).Error("invalid replay statuses", "error", err)
		return
	}
	if len(skipReasons) > 0 {
		statuses = append(statuses, enums.RunStatusSkipped)
	}
	query := cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: consts.DevServerAccountID, WorkspaceID: consts.DevServerEnvID,
			FunctionID: []uuid.UUID{fn.ID}, TimeField: enums.TraceRunTimeQueuedAt,
			From: opts.From, Until: opts.Until, Status: statuses,
		},
		Order: []cqrs.GetTraceRunOrder{{Field: enums.TraceRunTimeQueuedAt, Direction: enums.TraceRunOrderDesc}},
		Items: replayPageSize, IncludeSkipped: len(skipReasons) > 0,
	}
	for {
		runs, err := p.data.GetRuns(ctx, query)
		if err != nil {
			logger.StdlibLogger(ctx).Error("failed to query replay runs", "function_id", fn.ID, "error", err)
			break
		}
		for _, run := range runs {
			if !slices.Contains(statuses, run.Status) || (run.Status == enums.RunStatusSkipped && (run.SkipReason == nil || !slices.Contains(skipReasons, *run.SkipReason))) {
				continue
			}
			if err := p.replayRun(ctx, replayID, fn, run); err != nil {
				logger.StdlibLogger(ctx).Error("failed to schedule replay run", "function_id", fn.ID, "run_id", run.RunID, "error", err)
			}
		}
		if len(runs) < replayPageSize {
			break
		}
		next := runs[len(runs)-1].Cursor
		if next == "" || next == query.Cursor {
			logger.StdlibLogger(ctx).Error("replay query did not advance", "function_id", fn.ID)
			break
		}
		query.Cursor = next
	}
}

func (p *replayProvider) replayRun(ctx context.Context, replayID uuid.UUID, fn inngest.DeployedFunction, run *cqrs.TraceRun) error {
	if run.AccountID != consts.DevServerAccountID || run.WorkspaceID != consts.DevServerEnvID || run.FunctionID != fn.ID {
		return apiv2.ErrRunNotFound
	}
	runID, err := ulid.Parse(run.RunID)
	if err != nil {
		return fmt.Errorf("invalid original run ID: %w", err)
	}
	if len(run.TriggerIDs) == 0 {
		return fmt.Errorf("original run has no trigger events")
	}
	ids := make([]ulid.ULID, len(run.TriggerIDs))
	for i, triggerID := range run.TriggerIDs {
		ids[i], err = ulid.Parse(triggerID)
		if err != nil {
			return fmt.Errorf("invalid original event ID: %w", err)
		}
	}
	rows, err := p.data.GetEventsByInternalIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("load original events: %w", err)
	}
	byID := make(map[ulid.ULID]*cqrs.Event, len(rows))
	for _, row := range rows {
		byID[row.InternalID()] = row
	}
	events := make([]event.TrackedEvent, len(ids))
	for i, id := range ids {
		row, ok := byID[id]
		if !ok {
			return fmt.Errorf("original event %s not found", id)
		}
		events[i] = event.NewBaseTrackedEventWithID(row.Event(), id)
	}
	var batchID *ulid.ULID
	if run.BatchID != nil {
		// Local event_batches is keyed by batch ID, so each new run needs its
		// own batch association even though it keeps the original events.
		id := ulid.Make()
		batchID = &id
	}
	_, _, err = p.scheduler.Schedule(ctx, execution.ScheduleRequest{
		Function: fn.Function, AppID: fn.AppID, AppName: fn.AppName,
		AccountID: consts.DevServerAccountID, WorkspaceID: consts.DevServerEnvID,
		OriginalRunID: &runID, ReplayID: &replayID, BatchID: batchID,
		Events: events, PreventRateLimit: true, PreventDebounce: true,
	})
	return err
}
