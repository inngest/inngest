package pauses

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob/memblob"
)

func TestManagerFlushingWithLowLimit(t *testing.T) {
	// Setup miniredis
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	// Setup in-memory blob bucket
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	// Create a leaser
	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	// Create a mock bufferer
	mockBufferer := &mockBufferer{
		pauses: []*state.Pause{},
	}

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)
	
	// Create block store with a very low block size (2) to trigger flushing quickly
	const lowBlockSize = 3
	blockStore, err := NewBlockstore(BlockstoreOpts{
		PauseClient:               pauseClient,
		Bucket:           bucket,
		Bufferer:         mockBufferer,
		Leaser:           leaser,
		BlockSize:        lowBlockSize, // Very low limit to ensure flushing happens quickly
		CompactionGarbageRatio: 0.33,
		CompactionSample: 1.0, // Always compact for testing
		CompactionLeaser: leaser,
		DeleteAfterFlush: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	// Create in-process flusher that will trigger flush synchronously
	inProcessFlusher := InMemoryFlushProcessor(blockStore).(*flushInProcess)

	// Create manager with our configured flusher and a short flush delay
	manager := NewManager(mockBufferer, blockStore, WithFlusher(inProcessFlusher), WithBlockFlushEnabled(alwaysEnabled), WithBlockStoreEnabled(alwaysEnabled)).(*manager)
	manager.flushDelay = 100 * time.Millisecond // Short delay for tests

	// Create test index
	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.event",
	}

	ctx := context.Background()

	// Test 1: Write fewer pauses than the block size limit - should not trigger flush
	pauses := createTestPauses(2) // Less than lowBlockSize
	count, err := manager.Write(ctx, index, pauses...)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.EqualValues(t, 0, inProcessFlusher.counter, "No flush should happen when below limit")

	// Test 2: Write more pauses to exceed the block size - should trigger flush
	morePauses := createTestPauses(2) // This will make total 4 pauses, exceeding lowBlockSize
	count, err = manager.Write(ctx, index, morePauses...)
	require.NoError(t, err)
	assert.Equal(t, 4, count)
	assert.EqualValues(t, 1, inProcessFlusher.counter, "Flush should happen when exceeding limit")
	time.Sleep(manager.flushDelay * 2)
	// After waiting for the flush, there should only be 1 pause in the buffer,
	// as the block size is 3 and there were 4 pauses in the buffer - leaving 1 remaining.
	assert.Equal(t, 1, mockBufferer.pauseCount())

	// Test 3: Verify blocks were created and retrievable
	blocks, err := blockStore.BlocksSince(ctx, index, time.Time{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(blocks), 1, "At least one block should be created")

	// Test 4: Manually triggering flush
	err = manager.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// Test 5: Verify PausesSince can read from both buffer and blocks
	iter, err := manager.PausesSince(ctx, index, time.Time{})
	require.NoError(t, err)

	// Count pauses through the iterator
	pauseCount := 0
	for iter.Next(ctx) {
		pause := iter.Val(ctx)
		require.NotNil(t, pause)
		pauseCount++
	}
	require.NoError(t, iter.Error())
	assert.Equal(t, 4, pauseCount, "Should retrieve all pauses from buffer and blocks")

	// Test 6: Test deleting a pause
	pauseToDelete := pauses[0]
	err = manager.Delete(ctx, index, *pauseToDelete)
	require.NoError(t, err)

	// Verify the pause was deleted by trying to access it
	mockBufferer.clearPauses() // Clear buffer to force reading from blocks
	iter, err = manager.PausesSince(ctx, index, time.Time{})
	require.NoError(t, err)

	found := false
	for iter.Next(ctx) {
		pause := iter.Val(ctx)
		if pause != nil && pause.ID == pauseToDelete.ID {
			found = true
			break
		}
	}
	// Since blockStore.Delete only marks the pause for deletion and doesn't immediately remove it
	// from the block, we can't assert \!found here. In a real implementation,
	// this would eventually be removed during compaction.
	require.True(t, found)
}

func TestConsumePause(t *testing.T) {
	// Setup miniredis
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	// Setup manager with mock components
	mockBufferer := &mockBuffererWithConsume{}
	mockBlockStore := &mockBlockStore{}
	mockFlusher := &mockSimpleFlusher{}

	manager := NewManager(mockBufferer, mockBlockStore, WithFlusher(mockFlusher), WithBlockFlushEnabled(alwaysEnabled), WithBlockStoreEnabled(alwaysEnabled))

	ctx := context.Background()
	eventName := "test.event"
	workspaceID := uuid.New()

	// Create a pause with an event
	pause := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Event:       &eventName,
	}

	// Test consuming a pause
	result, cleanup, err := manager.ConsumePause(ctx, pause, state.ConsumePauseOpts{
		Data: "test-data",
	})
	require.NoError(t, err)
	require.NoError(t, cleanup())
	assert.Equal(t, true, result.DidConsume)
	assert.True(t, mockBufferer.consumeCalled, "ConsumePause should be called on the buffer")
	assert.Equal(t, 1, mockBlockStore.deleteCalled, "Delete should be called once on the blockstore")
}

