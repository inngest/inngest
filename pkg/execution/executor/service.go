package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/service"
	"github.com/oklog/ulid/v2"
	"github.com/xhit/go-str2duration/v2"
)

type Opt func(s *svc)

func WithExecutionLoader(l cqrs.ExecutionLoader) func(s *svc) {
	return func(s *svc) {
		s.data = l
	}
}

func WithState(sm state.Manager) func(s *svc) {
	return func(s *svc) {
		s.state = sm
	}
}

func WithServiceExecutor(exec Executor) func(s *svc) {
	return func(s *svc) {
		s.exec = exec
	}
}

func WithExecutorOpts(opts ...ExecutorOpt) func(s *svc) {
	return func(s *svc) {
		s.opts = opts
	}
}

func WithServiceQueue(q queue.Queue) func(s *svc) {
	return func(s *svc) {
		s.queue = q
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
	data cqrs.ExecutionLoader
	// state allows us to record step results
	state state.Manager
	// queue allows us to enqueue next steps.
	queue queue.Queue
	// exec runs the specific actions.
	exec Executor

	wg sync.WaitGroup

	opts []ExecutorOpt
}

func (s *svc) Name() string {
	return "executor"
}

func (s *svc) Pre(ctx context.Context) error {
	var err error

	if s.state == nil {
		return fmt.Errorf("no state provided")
	}

	if s.queue == nil {
		return fmt.Errorf("no queue provided")
	}

	failureHandler, err := s.getFailureHandler(ctx)
	if err != nil {
		return fmt.Errorf("failed to create failure handler: %w", err)
	}
	s.exec.SetFailureHandler(failureHandler)

	return nil
}

func (s *svc) Executor() Executor {
	return s.exec
}

func (s *svc) getFailureHandler(ctx context.Context) (func(context.Context, state.Identifier, state.State, state.DriverResponse) error, error) {
	pb, err := pubsub.NewPublisher(ctx, s.config.EventStream.Service)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	topicName := s.config.EventStream.Service.Concrete.TopicName()

	return func(ctx context.Context, id state.Identifier, s state.State, r state.DriverResponse) error {
		now := time.Now()
		evt := event.Event{
			ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
			Name:      event.FnFailedName,
			Timestamp: now.UnixMilli(),
			Data: map[string]interface{}{
				"function_id": s.Function().Slug,
				"run_id":      id.RunID.String(),
				"error":       r.UserError(),
				"event":       s.Event(),
			},
		}

		byt, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("error marshalling failure event: %w", err)
		}

		err = pb.Publish(
			ctx,
			topicName,
			pubsub.Message{
				Name:      event.EventReceivedName,
				Data:      string(byt),
				Timestamp: now,
			},
		)
		if err != nil {
			return fmt.Errorf("error publishing failure event: %w", err)
		}

		return nil
	}, nil
}

