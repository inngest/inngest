package batch

import (
	"context"
	"crypto/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("github.com/karlseguin/ccache/v2.(*Cache).worker"),
	)
}

func TestBulkAppend(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	bm := NewRedisBatchManager(bc, nil)

	// Cast to *redisBatchManager to access BulkAppend
	rbm, ok := bm.(*redisBatchManager)
	require.True(t, ok)

	accountId := uuid.New()
	workspaceId := uuid.New()
	appId := uuid.New()
	fnId := uuid.New()

	fn := inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
		},
	}

	t.Run("bulk append creates new batch", func(t *testing.T) {
		items := make([]BatchItem, 3)
		for i := 0; i < 3; i++ {
			items[i] = BatchItem{
				AccountID:       accountId,
				WorkspaceID:     workspaceId,
				AppID:           appId,
				FunctionID:      fnId,
				FunctionVersion: 1,
				EventID:         ulid.MustNew(ulid.Now(), rand.Reader),
				Event: event.Event{
					Name: "test/event",
					Data: map[string]any{"index": i},
				},
			}
		}

		result, err := rbm.BulkAppend(context.Background(), items, fn)
		require.NoError(t, err)
		require.Equal(t, "new", result.Status)
		require.NotEmpty(t, result.BatchID)
		require.Equal(t, 3, result.Committed)
		require.Equal(t, 0, result.Duplicates)

		// Verify items are in Redis
		info, err := rbm.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Len(t, info.Items, 3)
	})

	t.Run("bulk append with duplicates", func(t *testing.T) {
		newFnId := uuid.New()
		fn := inngest.Function{
			ID: newFnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		eventID := ulid.MustNew(ulid.Now(), rand.Reader)
		items := []BatchItem{
			{
				AccountID:   accountId,
				WorkspaceID: workspaceId,
				AppID:       appId,
				FunctionID:  newFnId,
				EventID:     eventID,
				Event:       event.Event{Name: "test/event", Data: map[string]any{"a": 1}},
			},
			{
				AccountID:   accountId,
				WorkspaceID: workspaceId,
				AppID:       appId,
				FunctionID:  newFnId,
				EventID:     eventID, // Duplicate
				Event:       event.Event{Name: "test/event", Data: map[string]any{"a": 2}},
			},
			{
				AccountID:   accountId,
				WorkspaceID: workspaceId,
				AppID:       appId,
				FunctionID:  newFnId,
				EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
				Event:       event.Event{Name: "test/event", Data: map[string]any{"a": 3}},
			},
		}

		result, err := rbm.BulkAppend(context.Background(), items, fn)
		require.NoError(t, err)
		require.Equal(t, "new", result.Status)
		require.Equal(t, 2, result.Committed)
		require.Equal(t, 1, result.Duplicates)

		// Verify only 2 items in Redis
		info, err := rbm.GetBatchInfo(context.Background(), newFnId, "")
		require.NoError(t, err)
		require.Len(t, info.Items, 2)
	})

	t.Run("bulk append fills batch", func(t *testing.T) {
		newFnId := uuid.New()
		fn := inngest.Function{
			ID: newFnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 5,
				Timeout: "60s",
			},
		}

		items := make([]BatchItem, 5)
		for i := 0; i < 5; i++ {
			items[i] = BatchItem{
				AccountID:   accountId,
				WorkspaceID: workspaceId,
				AppID:       appId,
				FunctionID:  newFnId,
				EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
				Event:       event.Event{Name: "test/event", Data: map[string]any{"i": i}},
			}
		}

		result, err := rbm.BulkAppend(context.Background(), items, fn)
		require.NoError(t, err)
		require.Equal(t, "full", result.Status)
		require.Equal(t, 5, result.Committed)
	})

	t.Run("bulk append with overflow", func(t *testing.T) {
		newFnId := uuid.New()
		fn := inngest.Function{
			ID: newFnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 3,
				Timeout: "60s",
			},
		}

		// Add 5 items to a batch with max size 3
		items := make([]BatchItem, 5)
		for i := 0; i < 5; i++ {
			items[i] = BatchItem{
				AccountID:   accountId,
				WorkspaceID: workspaceId,
				AppID:       appId,
				FunctionID:  newFnId,
				EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
				Event:       event.Event{Name: "test/event", Data: map[string]any{"i": i}},
			}
		}

		result, err := rbm.BulkAppend(context.Background(), items, fn)
		require.NoError(t, err)
		require.Equal(t, "overflow", result.Status)
		require.Equal(t, 5, result.Committed)
		require.Equal(t, 2, result.OverflowCount)
		require.NotEmpty(t, result.NextBatchID)
		require.NotEqual(t, result.BatchID, result.NextBatchID)
	})
}

