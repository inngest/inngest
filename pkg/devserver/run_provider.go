package devserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

type runProvider struct {
	data      runProviderDataReader
	q         db.Querier
	scheduler apiv2.FunctionScheduler
}

type runProviderDataReader interface {
	runSpanReader
	GetFunctionRun(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, runID ulid.ULID) (*cqrs.FunctionRun, error)
	GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*cqrs.Function, error)
	GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*cqrs.Event, error)
}

type runSpanReader interface {
	GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error)
}

func NewRunProvider(data runProviderDataReader, q db.Querier, scheduler apiv2.FunctionScheduler) apiv2.RunProvider {
	return &runProvider{data: data, q: q, scheduler: scheduler}
}

func (p *runProvider) GetRun(ctx context.Context, runID ulid.ULID, _ apiv2.GetRunOpts) (*cqrs.FunctionRun, error) {
	return functionRunFromSpan(ctx, p.data, runID)
}

func (p *runProvider) GetRuns(ctx context.Context, opts apiv2.GetRunsOpts) (*apiv2.GetRunsResult, error) {
	rows, err := p.q.GetRuns(ctx, db.GetRunsParams{
		EventID:       opts.EventID,
		Cursor:        opts.Cursor,
		Limit:         int64(opts.Limit + 1),
		IncludeOutput: opts.IncludeOutput,
	})
	if err != nil {
		return nil, err
	}

	hasMore := len(rows) > opts.Limit
	if hasMore {
		rows = rows[:opts.Limit]
	}

	runs := make([]*apiv2.RunListItem, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, runListItemFromRow(row, opts.IncludeOutput))
	}

	return &apiv2.GetRunsResult{
		Runs:    runs,
		HasMore: hasMore,
	}, nil
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

func runListItemFromRow(row *db.RunListItemRow, includeOutput bool) *apiv2.RunListItem {
	fn := inngest.Function{}
	_ = json.Unmarshal([]byte(row.FunctionConfig), &fn)

	functionName := fn.Name
	if functionName == "" {
		functionName = row.FunctionName
	}

	appID := row.AppName
	if appID == "" {
		appID = row.FunctionAppID.String()
	}

	run := &apiv2.RunListItem{
		RunID:        row.FunctionRun.RunID,
		RunStartedAt: row.FunctionRun.RunStartedAt,
		EventID:      row.FunctionRun.EventID,
		FunctionID:   publicRunListFunctionID(row.AppName, row.FunctionSlug, fn.Slug),
		FunctionName: functionName,
		AppID:        appID,
	}

	if !row.FunctionRun.BatchID.IsZero() {
		run.BatchID = &row.FunctionRun.BatchID
	}
	if row.FunctionRun.Cron.Valid {
		run.Cron = &row.FunctionRun.Cron.String
	}
	if row.FunctionFinish.Status.Valid {
		run.Status, _ = enums.RunStatusString(row.FunctionFinish.Status.String)
		if row.FunctionFinish.CreatedAt.Valid {
			run.EndedAt = &row.FunctionFinish.CreatedAt.Time
		}
		if includeOutput && len(row.Output) > 0 {
			run.Output = publicRunOutput(row.Output)
		}
	}

	return run
}

func publicRunListFunctionID(appID string, storedFunctionID string, configFunctionID string) string {
	if configFunctionID != "" && configFunctionID != storedFunctionID {
		return configFunctionID
	}

	functionID := configFunctionID
	if functionID == "" {
		functionID = storedFunctionID
	}
	if appID != "" {
		return strings.TrimPrefix(functionID, appID+"-")
	}
	return functionID
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
		Op    string          `json:"op"`
		Data  json.RawMessage `json:"data"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(output, &opcodes); err != nil {
		return util.EnsureJSON(output)
	}

	for i := len(opcodes) - 1; i >= 0; i-- {
		if opcodes[i].Op == enums.OpcodeRunComplete.String() || opcodes[i].Op == enums.OpcodeSyncRunComplete.String() {
			return util.EnsureJSON(opcodes[i].Data)
		}
		if opcodes[i].Op == enums.OpcodeRunError.String() {
			wrapped, err := json.Marshal(map[string]json.RawMessage{execution.StateErrorKey: opcodes[i].Error})
			if err != nil {
				return util.EnsureJSON(output)
			}
			return util.EnsureJSON(wrapped)
		}
	}

	return util.EnsureJSON(output)
}
