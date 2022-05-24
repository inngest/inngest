package devserver

import (
	"context"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/rs/zerolog"
)

func NewLoggingExecutor(e executor.Executor, l *zerolog.Logger) executor.Executor {
	log := l.With().Str("caller", "executor").Logger()
	return loggingExecutor{Executor: e, log: &log}
}

type loggingExecutor struct {
	executor.Executor

	log *zerolog.Logger
}

func (l loggingExecutor) Execute(ctx context.Context, id state.Identifier, from string) ([]inngest.Edge, error) {
	l.log.Info().
		Str("run_id", id.RunID.String()).
		Str("step", from).
		Msg("executing step")

	edges, err := l.Executor.Execute(ctx, id, from)

	l.log.Info().
		Str("run_id", id.RunID.String()).
		Str("step", from).
		Msg("executed step")

	return edges, err
}
