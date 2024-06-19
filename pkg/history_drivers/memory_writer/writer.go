package memory_writer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/history_drivers/memory_store"
	"github.com/inngest/inngest/pkg/inngest/log"
)

func NewWriter(ctx context.Context) history.Driver {
	if len(memory_store.Singleton.Data) == 0 {
		l := log.From(ctx).With().Str("caller", "memory_writer").Logger()

		// read data from file and populate memory_store.Singleton
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", consts.DevServerTempDir, consts.DevServerHistoryFile))
		if err != nil {
			if os.IsNotExist(err) {
				goto end
			}
			l.Error().Err(err).Msg("failed to read history file")
		}

		err = json.Unmarshal(file, &memory_store.Singleton.Data)
		if err != nil {
			l.Error().Err(err).Msg("failed to unmarshal history file")
		}

		humanSize := fmt.Sprintf("%.2fKB", float64(len(file))/1024)
		l.Info().Str("size", humanSize).Msg("imported history snapshot")
	}

end:
	return &writer{
		store: memory_store.Singleton,
	}
}

type writer struct {
	store *memory_store.RunStore
}

func (w *writer) Close(ctx context.Context) error {
	l := log.From(ctx).With().Str("caller", "memory_writer").Logger()

	b, err := json.Marshal(w.store.Data)
	if err != nil {
		l.Error().Err(err).Msg("error marshalling history data for export")
		return err
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s", consts.DevServerTempDir, consts.DevServerHistoryFile), b, 0644)
	if err != nil {
		l.Error().Err(err).Msg("error writing history data to file")
		return err
	}

	humanSize := fmt.Sprintf("%.2fKB", float64(len(b))/1024)
	l.Info().Str("size", humanSize).Msg("exported history snapshot")

	return nil
}

func (w *writer) Write(
	ctx context.Context,
	item history.History,
) error {
	w.store.Mu.Lock()
	defer w.store.Mu.Unlock()

	if item.Type == enums.HistoryTypeFunctionScheduled.String() ||
		item.Type == enums.HistoryTypeFunctionStarted.String() {
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

	if item.Result != nil {
		run.Run.Output = &item.Result.Output
	}

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
	run.Run.Cron = item.Cron
	run.Run.EventID = item.EventID
	run.Run.ID = item.RunID
	run.Run.OriginalRunID = item.OriginalRunID

	if item.Result != nil {
		run.Run.Output = &item.Result.Output
	}

	var status enums.RunStatus
	switch item.Type {
	case enums.HistoryTypeFunctionScheduled.String():
		status = enums.RunStatusScheduled
	case enums.HistoryTypeFunctionStarted.String():
		status = enums.RunStatusRunning
	default:
		log.From(ctx).Error().Str("type", item.Type).
			Msg("unknown history type")
	}

	run.Run.StartedAt = time.UnixMilli(int64(item.RunID.Time()))
	run.Run.Status = status
	run.Run.WorkflowID = item.FunctionID
	run.Run.WorkspaceID = item.WorkspaceID
	run.Run.WorkflowVersion = int(item.FunctionVersion)
	w.store.Data[item.RunID] = run
}

func timePtr(t time.Time) *time.Time {
	return &t
}
