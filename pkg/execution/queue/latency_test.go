package queue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestLatencyQueueName(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{1, "ffffffff-ffff-ffff-ffff-fffffffffff1"},
		{2, "ffffffff-ffff-ffff-ffff-fffffffffff2"},
		{10, "ffffffff-ffff-ffff-ffff-fffffffffffa"},
		{15, "ffffffff-ffff-ffff-ffff-ffffffffffff"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.expected, latencyQueueName(tt.n))
	}
}

func TestIsLatencyPartition(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{latencyQueueName(1), true},
		{latencyQueueName(2), true},
		{latencyQueueName(15), true},
		{"ffffffff-ffff-ffff-ffff-fffffffffff0", true},
		{"ffffffff-ffff-ffff-ffff-ffffffffffff", true},
		// Real UUIDs should never match.
		{"00000000-0000-0000-0000-000000000000", false},
		{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", false},
		// Differs before the final suffix — not a latency partition.
		{"ffffffff-ffff-ffff-ffff-ffffffffffe1", false},
		// Empty and short strings.
		{"", false},
		{"ffffffff", false},
	}
	for _, tt := range tests {
		require.Equal(t, tt.expected, IsLatencyPartition(tt.id), "IsLatencyPartition(%q)", tt.id)
	}
}

func TestWithLatencyPartition(t *testing.T) {
	t.Run("sets defaults", func(t *testing.T) {
		called := false
		opts := NewQueueOptions(WithLatencyPartition(LatencyPartitionOptions{
			Callback: func(ctx context.Context, info RunInfo) {
				called = true
			},
		}))
		require.NotNil(t, opts.latencyPartition)
		require.Equal(t, 1, opts.latencyPartition.Partitions)
		require.Equal(t, 5*time.Second, opts.latencyPartition.Interval)
		require.NotNil(t, opts.latencyPartition.Callback)

		// Verify callback is the one we passed
		opts.latencyPartition.Callback(context.Background(), RunInfo{})
		require.True(t, called)
	})

	t.Run("forces single partition", func(t *testing.T) {
		opts := NewQueueOptions(WithLatencyPartition(LatencyPartitionOptions{
			Partitions: 5,
			Interval:   time.Second,
		}))
		require.Equal(t, 1, opts.latencyPartition.Partitions)
	})

	t.Run("respects custom interval", func(t *testing.T) {
		opts := NewQueueOptions(WithLatencyPartition(LatencyPartitionOptions{
			Interval: 10 * time.Second,
		}))
		require.Equal(t, 10*time.Second, opts.latencyPartition.Interval)
	})

	t.Run("defaults negative interval", func(t *testing.T) {
		opts := NewQueueOptions(WithLatencyPartition(LatencyPartitionOptions{
			Interval: -1 * time.Second,
		}))
		require.Equal(t, 5*time.Second, opts.latencyPartition.Interval)
	})
}

