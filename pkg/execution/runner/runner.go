// Package runner provides a high level workflow runner, comprising of a state manager,
// executor, and enqueuer to manage running a workflow from beginning to end.
package runner

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/backoff"
	"github.com/inngest/inngestctl/pkg/execution/driver"
	"github.com/inngest/inngestctl/pkg/execution/executor"
	"github.com/inngest/inngestctl/pkg/execution/state"
	"github.com/inngest/inngestctl/pkg/execution/state/inmemory"
	"github.com/oklog/ulid"
	"github.com/xhit/go-str2duration/v2"
)

// New returns a new runner which executes workflows in-memory.  This is NOT, EVER, IN
// ANY WAY SHAPE OR FORM SUITABLE FOR PRODUCTION.  Use this locally to test your stuff,
// please, and only to test.
func NewInMemoryRunner(sm inmemory.Queue, exec executor.Executor) *InMemoryRunner {
	return &InMemoryRunner{
		sm:   sm,
		exec: exec,
		wg:   &sync.WaitGroup{},
	}
}

// InMemoryRunner represents a runner that does some funky shit for my peeps please.
type InMemoryRunner struct {
	sm   inmemory.Queue
	exec executor.Executor

	// The waitgroup allows us to terminate the inmemory runner when all nodes in
	// a single function is complete, instead of blocking forever.
	wg *sync.WaitGroup
}

// NewRun initializes a new run for the given workflow.
func (i *InMemoryRunner) NewRun(ctx context.Context, f inngest.Workflow, data map[string]interface{}) (*state.Identifier, error) {
	state, err := i.sm.New(ctx, f, ulid.MustNew(ulid.Now(), rand.Reader), data)
	if err != nil {
		return nil, fmt.Errorf("error initializing new run: %w", err)
	}
	id := state.Identifier()

	i.wg.Add(1)
	i.sm.Enqueue(inmemory.QueueItem{
		ID:   id,
		Edge: inngest.SourceEdge,
	}, time.Now())

	return &id, nil
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
	i.wg.Wait()
	return err
}

func (i *InMemoryRunner) run(ctx context.Context, item inmemory.QueueItem) error {
	defer func() {
		i.wg.Done()
	}()

	children, err := i.exec.Execute(ctx, item.ID, item.Edge.Incoming)

	if err != nil {
		resp := driver.Response{}
		if !errors.As(err, &resp) || resp.Retryable() {
			next := item
			next.ErrorCount += 1

			at := backoff.LinearJitterBackoff(next.ErrorCount)

			// XXX: When we add max retries to steps, read the step from the
			// state store here to chech for the step's retry data.
			//
			// For now, we retry steps of a function up to 3 times.
			if next.ErrorCount < 3 {
				i.wg.Add(1)
				i.sm.Enqueue(next, at)
				return nil
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

		// Add to the waitgroup, ensuring that the runner blocks until the enqueued item
		// is finished.
		i.wg.Add(1)

		// Enqueue the next child in our in-memory state queue.
		i.sm.Enqueue(inmemory.QueueItem{
			ID:   item.ID,
			Edge: next,
		}, at)
	}

	return nil
}
