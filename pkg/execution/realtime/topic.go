package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/redis/rueidis"
)

// readyErr is published by closing ready; every initial subscriber observes
// the same result. done includes Redis cleanup, not just Receive returning.
type topicWorker struct {
	ctx         context.Context
	cancel      context.CancelFunc
	ready       chan struct{}
	readyOnce   sync.Once
	readyErr    error
	initialized atomic.Bool
	done        chan struct{}
}

func (w *topicWorker) signalReady(err error) {
	w.readyOnce.Do(func() {
		w.readyErr = err
		close(w.ready)
	})
}

// startTopic and stopTopic are called with b.l held.
func (b *broadcaster) startTopic(t Topic) *topicWorker {
	previous := b.topicWorkers[t.String()]
	ctx, cancel := context.WithCancel(context.Background())
	worker := &topicWorker{
		ctx: ctx, cancel: cancel,
		ready: make(chan struct{}), done: make(chan struct{}),
	}
	b.topicWorkers[t.String()] = worker
	b.wg.Go(func() { b.runTopic(t, worker, previous) })
	return worker
}

func (b *broadcaster) stopTopic(t Topic) {
	if worker := b.topicWorkers[t.String()]; worker != nil {
		worker.cancel()
	}
}

func (b *broadcaster) runTopic(t Topic, worker, previous *topicWorker) {
	attempted := false
	defer func() {
		if r := recover(); r != nil {
			worker.signalReady(fmt.Errorf("panic in runTopic: %v", r))
			logger.StdlibLogger(worker.ctx).Error("panic in redis topic subscriber", "panic", r)
		}
		var subs []Subscription
		if worker.initialized.Load() && worker.ctx.Err() == nil {
			subs = b.detachFailedTopic(t, worker)
		}
		worker.cancel()
		worker.signalReady(worker.ctx.Err())
		if previous != nil {
			// Even a cancelled replacement carries its predecessor's cleanup
			// barrier forward to a third generation of this topic.
			<-previous.done
		}
		if attempted {
			// A disconnected Redis must not hold up topic replacement forever.
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			cmd := b.subc.B().Unsubscribe().Channel(t.String()).Build()
			if err := b.subc.Do(ctx, cmd).Error(); err != nil {
				logger.StdlibLogger(ctx).Warn("failed to unsubscribe from redis topic", "error", err)
			}
			cancel()
		}
		b.l.Lock()
		if b.topicWorkers[t.String()] == worker {
			delete(b.topicWorkers, t.String())
		}
		b.l.Unlock()
		close(worker.done)
		b.closeSubscriptions(context.Background(), subs)
	}()

	if previous != nil {
		// In particular, wait for the old UNSUBSCRIBE before sending SUBSCRIBE.
		select {
		case <-previous.done:
		case <-worker.ctx.Done():
			return
		}
	}
	if worker.ctx.Err() != nil {
		return
	}
	attempted = true
	err := b.receiveTopic(worker.ctx, t, func() {
		worker.initialized.Store(true)
		worker.signalReady(nil)
	})
	if !worker.initialized.Load() {
		if err == nil {
			err = fmt.Errorf("Redis subscription ended before confirmation")
		}
		worker.signalReady(err)
		return
	}
	for worker.ctx.Err() == nil {
		metrics.IncrRealtimeRedisErrorsTotal(worker.ctx, metrics.CounterOpt{
			PkgName: "realtime", Tags: map[string]any{"op": "subscribe"},
		})
		logger.StdlibLogger(worker.ctx).Warn("redis topic subscription interrupted", "error", err, "topic", t)
		var recovered bool
		recovered, err = b.recoverTopic(worker.ctx, t)
		if !recovered {
			return
		}
	}
}

// recoverTopic bounds time until confirmation, rather than the lifetime of a
// healthy Receive. Rueidis may itself retry indefinitely before confirming.
func (b *broadcaster) recoverTopic(ctx context.Context, t Topic) (bool, error) {
	started := time.Now()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	timer := time.AfterFunc(b.subscriptionRecoveryTimeout, cancel)
	defer timer.Stop()
	var confirmed atomic.Bool
	backoff := 100 * time.Millisecond
	for ctx.Err() == nil {
		delay := time.NewTimer(backoff/2 + time.Duration(rand.Int64N(int64(backoff/2))))
		select {
		case <-ctx.Done():
			delay.Stop()
			return false, ctx.Err()
		case <-delay.C:
		}
		metrics.IncrRealtimeSubscriptionRecoveryTotal(ctx, metrics.CounterOpt{
			PkgName: "realtime", Tags: map[string]any{"result": "attempt"},
		})
		err := b.receiveTopic(ctx, t, func() {
			if ctx.Err() == nil && !confirmed.Load() && timer.Stop() && confirmed.CompareAndSwap(false, true) {
				metrics.IncrRealtimeSubscriptionRecoveryTotal(ctx, metrics.CounterOpt{
					PkgName: "realtime", Tags: map[string]any{"result": "recovered"},
				})
				metrics.HistogramRealtimeSubscriptionRecoveryDuration(ctx, time.Since(started).Milliseconds(), metrics.HistogramOpt{PkgName: "realtime"})
			}
		})
		if confirmed.Load() {
			return true, err
		}
		backoff = min(backoff*2, 2*time.Second)
	}
	return false, ctx.Err()
}

func (b *broadcaster) detachFailedTopic(t Topic, worker *topicWorker) []Subscription {
	b.l.Lock()
	defer b.l.Unlock()
	topic, ok := b.topics[t.String()]
	if !ok || topic.worker != worker {
		return nil
	}
	var ids []uuid.UUID
	topic.eachSubscription(func(s Subscription) { ids = append(ids, s.ID()) })
	var subs []Subscription
	for _, id := range ids {
		if sub := b.removeSubscription(id); sub != nil {
			subs = append(subs, sub)
		}
	}
	metrics.IncrRealtimeSubscriptionRecoveryTotal(context.Background(), metrics.CounterOpt{
		PkgName: "realtime", Tags: map[string]any{"result": "exhausted"},
	})
	return subs
}

func (b *broadcaster) receiveTopic(ctx context.Context, t Topic, onReady func()) error {
	hookCtx := rueidis.WithOnSubscriptionHook(ctx, func(s rueidis.PubSubSubscription) {
		if s.Kind == "subscribe" && s.Channel == t.String() {
			onReady()
		}
	})
	cmd := b.subc.B().Subscribe().Channel(t.String()).Build()
	return b.subc.Receive(hookCtx, cmd, func(msg rueidis.PubSubMessage) {
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

}
