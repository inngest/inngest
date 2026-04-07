package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MauriceGit/skiplist"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
	"github.com/redis/rueidis"
	"github.com/sourcegraph/conc"
)

const (
	redisPublishAttempts = 3
	redisRetryInterval   = 2 * time.Second
	redisPublishTimeout  = 10 * time.Second

	// redisRawDataPrefix and redisChunkPrefix distinguish message types in
	// Redis pub/sub. Structured JSON messages have no prefix and always start
	// with '{', so there is no ambiguity.
	redisRawDataPrefix = "RAW:"
	redisChunkPrefix   = "CHUNK:"
)

var (
	// ErrBroadcasterClosed is used when connecting to a broadcaster that is closing,
	// and not accepting new connections.
	ErrBroadcasterClosed = fmt.Errorf("broadcaster is closed")

	ShutdownGracePeriod = time.Minute * 5
	MaxWriteAttempts    = 3
	MaxKeepaliveErrors  = 3
	WriteRetryInterval  = 3 * time.Second
	KeepaliveInterval   = 15 * time.Second
)

// BroadcasterOpts configures optional broadcaster behaviour.
type BroadcasterOpts struct {
	ShutdownGracePeriod time.Duration
}

// NewRedisBroadcaster returns a broadcaster that uses Redis pub/sub for
// cross-instance messaging. Publishers publish to Redis, and each instance's
// subscriber goroutines receive and fan out messages to local subscriptions.
//
// pubc is used for publishing; subc is used for subscribing (a Redis client
// cannot do both once subscribed).
func NewRedisBroadcaster(pubc, subc rueidis.Client, opts ...BroadcasterOpts) Broadcaster {
	gracePeriod := ShutdownGracePeriod
	if len(opts) > 0 && opts[0].ShutdownGracePeriod > 0 {
		gracePeriod = opts[0].ShutdownGracePeriod
	}
	return &broadcaster{
		closing:             0,
		subs:                map[uuid.UUID]*activesub{},
		topics:              map[string]topicsub{},
		l:                   &sync.RWMutex{},
		pubc:                pubc,
		subc:                subc,
		topicCancelFuncs:    map[string]context.CancelFunc{},
		shutdownGracePeriod: gracePeriod,
	}
}

// broadcaster represents a set of subscriptions for one or more topics.
//
// A broadcaster maintains connections to an external realtime subscriber in-memory,
// eg. actual live WebSocket or HTTP connections.  For each connection, it retains which
// topics the subscription is currently interested in.
//
// The broadcaster receives events via Redis Pub/Sub, and pushes the events to
// the Subscription.
//
// When the broadcaster shuts down, it sends a shutdown message to all subscribers.
// Subscribers should reconnect immediately (routed to a healthy broadcaster) to prevent
// lost messages.
type broadcaster struct {
	closing int32

	// l protects subs and topics.
	l *sync.RWMutex

	// subs is a map of subscription IDs to all active topic subscriptions.
	subs map[uuid.UUID]*activesub

	// topics is a list of all subscribed topics, as a skiplist of
	// subscription IDs
	topics map[string]topicsub

	wg conc.WaitGroup

	// pubc is the client connected to Redis for publishing messages.
	pubc rueidis.Client

	// subc is the client connected to Redis for subscribing to pub-sub streams.
	subc rueidis.Client

	// topicCancelMu protects topicCancelFuncs.
	topicCancelMu sync.Mutex

	// topicCancelFuncs maps topic keys to the cancel function for their
	// `runTopic` goroutine. Used by `stopTopic` (last subscriber leaves) and
	// `Close` (broadcaster shutdown) to terminate the Redis subscription.
	topicCancelFuncs map[string]context.CancelFunc

	shutdownGracePeriod time.Duration
}

