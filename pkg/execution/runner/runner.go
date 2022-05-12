// Package runner provides a high level workflow runner, comprising of a state manager,
// executor, and enqueuer to manage running a workflow from beginning to end.
package runner

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/pkg/execution/driver"
	"github.com/inngest/inngestctl/pkg/execution/executor"
	"github.com/inngest/inngestctl/pkg/execution/state"
	"github.com/oklog/ulid"
)

// New returns a new runner which executes workflows as they're added to the state manager
// as available.
func NewInMemoryRunner(sm state.Manager, exec executor.Executor) *InMemoryRunner {
	return &InMemoryRunner{
		sm:   sm,
		exec: exec,
		// execute from the trigger onwards.
		available: []string{inngest.TriggerName},
	}
}

// InMemoryRunner represents a runner that does some funky shit for my peeps please.
type InMemoryRunner struct {
	sm   state.Manager
	exec executor.Executor

	identifier state.Identifier

	// available reperesents functions that are available to be executed.  We
	// traverse the step functions from the trigger downards in a BFS, executing
	// each function in parallel.  Once each function is complete, its children
	// are pushed to the available slice for execution.
	available []string
}

// NewRun initializes a new run for the given workflow.
func (i *InMemoryRunner) NewRun(ctx context.Context, f inngest.Workflow) (*state.Identifier, error) {
	state, err := i.sm.New(ctx, f, ulid.MustNew(ulid.Now(), rand.Reader))
	if err != nil {
		return nil, fmt.Errorf("error initializing new run: %w", err)
	}
	id := state.Identifier()
	return &id, nil
}

// Execute runs all available tasks, blocking until terminated.
func (i *InMemoryRunner) Execute(ctx context.Context, id state.Identifier) error {
	// TODO: block.

	// Iterate over all available actions and execute.
	for len(i.available) > 0 {
		next := i.available[0]
		i.available = i.available[1:]

		children, err := i.exec.Execute(ctx, id, next)
		if err != nil {

			resp := driver.Response{}
			if !errors.As(err, &resp) || resp.Retryable() {
				// TODO: If this is retryable, schedule
				// up to N runs based off of the func.
			}
			return fmt.Errorf("execution error: %s", err)
		}
		i.available = append(i.available, children...)
	}

	// TODO: Wait for pauses.
	return nil
}
