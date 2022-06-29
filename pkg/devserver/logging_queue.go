package devserver

import (
	"context"
	"time"

	"github.com/inngest/inngest-cli/pkg/execution/queue"
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

func (l loggingQueue) Enqueue(ctx context.Context, item queue.Item, at time.Time) error {
	l.log.Info().
		Str("run_id", item.Identifier.RunID.String()).
		Interface("payload", item.Payload).
		Interface("error_count", item.ErrorCount).
		Interface("at", at).
		Msg("enqueueing step")
	return l.Queue.Enqueue(ctx, item, at)
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
