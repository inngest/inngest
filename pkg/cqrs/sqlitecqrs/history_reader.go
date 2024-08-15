package sqlitecqrs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/sqlitecqrs/sqlc"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/history_reader"
	"github.com/inngest/inngest/pkg/usage"
	"github.com/oklog/ulid/v2"
)

func NewHistoryReader(db *sql.DB) history_reader.Reader {
	return &reader{
		q: sqlc.New(db),
	}
}

type reader struct {
	q *sqlc.Queries
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

	return &item.Result.String, nil
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
		endedAt *time.Time
		output  *string
		status  = enums.RunStatusRunning
	)

	if finish != nil && finish.Status.Valid {
		status, _ = enums.RunStatusString(finish.Status.String)
		output = &finish.Output.String
		endedAt = &finish.CreatedAt.Time
	}

	return &history_reader.Run{
		AccountID:       uuid.UUID{},
		BatchID:         &item.BatchID,
		EndedAt:         endedAt,
		EventID:         item.EventID,
		ID:              item.RunID,
		OriginalRunID:   &item.OriginalRunID,
		Output:          output,
		StartedAt:       item.RunStartedAt,
		Status:          status,
		WorkflowID:      item.FunctionID,
		WorkspaceID:     uuid.UUID{},
		WorkflowVersion: int(item.FunctionVersion),
		Cron:            &item.Cron.String,
	}, nil
}

func sqlToRunHistory(item *sqlc.History) (*history_reader.RunHistory, error) {
	var cancel *history_reader.RunHistoryCancel
	if item.CancelRequest.Valid {
		cancel = &history_reader.RunHistoryCancel{
			// EventID:    item.Cancel.EventID,
			// Expression: item.Cancel.Expression,
			// UserID:     item.Cancel.UserID,
		}
	}

	historyType, err := enums.HistoryTypeString(item.Type)
	if err != nil {
		return nil, fmt.Errorf("invalid history type: %w", err)
	}

	var result *history_reader.RunHistoryResult
	if item.Result.Valid {
		result = &history_reader.RunHistoryResult{
			//
		}
	}

	var sleep *history_reader.RunHistorySleep
	if item.Sleep.Valid {
		sleep = &history_reader.RunHistorySleep{
			// Until: item.Sleep.Until,
		}
	}

	var waitForEvent *history_reader.RunHistoryWaitForEvent
	if item.WaitForEvent.Valid {
		waitForEvent = &history_reader.RunHistoryWaitForEvent{
			// EventName:  item.WaitForEvent.EventName,
			// Expression: item.WaitForEvent.Expression,
			// Timeout:    item.WaitForEvent.Timeout,
		}
	}

	var waitResult *history_reader.RunHistoryWaitResult
	if item.WaitResult.Valid {
		waitResult = &history_reader.RunHistoryWaitResult{
			// EventID: item.WaitResult.EventID,
			// Timeout: item.WaitResult.Timeout,
		}
	}

	var invokeFunction *history_reader.RunHistoryInvokeFunction
	if item.InvokeFunction.Valid {
		invokeFunction = &history_reader.RunHistoryInvokeFunction{
			// CorrelationID: item.InvokeFunction.CorrelationID,
			// EventID:       item.InvokeFunction.EventID,
			// FunctionID:    item.InvokeFunction.FunctionID,
			// Timeout:       item.InvokeFunction.Timeout,
		}
	}

	var invokeFunctionResult *history_reader.RunHistoryInvokeFunctionResult
	if item.InvokeFunctionResult.Valid {
		invokeFunctionResult = &history_reader.RunHistoryInvokeFunctionResult{
			// EventID: item.InvokeFunctionResult.EventID,
			// RunID:   item.InvokeFunctionResult.RunID,
			// Timeout: item.InvokeFunctionResult.Timeout,
		}
	}

	return &history_reader.RunHistory{
		Attempt:         item.Attempt,
		Cancel:          cancel,
		CreatedAt:       item.CreatedAt,
		FunctionVersion: item.FunctionVersion,
		// GroupID:              item.GroupID,
		ID:                   item.ID,
		InvokeFunction:       invokeFunction,
		InvokeFunctionResult: invokeFunctionResult,
		Result:               result,
		RunID:                item.RunID,
		Sleep:                sleep,
		// StepName:             item.StepName,
		// StepType:             item.StepType,
		Type: historyType,
		// URL:                  item.URL,
		WaitForEvent: waitForEvent,
		WaitResult:   waitResult,
	}, nil
}
