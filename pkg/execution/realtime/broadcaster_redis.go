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

type redisBroadcaster struct {
	// embed a broadcaster for in-memory mamangement.
	*broadcaster

	// c is the raw client connected to Redis, allowing us to manage pub-sub streams.
	c rueidis.Client
}

// Subscribe is called with an active Websocket connection to subscribe the conn to multiple topics.
func (b *redisBroadcaster) Subscribe(ctx context.Context, s Subscription, topics []Topic) error {
	err := b.subscribe(
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
				// Unsubscribe in a goroutine, so that we can eventually lock
				// after the subscribe call finsihes.
				go b.Unsubscribe(ctx, s.ID(), []Topic{t})
				return
			}

			logger.StdlibLogger(ctx).Debug(
				"subscribing to realtime redis pubsub",
				"topic", t,
				"subscription_id", s.ID(),
			)
		},
		func(t Topic) {
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
	cmd := b.c.B().Subscribe().Channel(t.String()).Build()
	err := b.c.Receive(ctx, cmd, func(msg rueidis.PubSubMessage) {
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
			cmd := b.c.B().Publish().Channel(t.String()).Message(string(content)).Build()
			for i := 0; i < redisPublishAttempts; i++ {
				err := b.c.Do(ctx, cmd).Error()
				if err == nil {
					break
				}
				logger.StdlibLogger(ctx).Error(
					"error publishing to realtime redis pubsub",
					"error", err,
				)
				<-time.After(redisRetryInterval)
			}
		}(t)
	}
}
