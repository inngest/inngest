package base_cqrs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	sqlc_postgres "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/history_reader"
	"github.com/inngest/inngest/pkg/usage"
	"github.com/oklog/ulid/v2"
)

func NewHistoryReader(db *sql.DB, driver string, o sqlc_postgres.NewNormalizedOpts) history_reader.Reader {
	return &reader{
		q: NewQueries(db, driver, o),
	}
}

type reader struct {
	q sqlc.Querier
}

func (r *reader) CountRuns(
	ctx context.Context,
	opts history_reader.CountRunOpts,
) (int, error) {
	count, err := r.q.HistoryCountRuns(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count runs: %w", err)
	}

	return int(count), nil
}

func (r *reader) GetRun(
	ctx context.Context,
	runID ulid.ULID,
	opts history_reader.GetRunOpts,
) (history_reader.Run, error) {
	rawRun, err := r.q.GetFunctionRun(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return history_reader.Run{}, history_reader.ErrNotFound
		}

		return history_reader.Run{}, fmt.Errorf("failed to get run: %w", err)
	}

	run, err := sqlToRun(&rawRun.FunctionRun, &rawRun.FunctionFinish)
	if err != nil {
		return history_reader.Run{}, fmt.Errorf("failed to convert run: %w", err)
	}

	return *run, nil
}

func (r *reader) GetFunctionRun(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	runID ulid.ULID,
) (*cqrs.FunctionRun, error) {
	run, err := r.GetRun(ctx, runID, history_reader.GetRunOpts{
		AccountID:   accountID,
		WorkspaceID: &workspaceID,
	})
	if err != nil {
		return nil, err
	}
	return run.ToCQRS(), nil
}

// For the API - CQRS return
func (r *reader) GetFunctionRunsFromEvents(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	eventIDs []ulid.ULID,
) ([]*cqrs.FunctionRun, error) {
	runs, err := r.q.GetFunctionRunsFromEvents(ctx, eventIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get function runs: %w", err)
	}

	result := []*cqrs.FunctionRun{}
	for _, rawRun := range runs {
		run, err := sqlToRun(&rawRun.FunctionRun, &rawRun.FunctionFinish)
		if err != nil {
			return nil, fmt.Errorf("failed to convert run: %w", err)
		}

		result = append(result, run.ToCQRS())
	}

	return result, nil
}

func (r *reader) GetRunHistory(
	ctx context.Context,
	runID ulid.ULID,
	opts history_reader.GetRunOpts,
) ([]*history_reader.RunHistory, error) {
	rows, err := r.q.GetFunctionRunHistory(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, history_reader.ErrNotFound
		}

		return nil, fmt.Errorf("failed to get run history: %w", err)
	}

	var items []*history_reader.RunHistory
	for _, row := range rows {
		historyItem, err := sqlToRunHistory(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert history item: %w", err)
		}
		items = append(items, historyItem)
	}

	return items, nil
}

func (r *reader) GetRunHistoryItemOutput(
	ctx context.Context,
	historyID ulid.ULID,
	opts history_reader.GetHistoryOutputOpts,
) (*string, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	item, err := r.q.GetHistoryItem(ctx, historyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, history_reader.ErrNotFound
		}

		return nil, fmt.Errorf("failed to get history item: %w", err)
	}

	if !item.Result.Valid {
		return nil, history_reader.ErrNotFound
	}

	var (
		result            *string
		execHistoryResult history.Result
	)

	if err := json.Unmarshal([]byte(item.Result.String), &execHistoryResult); err == nil {
		result = &execHistoryResult.Output
	}

	return result, nil
}

func (r *reader) GetRuns(
	ctx context.Context,
	opts history_reader.GetRunsOpts,
) ([]history_reader.Run, error) {
	runs, err := r.q.GetFunctionRuns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get runs: %w", err)
	}

	var result []history_reader.Run
	for _, run := range runs {
		r, err := sqlToRun(&run.FunctionRun, &run.FunctionFinish)
		if err != nil {
			return nil, fmt.Errorf("failed to convert run: %w", err)
		}

		result = append(result, *r)
	}

	return result, nil
}