func TestWrapRunFuncWithLatency(t *testing.T) {
	t.Run("returns original func when no latency config", func(t *testing.T) {
		qp := &queueProcessor{
			QueueOptions: NewQueueOptions(),
		}
		original := func(ctx context.Context, info RunInfo, item Item) (RunResult, error) {
			return RunResult{}, nil
		}
		wrapped := qp.wrapRunFuncWithLatency(original)

		// When latencyPartition is nil, the wrapped func should be the same pointer.
		// We can't compare func pointers directly, so just verify it works.
		res, err := wrapped(context.Background(), RunInfo{}, Item{Kind: KindEdge})
		require.NoError(t, err)
		require.Equal(t, RunResult{}, res)
	})

	t.Run("returns original func when callback is nil", func(t *testing.T) {
		qp := &queueProcessor{
			QueueOptions: NewQueueOptions(WithLatencyPartition(LatencyPartitionOptions{
				Interval: time.Second,
				Callback: nil,
			})),
		}
		original := func(ctx context.Context, info RunInfo, item Item) (RunResult, error) {
			return RunResult{ScheduledImmediateJob: true}, nil
		}
		wrapped := qp.wrapRunFuncWithLatency(original)

		res, err := wrapped(context.Background(), RunInfo{}, Item{Kind: KindEdge})
		require.NoError(t, err)
		require.True(t, res.ScheduledImmediateJob)
	})

	t.Run("intercepts latency tracking items", func(t *testing.T) {
		var capturedInfo RunInfo
		qp := &queueProcessor{
			QueueOptions: NewQueueOptions(WithLatencyPartition(LatencyPartitionOptions{
				Interval: time.Second,
				Callback: func(ctx context.Context, info RunInfo) {
					capturedInfo = info
				},
			})),
		}

		originalCalled := false
		original := func(ctx context.Context, info RunInfo, item Item) (RunResult, error) {
			originalCalled = true
			return RunResult{}, nil
		}
		wrapped := qp.wrapRunFuncWithLatency(original)

		expectedInfo := RunInfo{
			Latency:      42 * time.Millisecond,
			SojournDelay: 10 * time.Millisecond,
		}
		res, err := wrapped(context.Background(), expectedInfo, Item{Kind: KindLatencyTrack})

		require.NoError(t, err)
		require.Equal(t, RunResult{}, res)
		require.Equal(t, expectedInfo, capturedInfo)
		require.False(t, originalCalled, "original RunFunc should not be called for latency items")
	})

	t.Run("passes non-latency items to original func", func(t *testing.T) {
		qp := &queueProcessor{
			QueueOptions: NewQueueOptions(WithLatencyPartition(LatencyPartitionOptions{
				Interval: time.Second,
				Callback: func(ctx context.Context, info RunInfo) {
					t.Fatal("callback should not be called for non-latency items")
				},
			})),
		}

		originalCalled := false
		original := func(ctx context.Context, info RunInfo, item Item) (RunResult, error) {
			originalCalled = true
			return RunResult{ScheduledImmediateJob: true}, nil
		}
		wrapped := qp.wrapRunFuncWithLatency(original)

		for _, kind := range []string{KindStart, KindEdge, KindSleep, KindPause, KindDebounce} {
			originalCalled = false
			res, err := wrapped(context.Background(), RunInfo{}, Item{Kind: kind})
			require.NoError(t, err)
			require.True(t, originalCalled, "original RunFunc should be called for kind %q", kind)
			require.True(t, res.ScheduledImmediateJob)
		}
	})
}

