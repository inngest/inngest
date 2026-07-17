package devserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	state "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

type runProvider struct {
	data      runProviderDataReader
	scheduler runProviderExecutor
}

type runProviderExecutor interface {
	apiv2.FunctionScheduler
	Cancel(ctx context.Context, id state.ID, req execution.CancelRequest) error
}

// Run filters use an exclusive upper bound, so this represents no requested bound.
var maxRunListTime = time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)

type runProviderDataReader interface {
	runSpanReader
	GetFunctionRun(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, runID ulid.ULID) (*cqrs.FunctionRun, error)
	GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*cqrs.Function, error)
	GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*cqrs.Event, error)
	GetRuns(ctx context.Context, opts cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error)
}

type runSpanReader interface {
	GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error)
}

func NewRunProvider(data runProviderDataReader, scheduler runProviderExecutor) apiv2.RunProvider {
	return &runProvider{data: data, scheduler: scheduler}
}

func (p *runProvider) Cancel(ctx context.Context, runID ulid.ULID) error {
	fnrun, err := p.data.GetFunctionRun(ctx, consts.DevServerAccountID, consts.DevServerEnvID, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiv2.ErrRunNotFound
		}
		return err
	}

	if fnrun.Status == enums.RunStatusCancelled {
		return apiv2.ErrRunAlreadyCancelled
	}
	if enums.RunStatusEnded(fnrun.Status) {
		return apiv2.ErrRunEnded
	}

	return p.scheduler.Cancel(ctx, state.ID{
		RunID:      runID,
		FunctionID: fnrun.FunctionID,
		Tenant: state.Tenant{
			EnvID:     consts.DevServerEnvID,
			AccountID: consts.DevServerAccountID,
		},
	}, execution.CancelRequest{})
}

func (p *runProvider) GetRun(ctx context.Context, runID ulid.ULID, _ apiv2.GetRunOpts) (*cqrs.FunctionRun, error) {
	return functionRunFromSpan(ctx, p.data, runID)
}

func (p *runProvider) GetRuns(ctx context.Context, opts apiv2.GetRunsOpts) (*apiv2.GetRunsResult, error) {
	timeField, err := cqrsRunTimeField(opts.TimeField)
	if err != nil {
		return nil, err
	}

	from := time.Time{}
	if opts.From != nil {
		from = *opts.From
	}
	until := maxRunListTime
	if opts.Until != nil {
		until = *opts.Until
	}
	eventIDs := []ulid.ULID{}
	if opts.EventID != ulid.Zero {
		eventIDs = append(eventIDs, opts.EventID)
	}

	rows, err := p.data.GetRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID:    consts.DevServerAccountID,
			WorkspaceID:  consts.DevServerEnvID,
			AppName:      opts.AppIDs,
			FunctionSlug: opts.FunctionIDs,
			EventID:      eventIDs,
			TimeField:    timeField,
			From:         from,
			Until:        until,
			Status:       opts.Status,
			IsDeferred:   opts.IsDeferred,
		},
		Order: []cqrs.GetTraceRunOrder{{
			Field:     timeField,
			Direction: cqrsRunOrder(opts.Order),
		}},
		Cursor:        opts.Cursor,
		Items:         uint(opts.Limit + 1),
		IncludeOutput: opts.IncludeOutput,
	})
	if err != nil {
		return nil, err
	}

	runs := make([]*apiv2.RunListItem, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, runListItemFromCQRS(row, opts.IncludeOutput))
	}

	hasMore := len(runs) > opts.Limit
	if hasMore {
		runs = runs[:opts.Limit]
	}

	return &apiv2.GetRunsResult{
		Runs:    runs,
		HasMore: hasMore,
	}, nil
}

func cqrsRunTimeField(field apiv2.RunTimeField) (enums.TraceRunTime, error) {
	switch field {
	case apiv2.RunTimeFieldQueuedAt:
		return enums.TraceRunTimeQueuedAt, nil
	case apiv2.RunTimeFieldStartedAt:
		return enums.TraceRunTimeStartedAt, nil
	case apiv2.RunTimeFieldEndedAt:
		return enums.TraceRunTimeEndedAt, nil
	default:
		return enums.TraceRunTimeQueuedAt, fmt.Errorf("unsupported run time field: %d", field)
	}
}

func cqrsRunOrder(order apiv2.OrderDirection) enums.TraceRunOrder {
	if order == apiv2.OrderDirectionAsc {
		return enums.TraceRunOrderAsc
	}
	return enums.TraceRunOrderDesc
}