func TestBufferedBatchManager(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)

	t.Run("blocking append with timer flush", func(t *testing.T) {
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(100*time.Millisecond, 100), // High size so timer triggers flush
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		bi := BatchItem{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			FunctionID:  fnId,
			EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
			Event:       event.Event{Name: "test/event", Data: map[string]any{"a": 1}},
		}

		start := time.Now()
		result, err := buffered.Append(context.Background(), bi, fn)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, enums.BatchAppend, result.Status) // Buffer handles scheduling internally

		// Should have waited for timer (100ms)
		require.GreaterOrEqual(t, elapsed, 90*time.Millisecond)

		// Verify item is in Redis
		info, err := buffered.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Len(t, info.Items, 1)
	})

	t.Run("blocking append with size flush", func(t *testing.T) {
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(5*time.Second, 3), // Long timer so size triggers flush
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		var wg sync.WaitGroup
		results := make([]*BatchAppendResult, 3)
		errors := make([]error, 3)

		// Launch 3 goroutines that will all block until buffer flushes
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				bi := BatchItem{
					AccountID:   uuid.New(),
					WorkspaceID: uuid.New(),
					AppID:       uuid.New(),
					FunctionID:  fnId,
					EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
					Event:       event.Event{Name: "test/event", Data: map[string]any{"idx": idx}},
				}
				results[idx], errors[idx] = buffered.Append(context.Background(), bi, fn)
			}(i)
		}

		wg.Wait()

		// All should succeed
		for i := 0; i < 3; i++ {
			require.NoError(t, errors[i])
			require.NotNil(t, results[i])
		}

		// Verify all items are in Redis
		info, err := buffered.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Len(t, info.Items, 3)
	})

	t.Run("local dedup blocks until flush then returns item exists", func(t *testing.T) {
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(100*time.Millisecond, 100),
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		eventID := ulid.MustNew(ulid.Now(), rand.Reader)
		bi := BatchItem{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			FunctionID:  fnId,
			EventID:     eventID,
			Event:       event.Event{Name: "test/event"},
		}

		// First append starts blocking (we don't wait for it)
		go func() {
			_, appendErr := buffered.Append(context.Background(), bi, fn)
			require.NoError(t, appendErr)
		}()

		// Give time for first append to add to buffer
		time.Sleep(10 * time.Millisecond)

		// Second append with same event ID should block until flush
		// completes and then return BatchItemExists
		result, err := buffered.Append(context.Background(), bi, fn)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, enums.BatchItemExists, result.Status)
	})

	t.Run("context cancellation unblocks", func(t *testing.T) {
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(5*time.Second, 100),
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		bi := BatchItem{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			FunctionID:  fnId,
			EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
			Event:       event.Event{Name: "test/event"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		result, err := buffered.Append(ctx, bi, fn)
		elapsed := time.Since(start)

		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Less(t, elapsed, 200*time.Millisecond, "should unblock on context cancel")
	})

	t.Run("close flushes pending buffers", func(t *testing.T) {
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(5*time.Second, 100),
		)

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		bi := BatchItem{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			FunctionID:  fnId,
			EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
			Event:       event.Event{Name: "test/event"},
		}

		var appendErr error
		var appendResult *BatchAppendResult
		done := make(chan struct{})

		go func() {
			appendResult, appendErr = buffered.Append(context.Background(), bi, fn)
			close(done)
		}()

		// Give time for append to add to buffer
		time.Sleep(20 * time.Millisecond)

		// Close should flush the buffer
		closeErr := buffered.Close()
		require.NoError(t, closeErr)

		// Wait for append to complete
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("append did not complete after close")
		}

		require.NoError(t, appendErr)
		require.NotNil(t, appendResult)

		// Verify item is in Redis - use a fresh manager without buffering for verification
		verifier := NewRedisBatchManager(bc, nil)
		defer verifier.Close()
		info, err := verifier.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Len(t, info.Items, 1)
	})

	t.Run("without buffering appends directly", func(t *testing.T) {
		// Test that without WithBuffering, Append goes directly to Redis (no blocking)
		direct := NewRedisBatchManager(bc, nil, WithoutBuffer())
		defer direct.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		bi := BatchItem{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			FunctionID:  fnId,
			EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
			Event:       event.Event{Name: "test/event"},
		}

		start := time.Now()
		result, err := direct.Append(context.Background(), bi, fn)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should be fast (no buffering delay)
		require.Less(t, elapsed, 50*time.Millisecond)

		// Verify item is in Redis
		info, err := direct.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Len(t, info.Items, 1)
	})
}