func (r *reader) GetRunsByEventID(
	ctx context.Context,
	eventID ulid.ULID,
	opts history_reader.GetRunsByEventIDOpts,
) ([]history_reader.Run, error) {
	runs, err := r.q.GetFunctionRunsFromEvents(ctx, []ulid.ULID{eventID})
	if err != nil {
		return nil, fmt.Errorf("failed to get runs by event ID: %w", err)
	}

	var result []history_reader.Run
	for _, run := range runs {
		r, err := sqlToRun(&run.FunctionRun, &run.FunctionFinish)
		if err != nil {
			return nil, fmt.Errorf("failed to convert run: %w", err)
		}

		result = append(result, *r)
	}

	return result, nil
}

func (r *reader) GetSkippedRunsByEventID(
	_ context.Context,
	_ ulid.ULID,
	_ history_reader.GetRunsByEventIDOpts,
) ([]history_reader.SkippedRun, error) {
	return nil, errors.New("not implemented")
}

func (r *reader) GetUsage(
	ctx context.Context,
	opts history_reader.GetUsageOpts,
) ([]usage.UsageSlot, error) {
	return nil, errors.New("not implemented")
}

func (r *reader) GetReplayRuns(ctx context.Context, opts history_reader.GetReplayRunsOpts) ([]history_reader.ReplayRun, error) {
	return nil, errors.New("not implemented")
}

func (r *reader) CountReplayRuns(ctx context.Context, opts history_reader.CountReplayRunsOpts) (history_reader.ReplayRunCounts, error) {
	return history_reader.ReplayRunCounts{}, errors.New("not implemented")
}

func (r *reader) GetActiveRunIDs(
	ctx context.Context,
	opts history_reader.GetActiveRunIDsOpts,
) ([]ulid.ULID, error) {
	return nil, errors.New("not implemented")
}

func (r *reader) CountActiveRuns(
	ctx context.Context,
	opts history_reader.CountActiveRunsOpts,
) (int, error) {
	return 0, errors.New("not implemented")
}

func sqlToRun(item *sqlc.FunctionRun, finish *sqlc.FunctionFinish) (*history_reader.Run, error) {
	if item == nil {
		return nil, history_reader.ErrNotFound
	}

	var (
		endedAt       *time.Time
		batchID       *ulid.ULID
		output        *string
		status        = enums.RunStatusRunning
		originalRunID *ulid.ULID
		cron          *string
	)

	if finish != nil && finish.Status.Valid {
		status, _ = enums.RunStatusString(finish.Status.String)
		output = &finish.Output.String
		endedAt = &finish.CreatedAt.Time
	}

	if !item.BatchID.IsZero() {
		batchID = &item.BatchID
	}

	if !item.OriginalRunID.IsZero() {
		originalRunID = &item.OriginalRunID
	}

	if item.Cron.Valid && item.Cron.String != "" {
		cron = &item.Cron.String
	}

	return &history_reader.Run{
		AccountID:       nilUUID,
		BatchID:         batchID,
		EndedAt:         endedAt,
		EventID:         item.EventID,
		ID:              item.RunID,
		OriginalRunID:   originalRunID,
		Output:          output,
		StartedAt:       item.RunStartedAt,
		Status:          status,
		WorkflowID:      item.FunctionID,
		WorkspaceID:     nilUUID,
		WorkflowVersion: int(item.FunctionVersion),
		Cron:            cron,
	}, nil
}

