package realtime

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MauriceGit/skiplist"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
	"github.com/sourcegraph/conc"
)

var (
	// ErrBroadcasterClosed is used when connecting to a braodcaster that is closing,
	// and not accepting new connections.
	ErrBroadcasterClosed = fmt.Errorf("broadcaster is closed")

	ShutdownGracePeriod = time.Minute * 5
	MaxWriteAttempts    = 3
	MaxKeepaliveErrors  = 3
	WriteRetryInterval  = 3 * time.Second
	KeepaliveInterval   = 15 * time.Second
)

// NewInProcessBroadcaster is a single broadcaster which manages active subscriptions
// in-memory and broadcasts to connected subscribers.
//
// This fulfils the Broadcaster interface.
func NewInProcessBroadcaster() *broadcaster {
	return newBroadcaster()
}

func newBroadcaster() *broadcaster {
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

	wg conc.WaitGroup

	// TopicStart is called when the first subscriber connects to a topic.
	TopicStart func(ctx context.Context, t Topic) error
	// TopicStop is called when the last subscriber disconnects from a topic.
	TopicStop func(ctx context.Context, t Topic) error
}

func (b *broadcaster) Subscribe(ctx context.Context, s Subscription, topics []streamingtypes.Topic) error {
	if len(topics) == 0 {
		return nil
	}
	return b.subscribe(ctx, s, topics)
}

// subscribe ensures that a given Subscription is subscribed to the provided topics.
func (b *broadcaster) subscribe(
	ctx context.Context,
	s Subscription,
	topics []Topic,
) error {
	if len(topics) == 0 {
		return nil
	}
	if atomic.LoadInt32(&b.closing) == 1 {
		return ErrBroadcasterClosed
	}

	b.l.Lock()
	defer b.l.Unlock()

	for _, t := range topics {
		topicHash := t.String()
		topicsubs, ok := b.topics[topicHash]
		if !ok {
			sl := skiplist.New()
			topicsubs = topicsub{
				Topic:         t,
				subscriptions: &sl,
			}
		}

		// If we already have this subscription, we don't want to insert it into
		// the topic slice again.  It's okay to check this as we only update the
		// active subscriptions after manipulating the topic map (see below).
		var seen bool
		if as, ok := b.subs[s.ID()]; ok {
			_, seen = as.Topics[topicHash]
		}

		// We haven't seen the topic before.  The topics map is used when publishing,
		// and we only want to insert the same subscription once.  This ensures that
		// even if a single Subscription calls subscribe to the same topic, we only send
		// a single message. Note that a Subscription represents a single connection,
		// meaning we only send a single message per eg. websocket connection.
		if !seen {
			topicsubs.subscriptions.Insert(skiplistSub{s})
			topicsubs.RefCount++
			b.topics[topicHash] = topicsubs

			if topicsubs.RefCount == 1 && b.TopicStart != nil {
				// We must release the lock while calling TopicStart to avoid deadlocks
				// if TopicStart calls back into broadcaster (though it shouldn't).
				// However, releasing the lock here is dangerous as state might change.
				// For now, we assume TopicStart is non-blocking and doesn't call back.
				// redisBroadcaster.TopicStart spawns a goroutine, so it is fast.
				if err := b.TopicStart(ctx, t); err != nil {
					// Rollback
					topicsubs.subscriptions.Delete(skiplistSub{s})
					topicsubs.RefCount--
					if topicsubs.RefCount == 0 {
						delete(b.topics, topicHash)
					} else {
						b.topics[topicHash] = topicsubs
					}
					return fmt.Errorf("error starting topic %s: %w", t.String(), err)
				}
			}
		}
	}

	if as, ok := b.subs[s.ID()]; ok {
		as.AddTopics(topics...)
	} else {
		as = &activesub{Subscription: s}
		as.AddTopics(topics...)
		b.subs[s.ID()] = as
		// This is the first time we've seen a subscription.  Send
		// keepalives after an interval to ensure that the connection
		// remains open during periods of inactivity.
		b.wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					logger.StdlibLogger(ctx).Error("panic in keepalive", "panic", r, "sub_id", s.ID())
				}
			}()
			b.keepalive(ctx, s.ID())
		})

		b.recordConnectionMetrics(ctx)
	}

	metrics.HistogramRealtimeSubscriptionTopicsCount(ctx, int64(len(topics)), metrics.HistogramOpt{
		PkgName: "realtime",
		Tags:    map[string]any{},
	})

	return nil
}

