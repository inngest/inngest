package pubsub

import (
	"context"
	"fmt"
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

func WithRedis(clientConfig rueidis.ClientOption, listen bool, opts RedisPubSubConnectorOpts) ConnectorOpt {
	return func(ctx context.Context) (Connector, error) {
		rc, err := rueidis.NewClient(clientConfig)
		if err != nil {
			return nil, fmt.Errorf("error initializing redis client for connector: %w", err)
		}

		connector, err := newRedisPubSubConnector(rc, opts), nil
		if listen {
			go func() {
				if err := connector.Wait(ctx); err != nil {
					opts.Logger.Error("error waiting for pubsub messages", "error", err)
				}
			}()
		}
		return connector, err
	}
}