func (p *runProvider) Rerun(ctx context.Context, runID ulid.ULID, opts apiv2.RerunOpts) (ulid.ULID, error) {
	fnrun, err := p.data.GetFunctionRun(ctx, consts.DevServerAccountID, consts.DevServerEnvID, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ulid.ULID{}, apiv2.ErrRunNotFound
		}
		return ulid.ULID{}, err
	}

	fnCQRS, err := p.data.GetFunctionByInternalUUID(ctx, fnrun.FunctionID)
	if err != nil {
		return ulid.ULID{}, err
	}

	fn, err := fnCQRS.InngestFunction()
	if err != nil {
		return ulid.ULID{}, err
	}

	evt, err := p.data.GetEventByInternalID(ctx, fnrun.EventID)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("failed to get run event: %w", err)
	}

	var fromStep *execution.ScheduleRequestFromStep
	if opts.FromStep != nil {
		fromStep = &execution.ScheduleRequestFromStep{
			StepID: opts.FromStep.StepID,
		}

		if len(opts.FromStep.Input) > 0 {
			if opts.FromStep.Input[0] != '[' {
				return ulid.ULID{}, fmt.Errorf("input is not a valid JSON array")
			}
			fromStep.Input = json.RawMessage(opts.FromStep.Input)
		}
	}

	originalRunID := &fnrun.RunID
	if fnrun.OriginalRunID != nil {
		originalRunID = fnrun.OriginalRunID
	}

	newRunID, _, err := p.scheduler.Schedule(ctx, execution.ScheduleRequest{
		Function: *fn,
		AppID:    fnCQRS.AppID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(evt.Event(), evt.InternalID()),
		},
		OriginalRunID:    originalRunID,
		AccountID:        consts.DevServerAccountID,
		FromStep:         fromStep,
		WorkspaceID:      consts.DevServerEnvID,
		PreventRateLimit: true,
	})
	if err != nil {
		switch {
		case errors.Is(err, executor.ErrRerunStepNotFound):
			return ulid.ULID{}, apiv2.ErrRerunStepNotFound
		case errors.Is(err, executor.ErrRerunStepAmbiguous):
			return ulid.ULID{}, apiv2.ErrRerunStepAmbiguous
		}
		return ulid.ULID{}, err
	}
	if newRunID == nil {
		return ulid.ULID{}, fmt.Errorf("rerun did not return run ID")
	}

	return *newRunID, nil
}

func functionRunFromSpan(ctx context.Context, reader runSpanReader, runID ulid.ULID) (*cqrs.FunctionRun, error) {
	root, err := reader.GetSpansByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, errors.New("run not found")
	}
	if root.Attributes == nil {
		root.Attributes = &meta.ExtractedValues{}
	}

	span, err := loader.ConvertRunSpan(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("error converting run span: %w", err)
	}

	eventID, err := runEventID(root)
	if err != nil {
		return nil, err
	}

	startedAt := root.StartTime
	if span.StartedAt != nil {
		startedAt = *span.StartedAt
	}

	status := enums.RunStatusRunning
	if root.Status != enums.StepStatusUnknown {
		status = enums.StepStatusToRunStatus(root.Status)
	}

	run := &cqrs.FunctionRun{
		RunID:        runID,
		RunStartedAt: startedAt,
		FunctionID:   root.GetFunctionID(),
		EventID:      eventID,
		Status:       status,
		EndedAt:      span.EndedAt,
	}

	if root.Attributes.BatchID != nil {
		run.BatchID = root.Attributes.BatchID
	}
	if root.Attributes.CronSchedule != nil {
		run.Cron = root.Attributes.CronSchedule
	}

	return run, nil
}

func runEventID(root *cqrs.OtelSpan) (ulid.ULID, error) {
	if root.Attributes == nil || root.Attributes.EventIDs == nil || len(*root.Attributes.EventIDs) == 0 {
		return ulid.Zero, errors.New("run span missing event ID")
	}

	eventID, err := ulid.Parse((*root.Attributes.EventIDs)[0])
	if err != nil {
		return ulid.Zero, fmt.Errorf("invalid run event ID: %w", err)
	}

	return eventID, nil
}

func runListItemFromCQRS(row *cqrs.TraceRun, includeOutput bool) *apiv2.RunListItem {
	runID, _ := ulid.Parse(row.RunID)
	run := &apiv2.RunListItem{
		RunID:        runID,
		Cursor:       row.Cursor,
		RunStartedAt: row.StartedAt,
		FunctionID:   row.FunctionID.String(),
		AppID:        row.AppID.String(),
		Status:       row.Status,
	}
	if len(row.TriggerIDs) > 0 {
		run.EventID, _ = ulid.Parse(row.TriggerIDs[0])
	}
	if row.FunctionSlug != "" {
		run.FunctionID = row.FunctionSlug
	}
	if row.FunctionName != "" {
		run.FunctionName = row.FunctionName
	}
	if row.AppName != "" {
		run.AppID = row.AppName
	}
	if row.BatchID != nil {
		run.BatchID = row.BatchID
	}
	if row.CronSchedule != nil {
		run.Cron = row.CronSchedule
	}
	if enums.RunStatusEnded(row.Status) && !row.EndedAt.IsZero() {
		run.EndedAt = &row.EndedAt
	}
	if includeOutput && len(row.Output) > 0 {
		run.Output = publicRunOutput(row.Output)
	}

	return run
}

func publicRunOutput(raw []byte) json.RawMessage {
	output := util.EnsureJSON(json.RawMessage(raw))

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(output, &envelope); err == nil {
		if data, ok := envelope["data"]; ok {
			output = data
		}
	}

	var opcodes []struct {
		Op   string          `json:"op"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(output, &opcodes); err != nil {
		return util.EnsureJSON(output)
	}

	for i := len(opcodes) - 1; i >= 0; i-- {
		if opcodes[i].Op == enums.OpcodeRunComplete.String() || opcodes[i].Op == enums.OpcodeSyncRunComplete.String() {
			return util.EnsureJSON(opcodes[i].Data)
		}
	}

	return util.EnsureJSON(output)
}
