package queuefactory

import (
	"context"
	"fmt"

	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/execution/queue"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
)

func NewQueue(ctx context.Context, conf config.Queue) (queue.Queue, error) {
	// TODO: Move to an init() based registration method, in which
	// drivers register their backend name and config struct when
	// imported.  This lets us remove the hard-coded dependencies
	// between config, implementations, and this factory.
	// us remove this hard coding
	switch conf.Service.Backend {
	case "inmemory":
		// TODO: Move this into its own package, separating it from state.
		return inmemory.NewSingletonStateManager(), nil
	default:
		return nil, fmt.Errorf("unknown queue backend: %s", conf.Service.Backend)
	}
}
