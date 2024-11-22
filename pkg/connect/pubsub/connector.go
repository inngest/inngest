package pubsub

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/logger"
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

func WithRedis(opt rueidis.ClientOption, listen bool) ConnectorOpt {
	return func(ctx context.Context) (Connector, error) {
		rc, err := rueidis.NewClient(opt)
		if err != nil {
			return nil, fmt.Errorf("error initializing redis client for connector: %w", err)
		}

		connector, err := NewRedisPubSubConnector(rc), nil
		if listen {
			go func() {
				if err := connector.Wait(ctx); err != nil {
					logger.StdlibLogger(ctx).Error("error waiting for pubsub messages", "error", err)
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
