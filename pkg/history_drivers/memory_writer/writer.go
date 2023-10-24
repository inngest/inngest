package memory_writer

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/history_drivers/memory_store"
	"github.com/inngest/inngest/pkg/inngest/log"
)

func NewWriter() history.Driver {
	return &writer{
		store: memory_store.Singleton,
	}
}

type writer struct {
	store *memory_store.RunStore
}

func (w *writer) Close() error {
	return nil
}

func (w *writer) Write(
	ctx context.Context,
	item history.History,
) error {
	w.store.Mu.Lock()
	defer w.store.Mu.Unlock()

	if item.Type == enums.HistoryTypeFunctionStarted.String() {
		w.writeWorkflowStart(ctx, item)
	} else if item.Type == enums.HistoryTypeFunctionCancelled.String() ||
		item.Type == enums.HistoryTypeFunctionCompleted.String() ||
		item.Type == enums.HistoryTypeFunctionFailed.String() {
		w.writeWorkflowEnd(ctx, item)
	}

	w.writeHistory(ctx, item)
	return nil
}

func (w *writer) writeHistory(
	ctx context.Context,
	item history.History,
) {
	run := w.store.Data[item.RunID]
	run.History = append(run.History, item)
	w.store.Data[item.RunID] = run
}

func (w *writer) writeWorkflowEnd(
	ctx context.Context,
	item history.History,
) {
	var status enums.RunStatus
	switch item.Type {
	case enums.HistoryTypeFunctionCancelled.String():
		status = enums.RunStatusCancelled
	case enums.HistoryTypeFunctionCompleted.String():
		status = enums.RunStatusCompleted
	case enums.HistoryTypeFunctionFailed.String():
		status = enums.RunStatusFailed
	default:
		log.From(ctx).Error().Str("type", item.Type).
			Msg("unknown history type")
	}

	run := w.store.Data[item.RunID]
	run.Run.EndedAt = timePtr(time.Now())
	run.Run.Status = status
	w.store.Data[item.RunID] = run
}

func (w *writer) writeWorkflowStart(
	ctx context.Context,
	item history.History,
) {
	run := w.store.Data[item.RunID]
	run.Run.AccountID = item.AccountID
	run.Run.BatchID = item.BatchID
	run.Run.EventID = item.EventID
	run.Run.ID = item.RunID
	run.Run.OriginalRunID = item.OriginalRunID
	run.Run.StartedAt = time.UnixMilli(int64(item.RunID.Time()))
	run.Run.Status = enums.RunStatusRunning
	run.Run.WorkflowID = item.FunctionID
	run.Run.WorkspaceID = item.WorkspaceID
	run.Run.WorkflowVersion = int(item.FunctionVersion)
	w.store.Data[item.RunID] = run
}

func timePtr(t time.Time) *time.Time {
	return &t
}
