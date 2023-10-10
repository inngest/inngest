package memory_reader

import (
	"context"
	"errors"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/history_drivers/memory_store"
	"github.com/inngest/inngest/pkg/history_reader"
	"github.com/inngest/inngest/pkg/usage"
	"github.com/oklog/ulid/v2"
)

func NewReader() *reader {
	return &reader{
		store: memory_store.Singleton,
	}
}

type reader struct {
	store *memory_store.RunStore
}

func (r *reader) CountRuns(
	ctx context.Context,
	opts history_reader.CountRunOpts,
) (int, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()

	return len(r.store.Data), nil
}

func (r *reader) GetRun(
	ctx context.Context,
	runID ulid.ULID,
	opts history_reader.GetRunOpts,
) (history_reader.Run, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()

	run, ok := r.store.Data[runID]
	if !ok {
		return history_reader.Run{}, history_reader.ErrNotFound
	}

	return run.Run, nil
}

func (r *reader) GetRunHistory(
	ctx context.Context,
	runID ulid.ULID,
	opts history_reader.GetRunOpts,
) ([]*history_reader.RunHistory, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()

	run, ok := r.store.Data[runID]
	if !ok {
		return nil, history_reader.ErrNotFound
	}

	var items []*history_reader.RunHistory
	for _, item := range run.History {
		historyItem, err := toRunHistory(item)
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
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()

	if err := opts.Validate(); err != nil {
		return nil, err
	}

	run, ok := r.store.Data[opts.RunID]
	if !ok {
		return nil, history_reader.ErrNotFound
	}

	for _, item := range run.History {
		if item.ID == historyID {
			if item.Result == nil {
				return nil, nil
			}

			return &item.Result.Output, nil
		}
	}

	return nil, history_reader.ErrNotFound
}

func (r *reader) GetRuns(
	ctx context.Context,
	opts history_reader.GetRunsOpts,
) ([]history_reader.Run, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()

	var runs []history_reader.Run
	for _, run := range r.store.Data {
		runs = append(runs, run.Run)
	}

	return runs, nil
}

func (r *reader) GetRunsByEventID(
	ctx context.Context,
	eventID ulid.ULID,
	opts history_reader.GetRunsByEventIDOpts,
) ([]history_reader.Run, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()

	var runs []history_reader.Run
	for _, run := range r.store.Data {
		if run.Run.EventID == eventID {
			runs = append(runs, run.Run)
		}
	}

	return runs, nil
}

func (r *reader) GetUsage(
	ctx context.Context,
	opts history_reader.GetUsageOpts,
) ([]usage.UsageSlot, error) {
	r.store.Mu.RLock()
	defer r.store.Mu.RUnlock()

	return nil, errors.New("not implemented")
}

func toRunHistory(item history.History) (*history_reader.RunHistory, error) {
	var cancel *history_reader.RunHistoryCancel
	if item.Cancel != nil {
		cancel = &history_reader.RunHistoryCancel{
			EventID:    item.Cancel.EventID,
			Expression: item.Cancel.Expression,
			UserID:     item.Cancel.UserID,
		}
	}

	historyType, err := enums.HistoryTypeString(item.Type)
	if err != nil {
		return nil, fmt.Errorf("invalid history type: %w", err)
	}

	var result *history_reader.RunHistoryResult
	if item.Result != nil {
		result = &history_reader.RunHistoryResult{
			// TODO
		}
	}

	var sleep *history_reader.RunHistorySleep
	if item.Sleep != nil {
		sleep = &history_reader.RunHistorySleep{
			Until: item.Sleep.Until,
		}
	}

	var waitForEvent *history_reader.RunHistoryWaitForEvent
	if item.WaitForEvent != nil {
		waitForEvent = &history_reader.RunHistoryWaitForEvent{
			EventName:  item.WaitForEvent.EventName,
			Expression: item.WaitForEvent.Expression,
			Timeout:    item.WaitForEvent.Timeout,
		}
	}

	var waitResult *history_reader.RunHistoryWaitResult
	if item.WaitResult != nil {
		waitResult = &history_reader.RunHistoryWaitResult{
			EventID: item.WaitResult.EventID,
			Timeout: item.WaitResult.Timeout,
		}
	}

	return &history_reader.RunHistory{
		Attempt:         item.Attempt,
		Cancel:          cancel,
		CreatedAt:       item.CreatedAt,
		FunctionVersion: item.FunctionVersion,
		GroupID:         item.GroupID,
		ID:              item.ID,
		Result:          result,
		RunID:           item.RunID,
		Sleep:           sleep,
		StepName:        item.StepName,
		Type:            historyType,
		URL:             item.URL,
		WaitForEvent:    waitForEvent,
		WaitResult:      waitResult,
	}, nil
}