// Unsubscribe removes a subscription from specific topics.
func (b *broadcaster) Unsubscribe(ctx context.Context, subID uuid.UUID, topics []Topic) error {
	if atomic.LoadInt32(&b.closing) == 1 {
		// Already happening, so ignore.
		return ErrBroadcasterClosed
	}

	b.l.Lock()
	defer b.l.Unlock()

	as, ok := b.subs[subID]
	if !ok {
		return nil
	}

	// Delete all subscriptions from the topic lookup
	for _, t := range topics {
		str := t.String()

		// Check to see if this active subscription is subscribed to the given
		// topic.  If not, we're not going to bother.
		subs, ok := b.topics[str]
		if !ok {
			continue
		}

		if _, ok := as.Topics[str]; !ok {
			continue
		}

		// Remove this from the subscription list
		subs.subscriptions.Delete(skiplistSub{as.Subscription})
		delete(as.Topics, str)

		subs.RefCount--
		if subs.RefCount == 0 {
			if b.TopicStop != nil {
				if err := b.TopicStop(ctx, t); err != nil {
					logger.StdlibLogger(ctx).Warn("error stopping topic", "error", err, "topic", t)
				}
			}
			delete(b.topics, str)
		} else {
			// Update the map with new RefCount
			b.topics[str] = subs
		}
	}

	return nil
}

// CloseSubscription shuts down a subscription, removing it from all topics and removing the subscription
// from the subscription map.
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

		subs.RefCount--
		if subs.RefCount == 0 {
			if b.TopicStop != nil {
				if err := b.TopicStop(ctx, t); err != nil {
					logger.StdlibLogger(ctx).Warn("error stopping topic", "error", err, "topic", t)
				}
			}
			delete(b.topics, str)
		} else {
			b.topics[str] = subs
		}
	}

	// Then remove the subscription from our subscription map
	delete(b.subs, subscriptionID)

	b.recordConnectionMetrics(ctx)
	return nil
}

func (b *broadcaster) recordConnectionMetrics(ctx context.Context) {
	// This function assumes b.l is already locked by the caller.
	websocketCount := 0
	sseCount := 0

	for _, as := range b.subs {
		sub := as.Subscription
		if sub.Protocol() == "websocket" {
			websocketCount++
		} else if sub.Protocol() == "sse" {
			sseCount++
		}
	}

	metrics.GaugeRealtimeConnectionsActive(ctx, int64(websocketCount), metrics.GaugeOpt{
		PkgName: "realtime",
		Tags: map[string]any{
			"protocol": "websocket",
		},
	})

	metrics.GaugeRealtimeConnectionsActive(ctx, int64(sseCount), metrics.GaugeOpt{
		PkgName: "realtime",
		Tags: map[string]any{
			"protocol": "sse",
		},
	})
}