// topicReady pairs a topic with its ready channel from startTopic, used to
// wait for Redis subscription confirmation after releasing the lock.
type topicReady struct {
	topic Topic
	ready <-chan error
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

	var pendingTopics []topicReady
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
			topicsubs.refCount++
			b.topics[topicHash] = topicsubs

			if topicsubs.refCount == 1 {
				// Launch the goroutine now (fast, no I/O). Collect the ready
				// channel to wait on after releasing the lock.
				pendingTopics = append(pendingTopics, topicReady{
					topic: t,
					ready: b.startTopic(t),
				})
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

	b.l.Unlock()

	// Wait for Redis subscription confirmations outside the lock. Do this
	// outside the lock in case any of the topics take a long time to be ready.
	var anyErr error
	for _, pt := range pendingTopics {
		select {
		case err := <-pt.ready:
			if err != nil {
				anyErr = fmt.Errorf("error starting topic %s: %w", pt.topic.String(), err)
			}
		case <-ctx.Done():
			anyErr = fmt.Errorf("context canceled waiting for topic %s: %w", pt.topic.String(), ctx.Err())
		}
	}

	if anyErr != nil {
		b.rollbackPendingTopics(s, pendingTopics)
		return anyErr
	}

	return nil
}

// rollbackPendingTopics undoes topic and subscription registrations for all
// pending topics after a `Subscribe` failure. It removes the subscription from
// each topic, cleans up the subscriber record, and stops `runTopic` goroutines
// for topics that have no remaining subscribers.
func (b *broadcaster) rollbackPendingTopics(s Subscription, pendingTopics []topicReady) {
	var topicsToStop []Topic
	b.l.Lock()
	for _, pt := range pendingTopics {
		topicHash := pt.topic.String()
		topicsubs, ok := b.topics[topicHash]
		if !ok {
			// Already removed by a concurrent `Unsubscribe`.
			continue
		}
		topicsubs.subscriptions.Delete(skiplistSub{s})
		topicsubs.refCount--
		if topicsubs.refCount == 0 {
			delete(b.topics, topicHash)
			topicsToStop = append(topicsToStop, pt.topic)
		} else {
			b.topics[topicHash] = topicsubs
		}
		if as, ok := b.subs[s.ID()]; ok {
			delete(as.Topics, pt.topic.String())
			if len(as.Topics) == 0 {
				delete(b.subs, s.ID())
			}
		}
	}
	b.l.Unlock()

	for _, t := range topicsToStop {
		b.stopTopic(t)
	}
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

		subs.refCount--
		if subs.refCount == 0 {
			b.stopTopic(t)
			delete(b.topics, str)
		} else {
			// Update the map with new refCount
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

		subs.refCount--
		if subs.refCount == 0 {
			b.stopTopic(t)
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
	if !atomic.CompareAndSwapInt32(&b.closing, 0, 1) {
		return ErrBroadcasterClosed
	}

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

		// After the grace period, close all subscriber connections and cancel
		// all `runTopic` goroutines so `wg.WaitAndRecover` can return.
		time.Sleep(b.shutdownGracePeriod)

		// Close subs
		b.l.RLock()
		for _, s := range b.subs {
			if err := s.Subscription.Close(); err != nil {
				logger.StdlibLogger(ctx).Warn("error closing realtime subscription", "error", err)
			}
		}
		b.l.RUnlock()

		// Cancel topics
		b.topicCancelMu.Lock()
		for _, cancel := range b.topicCancelFuncs {
			cancel()
		}
		clear(b.topicCancelFuncs)
		b.topicCancelMu.Unlock()
	}()

	return nil
}

// Publish publishes a message to Redis pub/sub. The message is delivered to
// local subscriptions via the runTopic subscriber goroutine.
func (b *broadcaster) Publish(ctx context.Context, m Message) {
	metrics.IncrRealtimeMessagesPublishedTotal(ctx, metrics.CounterOpt{
		PkgName: "realtime",
		Tags: map[string]any{
			"broadcaster": "redis",
			"stage":       "ingest",
			"payload":     "structured",
		},
	})

	content, err := json.Marshal(m)
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error marshalling for realtime redis pubsub",
			"error", err,
		)
		return
	}

	pubCtx, cancel := publishCtx(ctx)
	defer cancel()
	for _, t := range m.Topics() {
		b.redisPublish(pubCtx, t.String(), string(content))
	}
}

