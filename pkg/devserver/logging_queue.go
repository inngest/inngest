package devserver

import (
	"context"
	"time"

	"github.com/inngest/inngestctl/pkg/execution/state"
	"github.com/inngest/inngestctl/pkg/execution/state/inmemory"
	"github.com/rs/zerolog"
)

func NewLoggingQueue(l *zerolog.Logger) inmemory.Queue {
	log := l.With().Str("caller", "state").Logger()

	sm := inmemory.NewStateManager()

	return loggingQueue{
		log:   &log,
		Queue: sm,
	}
}

type loggingQueue struct {
	inmemory.Queue

	log *zerolog.Logger
}

func (l loggingQueue) Enqueue(item inmemory.QueueItem, at time.Time) {
	l.log.Info().
		Str("run_id", item.ID.RunID.String()).
		Str("step", item.Edge.Incoming).
		Interface("edge", item.Edge).
		Interface("error_count", item.ErrorCount).
		Interface("at", at).
		Msg("enqueueing step")
	l.Queue.Enqueue(item, at)
}

func (l loggingQueue) SaveActionOutput(ctx context.Context, i state.Identifier, actionID string, data map[string]interface{}) (state.State, error) {
	l.log.Info().
		Str("run_id", i.RunID.String()).
		Str("step", actionID).
		Interface("data", data).
		Msg("recording step output")

	return l.Queue.SaveActionOutput(ctx, i, actionID, data)
}

func (l loggingQueue) SaveActionError(ctx context.Context, i state.Identifier, actionID string, err error) (state.State, error) {
	l.log.Warn().
		Str("run_id", i.RunID.String()).
		Str("step", actionID).
		Err(err).
		Msg("recording step error")

	return l.Queue.SaveActionError(ctx, i, actionID, err)
}
