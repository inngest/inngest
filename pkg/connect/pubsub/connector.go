package pubsub

import (
	"context"
	"fmt"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"log/slog"

	"github.com/redis/rueidis"
)

type Connector interface {
	RequestReceiver
	RequestForwarder
}

type ConnectorOpt func(ctx context.Context) (Connector, error)

func NewConnector(ctx context.Context, initialize ConnectorOpt) (Connector, error) {
	return initialize(ctx)
}

func WithRedis(opt rueidis.ClientOption, logger *slog.Logger, tracer trace.ConditionalTracer, stateMan state.StateManager, listen bool) ConnectorOpt {
	return func(ctx context.Context) (Connector, error) {
		rc, err := rueidis.NewClient(opt)
		if err != nil {
			return nil, fmt.Errorf("error initializing redis client for connector: %w", err)
		}

		connector, err := newRedisPubSubConnector(rc, logger, tracer, stateMan), nil
		if listen {
			go func() {
				if err := connector.Wait(ctx); err != nil {
					logger.Error("error waiting for pubsub messages", "error", err)
				}
			}()
		}
		return connector, err
	}
}