// Write publishes raw bytes to Redis pub/sub for the given envID and channel.
// It publishes to the well-known "$stream" topic, which is the only topic used
// by the "/realtime/publish/tee" endpoint.
func (b *broadcaster) Write(ctx context.Context, envID uuid.UUID, channel string, data []byte) {
	metrics.IncrRealtimeMessagesPublishedTotal(ctx, metrics.CounterOpt{
		PkgName: "realtime",
		Tags: map[string]any{
			"broadcaster": "redis",
			"stage":       "ingest",
			"payload":     "raw",
		},
	})

	rawMessage := redisRawDataPrefix + string(data)
	key := streamingtypes.Topic{
		Kind:    streamingtypes.TopicKindRun,
		EnvID:   envID,
		Channel: channel,
		Name:    streamingtypes.TopicNameStream,
	}.String()
	pubCtx, cancel := publishCtx(ctx)
	defer cancel()
	b.redisPublish(pubCtx, key, rawMessage)
}

// PublishChunk publishes streams of data to any subscribers for a given
// datastream.
func (b *broadcaster) PublishChunk(ctx context.Context, m Message, c Chunk) {
	content, err := json.Marshal(c)
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error marshalling chunk for realtime redis pubsub",
			"error", err,
		)
		return
	}

	chunkMessage := redisChunkPrefix + string(content)
	pubCtx, cancel := publishCtx(ctx)
	defer cancel()
	for _, t := range m.Topics() {
		b.redisPublish(pubCtx, t.String(), chunkMessage)
	}
}

// publishCtx returns a context for Redis publish operations. It severs the
// caller's cancel signal so that a client disconnect doesn't abort delivery
// mid-publish, but applies a timeout so we don't block indefinitely if Redis is
// down.
func publishCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), redisPublishTimeout)
}

