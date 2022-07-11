package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/pkg/backoff"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/coredata"
	"github.com/inngest/inngest-cli/pkg/execution/driver"
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

	// Create drivers based off of the available config
	var drivers = []driver.Driver{}
	for _, driverConfig := range s.config.Execution.Drivers {
		d, err := driverConfig.NewDriver()
		if err != nil {
			return err
		}
		drivers = append(drivers, d)
	}

	// XXX: Configure executor & drivers via config.
	s.exec, err = NewExecutor(
		WithActionLoader(s.data),
		WithStateManager(s.state),
		WithRuntimeDrivers(
			drivers...,
		),
		WithLogger(logger.From(ctx)),
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *svc) Run(ctx context.Context) error {
	logger.From(ctx).Info().Msg("subscribing to function queue")
	return s.queue.Run(ctx, func(ctx context.Context, item queue.Item) error {
		// Don't stop the service on errors.
		_ = s.handleQueueItem(ctx, item)
		return nil
	})
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

	l.Info().Interface("edge", edge).Msg("processing step")

	resp, err := s.exec.Execute(ctx, item.Identifier, edge.Incoming, item.ErrorCount)
	if err != nil {
		// The executor usually returns a state.DriverResponse if the step's
		// response was an error.  In this case, the executor itself handles
		// whether the step has been retried the max amount of times, as the
		// executor has the workflow & step config.
		//
		// Accordingly, we check if the driver's response is retryable here;
		// this will let us know whether we can re-enqueue.
		//
		// If the error is not of type response error, we assume the step is
		// always retryable.
		_, isResponseError := err.(*state.DriverResponse)
		if (resp != nil && resp.Retryable()) || !isResponseError {
			next := item
			next.ErrorCount += 1
			at := backoff.LinearJitterBackoff(next.ErrorCount)
			l.Info().Interface("edge", next).Time("at", at).Msg("enqueueing retry")
			if err := s.queue.Enqueue(ctx, next, at); err != nil {
				return err
			}
			return nil
		}

		// This is a non-retryable error.  Finalize this step.
		l.Warn().Interface("edge", edge).Msg("step permanently failed")
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