func sqlToRunHistory(item *sqlc.History) (*history_reader.RunHistory, error) {
	historyType, err := enums.HistoryTypeString(item.Type)
	if err != nil {
		return nil, fmt.Errorf("invalid history type: %w", err)
	}

	var (
		cancel *history_reader.RunHistoryCancel

		groupID *uuid.UUID

		stepName *string

		url *string

		stepType *enums.HistoryStepType

		result            *history_reader.RunHistoryResult
		execHistoryResult *history.Result

		sleep            *history_reader.RunHistorySleep
		execHistorySleep *history.Sleep

		waitForEvent            *history_reader.RunHistoryWaitForEvent
		execHistoryWaitForEvent *history.WaitForEvent

		waitResult            *history_reader.RunHistoryWaitResult
		execHistoryWaitResult *history.WaitResult

		invokeFunction            *history_reader.RunHistoryInvokeFunction
		execHistoryInvokeFunction *history.InvokeFunction

		invokeFunctionResult            *history_reader.RunHistoryInvokeFunctionResult
		execHistoryInvokeFunctionResult *history.InvokeFunctionResult
	)

	if item.CancelRequest.Valid {
		_ = json.Unmarshal([]byte(item.CancelRequest.String), &cancel)
	}

	if item.GroupID.Valid {
		if gid, err := uuid.Parse(item.GroupID.String); err == nil {
			groupID = &gid
		}
	}

	if item.StepName.Valid {
		stepName = &item.StepName.String
	}

	if item.Url.Valid {
		url = &item.Url.String
	}

	if item.StepType.Valid {
		if st, err := enums.HistoryStepTypeString(item.StepType.String); err == nil {
			stepType = &st
		}
	}

	if item.Result.Valid {
		if err := json.Unmarshal([]byte(item.Result.String), &execHistoryResult); err == nil && execHistoryResult != nil {
			result = &history_reader.RunHistoryResult{
				DurationMS:  execHistoryResult.DurationMS,
				ErrorCode:   execHistoryResult.ErrorCode,
				Framework:   execHistoryResult.Framework,
				Platform:    execHistoryResult.Platform,
				SDKLanguage: execHistoryResult.SDKLanguage,
				SDKVersion:  execHistoryResult.SDKVersion,
				SizeBytes:   execHistoryResult.SizeBytes,
			}
		}
	}

	if item.Sleep.Valid {
		if err := json.Unmarshal([]byte(item.Sleep.String), &execHistorySleep); err == nil && execHistorySleep != nil {
			sleep = &history_reader.RunHistorySleep{
				Until: execHistorySleep.Until,
			}
		}
	}

	if item.WaitForEvent.Valid {
		if err := json.Unmarshal([]byte(item.WaitForEvent.String), &execHistoryWaitForEvent); err == nil && execHistoryWaitForEvent != nil {
			waitForEvent = &history_reader.RunHistoryWaitForEvent{
				EventName:  execHistoryWaitForEvent.EventName,
				Expression: execHistoryWaitForEvent.Expression,
				Timeout:    execHistoryWaitForEvent.Timeout,
			}
		}
	}

	if item.WaitResult.Valid {
		if err := json.Unmarshal([]byte(item.WaitResult.String), &execHistoryWaitResult); err == nil && execHistoryWaitResult != nil {
			waitResult = &history_reader.RunHistoryWaitResult{
				EventID: execHistoryWaitResult.EventID,
				Timeout: execHistoryWaitResult.Timeout,
			}
		}
	}

	if item.InvokeFunction.Valid {
		if err := json.Unmarshal([]byte(item.InvokeFunction.String), &execHistoryInvokeFunction); err == nil && execHistoryInvokeFunction != nil {
			invokeFunction = &history_reader.RunHistoryInvokeFunction{
				CorrelationID: execHistoryInvokeFunction.CorrelationID,
				EventID:       execHistoryInvokeFunction.EventID,
				FunctionID:    execHistoryInvokeFunction.FunctionID,
				Timeout:       execHistoryInvokeFunction.Timeout,
			}
		}
	}

	if item.InvokeFunctionResult.Valid {
		if err := json.Unmarshal([]byte(item.InvokeFunctionResult.String), &execHistoryInvokeFunctionResult); err == nil && execHistoryInvokeFunctionResult != nil {
			invokeFunctionResult = &history_reader.RunHistoryInvokeFunctionResult{
				EventID: execHistoryInvokeFunctionResult.EventID,
				RunID:   execHistoryInvokeFunctionResult.RunID,
				Timeout: execHistoryInvokeFunctionResult.Timeout,
			}
		}
	}

	return &history_reader.RunHistory{
		Attempt:              item.Attempt,
		Cancel:               cancel,
		CreatedAt:            item.CreatedAt,
		FunctionVersion:      item.FunctionVersion,
		GroupID:              groupID,
		ID:                   item.ID,
		InvokeFunction:       invokeFunction,
		InvokeFunctionResult: invokeFunctionResult,
		Result:               result,
		RunID:                item.RunID,
		Sleep:                sleep,
		StepName:             stepName,
		StepType:             stepType,
		Type:                 historyType,
		URL:                  url,
		WaitForEvent:         waitForEvent,
		WaitResult:           waitResult,
	}, nil
}