func (b *broadcaster) Close(ctx context.Context) error {
	if atomic.LoadInt32(&b.closing) == 1 {
		return ErrBroadcasterClosed
	}

	atomic.StoreInt32(&b.closing, 1)

	msg := Message{
		Kind:      streamingtypes.MessageKindClosing,
		CreatedAt: time.Now(),
	}

	// Send a close notification to all active subscriptions.
	b.l.RLock()
	defer b.l.RUnlock()
	for _, s := range b.subs {
		b.publishTo(ctx, s.Subscription, msg)
	}

	go func() {
		defer func() {
			// Ensure we wait for all background routines to finish.
			// Since we recover inside them, Wait() shouldn't panic, but let's be safe.
			if r := b.wg.WaitAndRecover(); r != nil {
				logger.StdlibLogger(ctx).Error("panic waiting for broadcaster shutdown", "panic", r)
			}
		}()

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

func (b *broadcaster) Write(ctx context.Context, envID uuid.UUID, channel string, data []byte) {
	b.l.RLock()
	defer b.l.RUnlock()

	// Find all subscriptions for this channel across all topics
	// Since we don't have a specific topic, we'll write to any subscription
	// that has a matching channel in any of its topics
	for _, topicSub := range b.topics {
		if topicSub.EnvID == envID && topicSub.Channel == channel {
			topicSub.eachSubscription(func(s Subscription) {
				// Use Write() to forward raw bytes directly
				if err := s.Write(data); err != nil {
					logger.StdlibLogger(ctx).Warn(
						"error writing raw data to subscription",
						"error", err,
						"channel", channel,
						"sub_id", s.ID(),
					)
				}
			})
		}
	}
}

func (b *broadcaster) Publish(ctx context.Context, m Message) {
	b.l.RLock()
	defer b.l.RUnlock()

	metrics.IncrRealtimeMessagesPublishedTotal(ctx, metrics.CounterOpt{
		PkgName: "realtime",
		Tags:    map[string]any{},
	})

	wg := conc.WaitGroup{}
	for _, t := range m.Topics() {
		tid := t.String()
		found, ok := b.topics[tid]

		if !ok {
			continue
		}

		wg.Go(func() {
			t := found
			// Ensure we set the correct topic name for the given topic.
			// Messages always have a custom topic name (eg. the step name),
			// but are broadcast to internal topics such as "$step";  we need
			// to update that for each topic here.
			msg := m
			msg.Topic = t.Name

			t.eachSubscription(func(s Subscription) {
				b.publishTo(ctx, s, msg)
			})
		})
	}

	wg.Wait()
}

func (b *broadcaster) publishToTopic(ctx context.Context, t Topic, m Message) {
	b.l.RLock()
	defer b.l.RUnlock()

	metrics.IncrRealtimeMessagesPublishedTotal(ctx, metrics.CounterOpt{
		PkgName: "realtime",
		Tags:    map[string]any{},
	})

	tid := t.String()
	found, ok := b.topics[tid]
	if !ok {
		return
	}

	msg := m
	msg.Topic = t.Name

	found.eachSubscription(func(s Subscription) {
		b.publishTo(ctx, s, msg)
	})
}

func (b *broadcaster) writeToTopic(ctx context.Context, t Topic, data []byte) {
	b.l.RLock()
	defer b.l.RUnlock()

	tid := t.String()
	found, ok := b.topics[tid]
	if !ok {
		return
	}

	found.eachSubscription(func(s Subscription) {
		// Use Write() to forward raw bytes directly
		if err := s.Write(data); err != nil {
			logger.StdlibLogger(ctx).Warn(
				"error writing raw data to subscription",
				"error", err,
				"topic", t.String(),
				"sub_id", s.ID(),
			)
		}
	})
}

// PublishStream publishes streams of data to any subscribers for a given datastream.
func (b *broadcaster) PublishChunk(ctx context.Context, m Message, c Chunk) {
	b.l.RLock()
	defer b.l.RUnlock()

	wg := conc.WaitGroup{}
	for _, t := range m.Topics() {
		tid := t.String()
		found, ok := b.topics[tid]
		if !ok {
			continue
		}

		wg.Go(func() {
			t := found
			t.eachSubscription(func(s Subscription) {
				b.publishStreamTo(ctx, s, c)
			})
		})
	}

	wg.Wait()
}

// publishTo publishes a message to a subscription, keeping track of retries if the
// write fails.
func (b *broadcaster) publishTo(ctx context.Context, s Subscription, m Message) {
	b.doPublish(ctx, s, func() error {
		return s.WriteMessage(m)
	})
}

// publishStreamTo publishes a message to a subscription, keeping track of retries if the
// write fails.
func (b *broadcaster) publishStreamTo(ctx context.Context, s Subscription, c Chunk) {
	b.doPublish(ctx, s, func() error {
		return s.WriteChunk(c)
	})
}

// doPublish publishes a message or stream to a subscription,
// keeping track of retries if the write fails.
func (b *broadcaster) doPublish(ctx context.Context, s Subscription, f func() error) {
	if err := f(); err == nil {
		return
	}

	// If this failed to write, attempt to resend the message until
	// max attempts pass.
	b.wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				logger.StdlibLogger(ctx).Error("panic in doPublish retry", "panic", r, "sub_id", s.ID())
			}
		}()

		var err error
		for att := 1; att < MaxWriteAttempts; att++ {
			<-time.After(WriteRetryInterval)
			if err = f(); err == nil {
				return
			}
		}
		metrics.IncrRealtimeMessageDeliveryFailuresTotal(ctx, metrics.CounterOpt{
			PkgName: "realtime",
			Tags: map[string]any{
				"protocol": s.Protocol(),
				"reason":   "write_failed",
			},
		})
		logger.StdlibLogger(ctx).Warn(
			"error publishing to subscription",
			"error", err,
			"subscription_id", s.ID(),
			"protocol", s.Protocol(),
		)
		// TODO: mark the subscription as failing.  If the subscription
		// continues to fail, ensure we close the subscription.
	})
}

// keepalive sends keepalives to the subscription within a specific interval, ensuring
// that the connection is active.
func (b *broadcaster) keepalive(ctx context.Context, subID uuid.UUID) {
	errCount := 0

	for {
		// ensure the subscription ID exists, else it has been closed.
		b.l.RLock()
		sub, ok := b.subs[subID]
		if !ok {
			return
		}
		b.l.RUnlock()

		err := sub.SendKeepalive(Message{
			Kind:      streamingtypes.MessageKindPing,
			CreatedAt: time.Now(),
		})
		if err == nil {
			// reset the error count on success.
			errCount = 0
		}
		if err != nil {
			errCount += 1
			metrics.IncrRealtimeKeepaliveFailuresTotal(ctx, metrics.CounterOpt{
				PkgName: "realtime",
				Tags: map[string]any{
					"protocol": sub.Protocol(),
				},
			})
		}
		if errCount == MaxKeepaliveErrors {
			// Close this subscription and quit.
			logger.StdlibLogger(ctx).Warn(
				"max failed keepalives reached",
				"error", err,
				"subscription_id", subID,
				"protocol", sub.Protocol(),
			)
			_ = b.CloseSubscription(ctx, subID)
			return
		}

		<-time.After(KeepaliveInterval)
	}
}

// activesub represents an active subscription with interest in one or
// more Topics, for lookup from subscriber -> topics.
type activesub struct {
	Subscription

	// Topics lists all topics that the subscription is interested in
	Topics map[string]Topic
}

func (a *activesub) AddTopics(t ...Topic) {
	if a.Topics == nil {
		a.Topics = map[string]Topic{}
	}

	for _, item := range t {
		a.Topics[item.String()] = item
	}
}

// topicsub represents subscriptions to a particular topic, for lookup
// by topic -> subscribers.
type topicsub struct {
	Topic

	subscriptions *skiplist.SkipList
	RefCount      int
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
