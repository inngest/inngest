package devserver

import (
	"context"

	"github.com/inngest/inngest-cli/pkg/execution/driver"
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

func (l loggingExecutor) Execute(ctx context.Context, id state.Identifier, from string) (*driver.Response, error) {
	l.log.Info().
		Str("run_id", id.RunID.String()).
		Str("step", from).
		Msg("executing step")

	resp, err := l.Executor.Execute(ctx, id, from)

	if err == nil {
		l.log.Info().
			Str("run_id", id.RunID.String()).
			Str("step", from).
			Interface("response", resp).
			Msg("executed step")
	} else {
		retryable := false
		if resp != nil {
			retryable = resp.Retryable()
		}

		l.log.Info().
			Str("run_id", id.RunID.String()).
			Str("step", from).
			Err(err).
			Interface("response", resp).
			Bool("retryable", retryable).
			Msg("error executing step")
	}

	return resp, err
}