func (s *svc) Run(ctx context.Context) error {
	logger.From(ctx).Info().Msg("subscribing to function queue")
	return s.queue.Run(ctx, func(ctx context.Context, item queue.Item) error {
		logger.From(ctx).Info().Interface("item", item).Msg("processing queue item")
		// Don't stop the service on errors.
		s.wg.Add(1)
		defer s.wg.Done()

		var err error
		switch item.Kind {
		case queue.KindEdge, queue.KindSleep:
			err = s.handleQueueItem(ctx, item)
		case queue.KindPause:
			err = s.handlePauseTimeout(ctx, item)
		default:
			err = fmt.Errorf("unknown payload type: %T", item.Payload)
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
	l := logger.From(ctx).With().
		Str("run_id", item.Identifier.RunID.String()).
		Int("attempt", item.Attempt).
		Logger()

	payload, err := queue.GetEdge(item)
	if err != nil {
		return fmt.Errorf("unable to get edge from queue item: %w", err)
	}
	edge := payload.Edge

	l.Info().Interface("payload", payload).Msg("processing step")

	// If this is of type sleep, ensure that we save "nil" within the state store
	// for the outgoing edge ID.  This ensures that we properly increase the stack
	// for `tools.sleep` within generator functions.
	var stackIdx int
	if item.Kind == queue.KindSleep {
		stackIdx, err = s.state.SaveResponse(ctx, item.Identifier, state.DriverResponse{
			Step: inngest.Step{ID: edge.Outgoing}, // XXX: Save edge name here.
		}, 0)
		if err != nil {
			return err
		}
	} else if edge.Outgoing != inngest.TriggerName {
		// Load the position within the stack for standard edges.
		stackIdx, err = s.state.StackIndex(ctx, item.Identifier.RunID, edge.Outgoing)
		if err != nil {
			return fmt.Errorf("unable to find stack index: %w", err)
		}
	}

	resp, _, err := s.exec.Execute(ctx, item.Identifier, edge, item.Attempt, stackIdx)

	// Check if the execution is cancelled, and if so finalize and terminate early.
	// This prevents steps from scheduling children.
	if err == state.ErrFunctionCancelled {
		_ = s.state.Finalized(ctx, item.Identifier, edge.Incoming, item.Attempt)
		return nil
	}

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
		if (resp == nil || resp.Retryable()) && queue.ShouldRetry(nil, item.Attempt, item.GetMaxAttempts()) {
			return err
		}

		// This is a non-retryable error.  Finalize this step.
		l.Warn().Interface("edge", edge).Err(err).Msg("step permanently failed")
		if err := s.state.Finalized(ctx, item.Identifier, edge.Incoming, item.Attempt); err != nil {
			return fmt.Errorf("unable to finalize step: %w", err)
		}
		return nil
	}

	// If this is a generator step, we need to re-invoke the current function.
	if resp != nil && resp.Generator != nil {
		// We're re-invoking the current step again.  Generator steps do not have
		// their own "step output" until the end of the function;  instead, each
		// sub-step within the generator yields a new output with its own step ID.
		//
		// We keep invoking Generator-based functions until they provide no more
		// yields, signalling they're done.
		err := s.exec.HandleGeneratorResponse(ctx, resp.Generator, item)
		if err != nil {
			return fmt.Errorf("unable to schedule generator response: %w", err)
		}
		// Finalize this step early, as we don't need to re-invoke anything else or
		// load children until generators complete.
		return s.state.Finalized(ctx, item.Identifier, edge.Incoming, item.Attempt)
	}

	run, err := s.state.Load(ctx, item.Identifier.RunID)
	if err != nil {
		return fmt.Errorf("unable to load run: %w", err)
	}

	children, err := state.DefaultEdgeEvaluator.AvailableChildren(ctx, run, edge.Incoming)
	if err != nil {
		return fmt.Errorf("unable to evaluate available children: %w", err)
	}

	l.Trace().Int("len", len(children)).Msg("evaluated children")

	for _, next := range children {
		retries := next.Step.RetryCount()
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
			//
			// XXX: Should this be a part of saving a pause?  Maybe we should
			// always increase scheduled count atomically here.
			if err := s.state.Scheduled(ctx, item.Identifier, next.Incoming, 0, nil); err != nil {
				return fmt.Errorf("unable to schedule async edge: %w", err)
			}

			l.Debug().Interface("edge", next).Msg("saving pause")
			pauseID := uuid.New()
			expires := time.Now().Add(dur)
			err = s.state.SavePause(ctx, state.Pause{
				ID:         pauseID,
				Identifier: run.Identifier(),
				Outgoing:   next.Outgoing,
				Incoming:   next.Incoming,
				Expires:    state.Time(expires),
				Event:      &am.Event,
				Expression: am.Match,
				OnTimeout:  am.OnTimeout,
			})
			if err != nil {
				return fmt.Errorf("error saving edge pause: %w", err)
			}

			l.Debug().Interface("edge", next).Msg("scheduling pause timeout")
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
				MaxAttempts: &retries,
			}, expires); err != nil {
				return fmt.Errorf("unable to enqueue pause timeout: %w", err)
			}
			continue
		}

		at := time.Now()
		if next.Metadata != nil && next.Metadata.Wait != nil {
			dur, err := ParseWait(ctx, *next.Metadata.Wait, run, edge.Incoming)
			if err != nil {
				return fmt.Errorf("unable to parse wait: %w", err)
			}
			at = at.Add(dur)
		}

		l.Debug().Str("outgoing", next.Outgoing).Time("at", at).Msg("scheduling next step")

		// Enqueue the next child in our queue.
		if err := s.queue.Enqueue(ctx, queue.Item{
			Kind:       queue.KindEdge,
			Identifier: item.Identifier,
			Payload: queue.PayloadEdge{
				Edge: next.Edge,
			},
			MaxAttempts: &retries,
		}, at); err != nil {
			return fmt.Errorf("unable to enqueue next step: %w", err)
		}

		// Increase the waitgroup counter.
		// Unfortunately, the backing queue and the state store may be different
		// backing services.  Therefore, we can never guarantee that enqueueing an
		// item increases the scheduled count.
		//
		// Hopefully, if the backing implementation is the same (eg. a database which
		// hosts the queue and the state store), Enqueue increases the pending count
		// and this is a no-op - things should be atomic where possible.
		if err := s.state.Scheduled(ctx, item.Identifier, next.Incoming, 0, &at); err != nil {
			return fmt.Errorf("unable to schedule next step: %w", err)
		}
	}

	// Mark this step as finalized.
	//
	// This must happen after everything is enqueued, else the scheduled <> finalized count
	// is out of order.
	if err := s.state.Finalized(ctx, item.Identifier, edge.Incoming, item.Attempt); err != nil {
		return fmt.Errorf("unable to finalize step: %w", err)
	}

	l.Debug().Interface("edge", edge).Msg("step complete")
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
	if pause == nil {
		return nil
	}

	if err := s.state.ConsumePause(ctx, pause.ID, nil); err != nil {
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
		if err := s.state.Finalized(ctx, item.Identifier, pause.Edge().Incoming, 0); err != nil {
			return err
		}
	}

	return nil
}
