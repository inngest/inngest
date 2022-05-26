// Package runner provides a high level workflow runner, comprising of a state manager,
// executor, and enqueuer to manage running a workflow from beginning to end.
package runner

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/backoff"
	"github.com/inngest/inngest-cli/pkg/execution/driver"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/oklog/ulid/v2"
	"github.com/xhit/go-str2duration/v2"
)

// New returns a new runner which executes workflows in-memory.  This is NOT, EVER, IN
// ANY WAY SHAPE OR FORM SUITABLE FOR PRODUCTION.  Use this locally to test your stuff,
// please, and only to test.
func NewInMemoryRunner(sm inmemory.Queue, exec executor.Executor) *InMemoryRunner {
	return &InMemoryRunner{
		sm:    sm,
		exec:  exec,
		waits: map[ulid.ULID]*sync.WaitGroup{},
	}
}

// InMemoryRunner represents a runner which coordinates steps enqueued within an
// in memory queue, and executing the steps within the executor.
type InMemoryRunner struct {
	sm   inmemory.Queue
	exec executor.Executor

	// In a dev server, we want to wait until all current steps of a state.Identifier
	// are complete.  We create a new waitgroup per identifier to wait for the steps.
	waits map[ulid.ULID]*sync.WaitGroup
}

// NewRun initializes a new run for the given workflow.
func (i *InMemoryRunner) NewRun(ctx context.Context, f inngest.Workflow, data map[string]interface{}) (*state.Identifier, error) {
	state, err := i.sm.New(ctx, f, ulid.MustNew(ulid.Now(), rand.Reader), data)
	if err != nil {
		return nil, fmt.Errorf("error initializing new run: %w", err)
	}
	id := state.Identifier()

	i.Enqueue(ctx, inmemory.QueueItem{
		ID:   id,
		Edge: inngest.SourceEdge,
	}, time.Now())

	return &id, nil
}

func (i InMemoryRunner) Enqueue(ctx context.Context, item inmemory.QueueItem, at time.Time) {
	if _, ok := i.waits[item.ID.RunID]; !ok {
		i.waits[item.ID.RunID] = &sync.WaitGroup{}
	}

	// Add to the waitgroup, ensuring that the runner blocks until the enqueued item
	// is finished.
	i.waits[item.ID.RunID].Add(1)
	i.sm.Enqueue(item, at)
}

// Execute runs all available tasks, blocking until terminated.
func (i *InMemoryRunner) Execute(ctx context.Context, id state.Identifier) error {
	var err error
	go func() {
		for item := range i.sm.Channel() {
			// We could terminate the executor here on error
			_ = i.run(ctx, item)
		}
	}()

	// Wait for all items in the queue to be complete.
	i.waits[id.RunID].Wait()
	return err
}

func (i *InMemoryRunner) run(ctx context.Context, item inmemory.QueueItem) error {
	defer func() {
		i.waits[item.ID.RunID].Done()
	}()

	resp, children, err := i.exec.Execute(ctx, item.ID, item.Edge.Incoming)

	if err != nil {
		// If the error is not of type response error, we can assume that this is
		// always retryable.
		_, isResponseError := err.(*driver.Response)
		if (resp != nil && resp.Retryable()) || !isResponseError {
			next := item
			next.ErrorCount += 1

			at := backoff.LinearJitterBackoff(next.ErrorCount)

			// XXX: When we add max retries to steps, read the step from the
			// state store here to chech for the step's retry data.
			//
			// For now, we retry steps of a function up to 3 times.
			if next.ErrorCount < 3 {
				i.Enqueue(ctx, next, at)
			}
		}

		return fmt.Errorf("execution error: %s", err)
	}

	for _, next := range children {
		at := time.Now()
		if next.Metadata.Wait != nil {
			dur, err := str2duration.ParseDuration(*next.Metadata.Wait)
			if err != nil {
				return fmt.Errorf("invalid wait duration: %s", *next.Metadata.Wait)
			}
			at = at.Add(dur)
		}

		// Enqueue the next child in our in-memory state queue.
		i.Enqueue(ctx, inmemory.QueueItem{
			ID:   item.ID,
			Edge: next,
		}, at)
	}

	return nil
}
