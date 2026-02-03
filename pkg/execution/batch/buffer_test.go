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
)

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
		require.Equal(t, enums.BatchNew, result.Status)

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

	t.Run("local dedup returns immediately", func(t *testing.T) {
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
			_, err = buffered.Append(context.Background(), bi, fn)
			require.NoError(t, err)
		}()

		// Give time for first append to add to buffer
		time.Sleep(10 * time.Millisecond)

		// Second append with same event ID should return immediately
		start := time.Now()
		result, err := buffered.Append(context.Background(), bi, fn)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, enums.BatchItemExists, result.Status)
		require.Less(t, elapsed, 50*time.Millisecond, "dedup should return immediately")
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
		err = buffered.Close()
		require.NoError(t, err)

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
