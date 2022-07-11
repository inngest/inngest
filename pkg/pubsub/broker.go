package pubsub

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/inngest/inngest-cli/inngest/log"
	"github.com/inngest/inngest-cli/pkg/config"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/gcppubsub"
	_ "gocloud.dev/pubsub/mempubsub"
	_ "gocloud.dev/pubsub/natspubsub"
	"golang.org/x/sync/semaphore"
)

func NewPublisher(ctx context.Context, c config.MessagingService) (Publisher, error) {
	return NewPublishSubscriber(ctx, c)
}

func NewSubscriber(ctx context.Context, c config.MessagingService) (Subscriber, error) {
	return NewPublishSubscriber(ctx, c)
}

func NewPublishSubscriber(ctx context.Context, c config.MessagingService) (PublishSubscriber, error) {
	b := &broker{
		conf:   c,
		topics: map[string]*pubsub.Topic{},
		tl:     &sync.RWMutex{},
		mux:    pubsub.DefaultURLMux(),
	}
	return b, nil
}

// broker implements the PublishSubscriber interface using drivers to abstract
// the backing implementation.
type broker struct {
	conf config.MessagingService
	// topics stores a list of opened topics.
	topics map[string]*pubsub.Topic
	// tl locks topics within the broker.
	tl *sync.RWMutex
	// mux...
	mux *pubsub.URLMux
}

// Publish publishes an event on the given topic.  It does not check that a subscriber exists
// to the topic before sending, nor does it check that the topic/queue exists on the backend
// implementation.
func (b *broker) Publish(ctx context.Context, topic string, m Message) error {
	t, err := b.openPublishTopic(ctx, topic)
	if err != nil {
		return err
	}

	body, err := m.Encode()
	if err != nil {
		return fmt.Errorf("error encoding message: %w", err)
	}

	// convert message to pubsub message.
	wrapped := &pubsub.Message{
		Body: body,
		Metadata: map[string]string{
			"name":    m.Name,
			"version": m.Version,
		},
	}

	log.From(ctx).Debug().Interface("event", m.Name).Str("topic", topic).Msg("publishing event")

	if err = t.Send(ctx, wrapped); err != nil {
		return fmt.Errorf("error publishing event: %w", err)
	}

	return nil
}

// Subscribe subscribes to a topic, invoking the given run function consecutively
// in a single threaded manner each time an event is received.
func (b *broker) Subscribe(ctx context.Context, topic string, run PerformFunc) error {
	return b.SubscribeN(ctx, topic, run, 1)
}

// Subscribe subscribes to the given topic and runs the passed function each time
// an event is received.  It blocks until the given context is cancelled, and returns
// a nil error when shutting down from a cancelled context.
func (b *broker) SubscribeN(ctx context.Context, topic string, run PerformFunc, concurrency int64) error {
	url := b.conf.TopicURL(topic, config.URLTypeSubscribe)

	subs, err := b.mux.OpenSubscription(ctx, url)
	if err != nil {
		// NOTE: Most message systems require the topic to be created via config;
		// this may error if you haven't yet created the particular topic.
		return fmt.Errorf("error opening subscription: %w", err)
	}

	// we need a waitgroup to wait for all in-flight events to be processed
	// when the context is cancelled.  we can't re-acquire the semaphore after
	// the context is cancelled;  it'll return an error.
	wg := sync.WaitGroup{}

	// In order to subscribe N times, we want to create a weighted semaphore
	// versus creating a buffered channel.  The weighted semaphore will allow
	// us to adjust the semaphore's capacity depending on availability in the
	// future.
	//
	// Also, and more importantly, a typical pattern for buffered channels is
	// to acquire a message from the backing pubsub system then place this onto
	// a queue, blocking until capacity.  In this case, what happens if we close
	// the server while waiting for queue capacity?  We've already received an
	// event, and if the backing implementation doesn't support retries via NACKs
	// we've lost the message.
	//
	// By using a semaphore, we can block prior to receiving messages at all,
	// removing this condition.
	sem := semaphore.NewWeighted(math.MaxInt64)

	// The for loop will handle multiple messages at once.  When we have unrecoverable
	// errors we'll break out of the loop and wait for all in-flight messages to be
	// completed.
	var unrecoverableErr error

	for {
		if unrecoverableErr != nil {
			break
		}

		// We always have to check the context err, as semaphores can be acquired and return
		// no error after the context is cancelled.  It only errors if we're blocking and waiting
		// to acquire tokens.
		if err := sem.Acquire(ctx, math.MaxInt64/concurrency); err != nil || ctx.Err() != nil {
			// The subscription closed.
			unrecoverableErr = err
			break
		}

		msg, err := subs.Receive(ctx)
		if err != nil {
			// This is an unrecoverable error.
			unrecoverableErr = err
			// Break out of the loop and wait for all existing functions to complete.
			break
		}

		wg.Add(1)
		go func(msg *pubsub.Message) {
			defer sem.Release(math.MaxInt64 / concurrency)
			defer wg.Done()

			m := &Message{}
			if err := m.Decode(msg.Body); err != nil {
				// TODO: log.
				if msg.Nackable() {
					msg.Nack()
				} else {
					msg.Ack() // Unfortunately have to ack these if its not nackable.
				}
			}

			// Run the message only if we've decoded items.
			err := run(ctx, *m)
			if err == nil {
				msg.Ack()
				return
			}

			if msg.Nackable() {
				msg.Nack()
			} else {
				msg.Ack() // Unfortunately have to ack these if its not nackable.
			}
		}(msg)
	}

	if errors.Is(unrecoverableErr, context.Canceled) {
		// There's no need to error here, and an implicit race on sem acquisition
		// in which we only set the ctx cancelled error if it came from sem.Acquire,
		// not if ctx.Err is done.
		// We don't want to return an error either way.
		return nil
	}

	return unrecoverableErr
}

// openPublishTopic calls pubsub.OpenTopic once for a given event for publishing.
// It caches the opened *pubsub.Topic inside a map on the manager.
func (b *broker) openPublishTopic(ctx context.Context, topic string) (*pubsub.Topic, error) {
	url := b.conf.TopicURL(topic, config.URLTypePublish)

	b.tl.RLock()
	existing, ok := b.topics[url]
	b.tl.RUnlock()

	if ok {
		return existing, nil
	}

	t, err := b.mux.OpenTopic(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("error opening topic: %w", err)
	}

	b.tl.Lock()
	b.topics[url] = t
	b.tl.Unlock()

	return t, nil
}
