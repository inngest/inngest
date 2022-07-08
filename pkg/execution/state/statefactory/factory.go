package statefactory

import (
	"context"
	"fmt"

	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/inmemory"
	"github.com/inngest/inngest-cli/pkg/execution/state/redis_state"
)

func NewState(ctx context.Context, conf config.State) (state.Manager, error) {
	// TODO: Move to an init() based registration method, in which
	// drivers register their backend name and config struct when
	// imported.  This lets us remove the hard-coded dependencies
	// between config, implementations, and this factory.
	// us remove this hard coding
	switch conf.Service.Backend {
	case "inmemory":
		return inmemory.NewSingletonStateManager(), nil
	case "redis":
		rc, ok := conf.Service.Concrete.(*config.RedisState)
		if !ok {
			return nil, fmt.Errorf("provided redis background with no redis config")
		}
		return redis_state.New(
			ctx,
			redis_state.WithConnectOpts(rc.ConnectOpts()),
		)
	default:
		return nil, fmt.Errorf("unknown state backend: %s", conf.Service.Backend)
	}
}