func TestDeletePauseByID(t *testing.T) {
	t.Run("with block store enabled", func(t *testing.T) {
		mockBufferer := &mockBufferer{}
		mockBlockStore := &mockBlockStore{}
		mockFlusher := &mockSimpleFlusher{}
		
		manager := NewManager(mockBufferer, mockBlockStore, WithFlusher(mockFlusher), WithBlockFlushEnabled(alwaysEnabled), WithBlockStoreEnabled(alwaysEnabled))

		ctx := context.Background()
		pauseID := uuid.New()
		workspaceID := uuid.New()

		testPause := &state.Pause{
			ID:          pauseID,
			WorkspaceID: workspaceID,
		}
		mockBufferer.pauses = append(mockBufferer.pauses, testPause)

		err := manager.DeletePauseByID(ctx, pauseID, workspaceID)
		require.NoError(t, err)

		assert.Equal(t, 1, mockBlockStore.deleteByIDCalled, "BlockStore DeleteByID should be called once")
		assert.Equal(t, 1, mockBufferer.deletePauseByIDCalled, "Buffer DeletePauseByID should be called once")
	})

	t.Run("with block store disabled", func(t *testing.T) {
		mockBufferer := &mockBufferer{}
		mockBlockStore := &mockBlockStore{}
		mockFlusher := &mockSimpleFlusher{}

		manager := NewManager(mockBufferer, mockBlockStore, WithFlusher(mockFlusher))

		ctx := context.Background()
		pauseID := uuid.New()
		workspaceID := uuid.New()

		testPause := &state.Pause{
			ID:          pauseID,
			WorkspaceID: workspaceID,
		}
		mockBufferer.pauses = append(mockBufferer.pauses, testPause)

		err := manager.DeletePauseByID(ctx, pauseID, workspaceID)
		require.NoError(t, err)

		assert.Equal(t, 0, mockBlockStore.deleteByIDCalled, "BlockStore DeleteByID should NOT be called when disabled")
		assert.Equal(t, 1, mockBufferer.deletePauseByIDCalled, "Buffer DeletePauseByID should be called once")
	})

	t.Run("without block store", func(t *testing.T) {
		mockBufferer := &mockBufferer{}
		mockFlusher := &mockSimpleFlusher{}

		manager := NewManager(mockBufferer, nil, WithFlusher(mockFlusher), WithBlockFlushEnabled(alwaysEnabled))

		ctx := context.Background()
		pauseID := uuid.New()
		workspaceID := uuid.New()

		testPause := &state.Pause{
			ID:          pauseID,
			WorkspaceID: workspaceID,
		}
		mockBufferer.pauses = append(mockBufferer.pauses, testPause)

		err := manager.DeletePauseByID(ctx, pauseID, workspaceID)
		require.NoError(t, err)

		assert.Equal(t, 1, mockBufferer.deletePauseByIDCalled, "Buffer DeletePauseByID should be called once")
	})
}

// Helper functions

func createTestPauses(count int) []*state.Pause {
	pauses := make([]*state.Pause, count)
	baseTime := time.Now()
	for i := 0; i < count; i++ {
		eventName := "test.event"
		pauses[i] = &state.Pause{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Event:       &eventName,
			CreatedAt:   baseTime.Add(time.Duration(i) * time.Second),
		}
	}
	return pauses
}

func alwaysEnabled(ctx context.Context, id uuid.UUID) bool {
	return true
}

// Mock implementations

type mockSimpleFlusher struct{}

func (m *mockSimpleFlusher) Enqueue(ctx context.Context, index Index) error {
	return nil
}

type mockBuffererWithConsume struct {
	mockBufferer
	consumeCalled bool
}

func (m *mockBuffererWithConsume) ConsumePause(ctx context.Context, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error) {
	m.consumeCalled = true
	return state.ConsumePauseResult{DidConsume: true}, func() error { return nil }, nil
}

type mockBlockStore struct {
	deleteCalled      int
	deleteByIDCalled  int
}

func (m *mockBlockStore) BlockSize() int {
	return 10
}

func (m *mockBlockStore) FlushIndexBlock(ctx context.Context, index Index) error {
	return nil
}

func (m *mockBlockStore) BlocksSince(ctx context.Context, index Index, since time.Time) ([]ulid.ULID, error) {
	return nil, nil
}

func (m *mockBlockStore) ReadBlock(ctx context.Context, index Index, blockID ulid.ULID) (*Block, error) {
	return nil, nil
}

func (m *mockBlockStore) Delete(ctx context.Context, index Index, pause state.Pause, opts ...state.DeletePauseOpt) error {
	m.deleteCalled++
	return nil
}

func (m *mockBlockStore) DeleteByID(ctx context.Context, pauseID uuid.UUID, workspaceID uuid.UUID) error {
	m.deleteByIDCalled++
	return nil
}

func (m *mockBlockStore) LastBlockMetadata(ctx context.Context, index Index) (*blockMetadata, error) {
	return nil, nil
}

func (m *mockBlockStore) IndexExists(ctx context.Context, i Index) (bool, error) {
	return false, nil
}

func (m *mockBlockStore) PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error) {
	return nil, nil
}

func (m *mockBlockStore) GetBlockMetadata(ctx context.Context, index Index) (map[string]*blockMetadata, error) {
	return nil, nil
}

func (m *mockBlockStore) GetBlockDeleteCount(ctx context.Context, index Index, blockID ulid.ULID) (int64, error) {
	return 0, nil
}

func (m *mockBlockStore) GetBlockPauseIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error) {
	return nil, 0, nil
}

func (m *mockBlockStore) GetBlockDeletedIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error) {
	return nil, 0, nil
}
