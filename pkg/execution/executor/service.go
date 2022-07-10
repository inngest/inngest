package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/pkg/backoff"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/coredata"
	"github.com/inngest/inngest-cli/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest-cli/pkg/execution/queue"
	"github.com/inngest/inngest-cli/pkg/execution/queue/queuefactory"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/statefactory"
	"github.com/inngest/inngest-cli/pkg/logger"
	"github.com/inngest/inngest-cli/pkg/service"
	"github.com/xhit/go-str2duration/v2"
)

type Opt func(s *svc)

func WithExecutionLoader(l coredata.ExecutionLoader) func(s *svc) {
	return func(s *svc) {
		s.data = l
	}
}

func NewService(c config.Config, opts ...Opt) service.Service {
	svc := &svc{config: c}
	for _, o := range opts {
		o(svc)
	}
	return svc
}

type svc struct {
	config config.Config
	// data provides the ability to load action versions when running steps.
	data coredata.ExecutionLoader
	// state allows us to record step results
	state state.Manager
	// queue allows us to enqueue next steps.
	queue queue.Queue
	// exec runs the specific actions.
	exec Executor
}

func (s *svc) Name() string {
	return "executor"
}

func (s *svc) Pre(ctx context.Context) error {
	var err error

	if s.data == nil {
		s.data, err = coredata.NewFSLoader(ctx, ".")
		if err != nil {
			return err
		}
	}

	s.state, err = statefactory.NewState(ctx, s.config.State)
	if err != nil {
		return err
	}

	s.queue, err = queuefactory.NewQueue(ctx, s.config.Queue)
	if err != nil {
		return err
	}

	// Create our drivers.
	dd, err := dockerdriver.New()
	if err != nil {
		return err
	}

	// TODO: Configure executor & drivers via config.
	s.exec, err = NewExecutor(
		WithActionLoader(s.data),
		WithStateManager(s.state),
		WithRuntimeDrivers(
			dd,
		),
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *svc) Run(ctx context.Context) error {
	logger.From(ctx).Info().Msg("subscribing to function queue")
	return s.queue.Run(ctx, s.handleQueueItem)
}

func (s *svc) Stop(ctx context.Context) error {
	return nil
}

func (s *svc) handleQueueItem(ctx context.Context, item queue.Item) error {
	l := logger.From(ctx).With().Str("run_id", item.Identifier.RunID.String()).Logger()

	edge, err := queue.GetEdge(item)
	if err != nil {
		return err
	}

	l.Info().Interface("edge", edge).Msg("dequeueing step")

	resp, err := s.exec.Execute(ctx, item.Identifier, edge.Incoming, item.ErrorCount)
	if err != nil {
		l.Error().Err(err).Msg("error executing step")

		// If the error is not of type response error, we can assume that this is
		// always retryable.
		_, isResponseError := err.(*state.DriverResponse)
		if (resp != nil && resp.Retryable()) || !isResponseError {
			next := item
			next.ErrorCount += 1
			at := backoff.LinearJitterBackoff(next.ErrorCount)
			if err := s.queue.Enqueue(ctx, next, at); err != nil {
				return err
			}
		}

		// This is a non-retryable error.  Finalize this step.
		if err := s.state.Finalized(ctx, item.Identifier, edge.Incoming); err != nil {
			return err
		}
		return fmt.Errorf("execution error: %s", err)
	}

	run, err := s.state.Load(ctx, item.Identifier)
	if err != nil {
		return err
	}

	children, err := state.DefaultEdgeEvaluator.AvailableChildren(ctx, run, edge.Incoming)
	if err != nil {
		return err
	}

	for _, next := range children {
		// We want to wait for another event to come in to traverse this edge within the DAG.
		//
		// Create a new "pause", which informs the state manager that we're pausing the traversal
		// of this edge until later.
		//
		// The runner should load all pauses and automatically resume the traversal when a
		// matching event is received.
		if next.Metadata != nil && next.Metadata.AsyncEdgeMetadata != nil {
			am := next.Metadata.AsyncEdgeMetadata
			if am.Event == "" {
				return fmt.Errorf("no async edge event specified")
			}
			dur, err := str2duration.ParseDuration(am.TTL)
			if err != nil {
				return fmt.Errorf("error parsing async edge ttl '%s': %w", am.TTL, err)
			}

			err = s.state.SavePause(ctx, state.Pause{
				ID:         uuid.New(),
				Identifier: run.Identifier(),
				Outgoing:   next.Outgoing,
				Incoming:   next.Incoming,
				Expires:    time.Now().Add(dur),
				Event:      &am.Event,
				Expression: am.Match,
			})
			if err != nil {
				return fmt.Errorf("error saving edge pause: %w", err)
			}
			continue
		}

		at := time.Now()
		if next.Metadata != nil && next.Metadata.Wait != nil {
			dur, err := str2duration.ParseDuration(*next.Metadata.Wait)
			if err != nil {
				return fmt.Errorf("invalid wait duration: %s", *next.Metadata.Wait)
			}
			at = at.Add(dur)
		}

		l.Info().Str("outgoing", next.Outgoing).Time("at", at).Msg("scheduling next step")
		// Enqueue the next child in our in-memory state queue.
		if err := s.queue.Enqueue(ctx, queue.Item{
			Identifier: item.Identifier,
			Payload:    queue.PayloadEdge{Edge: next},
		}, at); err != nil {
			return err
		}
	}

	// Mark this step as finalized.
	//
	// This must happen after everything is enqueued, else the scheduled <> finalized count
	// is out of order.
	if err := s.state.Finalized(ctx, item.Identifier, edge.Incoming); err != nil {
		return err
	}

	l.Info().Interface("edge", edge).Msg("step complete")

	return nil
}
