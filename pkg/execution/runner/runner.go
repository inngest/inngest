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

	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/execution/driver"
	"github.com/inngest/inngestctl/pkg/execution/executor"
	"github.com/inngest/inngestctl/pkg/execution/state"
	"github.com/oklog/ulid"
	str2duration "github.com/xhit/go-str2duration/v2"
)

// New returns a new runner which executes workflows in-memory.  This is NOT, EVER, IN
// ANY WAY SHAPE OR FORM SUITABLE FOR PRODUCTION.  Use this locally to test your stuff,
// please, and only to test.
func NewInMemoryRunner(sm state.Manager, exec executor.Executor) *InMemoryRunner {
	return &InMemoryRunner{
		sm:   sm,
		exec: exec,
		// execute from the trigger onwards.
		available: []inngest.Edge{inngest.SourceEdge},
		queue:     make(chan inngest.Edge),
	}
}

// InMemoryRunner represents a runner that does some funky shit for my peeps please.
type InMemoryRunner struct {
	sm   state.Manager
	exec executor.Executor

	queue chan inngest.Edge
	wg    sync.WaitGroup
	err   error

	// available reperesents functions that are available to be executed.  We
	// traverse the step functions from the trigger downards in a BFS, executing
	// each function in parallel.  Once each function is complete, its children
	// are pushed to the available slice for execution.
	available []inngest.Edge
}

// NewRun initializes a new run for the given workflow.
func (i *InMemoryRunner) NewRun(ctx context.Context, f inngest.Workflow, data map[string]interface{}) (*state.Identifier, error) {
	state, err := i.sm.New(ctx, f, ulid.MustNew(ulid.Now(), rand.Reader), data)
	if err != nil {
		return nil, fmt.Errorf("error initializing new run: %w", err)
	}
	id := state.Identifier()
	return &id, nil
}

// Execute runs all available tasks, blocking until terminated.
func (i *InMemoryRunner) Execute(ctx context.Context, id state.Identifier) error {
	go func() {
		for next := range i.queue {
			go i.run(ctx, id, next)
		}
	}()

	// Iterate over all available actions and execute.
	for len(i.available) > 0 {
		next := i.available[0]
		i.available = i.available[1:]

		i.wg.Add(1)
		i.queue <- next
	}

	// Wait for all available steps to finish processing.
	i.wg.Wait()

	return i.err
}

func (i *InMemoryRunner) run(ctx context.Context, id state.Identifier, e inngest.Edge) {
	if e.Metadata.Wait != nil {
		dur, err := str2duration.ParseDuration(*e.Metadata.Wait)
		if err != nil {
			i.err = multierror.Append(i.err, fmt.Errorf("invalid edge duration '%s'", *e.Metadata.Wait))
		}
		<-time.After(dur)
	}

	children, err := i.exec.Execute(ctx, id, e.Incoming)
	if err != nil {

		resp := driver.Response{}
		if !errors.As(err, &resp) || resp.Retryable() {
			// TODO: Check state to see if we've reached max retry
			// TODO: Add exponential backoff.
			i.wg.Add(1)
			i.queue <- e
		}

		i.err = multierror.Append(i.err, fmt.Errorf("execution error: %s", err))
	}

	for _, item := range children {
		i.wg.Add(1)
		i.queue <- item
	}

	i.wg.Done()
}