func TestBufferedBatchManagerConcurrency(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	buffered := NewRedisBatchManager(bc, nil,
		WithBufferSettings(50*time.Millisecond, 20),
	)
	defer buffered.Close()

	fnId := uuid.New()
	fn := inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 100,
			Timeout: "60s",
		},
	}

	const numGoroutines = 50
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errorCount atomic.Int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			bi := BatchItem{
				AccountID:   uuid.New(),
				WorkspaceID: uuid.New(),
				AppID:       uuid.New(),
				FunctionID:  fnId,
				EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
				Event:       event.Event{Name: "test/event", Data: map[string]any{"idx": idx}},
			}
			result, err := buffered.Append(context.Background(), bi, fn)
			if err != nil {
				errorCount.Add(1)
				return
			}
			if result != nil && result.Status != enums.BatchItemExists {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	require.Equal(t, int32(0), errorCount.Load(), "should have no errors")
	require.Equal(t, int32(numGoroutines), successCount.Load(), "all appends should succeed")

	// Verify all items are in Redis
	info, err := buffered.GetBatchInfo(context.Background(), fnId, "")
	require.NoError(t, err)
	require.Len(t, info.Items, numGoroutines)
}

// TestBufferedFlushDurationClamping verifies that the buffer flush duration is clamped
// to the function's batch timeout when it's shorter than the buffer's maxDuration.
func TestBufferedFlushDurationClamping(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)

	t.Run("flushes at function batch timeout when shorter than buffer max", func(t *testing.T) {
		// Buffer has long maxDuration, but function has short batch timeout
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(5*time.Second, 100), // Long buffer duration
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "100ms", // Short batch timeout - should clamp flush duration
			},
		}

		bi := BatchItem{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			FunctionID:  fnId,
			EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
			Event:       event.Event{Name: "test/event", Data: map[string]any{"a": 1}},
		}

		start := time.Now()
		result, err := buffered.Append(context.Background(), bi, fn)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, enums.BatchAppend, result.Status) // Buffer handles scheduling internally

		// Should have flushed at ~100ms (function timeout), not 5s (buffer max)
		require.GreaterOrEqual(t, elapsed, 90*time.Millisecond, "should wait at least near the batch timeout")
		require.Less(t, elapsed, 500*time.Millisecond, "should not wait anywhere near the buffer maxDuration")
	})

	t.Run("uses buffer max when function timeout is longer", func(t *testing.T) {
		// Buffer has short maxDuration, function has long batch timeout
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(100*time.Millisecond, 100), // Short buffer duration
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s", // Long batch timeout - buffer max should be used
			},
		}

		bi := BatchItem{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			FunctionID:  fnId,
			EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
			Event:       event.Event{Name: "test/event", Data: map[string]any{"a": 1}},
		}

		start := time.Now()
		result, err := buffered.Append(context.Background(), bi, fn)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, enums.BatchAppend, result.Status) // Buffer handles scheduling internally

		// Should have flushed at ~100ms (buffer max), not waiting for 60s
		require.GreaterOrEqual(t, elapsed, 90*time.Millisecond)
		require.Less(t, elapsed, 500*time.Millisecond)
	})
}

