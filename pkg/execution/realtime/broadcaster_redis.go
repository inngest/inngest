package realtime

import (
	"context"
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/redis/rueidis"
)

const (
	redisPublishAttempts = 3
	redisRetryInterval   = 2 * time.Second
)

// NewRedisBroadcaster implements a decentralized broadcaster that allows publishing and fanout of
// messages from any internal service to clients connected via separate gateways.
//
// Publishers, such as the executor, can instantiate a Redis broadcaster to publish messages on any
// topic.  Gateways can instantiate broadcasters connected to the same Redis instance to forward
// messages to any connected subscribers.
//
// The messages pass from executors (calling .Publish) to gateways (susbcribed to redis pub/sub via
// .Subscribe calls), being sent to all interested subscribers.
func NewRedisBroadcaster(pubc, subc rueidis.Client) Broadcaster {
	return &redisBroadcaster{
		broadcaster: newBroadcaster(),
		pubc:        pubc,
		subc:        subc,
	}
}

type redisBroadcaster struct {
	// embed a broadcaster for in-memory mamangement.
	*broadcaster

	// pubc is the client connected to Redis for publishing messages.  These are separated
	// from subscribing, as once a client subscribes it cannot be used for publishing.
	pubc rueidis.Client
	// subc is the raw client connected to Redis, allowing us to subscribe to pub-sub streams.
	subc rueidis.Client
}

// Publish publishes a message to Redis' pub-sub.  This is then caught by any subscribers
// to the same Redis pub-sub channels, which push the message to any connected Subscriptions.
func (b *redisBroadcaster) Publish(ctx context.Context, m Message) {
	// Push the message to Redis' pub/sub so that all other replicas of the
	// broadcaster receive the same content.  This ensures that every subscription
	// publishes message data.
	content, err := json.Marshal(m)
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error marshalling for realtime redis pubsub",
			"error", err,
		)
		return
	}

	for _, t := range m.Topics() {
		go func(t Topic) {
			b.publish(ctx, t.String(), string(content))
		}(t)
	}
}

func (b *redisBroadcaster) PublishStream(ctx context.Context, m Message, data string) {
	for _, t := range m.Topics() {
		go func(t Topic) {
			b.publish(ctx, t.String(), string(m.Data)+":"+data)
		}(t)
	}
}

func (b *redisBroadcaster) publish(ctx context.Context, channel, message string) {
	cmd := b.pubc.B().Publish().Channel(channel).Message(message).Build()
	for i := 0; i < redisPublishAttempts; i++ {
		if i > 0 {
			<-time.After(redisRetryInterval)
		}
		if ctx.Err() != nil {
			return
		}
		err := b.pubc.Do(ctx, cmd).Error()
		if err == nil {
			return
		}
		logger.StdlibLogger(ctx).Warn(
			"error publishing to realtime redis pubsub",
			"channel", channel,
			"error", err,
			"attempt", i,
		)
	}
	logger.StdlibLogger(ctx).Warn(
		"failed to publish via realtime redis pubsub",
		"channel", channel,
	)
}

// Subscribe is called with an active Websocket connection to subscribe the conn to multiple topics.
func (b *redisBroadcaster) Subscribe(ctx context.Context, s Subscription, topics []Topic) error {
	err := b.broadcaster.subscribe(
		ctx,
		s,
		topics,
		func(ctx context.Context, t Topic) {
			// We are subscribing to a specific topic.  This context will be closed
			// when Unsubscribe is called on the broadcaster.
			//
			// This means that we are safe to use this function's context within a redis
			// pub/sub call, as the pub/sub Receive will stop when this context is closed
			// after the subscription finishes.
			if err := b.redisPubsub(ctx, s, t); err != nil {
				logger.StdlibLogger(ctx).Error(
					"error subscribing to realtime redis pubsub",
					"error", err,
					"topic", t,
					"subscription_id", s.ID(),
				)
				_ = b.Unsubscribe(ctx, s.ID(), []Topic{t})
				return
			}

			logger.StdlibLogger(ctx).Debug(
				"subscribing to realtime redis pubsub",
				"topic", t,
				"subscription_id", s.ID(),
			)
		},
		func(ctx context.Context, t Topic) {
			logger.StdlibLogger(ctx).Debug(
				"unsubscribed from realtime redis pubsub",
				"topic", t,
				"subscription_id", s.ID(),
			)
		},
	)
	return err
}

func (b *redisBroadcaster) redisPubsub(ctx context.Context, s Subscription, t Topic) error {
	cmd := b.subc.B().Subscribe().Channel(t.String()).Build()
	err := b.subc.Receive(ctx, cmd, func(msg rueidis.PubSubMessage) {
		// Unmarshal the message's contents into the Message struct, then send on
		// the backing websocket connection.
		m := Message{}
		err := json.Unmarshal([]byte(msg.Message), &m)
		if err != nil {
			logger.StdlibLogger(ctx).Error(
				"error unmarshalling for realtime redis pubsub",
				"error", err,
			)
			return
		}
		// Publish the message to the given subscription.  The underlying broadcaster
		// handles retries here.
		b.publishTo(ctx, s, m)
	})
	return err
}
