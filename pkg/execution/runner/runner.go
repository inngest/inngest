// Package runner provides a high level workflow runner, comprising of a state manager,
// executor, and enqueuer to manage running a workflow from beginning to end.
package runner

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/backoff"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/queue"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/oklog/ulid/v2"
	"github.com/xhit/go-str2duration/v2"
)

var (
	CompletePollInterval = 25 * time.Millisecond
)

// New returns a new runner which executes workflows in-memory.  This is NOT, EVER, IN
// ANY WAY SHAPE OR FORM SUITABLE FOR PRODUCTION.  Use this locally to test your stuff,
// please, and only to test.
func NewInMemoryRunner(sm inmemory.Queue, exec executor.Executor) *InMemoryRunner {
	return &InMemoryRunner{
		sm:    sm,
		queue: sm,
		exec:  exec,
	}
}

// InMemoryRunner represents a runner which coordinates steps enqueued within an
// in memory queue, and executing the steps within the executor.
type InMemoryRunner struct {
	sm    state.Manager
	queue queue.Queue
	exec  executor.Executor
}

// Start invokes the queue's Run function blocking and reading items from the queue
// for processing.
func (i *InMemoryRunner) Start(ctx context.Context) error {
	return i.queue.Run(ctx, i.run)
}

// NewRun initializes a new run for the given workflow.
func (i *InMemoryRunner) NewRun(ctx context.Context, f inngest.Workflow, data map[string]interface{}) (*state.Identifier, error) {
	id := state.Identifier{
		WorkflowID: f.UUID,
		RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
	}

	_, err := i.sm.New(ctx, f, id, data)
	if err != nil {
		return nil, fmt.Errorf("error initializing new run: %w", err)
	}

	err = i.Enqueue(ctx, queue.Item{
		Identifier: id,
		Payload:    queue.PayloadEdge{Edge: inngest.SourceEdge},
	}, time.Now())

	return &id, err
}

// Wait blocks until the given run has finished.
func (i InMemoryRunner) Wait(ctx context.Context, id state.Identifier) error {
	// TODO: Implement subscribe if the state store has a subscribe method.
	for {
		select {
		case <-time.After(CompletePollInterval):
			ok, err := i.sm.IsComplete(ctx, id)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (i InMemoryRunner) Enqueue(ctx context.Context, item queue.Item, at time.Time) error {
	edge, err := queue.GetEdge(item)
	if err != nil {
		return err
	}

	if err := i.queue.Enqueue(ctx, item, at); err != nil {
		return err
	}
	_ = i.sm.Scheduled(ctx, item.Identifier, edge.Incoming)
	return nil
}

// run coordinates the execution of items from the queue.
func (i *InMemoryRunner) run(ctx context.Context, item queue.Item) error {
	edge, err := queue.GetEdge(item)
	if err != nil {
		return err
	}

	resp, err := i.exec.Execute(ctx, item.Identifier, edge.Incoming, item.ErrorCount)
	if err != nil {
		// If the error is not of type response error, we can assume that this is
		// always retryable.
		_, isResponseError := err.(*state.DriverResponse)
		if (resp != nil && resp.Retryable()) || !isResponseError {
			next := item
			next.ErrorCount += 1
			at := backoff.LinearJitterBackoff(next.ErrorCount)
			if err := i.Enqueue(ctx, next, at); err != nil {
				return err
			}
		}

		// This is a non-retryable error.  Finalize this step.
		if err := i.sm.Finalized(ctx, item.Identifier, edge.Incoming); err != nil {
			return err
		}
		return fmt.Errorf("execution error: %s", err)
	}

	s, err := i.sm.Load(ctx, item.Identifier)
	if err != nil {
		return err
	}

	children, err := state.DefaultEdgeEvaluator.AvailableChildren(ctx, s, edge.Incoming)
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

			err = i.sm.SavePause(ctx, state.Pause{
				ID:         uuid.New(),
				Identifier: s.Identifier(),
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

		// Enqueue the next child in our in-memory state queue.
		if err := i.Enqueue(ctx, queue.Item{
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
	if err := i.sm.Finalized(ctx, item.Identifier, edge.Incoming); err != nil {
		return err
	}

	return nil
}