// TestBufferedIdempotence verifies that idempotence works correctly with buffering,
// both within a single buffer (in-memory dedup) and across flushes (Redis dedup).
func TestBufferedIdempotence(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)

	t.Run("cross-flush idempotence via Redis", func(t *testing.T) {
		// Use small buffer that flushes quickly
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(50*time.Millisecond, 1), // Flush after each item
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		eventID := ulid.MustNew(ulid.Now(), rand.Reader)
		bi := BatchItem{
			AccountID:   uuid.New(),
			WorkspaceID: uuid.New(),
			AppID:       uuid.New(),
			FunctionID:  fnId,
			EventID:     eventID,
			Event:       event.Event{Name: "test/event", Data: map[string]any{"a": 1}},
		}

		// First append - should succeed (buffer handles scheduling, returns Append status)
		result1, err := buffered.Append(context.Background(), bi, fn)
		require.NoError(t, err)
		require.Equal(t, enums.BatchAppend, result1.Status)

		// Second append with same eventID - should be detected as duplicate by Redis
		result2, err := buffered.Append(context.Background(), bi, fn)
		require.NoError(t, err)
		require.Equal(t, enums.BatchItemExists, result2.Status)

		// Verify only 1 item in Redis
		info, err := buffered.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Len(t, info.Items, 1)
	})

	t.Run("sequential appends with buffering", func(t *testing.T) {
		// Use timer-based flush to batch multiple items
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(100*time.Millisecond, 100),
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		accountID := uuid.New()
		workspaceID := uuid.New()
		appID := uuid.New()

		// Append first event
		bi1 := BatchItem{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			AppID:       appID,
			FunctionID:  fnId,
			EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
			Event:       event.Event{Name: "test/event", Data: map[string]any{"seq": 1}},
		}

		var wg sync.WaitGroup
		var results [3]*BatchAppendResult
		var errs [3]error

		// Launch all appends concurrently - they should all be buffered together
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[0], errs[0] = buffered.Append(context.Background(), bi1, fn)
		}()

		// Small delay to ensure ordering
		time.Sleep(5 * time.Millisecond)

		// Append second event
		bi2 := BatchItem{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			AppID:       appID,
			FunctionID:  fnId,
			EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
			Event:       event.Event{Name: "test/event", Data: map[string]any{"seq": 2}},
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[1], errs[1] = buffered.Append(context.Background(), bi2, fn)
		}()

		time.Sleep(5 * time.Millisecond)

		// Append duplicate of first event - should be caught by in-buffer dedup
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[2], errs[2] = buffered.Append(context.Background(), bi1, fn)
		}()

		wg.Wait()

		// First two should succeed
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])
		require.NoError(t, errs[2])

		// With buffering, scheduling is handled internally so all non-duplicate items return Append
		require.Equal(t, enums.BatchAppend, results[0].Status)
		require.Equal(t, enums.BatchAppend, results[1].Status)
		require.Equal(t, enums.BatchItemExists, results[2].Status)

		// Verify only 2 items in Redis
		info, err := buffered.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Len(t, info.Items, 2)
	})
}

