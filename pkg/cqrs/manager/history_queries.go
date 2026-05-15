package manager

import (
	"context"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/history_reader"
	"github.com/oklog/ulid/v2"
)

func (w wrapper) CountRuns(ctx context.Context, opts cqrs.CountRunOpts) (int, error) {
	return (&reader{q: w.q}).CountRuns(ctx, opts)
}

func (w wrapper) CountReplayRuns(ctx context.Context, opts cqrs.CountReplayRunsOpts) (cqrs.ReplayRunCounts, error) {
	return (&reader{q: w.q}).CountReplayRuns(ctx, opts)
}

func (w wrapper) GetHistoryRun(ctx context.Context, runID ulid.ULID, opts cqrs.GetRunOpts) (cqrs.Run, error) {
	return (&reader{q: w.q}).GetRun(ctx, runID, opts)
}

func (w wrapper) GetHistoryRuns(ctx context.Context, opts cqrs.GetRunsOpts) ([]cqrs.Run, error) {
	return (&reader{q: w.q}).GetRuns(ctx, opts)
}

func (w wrapper) GetReplayRuns(ctx context.Context, opts cqrs.GetReplayRunsOpts) ([]cqrs.ReplayRun, error) {
	return (&reader{q: w.q}).GetReplayRuns(ctx, opts)
}

func (w wrapper) GetRunHistory(ctx context.Context, runID ulid.ULID, opts cqrs.GetRunOpts) ([]*cqrs.RunHistory, error) {
	items, err := (&reader{q: w.q}).GetRunHistory(ctx, runID, opts)
	if err != nil {
		return nil, err
	}
	return toCQRSRunHistory(items), nil
}

func (w wrapper) GetRunHistoryItemOutput(ctx context.Context, historyID ulid.ULID, opts cqrs.GetHistoryOutputOpts) (*string, error) {
	return (&reader{q: w.q}).GetRunHistoryItemOutput(ctx, historyID, opts)
}

func (w wrapper) GetRunsByEventID(ctx context.Context, eventID ulid.ULID, opts cqrs.GetRunsByEventIDOpts) ([]cqrs.Run, error) {
	return (&reader{q: w.q}).GetRunsByEventID(ctx, eventID, opts)
}

func (w wrapper) GetSkippedRunsByEventID(ctx context.Context, eventID ulid.ULID, opts cqrs.GetRunsByEventIDOpts) ([]cqrs.SkippedRun, error) {
	return (&reader{q: w.q}).GetSkippedRunsByEventID(ctx, eventID, opts)
}

func (w wrapper) GetUsage(ctx context.Context, opts cqrs.GetUsageOpts) ([]cqrs.HistoryUsage, error) {
	return (&reader{q: w.q}).GetUsage(ctx, opts)
}

func (w wrapper) GetActiveRunIDs(ctx context.Context, opts cqrs.GetActiveRunIDsOpts) ([]ulid.ULID, error) {
	return (&reader{q: w.q}).GetActiveRunIDs(ctx, opts)
}

func (w wrapper) CountActiveRuns(ctx context.Context, opts cqrs.CountActiveRunsOpts) (int, error) {
	return (&reader{q: w.q}).CountActiveRuns(ctx, opts)
}

func toCQRSRunHistory(items []*history_reader.RunHistory) []*cqrs.RunHistory {
	result := make([]*cqrs.RunHistory, 0, len(items))
	for _, item := range items {
		if item == nil {
			result = append(result, nil)
			continue
		}

		result = append(result, &cqrs.RunHistory{
			Attempt:              item.Attempt,
			Cancel:               toCQRSRunHistoryCancel(item.Cancel),
			CreatedAt:            item.CreatedAt,
			FunctionVersion:      item.FunctionVersion,
			GroupID:              item.GroupID,
			ID:                   item.ID,
			InvokeFunction:       toCQRSRunHistoryInvokeFunction(item.InvokeFunction),
			InvokeFunctionResult: toCQRSRunHistoryInvokeFunctionResult(item.InvokeFunctionResult),
			Result:               toCQRSRunHistoryResult(item.Result),
			RunID:                item.RunID,
			Sleep:                toCQRSRunHistorySleep(item.Sleep),
			StepName:             item.StepName,
			StepType:             item.StepType,
			Type:                 item.Type,
			URL:                  item.URL,
			WaitForEvent:         toCQRSRunHistoryWaitForEvent(item.WaitForEvent),
			WaitResult:           toCQRSRunHistoryWaitResult(item.WaitResult),
		})
	}
	return result
}

func toCQRSRunHistoryCancel(item *history_reader.RunHistoryCancel) *cqrs.RunHistoryCancel {
	if item == nil {
		return nil
	}
	return &cqrs.RunHistoryCancel{
		EventID:    item.EventID,
		Expression: item.Expression,
		UserID:     item.UserID,
	}
}

func toCQRSRunHistoryResult(item *history_reader.RunHistoryResult) *cqrs.RunHistoryResult {
	if item == nil {
		return nil
	}
	return &cqrs.RunHistoryResult{
		DurationMS:  item.DurationMS,
		ErrorCode:   item.ErrorCode,
		Framework:   item.Framework,
		Platform:    item.Platform,
		SDKLanguage: item.SDKLanguage,
		SDKVersion:  item.SDKVersion,
		SizeBytes:   item.SizeBytes,
	}
}

func toCQRSRunHistorySleep(item *history_reader.RunHistorySleep) *cqrs.RunHistorySleep {
	if item == nil {
		return nil
	}
	return &cqrs.RunHistorySleep{Until: item.Until}
}

func toCQRSRunHistoryWaitForEvent(item *history_reader.RunHistoryWaitForEvent) *cqrs.RunHistoryWaitForEvent {
	if item == nil {
		return nil
	}
	return &cqrs.RunHistoryWaitForEvent{
		EventName:  item.EventName,
		Expression: item.Expression,
		Timeout:    item.Timeout,
	}
}

func toCQRSRunHistoryWaitResult(item *history_reader.RunHistoryWaitResult) *cqrs.RunHistoryWaitResult {
	if item == nil {
		return nil
	}
	return &cqrs.RunHistoryWaitResult{
		EventID: item.EventID,
		Timeout: item.Timeout,
	}
}

func toCQRSRunHistoryInvokeFunction(item *history_reader.RunHistoryInvokeFunction) *cqrs.RunHistoryInvokeFunction {
	if item == nil {
		return nil
	}
	return &cqrs.RunHistoryInvokeFunction{
		CorrelationID: item.CorrelationID,
		EventID:       item.EventID,
		FunctionID:    item.FunctionID,
		Timeout:       item.Timeout,
	}
}

func toCQRSRunHistoryInvokeFunctionResult(item *history_reader.RunHistoryInvokeFunctionResult) *cqrs.RunHistoryInvokeFunctionResult {
	if item == nil {
		return nil
	}
	return &cqrs.RunHistoryInvokeFunctionResult{
		EventID: item.EventID,
		RunID:   item.RunID,
		Timeout: item.Timeout,
	}
}
