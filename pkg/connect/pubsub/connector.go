package pubsub

import (
	"context"
	"github.com/redis/rueidis"
)

type Connector interface {
	RequestForwarder
}

type ConnectorOpt func(ctx context.Context) (Connector, error)

func NewConnector(ctx context.Context, initialize ConnectorOpt) (Connector, error) {
	return initialize(ctx)
}

func WithRedis(clientConfig rueidis.ClientOption, listen bool, opts RedisPubSubConnectorOpts) ConnectorOpt {
	return func(ctx context.Context) (Connector, error) {
		connector := newRedisPubSubConnector(opts)
		return connector, nil
	}
}
