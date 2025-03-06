package pubsub

import (
	"context"
	"errors"
	"fmt"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
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

func WithRedis(opt rueidis.ClientOption, logger *slog.Logger, tracer trace.ConditionalTracer, listen bool) ConnectorOpt {
	return func(ctx context.Context) (Connector, error) {
		rc, err := rueidis.NewClient(opt)
		if err != nil {
			return nil, fmt.Errorf("error initializing redis client for connector: %w", err)
		}

		connector, err := newRedisPubSubConnector(rc, logger, tracer), nil
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

func WithNoop() ConnectorOpt {
	return func(ctx context.Context) (Connector, error) {
		return noopConnector{}, nil
	}
}

// connectorWaiterSvc is a simple service that waits and shuts down the wait loop on context cancelation.
// This is used by Inngest Lite to handle graceful shutdowns for the forwarder.
type connectorWaiterSvc struct {
	connector Connector
	nested    []service.Service
}

func (c connectorWaiterSvc) Name() string {
	return "connector-waiter"
}

func (c connectorWaiterSvc) Pre(ctx context.Context) error {
	return nil
}

func (c connectorWaiterSvc) Run(ctx context.Context) error {
	if err := c.connector.Wait(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.From(ctx).Error().Err(err).Msg("error waiting for pubsub messages")
	}

	return service.StartAll(ctx, c.nested...)
}

func (c connectorWaiterSvc) Stop(ctx context.Context) error {
	return nil
}

func NewConnectorWaiterSvc(connector Connector, nested ...service.Service) service.Service {
	return &connectorWaiterSvc{connector: connector, nested: nested}
}