// publishToTopic delivers a structured message to all local subscriptions for a
// specific topic. Used by `runTopic` when receiving messages from Redis.
func (b *broadcaster) publishToTopic(ctx context.Context, t Topic, m Message) {
	b.l.RLock()
	defer b.l.RUnlock()

	metrics.IncrRealtimeMessagesPublishedTotal(ctx, metrics.CounterOpt{
		PkgName: "realtime",
		Tags: map[string]any{
			"stage":   "fanout",
			"payload": "structured",
		},
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

// writeToTopic delivers raw bytes to all local subscriptions for a specific
// topic. Used by `runTopic` when receiving raw messages from Redis.
func (b *broadcaster) writeToTopic(ctx context.Context, t Topic, data []byte) {
	b.l.RLock()
	defer b.l.RUnlock()

	metrics.IncrRealtimeMessagesPublishedTotal(ctx, metrics.CounterOpt{
		PkgName: "realtime",
		Tags: map[string]any{
			"stage":   "fanout",
			"payload": "raw",
		},
	})

	tid := t.String()
	found, ok := b.topics[tid]
	if !ok {
		return
	}

	found.eachSubscription(func(s Subscription) {
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

// publishChunkToTopic delivers a chunk to all local subscriptions for a
// specific topic. Used by `runTopic` when receiving chunk messages from Redis.
func (b *broadcaster) publishChunkToTopic(ctx context.Context, t Topic, c Chunk) {
	b.l.RLock()
	defer b.l.RUnlock()

	metrics.IncrRealtimeMessagesPublishedTotal(ctx, metrics.CounterOpt{
		PkgName: "realtime",
		Tags: map[string]any{
			"stage":   "fanout",
			"payload": "chunk",
		},
	})

	tid := t.String()
	found, ok := b.topics[tid]
	if !ok {
		return
	}

	found.eachSubscription(func(s Subscription) {
		b.publishStreamTo(ctx, s, c)
	})
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
		b.l.RUnlock()
		if !ok {
			return
		}

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

// startTopic is called when the first subscriber connects to a topic. It
// launches the `runTopic` goroutine and returns a channel that signals when the
// Redis subscription is confirmed (or failed). This is safe to call under `b.l`
// because it does not block on I/O: the caller should wait on the returned
// channel after releasing the lock.
func (b *broadcaster) startTopic(t Topic) <-chan error {
	b.topicCancelMu.Lock()
	defer b.topicCancelMu.Unlock()

	key := t.String()
	if _, ok := b.topicCancelFuncs[key]; ok {
		// Already running. Return a pre-resolved channel.
		ch := make(chan error, 1)
		ch <- nil
		return ch
	}

	bgCtx, cancel := context.WithCancel(context.Background())
	b.topicCancelFuncs[key] = cancel

	// runTopic blocks for the lifetime of the subscription (Receive loops
	// until ctx is cancelled), so it must run in a goroutine. The ready
	// channel signals when Redis confirms the subscription.
	ready := make(chan error, 1)
	b.wg.Go(func() {
		b.runTopic(bgCtx, t, ready)
	})
	return ready
}

// stopTopic is called when the last subscriber disconnects from a topic. It
// cancels the background subscriber goroutine.
func (b *broadcaster) stopTopic(t Topic) {
	b.topicCancelMu.Lock()
	defer b.topicCancelMu.Unlock()

	key := t.String()
	if cancel, ok := b.topicCancelFuncs[key]; ok {
		cancel()
		delete(b.topicCancelFuncs, key)
	}
}

// runTopic subscribes to a Redis pub/sub channel for the given topic and
// forwards received messages to local subscriptions. It signals ready once the
// Redis subscription is confirmed.
func (b *broadcaster) runTopic(ctx context.Context, t Topic, ready chan<- error) {
	var signalReady sync.Once

	defer func() {
		if r := recover(); r != nil {
			logger.StdlibLogger(ctx).Error("panic in redis topic subscriber", "panic", r, "topic", t)
			signalReady.Do(func() { ready <- fmt.Errorf("panic in runTopic: %v", r) })
		}

		// Explicitly unsubscribe from the topic to ensure that we don't keep
		// receiving messages on the connection.
		cmd := b.subc.B().Unsubscribe().Channel(t.String()).Build()
		if err := b.subc.Do(context.WithoutCancel(ctx), cmd).Error(); err != nil {
			logger.StdlibLogger(ctx).Warn("failed to unsubscribe from redis topic", "topic", t, "error", err)
		}
	}()

	// Use `WithOnSubscriptionHook` to signal readiness once Redis confirms the
	// subscription.
	hookCtx := rueidis.WithOnSubscriptionHook(ctx, func(s rueidis.PubSubSubscription) {
		signalReady.Do(func() { ready <- nil })
	})

	cmd := b.subc.B().Subscribe().Channel(t.String()).Build()
	err := b.subc.Receive(hookCtx, cmd, func(msg rueidis.PubSubMessage) {
		metrics.IncrRealtimeRedisOpsTotal(ctx, metrics.CounterOpt{
			PkgName: "realtime",
			Tags: map[string]any{
				"op": "receive",
			},
		})

		// Check if this is raw data (prefixed with redisRawDataPrefix)
		if strings.HasPrefix(msg.Message, redisRawDataPrefix) {
			metrics.IncrRealtimeRedisMessageTypesTotal(ctx, metrics.CounterOpt{
				PkgName: "realtime",
				Tags: map[string]any{
					"type": "raw",
				},
			})
			rawData := []byte(msg.Message[len(redisRawDataPrefix):])
			b.writeToTopic(ctx, t, rawData)
			return
		}

		// Check if this is a chunk (prefixed with redisChunkPrefix)
		if strings.HasPrefix(msg.Message, redisChunkPrefix) {
			metrics.IncrRealtimeRedisMessageTypesTotal(ctx, metrics.CounterOpt{
				PkgName: "realtime",
				Tags: map[string]any{
					"type": "chunk",
				},
			})
			var c Chunk
			if err := json.Unmarshal([]byte(msg.Message[len(redisChunkPrefix):]), &c); err != nil {
				logger.StdlibLogger(ctx).Error(
					"error unmarshalling chunk for realtime redis pubsub",
					"error", err,
				)
				return
			}
			b.publishChunkToTopic(ctx, t, c)
			return
		}

		metrics.IncrRealtimeRedisMessageTypesTotal(ctx, metrics.CounterOpt{
			PkgName: "realtime",
			Tags: map[string]any{
				"type": "structured",
			},
		})
		m := Message{}
		err := json.Unmarshal([]byte(msg.Message), &m)
		if err != nil {
			metrics.IncrRealtimeRedisErrorsTotal(ctx, metrics.CounterOpt{
				PkgName: "realtime",
				Tags: map[string]any{
					"op": "unmarshal",
				},
			})
			logger.StdlibLogger(ctx).Error(
				"error unmarshalling for realtime redis pubsub",
				"error", err,
			)
			return
		}
		b.publishToTopic(ctx, t, m)
	})

	// On the happy path the subscription hook already signaled ready and
	// `sync.Once` makes this a no-op. On the sad path (e.g. connection failure
	// before the hook fires) this is the first call, so we forward the error to
	// `startTopic` to unblock it.
	signalReady.Do(func() { ready <- err })

	// We only care about a non-nil error when the context is not cancelled.
	// Context cancellation is part of the happy path: context is cancelled when
	// the last subscriber disconnects (i.e. `stopTopic` calls `cancel()`).
	if err != nil && ctx.Err() == nil {
		logger.StdlibLogger(ctx).Error(
			"redis topic subscription error",
			"error", err,
			"topic", t,
		)
	}
}

// redisPublish publishes a message to a Redis pub/sub channel with retries.
func (b *broadcaster) redisPublish(ctx context.Context, channel, message string) {
	metrics.IncrRealtimeRedisOpsTotal(ctx, metrics.CounterOpt{
		PkgName: "realtime",
		Tags: map[string]any{
			"op": "publish",
		},
	})

	cmd := b.pubc.B().Publish().Channel(channel).Message(message).Build()
	retryTimer := time.NewTimer(0)
	retryTimer.Stop()
	defer retryTimer.Stop()
	for i := 0; i < redisPublishAttempts; i++ {
		if i > 0 {
			metrics.IncrRealtimeRedisOpsTotal(ctx, metrics.CounterOpt{
				PkgName: "realtime",
				Tags: map[string]any{
					"op":     "publish",
					"status": "retry",
				},
			})
			retryTimer.Reset(redisRetryInterval)
			select {
			case <-retryTimer.C:
			case <-ctx.Done():
			}
		}
		if ctx.Err() != nil {
			logger.StdlibLogger(ctx).Error(
				"error publishing to realtime redis pubsub; ctx closed",
				"channel", util.SanitizeLogField(channel),
				"error", ctx.Err(),
				"attempt", i,
			)
			return
		}
		err := b.pubc.Do(ctx, cmd).Error()
		if err == nil {
			return
		}
		logger.StdlibLogger(ctx).Warn(
			"error publishing to realtime redis pubsub",
			"channel", util.SanitizeLogField(channel),
			"error", err,
			"attempt", i,
		)
	}
	logger.StdlibLogger(ctx).Warn(
		"failed to publish via realtime redis pubsub",
		"channel", util.SanitizeLogField(channel),
	)
	metrics.IncrRealtimeRedisErrorsTotal(ctx, metrics.CounterOpt{
		PkgName: "realtime",
		Tags: map[string]any{
			"op": "publish",
		},
	})
}

//
// Data types
//

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
	refCount      int
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
