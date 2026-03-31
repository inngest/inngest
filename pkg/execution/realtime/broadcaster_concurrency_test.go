package realtime

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBroadcasterConcurrency(t *testing.T) {
	t.Run("Concurrent subscriptions to same topic trigger hook correctly", func(t *testing.T) {
		b := NewInProcessBroadcaster()
		var startCalls int32
		var stopCalls int32

		// Add delays to simulate work and increase race window
		b.TopicStart = func(ctx context.Context, t Topic) error {
			atomic.AddInt32(&startCalls, 1)
			time.Sleep(5 * time.Millisecond)
			return nil
		}
		b.TopicStop = func(ctx context.Context, t Topic) error {
			atomic.AddInt32(&stopCalls, 1)
			time.Sleep(5 * time.Millisecond)
			return nil
		}

		topic := Topic{Name: "shared-topic"}
		topics := []Topic{topic}
		concurrency := 50
		subs := make([]Subscription, concurrency)
		for i := 0; i < concurrency; i++ {
			subs[i] = NewInmemorySubscription(uuid.New(), nil)
		}

		wg := sync.WaitGroup{}

		// Phase 1: Concurrent Subscribe
		// We expect exactly 1 Start call because they are all the same topic and we hold the lock during the critical section.
		// However, the lock is per-broadcaster.
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				err := b.Subscribe(context.Background(), subs[i], topics)
				require.NoError(t, err)
			}(i)
		}
		wg.Wait()

		require.Equal(t, int32(1), atomic.LoadInt32(&startCalls), "TopicStart should be called exactly once")
		require.Equal(t, int32(0), atomic.LoadInt32(&stopCalls), "TopicStop should not be called yet")

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

		require.Equal(t, int32(1), atomic.LoadInt32(&startCalls))
		require.Equal(t, int32(1), atomic.LoadInt32(&stopCalls), "TopicStop should be called exactly once after all unsubscribes")
	})

	t.Run("TopicStart failure cleans up state", func(t *testing.T) {
		b := NewInProcessBroadcaster()
		b.TopicStart = func(ctx context.Context, t Topic) error {
			return errors.New("redis connection failed")
		}

		sub := NewInmemorySubscription(uuid.New(), nil)
		topic := Topic{Name: "fail-topic"}

		// 1. Subscribe fails
		err := b.Subscribe(context.Background(), sub, []Topic{topic})
		require.Error(t, err)
		require.Contains(t, err.Error(), "redis connection failed")

		// 2. Ensure we can retry and succeed if error resolves
		b.TopicStart = func(ctx context.Context, t Topic) error { return nil }
		err = b.Subscribe(context.Background(), sub, []Topic{topic})
		require.NoError(t, err)

		// 3. Ensure Unsubscribe triggers Stop (proving we are in a valid state)
		stopCalled := false
		b.TopicStop = func(ctx context.Context, t Topic) error {
			stopCalled = true
			return nil
		}
		err = b.Unsubscribe(context.Background(), sub.ID(), []Topic{topic})
		require.NoError(t, err)
		require.True(t, stopCalled)
	})

	t.Run("Interleaved Subscribe and Unsubscribe", func(t *testing.T) {
		// This test tries to break the RefCount by interleaving adds and removes.
		b := NewInProcessBroadcaster()
		var activeTopics int32

		b.TopicStart = func(ctx context.Context, t Topic) error {
			atomic.AddInt32(&activeTopics, 1)
			return nil
		}
		b.TopicStop = func(ctx context.Context, t Topic) error {
			atomic.AddInt32(&activeTopics, -1)
			return nil
		}

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
					// Subscribe
					err := b.Subscribe(context.Background(), sub, topics)
					require.NoError(t, err)

					// Slight random delay?
					// time.Sleep(time.Microsecond)

					// Unsubscribe
					err = b.Unsubscribe(context.Background(), sub.ID(), topics)
					require.NoError(t, err)
				}
			}()
		}
		wg.Wait()

		// After all churning, active topics should be 0
		require.Equal(t, int32(0), atomic.LoadInt32(&activeTopics), "Active topics should be 0 after churn")
	})
}