func TestRunLatencyTracker(t *testing.T) {
	t.Run("exits immediately when no latency config", func(t *testing.T) {
		qp := &queueProcessor{
			QueueOptions: NewQueueOptions(),
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Should return immediately without blocking.
		done := make(chan struct{})
		go func() {
			qp.runLatencyTracker(ctx)
			close(done)
		}()

		select {
		case <-done:
			// OK
		case <-time.After(time.Second):
			t.Fatal("runLatencyTracker should return immediately when latencyPartition is nil")
		}
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		fakeClock := clockwork.NewFakeClock()
		qp := &queueProcessor{
			QueueOptions: NewQueueOptions(
				WithLatencyPartition(LatencyPartitionOptions{
					Interval: 5 * time.Second,
					Callback: func(ctx context.Context, info RunInfo) {},
				}),
				WithClock(fakeClock),
			),
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			qp.runLatencyTracker(ctx)
			close(done)
		}()

		// Cancel context
		cancel()

		select {
		case <-done:
			// OK
		case <-time.After(time.Second):
			t.Fatal("runLatencyTracker should exit on context cancellation")
		}
	})
}

func TestEnqueueLatencyJob(t *testing.T) {
	t.Run("enqueues item with correct fields", func(t *testing.T) {
		fakeClock := clockwork.NewFakeClock()

		var enqueuedItem Item
		var enqueuedAt time.Time
		var enqueuedOpts EnqueueOpts
		var enqueueCalled atomic.Int32

		shard := &mockShardForIterator{name: "test"}
		qp := &queueProcessor{
			QueueOptions: NewQueueOptions(
				WithLatencyPartition(LatencyPartitionOptions{
					Partitions: 1,
					Interval:   time.Second,
					Callback:   func(ctx context.Context, info RunInfo) {},
				}),
				WithClock(fakeClock),
			),
			primaryQueueShard: shard,
			queueShardClients: map[string]QueueShard{"test": shard},
			shardSelector: func(ctx context.Context, accountId uuid.UUID, queueName *string) (QueueShard, error) {
				return shard, nil
			},
		}

		// Monkey-patch by wrapping: we can't easily mock Enqueue on queueProcessor
		// since it calls shard.EnqueueItem, which our mock already stubs.
		// Instead, verify the mock shard receives the call.

		// The mockShardForIterator.EnqueueItem is a stub that returns (i, nil).
		// We need a custom shard to capture calls.
		captureShard := &capturingShardForLatency{
			mockShardForIterator: mockShardForIterator{name: "test"},
			onEnqueue: func(item QueueItem, at time.Time, opts EnqueueOpts) {
				enqueuedItem = item.Data
				enqueuedAt = at
				enqueuedOpts = opts
				enqueueCalled.Add(1)
			},
		}

		qp.primaryQueueShard = captureShard
		qp.queueShardClients = map[string]QueueShard{"test": captureShard}
		qp.shardSelector = func(ctx context.Context, accountId uuid.UUID, queueName *string) (QueueShard, error) {
			return captureShard, nil
		}

		err := qp.enqueueLatencyJob(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, int32(1), enqueueCalled.Load())

		require.Equal(t, KindLatencyTrack, enqueuedItem.Kind)
		require.NotNil(t, enqueuedItem.QueueName)
		require.Equal(t, latencyQueueName(1), *enqueuedItem.QueueName)
		require.NotNil(t, enqueuedItem.JobID)
		require.Contains(t, *enqueuedItem.JobID, "ltrack-1-")
		// Enqueue truncates times to millisecond precision internally.
		require.Equal(t, fakeClock.Now().UnixMilli(), enqueuedAt.UnixMilli())
		require.NotNil(t, enqueuedOpts.IdempotencyPeriod)
		require.Equal(t, time.Second, *enqueuedOpts.IdempotencyPeriod)
	})
}

func TestWrapRunFuncWithLatencyConcurrency(t *testing.T) {
	// Verify the wrapper is safe for concurrent access.
	var mu sync.Mutex
	var infos []RunInfo

	qp := &queueProcessor{
		QueueOptions: NewQueueOptions(WithLatencyPartition(LatencyPartitionOptions{
			Interval: time.Second,
			Callback: func(ctx context.Context, info RunInfo) {
				mu.Lock()
				infos = append(infos, info)
				mu.Unlock()
			},
		})),
	}

	var originalCalls atomic.Int32
	original := func(ctx context.Context, info RunInfo, item Item) (RunResult, error) {
		originalCalls.Add(1)
		return RunResult{}, nil
	}
	wrapped := qp.wrapRunFuncWithLatency(original)

	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				// Latency tracking item
				_, _ = wrapped(context.Background(), RunInfo{
					Latency: time.Duration(i) * time.Millisecond,
				}, Item{Kind: KindLatencyTrack})
			} else {
				// Regular item
				_, _ = wrapped(context.Background(), RunInfo{}, Item{Kind: KindEdge})
			}
		}(i)
	}
	wg.Wait()

	mu.Lock()
	require.Len(t, infos, n/2)
	mu.Unlock()
	require.Equal(t, int32(n/2), originalCalls.Load())
}

// capturingShardForLatency extends mockShardForIterator to capture EnqueueItem calls.
type capturingShardForLatency struct {
	mockShardForIterator
	onEnqueue func(item QueueItem, at time.Time, opts EnqueueOpts)
}

func (c *capturingShardForLatency) EnqueueItem(ctx context.Context, i QueueItem, at time.Time, opts EnqueueOpts) (QueueItem, error) {
	if c.onEnqueue != nil {
		c.onEnqueue(i, at, opts)
	}
	return i, nil
}
