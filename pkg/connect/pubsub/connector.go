package pubsub

import (
	"fmt"

	"github.com/redis/rueidis"
)

type Connector interface {
	RequestReceiver
	RequestForwarder
}

type ConnectorOpt func() (Connector, error)

func NewConnector(initialize ConnectorOpt) (Connector, error) {
	return initialize()
}

func WithRedis(opt rueidis.ClientOption) ConnectorOpt {
	return func() (Connector, error) {
		rc, err := rueidis.NewClient(opt)
		if err != nil {
			return nil, fmt.Errorf("error initializing redis client for connector: %w", err)
		}

		return NewRedisPubSubConnector(rc), nil
	}
}
