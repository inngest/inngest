package devserver

import (
	"context"
	"time"

	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
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

func (l loggingQueue) SaveResponse(ctx context.Context, i state.Identifier, r state.DriverResponse, attempt int) (state.State, error) {
	l.log.Info().
		Str("run_id", i.RunID.String()).
		Str("step", r.Step.ID).
		Int("attepmt", attempt).
		Interface("response", r.Output).
		Err(r.Err).
		Msg("recording step response")

	return l.Queue.SaveResponse(ctx, i, r, attempt)
}
