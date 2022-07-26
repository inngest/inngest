package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coredata"
	inmemorydatastore "github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
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

	wg sync.WaitGroup
}

func (s *svc) Name() string {
	return "executor"
}

func (s *svc) Pre(ctx context.Context) error {
	var err error

	if s.data == nil {
		s.data, err = inmemorydatastore.NewFSLoader(ctx, ".")
		if err != nil {
			return err
		}
	}

	s.state, err = s.config.State.Service.Concrete.Manager(ctx)
	if err != nil {
		return err
	}

	logger.From(ctx).Info().Str("backend", s.config.Queue.Service.Backend).Msg("starting queue")
	s.queue, err = s.config.Queue.Service.Concrete.Queue()
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

	s.exec, err = NewExecutor(
		WithActionLoader(s.data),
		WithStateManager(s.state),
		WithRuntimeDrivers(
			drivers...,
		),
		WithLogger(logger.From(ctx)),
		WithConfig(s.config.Execution),
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
		s.wg.Add(1)
		defer s.wg.Done()

		var err error
		switch item.Kind {
		case queue.KindEdge:
			err = s.handleQueueItem(ctx, item)
		case queue.KindPause:
			err = s.handlePauseTimeout(ctx, item)
		default:
			err = fmt.Errorf("unknown payload type: %T", item.Payload)
		}

		if err != nil {
			logger.From(ctx).Error().Err(err).Interface("item", item).Msg("critical error handling queue item")
		}

		return err
	})
}

func (s *svc) Stop(ctx context.Context) error {
	// Wait for all in-flight queue runs to finish
	s.wg.Wait()
	return nil
}

func (s *svc) handleQueueItem(ctx context.Context, item queue.Item) error {
	l := logger.From(ctx).With().Str("run_id", item.Identifier.RunID.String()).Logger()

	edge, err := queue.GetEdge(item)
	if err != nil {
		return err
	}

	l.Info().Interface("edge", edge).Msg("processing step")

	_, err = s.exec.Execute(ctx, item.Identifier, edge.Incoming, item.ErrorCount)
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
		retry, isRetryable := err.(state.Retryable)
		if (isRetryable && retry.Retryable()) || !isRetryable {
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
		return nil
	}

	run, err := s.state.Load(ctx, item.Identifier)
	if err != nil {
		return err
	}

	children, err := state.DefaultEdgeEvaluator.AvailableChildren(ctx, run, edge.Incoming)
	if err != nil {
		return err
	}

	l.Trace().Int("len", len(children)).Msg("evaluated children")

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

			// This should also increase the waitgroup count, as we have an
			// edge that is outstanding.
			if err := s.state.Scheduled(ctx, item.Identifier, next.Incoming); err != nil {
				return err
			}

			pauseID := uuid.New()
			expires := time.Now().Add(dur)
			err = s.state.SavePause(ctx, state.Pause{
				ID:         pauseID,
				Identifier: run.Identifier(),
				Outgoing:   next.Outgoing,
				Incoming:   next.Incoming,
				Expires:    expires,
				Event:      &am.Event,
				Expression: am.Match,
				OnTimeout:  am.OnTimeout,
			})
			if err != nil {
				return fmt.Errorf("error saving edge pause: %w", err)
			}

			// Enqueue a timeout.  This will be handled within our queue;  if the
			// pause still exists at this time and has not been comsumed we will
			// continue traversing this edge if OnTimeout is true.
			if err := s.queue.Enqueue(ctx, queue.Item{
				Kind:       queue.KindPause,
				Identifier: item.Identifier,
				Payload: queue.PayloadPauseTimeout{
					PauseID:   pauseID,
					OnTimeout: am.OnTimeout,
				},
			}, expires); err != nil {
				return err
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

		// Enqueue the next child in our queue.
		if err := s.queue.Enqueue(ctx, queue.Item{
			Kind:       queue.KindEdge,
			Identifier: item.Identifier,
			Payload:    queue.PayloadEdge{Edge: next},
		}, at); err != nil {
			return err
		}

		// Increase the waitgroup counter.
		// Unfortunately, the backing queue and the state store may be different
		// backing services.  Therefore, we can never guarantee that enqueueing an
		// item increases the scheduled count.
		//
		// Hopefully, if the backing implementation is the same (eg. a database which
		// hosts the queue and the state store), Enqueue increases the pending count
		// and this is a no-op - things should be atomic where possible.
		//
		// TODO: Add a unit test to ensure WG is 0 at the end of execution.
		if err := s.state.Scheduled(ctx, item.Identifier, next.Outgoing); err != nil {
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

func (s *svc) handlePauseTimeout(ctx context.Context, item queue.Item) error {
	l := logger.From(ctx).With().Str("run_id", item.Identifier.RunID.String()).Logger()

	pauseTimeout, ok := item.Payload.(queue.PayloadPauseTimeout)
	if !ok {
		return fmt.Errorf("unable to get pause timeout form queue item: %T", item.Payload)
	}

	pause, err := s.state.PauseByID(ctx, pauseTimeout.PauseID)
	if err == state.ErrPauseNotFound {
		// This pause has been consumed.
		l.Debug().Interface("pause", pauseTimeout).Msg("consumed pause timeout ignored")
		return nil
	}
	if err != nil {
		return err
	}
	if pause == nil || pause.LeasedUntil != nil {
		return nil
	}

	if err := s.state.ConsumePause(ctx, pause.ID); err != nil {
		return fmt.Errorf("error consuming timeout pause: %w", err)
	}

	if pauseTimeout.OnTimeout {
		l.Info().Interface("pause", pauseTimeout).Interface("edge", pause.Edge()).Msg("scheduling pause timeout step")
		// Enqueue the next job to run.  We could handle this in the
		// same thread, but its safer to enable retries by re-enqueueing.
		if err := s.queue.Enqueue(ctx, queue.Item{
			Kind:       queue.KindEdge,
			Identifier: item.Identifier,
			Payload:    queue.PayloadEdge{Edge: pause.Edge()},
		}, time.Now()); err != nil {
			return fmt.Errorf("error enqueueing timeout step: %w", err)
		}
	} else {
		l.Info().Interface("pause", pauseTimeout).Interface("edge", pause.Edge()).Msg("ignoring pause timeout")
		// Finalize this action without it running.
		if err := s.state.Finalized(ctx, item.Identifier, pause.Edge().Incoming); err != nil {
			return err
		}
	}

	return nil
}
