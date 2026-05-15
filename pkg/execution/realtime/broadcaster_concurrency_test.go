package realtime

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBroadcasterConcurrency(t *testing.T) {
	t.Run("Concurrent subscriptions to same topic manage refcount correctly", func(t *testing.T) {
		b := newTestBroadcaster(t).(*broadcaster)

		topic := Topic{Name: "shared-topic"}
		topics := []Topic{topic}
		concurrency := 50
		subs := make([]Subscription, concurrency)
		for i := 0; i < concurrency; i++ {
			subs[i] = NewInmemorySubscription(uuid.New(), nil)
		}

		wg := sync.WaitGroup{}

		// Phase 1: Concurrent Subscribe
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				err := b.Subscribe(context.Background(), subs[i], topics)
				require.NoError(t, err)
			}(i)
		}
		wg.Wait()

		// All subscriptions should exist and the topic should have the correct refcount.
		b.l.RLock()
		require.Equal(t, concurrency, len(b.subs), "Should have all subscriptions")
		topicHash := topic.String()
		ts, ok := b.topics[topicHash]
		require.True(t, ok, "Topic should exist")
		require.Equal(t, concurrency, ts.refCount, "refCount should match number of subscribers")
		b.l.RUnlock()

		// Phase 2: Concurrent Unsubscribe
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				err := b.Unsubscribe(context.Background(), subs[i].ID(), topics)
				require.NoError(t, err)
			}(i)
		}
		wg.Wait()

		// After all unsubscribes, the topic should be removed entirely.
		b.l.RLock()
		_, ok = b.topics[topicHash]
		require.False(t, ok, "Topic should be removed after all unsubscribes")
		b.l.RUnlock()
	})

	t.Run("Interleaved Subscribe and Unsubscribe", func(t *testing.T) {
		// This test tries to break the refCount by interleaving adds and removes.
		b := newTestBroadcaster(t).(*broadcaster)

		topic := Topic{Name: "churn-topic"}
		topics := []Topic{topic}
		concurrency := 20
		iterations := 100

		wg := sync.WaitGroup{}
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sub := NewInmemorySubscription(uuid.New(), nil)
				for j := 0; j < iterations; j++ {
					err := b.Subscribe(context.Background(), sub, topics)
					require.NoError(t, err)

					err = b.Unsubscribe(context.Background(), sub.ID(), topics)
					require.NoError(t, err)
				}
			}()
		}
		wg.Wait()

		// After all churning, the topic should be removed.
		b.l.RLock()
		_, ok := b.topics[topic.String()]
		b.l.RUnlock()
		require.False(t, ok, "Topic should be removed after churn")
	})
}