// TestBufferedBatchFull verifies that batch full status is correctly returned with buffering.
func TestBufferedBatchFull(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)

	t.Run("batch fills via buffered append", func(t *testing.T) {
		// Buffer size matches batch max size so one flush fills the batch
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(5*time.Second, 5), // Flush when 5 items buffered
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 5, // Batch is full at 5
				Timeout: "60s",
			},
		}

		accountID := uuid.New()
		workspaceID := uuid.New()
		appID := uuid.New()

		var wg sync.WaitGroup
		results := make([]*BatchAppendResult, 5)
		errs := make([]error, 5)

		// Launch 5 appends concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				bi := BatchItem{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					FunctionID:  fnId,
					EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
					Event:       event.Event{Name: "test/event", Data: map[string]any{"idx": idx}},
				}
				results[idx], errs[idx] = buffered.Append(context.Background(), bi, fn)
			}(i)
		}

		wg.Wait()

		// All should succeed with BatchAppend status (buffer handles scheduling internally)
		for i := 0; i < 5; i++ {
			require.NoError(t, errs[i], "append %d failed", i)
			require.NotNil(t, results[i], "result %d is nil", i)
			require.Equal(t, enums.BatchAppend, results[i].Status, "result %d should be BatchAppend", i)
		}

		// When batch is full, the pointer rotates to a new batch.
		// Retrieve items using the batch ID from one of the results.
		batchID, err := ulid.Parse(results[0].BatchID)
		require.NoError(t, err)
		items, err := buffered.RetrieveItems(context.Background(), fnId, batchID)
		require.NoError(t, err)
		require.Len(t, items, 5, "filled batch should have 5 items")
	})

	t.Run("batch overflow via buffered append", func(t *testing.T) {
		// Buffer more items than batch can hold
		buffered := NewRedisBatchManager(bc, nil,
			WithBufferSettings(5*time.Second, 5), // Flush when 5 items buffered
		)
		defer buffered.Close()

		fnId := uuid.New()
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 3, // Batch max is 3, but we'll buffer 5
				Timeout: "60s",
			},
		}

		accountID := uuid.New()
		workspaceID := uuid.New()
		appID := uuid.New()

		var wg sync.WaitGroup
		results := make([]*BatchAppendResult, 5)
		errs := make([]error, 5)

		// Launch 5 appends concurrently - should overflow
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				bi := BatchItem{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					FunctionID:  fnId,
					EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
					Event:       event.Event{Name: "test/event", Data: map[string]any{"idx": idx}},
				}
				results[idx], errs[idx] = buffered.Append(context.Background(), bi, fn)
			}(i)
		}

		wg.Wait()

		// All should succeed
		for i := 0; i < 5; i++ {
			require.NoError(t, errs[i], "append %d failed", i)
			require.NotNil(t, results[i], "result %d is nil", i)
		}

		// All 5 events should be committed (3 in first batch, 2 in overflow)
		// The batch pointer now points to the overflow batch with 2 items
		info, err := buffered.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Len(t, info.Items, 2, "overflow batch should have 2 items")
	})
}

func TestBufferedBatchManagerMultipleBufferKeys(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	buffered := NewRedisBatchManager(bc, nil,
		WithBufferSettings(100*time.Millisecond, 100),
	)
	defer buffered.Close()

	// Function with batch key expression
	fnId := uuid.New()
	fn := inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
			Key:     strPtr("event.data.tenant"),
		},
	}

	var wg sync.WaitGroup

	// Send events to two different tenants
	tenants := []string{"tenant-a", "tenant-b"}
	for _, tenant := range tenants {
		wg.Add(1)
		go func(tenantName string) {
			defer wg.Done()
			bi := BatchItem{
				AccountID:   uuid.New(),
				WorkspaceID: uuid.New(),
				AppID:       uuid.New(),
				FunctionID:  fnId,
				EventID:     ulid.MustNew(ulid.Now(), rand.Reader),
				Event:       event.Event{Name: "test/event", Data: map[string]any{"tenant": tenantName}},
			}
			_, appendErr := buffered.Append(context.Background(), bi, fn)
			require.NoError(t, appendErr)
		}(tenant)
	}

	wg.Wait()

	// Each tenant should have their own batch
	for _, tenant := range tenants {
		info, err := buffered.GetBatchInfo(context.Background(), fnId, tenant)
		require.NoError(t, err)
		require.Len(t, info.Items, 1, "tenant %s should have 1 item", tenant)
	}
}
