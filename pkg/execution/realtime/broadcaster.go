package realtime

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MauriceGit/skiplist"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
)

var (
	// ErrBroadcasterClosed is used when connecting to a braodcaster that is closing,
	// and not accepting new connections.
	ErrBroadcasterClosed = fmt.Errorf("broadcaster is closed")

	ShutdownGracePeriod = time.Minute * 5
	MaxWriteAttempts    = 3
	WriteRetryInterval  = 5 * time.Second
)

// NewInProcessBroadcaster is a single broadcaster which is used for in-memory, in-process
// publishing.
func NewInProcessBroadcaster() Broadcaster {
	return &broadcaster{
		closing: 0,
		subs:    map[uuid.UUID]*activesub{},
		topics:  map[string]topicsub{},
		l:       &sync.RWMutex{},
	}
}

// broadcaster represents a set of subscriptions for one or more topics.
//
// A broadcaster maintains connections to an external realtime subscriber in-memory,
// eg. actual live WebSocket or HTTP connections.  For each connection, it retains which
// topics the subscription is currently interested in.
//
// The broadcaster then receives events from a publisher (implemented either directly or
// indirectly via Redis Pub/Sub), and pushes the events to the Subscription.
//
// When the braodcaster shuts down, it sends a shutdown message to all subscribers.
// Subscribers should reconnect immediately (routed to a healthy broadcaster) to prevent
// lost messages.
type broadcaster struct {
	closing int32

	// subs is a map of subscription IDs to all active topic subscriptions.
	subs map[uuid.UUID]*activesub

	// topics is a list of all subscribed topics, as a skiplist of
	// subscription IDs
	topics map[string]topicsub

	l *sync.RWMutex
}

func (b *broadcaster) Subscribe(ctx context.Context, s Subscription, topics []Topic) error {
	if atomic.LoadInt32(&b.closing) == 1 {
		return ErrBroadcasterClosed
	}

	b.l.Lock()
	defer b.l.Unlock()
	for _, t := range topics {
		str := t.String()
		subs, ok := b.topics[str]
		if !ok {
			sl := skiplist.New()
			subs = topicsub{
				Topic:         t,
				subscriptions: &sl,
			}
		}
		subs.subscriptions.Insert(skiplistSub{s})
		b.topics[str] = subs
	}

	if as, ok := b.subs[s.ID()]; ok {
		as.AddTopics(topics...)
	} else {
		as = &activesub{
			Subscription: s,
			Topics:       topics,
		}
		b.subs[s.ID()] = as
		// This is the first time we've seen a subscription.  Send
		// keepalives every 30 seconds to ensure that the connection
		// remains open during periods of inactivity.
		//
		// TODO: Implement above.
	}

	return nil
}

func (b *broadcaster) CloseSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	b.l.Lock()
	defer b.l.Unlock()

	as, ok := b.subs[subscriptionID]
	if !ok {
		return nil
	}

	if err := as.Close(); err != nil {
		return fmt.Errorf("error closing subscription: %w", err)
	}

	// Delete all subscriptions from the topic lookup
	for _, t := range as.Topics {
		str := t.String()
		subs, ok := b.topics[str]
		if !ok {
			continue
		}
		subs.subscriptions.Delete(skiplistSub{as.Subscription})
	}

	// Then remove the subscription from our subscription map
	delete(b.subs, subscriptionID)
	return nil
}

func (b *broadcaster) Close(ctx context.Context) error {
	if atomic.LoadInt32(&b.closing) == 1 {
		return ErrBroadcasterClosed
	}

	atomic.StoreInt32(&b.closing, 1)

	msg := Message{
		Kind:      MessageKindClosing,
		CreatedAt: time.Now(),
	}

	// Send a close notification to all active subscriptions.
	b.l.RLock()
	defer b.l.RUnlock()
	for _, s := range b.subs {
		b.publishTo(ctx, s.Subscription, msg)
	}

	go func() {
		// After 5 minutes, close all connections.
		<-time.After(ShutdownGracePeriod)

		b.l.RLock()
		defer b.l.RUnlock()
		for _, s := range b.subs {
			if err := s.Subscription.Close(); err != nil {
				logger.StdlibLogger(ctx).Warn("error closing realtime subscription", "error", err)
			}
		}
	}()

	return nil
}

func (b *broadcaster) Publish(ctx context.Context, m Message) {
	b.l.RLock()
	defer b.l.RUnlock()

	wg := sync.WaitGroup{}
	for _, t := range m.Topics() {
		tid := t.String()
		found, ok := b.topics[tid]
		if !ok {
			continue
		}

		wg.Add(1)
		go func(t topicsub) {
			defer wg.Done()
			t.eachSubscription(func(s Subscription) {
				b.publishTo(ctx, s, m)
			})
		}(found)
	}

	wg.Wait()
}

func (b *broadcaster) publishTo(ctx context.Context, s Subscription, m Message) {
	if err := s.WriteMessage(m); err == nil {
		return
	}

	// If this failed to write, attempt to resend the message until
	// max attempts pass.
	go func() {
		for att := 1; att < MaxWriteAttempts; att++ {
			<-time.After(WriteRetryInterval)
			if err := s.WriteMessage(m); err == nil {
				return
			}
		}
		// TODO: Log an error that this subscription failed, and handle
		// marking the subscription as failing.
	}()
}

// activesub represents an active subscription with interest in one or
// more Topics, for lookup from subscriber -> topics.
type activesub struct {
	Subscription

	// Topics lists all topics that the subscription is interested in
	Topics []Topic
}

func (a *activesub) AddTopics(t ...Topic) {
	a.Topics = append(a.Topics, t...)
}

// topicsub represents subscriptions to a particular topic, for lookup
// by topic -> subscribers.
type topicsub struct {
	Topic

	subscriptions *skiplist.SkipList
}

func (t topicsub) eachSubscription(f func(s Subscription)) {
	node := t.subscriptions.GetSmallestNode()
	if node == nil {
		return
	}

	// Run the given function on the first node.
	f(node.GetValue().(skiplistSub).Subscription)

	key := node.GetValue().ExtractKey()

	// Iterate through all next nodes, up to 5000 times.  It is not
	// allowed to have more than 5,000 subscribers per topic.
	next := t.subscriptions.Next(node)
	i := 0
	for next.GetValue().ExtractKey() != key && i < 5000 {
		f(next.GetValue().(skiplistSub).Subscription)
		next = t.subscriptions.Next(next)
		i++
	}
}

// skiplistSub wraps a Subscription to implement the SkipListEntry interface
type skiplistSub struct {
	Subscription
}

func (s skiplistSub) ExtractKey() float64 {
	return util.XXHashFloat(s.Subscription.ID())
}

func (s skiplistSub) String() string {
	return ""
}
