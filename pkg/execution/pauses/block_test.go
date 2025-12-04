package pauses

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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

func TestBlockID(t *testing.T) {
	// Create a block with a deterministic pause
	pauseID, err := uuid.Parse("00000001-0000-0000-0000-000000000001")
	require.NoError(t, err)

	pause := &state.Pause{
		ID: pauseID,
	}

	block := &Block{
		Index:  Index{WorkspaceID: uuid.New(), EventName: "test.event"},
		Pauses: []*state.Pause{pause},
	}

	metadata := &blockMetadata{
		Timeranges: [2]int64{100000, 200000}, // milliseconds
		Len:        1,
	}

	// Generate the block ID
	id := blockID(block, metadata)

	// Verify the timestamp part of the ULID matches our latest timestamp
	require.Equal(t, uint64(200000), id.Time())

	// Verify determinism by generating another ID with the same inputs
	id2 := blockID(block, metadata)
	require.Equal(t, id.String(), id2.String())

	t.Run("with a new pause ID", func(t *testing.T) {
		// Create a new block with a different pause ID
		pauseID2, err := uuid.Parse("00000001-0000-0000-0000-000000000002")
		require.NoError(t, err)

		block2 := &Block{
			Index: block.Index,
			Pauses: []*state.Pause{
				{ID: pauseID2},
			},
		}

		// Generate a block ID with the new pause
		id3 := blockID(block2, metadata)

		// Verify the IDs are different due to different pause IDs
		require.NotEqual(t, id.String(), id3.String())

		// But the timestamp part should still be the same
		require.Equal(t, uint64(200000), id3.Time())
	})

	t.Run("with a new timestamp", func(t *testing.T) {
		metadata.Timeranges[1] = 300000

		// Verify determinism by generating another ID with the same inputs
		id4 := blockID(block, metadata)
		require.NotEqual(t, id.String(), id4.String())

		// Verify the timestamp part of the ULID matches our latest timestamp
		require.Equal(t, uint64(300000), id4.Time())
	})
}

func TestBlockFlusher(t *testing.T) {
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

	// Create a mock bufferer that returns some test pauses with different timestamps
	now := time.Now()
	pause1 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now,
	}
	pause2 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(time.Second),
	}
	mockBufferer := &mockBufferer{
		pauses: []*state.Pause{
			pause1,
			pause2,
		},
	}

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	// Create block store
	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              mockBufferer,
		Leaser:                leaser,
		BlockSize:             2, // Small block size for testing
		CompactionGarbageRatio:       0.5,
		CompactionSample:      0.1,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	// Create test index
	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.event",
	}

	// Test flushing
	ctx := context.Background()
	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// Verify block was written
	blocks, err := store.BlocksSince(ctx, index, time.Time{})
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	// Read the block back
	block, err := store.ReadBlock(ctx, index, blocks[0])
	require.NoError(t, err)
	require.NotNil(t, block)
	require.Len(t, block.Pauses, 2)
	require.Equal(t, pause1.ID, block.Pauses[0].ID)
	require.Equal(t, pause2.ID, block.Pauses[1].ID)

	// Verify that the pauses are removed from the buffer after flushing
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		mockBufferer.mu.RLock()
		pausesLen := len(mockBufferer.pauses)
		mockBufferer.mu.RUnlock()
		assert.Equal(t, 0, pausesLen, "pauses should be removed from buffer after flushing")
	}, 5*time.Second, 200*time.Millisecond)
}

func TestBlockMetadata_SameTimestamps(t *testing.T) {
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

	// Create a mock bufferer that returns the same timestamp for all pauses
	sameTime := time.Now()
	mockBufferer := &mockBuffererSameTimestamp{
		timestamp: sameTime,
		pauses: []*state.Pause{
			{ID: uuid.New(), CreatedAt: sameTime},
			{ID: uuid.New(), CreatedAt: sameTime},
		},
	}

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	// Create block store
	store := &blockstore{
		pc:        pauseClient,
		buf:       mockBufferer,
		bucket:    bucket,
		blocksize: 2,
	}

	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.event",
	}

	block := &Block{
		Index:  index,
		Pauses: mockBufferer.pauses,
	}

	// Test that blockMetadata returns an error when earliest == latest
	_, err = store.blockMetadata(context.Background(), index, block)
	require.Error(t, err)
	require.Contains(t, err.Error(), "block boundaries should never be the same")
}

// mockBufferer implements the Bufferer interface for testing
type mockBufferer struct {
	mu     sync.RWMutex
	pauses []*state.Pause
}

func (m *mockBufferer) Write(ctx context.Context, index Index, pauses ...*state.Pause) (int, error) {
	// For testing purposes, we'll just append the pauses to our mock buffer
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pauses = append(m.pauses, pauses...)
	return len(m.pauses), nil
}

func (m *mockBufferer) PausesSince(ctx context.Context, index Index, since time.Time) (state.PauseIterator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Create a copy of pauses to avoid race conditions
	pausesCopy := make([]*state.Pause, len(m.pauses))
	copy(pausesCopy, m.pauses)
	return &mockPauseIterator{pauses: pausesCopy}, nil
}

func (m *mockBufferer) PausesSinceWithCreatedAt(ctx context.Context, index Index, since time.Time, limit int64) (state.PauseIterator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Create a copy of pauses to avoid race conditions
	pausesCopy := make([]*state.Pause, len(m.pauses))
	copy(pausesCopy, m.pauses)
	return &mockPauseIterator{pauses: pausesCopy}, nil
}

func (m *mockBufferer) PauseTimestamp(ctx context.Context, index Index, pause state.Pause) (time.Time, error) {
	return time.Now(), nil
}

func (m *mockBufferer) ConsumePause(ctx context.Context, p state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error) {
	return state.ConsumePauseResult{}, func() error { return nil }, fmt.Errorf("not implemented")
}

func (m *mockBufferer) Delete(ctx context.Context, index Index, pause state.Pause, opts ...state.DeletePauseOpt) error {
	// For testing purposes, we'll just remove the pause from our mock buffer
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.pauses {
		if p.ID == pause.ID {
			m.pauses = append(m.pauses[:i], m.pauses[i+1:]...)
			return nil
		}
	}
	return ErrNotInBuffer
}

func (m *mockBufferer) PauseByInvokeCorrelationID(ctx context.Context, workspaceID uuid.UUID, correlationID string) (*state.Pause, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *mockBufferer) PauseBySignalID(ctx context.Context, workspaceID uuid.UUID, signalID string) (*state.Pause, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *mockBufferer) PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.pauses {
		if p.ID == pauseID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("pause not found")
}

func (m *mockBufferer) BufferLen(ctx context.Context, i Index) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.pauses)), nil
}

func (m *mockBufferer) IndexExists(ctx context.Context, i Index) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pauses) > 0, nil
}

func (m *mockBufferer) DeletePauseByID(ctx context.Context, pauseID uuid.UUID, workspaceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.pauses {
		if p.ID == pauseID && p.WorkspaceID == workspaceID {
			m.pauses = append(m.pauses[:i], m.pauses[i+1:]...)
			return nil
		}
	}
	return state.ErrPauseNotFound
}

// Helper methods for thread-safe access in tests
func (m *mockBufferer) pauseCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pauses)
}

func (m *mockBufferer) clearPauses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pauses = nil
}

// mockPauseIterator implements the PauseIterator interface for testing
type mockPauseIterator struct {
	pauses []*state.Pause
	pos    int
}

func (m *mockPauseIterator) Count() int {
	return len(m.pauses)
}

func (m *mockPauseIterator) Next(ctx context.Context) bool {
	return m.pos < len(m.pauses)
}

func (m *mockPauseIterator) Val(ctx context.Context) *state.Pause {
	if m.pos >= len(m.pauses) {
		return nil
	}
	p := m.pauses[m.pos]
	m.pos++
	return p
}

func (m *mockPauseIterator) Error() error {
	return nil
}

func (m *mockPauseIterator) Index() int64 {
	return int64(m.pos)
}

// mockBuffererSameTimestamp is a special mock that returns the same timestamp for all pauses
type mockBuffererSameTimestamp struct {
	mu        sync.RWMutex
	timestamp time.Time
	pauses    []*state.Pause
}

func (m *mockBuffererSameTimestamp) Write(ctx context.Context, index Index, pauses ...*state.Pause) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pauses = append(m.pauses, pauses...)
	return len(m.pauses), nil
}

func (m *mockBuffererSameTimestamp) PausesSince(ctx context.Context, index Index, since time.Time) (state.PauseIterator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pausesCopy := make([]*state.Pause, len(m.pauses))
	copy(pausesCopy, m.pauses)
	return &mockPauseIterator{pauses: pausesCopy}, nil
}

func (m *mockBuffererSameTimestamp) PausesSinceWithCreatedAt(ctx context.Context, index Index, since time.Time, limit int64) (state.PauseIterator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pausesCopy := make([]*state.Pause, len(m.pauses))
	copy(pausesCopy, m.pauses)
	return &mockPauseIterator{pauses: pausesCopy}, nil
}

func (m *mockBuffererSameTimestamp) PauseTimestamp(ctx context.Context, index Index, pause state.Pause) (time.Time, error) {
	// Always return the same timestamp - this is what triggers the error condition
	return m.timestamp, nil
}

func (m *mockBuffererSameTimestamp) ConsumePause(ctx context.Context, p state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error) {
	return state.ConsumePauseResult{}, func() error { return nil }, fmt.Errorf("not implemented")
}

func (m *mockBuffererSameTimestamp) Delete(ctx context.Context, index Index, pause state.Pause, opts ...state.DeletePauseOpt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.pauses {
		if p.ID == pause.ID {
			m.pauses = append(m.pauses[:i], m.pauses[i+1:]...)
			return nil
		}
	}
	return ErrNotInBuffer
}

func (m *mockBuffererSameTimestamp) PauseByInvokeCorrelationID(ctx context.Context, workspaceID uuid.UUID, correlationID string) (*state.Pause, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *mockBuffererSameTimestamp) PauseBySignalID(ctx context.Context, workspaceID uuid.UUID, signalID string) (*state.Pause, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *mockBuffererSameTimestamp) PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.pauses {
		if p.ID == pauseID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("pause not found")
}

func (m *mockBuffererSameTimestamp) BufferLen(ctx context.Context, i Index) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.pauses)), nil
}

func (m *mockBuffererSameTimestamp) IndexExists(ctx context.Context, i Index) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pauses) > 0, nil
}

func (m *mockBuffererSameTimestamp) DeletePauseByID(ctx context.Context, pauseID uuid.UUID, workspaceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.pauses {
		if p.ID == pauseID && p.WorkspaceID == workspaceID {
			m.pauses = append(m.pauses[:i], m.pauses[i+1:]...)
			return nil
		}
	}
	return state.ErrPauseNotFound
}

func TestLastBlockMetadata(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	now := time.Now()
	mockBufferer := &mockBufferer{
		pauses: []*state.Pause{
			{ID: uuid.New(), CreatedAt: now},
			{ID: uuid.New(), CreatedAt: now.Add(time.Second)},
		},
	}

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              mockBufferer,
		Leaser:                leaser,
		BlockSize:             2,
		CompactionGarbageRatio:       0.5,
		CompactionSample:      0.1,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.event",
	}
	ctx := context.Background()

	t.Run("should return nil when no blocks exist", func(t *testing.T) {
		metadata, err := store.LastBlockMetadata(ctx, index)
		require.NoError(t, err)
		require.Nil(t, metadata)
	})

	t.Run("should return metadata after creating first block", func(t *testing.T) {
		err := store.FlushIndexBlock(ctx, index)
		require.NoError(t, err)

		metadata, err := store.LastBlockMetadata(ctx, index)
		require.NoError(t, err)
		require.NotNil(t, metadata)
		require.Equal(t, 2, metadata.Len)
		require.False(t, metadata.FirstTimestamp().IsZero())
		require.False(t, metadata.LastTimestamp().IsZero())

		// Verify timestamps match the CreatedAt we set
		require.Equal(t, now.UnixMilli(), metadata.FirstTimestamp().UnixMilli())
		require.Equal(t, now.Add(time.Second).UnixMilli(), metadata.LastTimestamp().UnixMilli())
	})

	t.Run("should return latest block metadata after creating second block", func(t *testing.T) {
		// Wait for buffer to be empty after first flush
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			mockBufferer.mu.RLock()
			pausesLen := len(mockBufferer.pauses)
			mockBufferer.mu.RUnlock()
			assert.Equal(t, 0, pausesLen, "buffer should be empty after first flush")
		}, 5*time.Second, 200*time.Millisecond)

		// Add two more pauses to trigger a second block
		mockBufferer.mu.Lock()
		laterTime := now.Add(2 * time.Second)
		mockBufferer.pauses = append(mockBufferer.pauses,
			&state.Pause{ID: uuid.New(), CreatedAt: laterTime},
			&state.Pause{ID: uuid.New(), CreatedAt: laterTime.Add(time.Second)})
		mockBufferer.mu.Unlock()

		// Get metadata from first block
		firstMetadata, err := store.LastBlockMetadata(ctx, index)
		require.NoError(t, err)
		require.NotNil(t, firstMetadata)

		// Create second block
		err = store.FlushIndexBlock(ctx, index)
		require.NoError(t, err)

		// Should now return the second block's metadata
		secondMetadata, err := store.LastBlockMetadata(ctx, index)
		require.NoError(t, err)
		require.NotNil(t, secondMetadata)
		require.Equal(t, 2, secondMetadata.Len)

		// Second block should have later timestamp than first
		require.True(t, secondMetadata.LastTimestamp().After(firstMetadata.LastTimestamp()))

		// Verify second block timestamps match the CreatedAt we set
		require.Equal(t, laterTime.UnixMilli(), secondMetadata.FirstTimestamp().UnixMilli())
		require.Equal(t, laterTime.Add(time.Second).UnixMilli(), secondMetadata.LastTimestamp().UnixMilli())
	})
}

func TestBlockstoreDelete(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	now := time.Now()
	pause1 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now,
	}
	pause2 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(time.Second),
	}

	mockBufferer := &mockBufferer{
		pauses: []*state.Pause{pause1, pause2},
	}

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              mockBufferer,
		Leaser:                leaser,
		BlockSize:             2,
		CompactionGarbageRatio:       1.0,
		CompactionSample:      1.0,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.event",
	}
	ctx := context.Background()

	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	t.Run("delete with CreatedAt timestamp", func(t *testing.T) {
		// Test normal pause deletion in a specific block:
		//
		// Block 1: [0s ----------- 1s]
		// Block 2: [2s ----------- 3s]
		// Block 3: [4s ----------- 5s] ← pause gets marked here
		// Block 4: [6s ----------- 7s]
		//
		// Normal pause at 4.5s should be marked for deletion only in block 3

		// Create additional blocks to have 4 total
		mockBufferer.mu.Lock()
		mockBufferer.pauses = []*state.Pause{
			{
				ID:          uuid.New(),
				WorkspaceID: index.WorkspaceID,
				CreatedAt:   now.Add(2 * time.Second),
			},
			{
				ID:          uuid.New(),
				WorkspaceID: index.WorkspaceID,
				CreatedAt:   now.Add(3 * time.Second),
			},
		}
		mockBufferer.mu.Unlock()
		err := store.FlushIndexBlock(ctx, index)
		require.NoError(t, err)

		mockBufferer.mu.Lock()
		mockBufferer.pauses = []*state.Pause{
			{
				ID:          uuid.New(),
				WorkspaceID: index.WorkspaceID,
				CreatedAt:   now.Add(4 * time.Second),
			},
			{
				ID:          uuid.New(),
				WorkspaceID: index.WorkspaceID,
				CreatedAt:   now.Add(5 * time.Second),
			},
		}
		mockBufferer.mu.Unlock()
		err = store.FlushIndexBlock(ctx, index)
		require.NoError(t, err)

		mockBufferer.mu.Lock()
		mockBufferer.pauses = []*state.Pause{
			{
				ID:          uuid.New(),
				WorkspaceID: index.WorkspaceID,
				CreatedAt:   now.Add(6 * time.Second),
			},
			{
				ID:          uuid.New(),
				WorkspaceID: index.WorkspaceID,
				CreatedAt:   now.Add(7 * time.Second),
			},
		}
		mockBufferer.mu.Unlock()
		err = store.FlushIndexBlock(ctx, index)
		require.NoError(t, err)

		// Create pause that should be in block 3
		testPause := state.Pause{
			ID:        uuid.New(),
			CreatedAt: now.Add(4*time.Second + 500*time.Millisecond),
		}

		err = store.Delete(ctx, index, testPause)
		require.NoError(t, err)

		blocks, err := store.BlocksSince(ctx, index, time.Time{})
		require.NoError(t, err)
		require.Len(t, blocks, 4)

		block3ID := blocks[2] // third block
		var foundCount int
		var foundBlocks []ulid.ULID
		for _, blockID := range blocks {
			deleteKey := blockDeleteKey(index, blockID)
			exists, err := rc.Do(ctx, rc.B().Sismember().Key(deleteKey).Member(testPause.ID.String()).Build()).AsBool()
			require.NoError(t, err)
			if exists {
				foundCount++
				foundBlocks = append(foundBlocks, blockID)
			}
		}
		require.Equal(t, 1, foundCount, "normal pause should be marked for deletion in exactly 1 block")
		require.Contains(t, foundBlocks, block3ID, "block 3 should contain the pause")
	})
}

func TestBoundaryPauseDelete(t *testing.T) {
	// Test boundary pause deletion across multiple blocks:
	//
	// Block 1: [0s ----------- 1s]
	// Block 2: [1.5s --------- 2s] ← ends at boundary
	// Block 3: [2s ----------- 3s] ← starts at boundary
	// Block 4: [4s ----------- 5s]
	// Block 5: [6s ----------- 7s]
	//
	// Boundary pause at 2s should be marked for deletion in blocks 2 & 3 only

	ctx := context.Background()
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	bucket := memblob.OpenBucket(nil)
	now := time.Now()

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	mockBufferer := &mockBufferer{}

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              mockBufferer,
		Leaser:                leaser,
		BlockSize:             2,
		CompactionGarbageRatio:       1.0,
		CompactionSample:      1.0,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.boundary",
	}

	// Create block 1
	mockBufferer.mu.Lock()
	mockBufferer.pauses = []*state.Pause{
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now,
		},
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(time.Second),
		},
	}
	mockBufferer.mu.Unlock()
	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	boundaryTime := now.Add(2 * time.Second)

	// Create block 2 (ends at boundary)
	mockBufferer.mu.Lock()
	mockBufferer.pauses = []*state.Pause{
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(time.Second + 500*time.Millisecond),
		},
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   boundaryTime,
		},
	}
	mockBufferer.mu.Unlock()
	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// Create block 3 (starts at boundary)
	mockBufferer.mu.Lock()
	mockBufferer.pauses = []*state.Pause{
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   boundaryTime,
		},
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   boundaryTime.Add(time.Second),
		},
	}
	mockBufferer.mu.Unlock()
	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// Create blocks 4 and 5
	mockBufferer.mu.Lock()
	mockBufferer.pauses = []*state.Pause{
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(4 * time.Second),
		},
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(5 * time.Second),
		},
	}
	mockBufferer.mu.Unlock()
	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	mockBufferer.mu.Lock()
	mockBufferer.pauses = []*state.Pause{
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(6 * time.Second),
		},
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(7 * time.Second),
		},
	}
	mockBufferer.mu.Unlock()
	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// Create pause at boundary timestamp
	boundaryPause := state.Pause{
		ID:        uuid.New(),
		CreatedAt: boundaryTime,
	}

	err = store.Delete(ctx, index, boundaryPause)
	require.NoError(t, err)

	// Get all blocks
	blocks, err := store.BlocksSince(ctx, index, time.Time{})
	require.NoError(t, err)
	require.Len(t, blocks, 5)

	block2ID := blocks[1]
	block3ID := blocks[2]
	var foundCount int
	var foundBlocks []ulid.ULID
	for _, blockID := range blocks {
		deleteKey := blockDeleteKey(index, blockID)
		exists, err := rc.Do(ctx, rc.B().Sismember().Key(deleteKey).Member(boundaryPause.ID.String()).Build()).AsBool()
		require.NoError(t, err)
		if exists {
			foundCount++
			foundBlocks = append(foundBlocks, blockID)
		}
	}
	require.Equal(t, 2, foundCount, "boundary pause should be marked for deletion in exactly 2 blocks")
	require.Contains(t, foundBlocks, block2ID, "block 2 should contain the boundary pause")
	require.Contains(t, foundBlocks, block3ID, "block 3 should contain the boundary pause")
}

func TestLegacyPauseDelete(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	bucket := memblob.OpenBucket(nil)
	now := time.Now()

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	pause1 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now,
	}
	pause2 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(time.Second),
	}

	mockBufferer := &mockBufferer{
		pauses: []*state.Pause{pause1, pause2},
	}

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              mockBufferer,
		Leaser:                leaser,
		BlockSize:             2,
		CompactionGarbageRatio:       1.0,
		CompactionSample:      1.0,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.legacy",
	}

	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// Create additional blocks
	mockBufferer.mu.Lock()
	mockBufferer.pauses = []*state.Pause{
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(2 * time.Second),
		},
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(3 * time.Second),
		},
	}
	mockBufferer.mu.Unlock()
	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	mockBufferer.mu.Lock()
	mockBufferer.pauses = []*state.Pause{
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(4 * time.Second),
		},
		{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
			CreatedAt:   now.Add(5 * time.Second),
		},
	}
	mockBufferer.mu.Unlock()
	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	pauseWithoutTime := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: index.WorkspaceID,
	}

	err = store.Delete(ctx, index, pauseWithoutTime)
	require.NoError(t, err)

	blocks, err := store.BlocksSince(ctx, index, time.Time{})
	require.NoError(t, err)
	require.Len(t, blocks, 3)

	var foundCount int
	for _, blockID := range blocks {
		deleteKey := blockDeleteKey(index, blockID)
		exists, err := rc.Do(ctx, rc.B().Sismember().Key(deleteKey).Member(pauseWithoutTime.ID.String()).Build()).AsBool()
		require.NoError(t, err)
		if exists {
			foundCount++
		}
	}
	require.Equal(t, 3, foundCount)
}

func TestBlockFlushOrderingBug(t *testing.T) {
	// Test that exposes ordering bug where pauses retrieved by Redis second-precision
	// can be out of order when using millisecond-precision CreatedAt timestamps.
	//
	// Bug scenario:
	// - Create 99 pauses within the same second (millisecond differences)
	// - Create 1 pause after a few seconds (for proper block boundaries)

	ctx := context.Background()
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	// Create Redis state manager with actual Redis backend
	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)

	sm, err := redis_state.New(
		ctx,
		redis_state.WithUnshardedClient(unshardedClient),
	)
	require.NoError(t, err)

	baseTime := time.Now()
	workspaceID := uuid.New()
	index := Index{
		WorkspaceID: workspaceID,
		EventName:   "test.ordering",
	}

	runID := ulid.MustNew(ulid.Now(), nil)
	expires := state.Time(baseTime.Add(time.Hour))
	eventName := "test.ordering"

	// Create 99 pauses within the same second with millisecond differences
	for i := 0; i < 99; i++ {
		pause := state.Pause{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			CreatedAt:   baseTime.Add(time.Duration(i) * time.Millisecond),
			Identifier: state.PauseIdentifier{
				RunID:      runID,
				AccountID:  workspaceID,
				FunctionID: uuid.New(),
			},
			Outgoing: "start",
			Incoming: "end",
			StepName: fmt.Sprintf("pause-%d", i),
			Expires:  expires,
			Event:    &eventName,
		}
		_, err := sm.SavePause(ctx, pause)
		require.NoError(t, err)
	}

	// Create the 100th pause after a few seconds for proper block boundaries
	lastPause := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		CreatedAt:   baseTime.Add(3 * time.Second),
		Identifier: state.PauseIdentifier{
			RunID:      runID,
			AccountID:  workspaceID,
			FunctionID: uuid.New(),
		},
		Outgoing: "start",
		Incoming: "end",
		StepName: "pause-last",
		Expires:  expires,
		Event:    &eventName,
	}
	_, err = sm.SavePause(ctx, lastPause)
	require.NoError(t, err)

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	bufferer := StateBufferer(sm)
	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              bufferer,
		Leaser:                leaser,
		BlockSize:             100,
		CompactionGarbageRatio:       0.03,
		CompactionSample:      1.0,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// Get the created block
	blocks, err := store.BlocksSince(ctx, index, time.Time{})
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	// Read the block back to check ordering
	block, err := store.ReadBlock(ctx, index, blocks[0])
	require.NoError(t, err)
	require.NotNil(t, block)
	require.Len(t, block.Pauses, 100)

	firstPauseTime := block.Pauses[0].CreatedAt
	lastPauseTime := block.Pauses[99].CreatedAt

	// Last pause should be the one created 3 seconds later
	require.Equal(t, baseTime.Add(3*time.Second).UnixMilli(), lastPauseTime.UnixMilli(),
		"Last pause should be the one created 3 seconds later")

	require.Equal(t, baseTime.UnixMilli(), firstPauseTime.UnixMilli(),
		"First pause should have the earliest timestamp, but ordering is wrong due to Redis second-precision vs millisecond-precision mismatch")

	// Verify all pauses are in chronological order
	for i := 1; i < len(block.Pauses); i++ {
		prevTime := block.Pauses[i-1].CreatedAt
		currTime := block.Pauses[i].CreatedAt
		require.True(t, !currTime.Before(prevTime),
			"Pauses should be in chronological order, but pause %d (%v) comes before pause %d (%v)",
			i, currTime.UnixMilli(), i-1, prevTime.UnixMilli())
	}
}

func TestCompaction(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	now := time.Now()
	pause1 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now,
	}
	pause2 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(500 * time.Millisecond),
	}
	pause3 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(time.Second),
	}
	pause4 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(1500 * time.Millisecond),
	}
	pause5 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(2 * time.Second),
	}

	mockBufferer := &mockBufferer{
		pauses: []*state.Pause{pause1, pause2, pause3, pause4, pause5},
	}

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	// Set compaction limit to 2, so 1 deletion won't trigger compaction
	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              mockBufferer,
		Leaser:                leaser,
		BlockSize:             5,
		CompactionGarbageRatio:       0.4,
		CompactionSample:      1.0,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.compact",
	}
	ctx := context.Background()

	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	blocks, err := store.BlocksSince(ctx, index, time.Time{})
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	blockID := blocks[0]
	deleteKey := blockDeleteKey(index, blockID)

	// Delete one pause - below compaction threshold of 2
	err = store.Delete(ctx, index, *pause1)
	require.NoError(t, err)

	// Force trigger compaction
	err = store.(*blockstore).compact(ctx, index)
	require.NoError(t, err)

	deleteCount, err := rc.Do(ctx, rc.B().Scard().Key(deleteKey).Build()).AsInt64()
	require.NoError(t, err)
	require.Equal(t, int64(1), deleteCount)

	block, err := store.ReadBlock(ctx, index, blockID)
	require.NoError(t, err)
	require.Len(t, block.Pauses, 5)

	// Delete second pause - meets compaction threshold of 2
	err = store.Delete(ctx, index, *pause5)
	require.NoError(t, err)

	deleteCount, err = rc.Do(ctx, rc.B().Scard().Key(deleteKey).Build()).AsInt64()
	require.NoError(t, err)
	require.Equal(t, int64(2), deleteCount)

	// Wait for async compaction to complete
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		block, err := store.ReadBlock(ctx, index, blockID)
		assert.NoError(t, err)
		assert.Len(t, block.Pauses, 3)

		deleteKey := blockDeleteKey(index, blockID)
		exists, err := rc.Do(ctx, rc.B().Exists().Key(deleteKey).Build()).AsBool()
		assert.NoError(t, err)
		assert.False(t, exists)

		// Assert that remaining pauses are pause2, pause3, and pause4 (pause1 and pause5 were deleted)
		assert.Equal(t, pause2.ID, block.Pauses[0].ID)
		assert.Equal(t, pause3.ID, block.Pauses[1].ID)
		assert.Equal(t, pause4.ID, block.Pauses[2].ID)
	}, 5*time.Second, 20*time.Millisecond)

	// Verify that block index score was updated to the new latest timestamp (even though the blockID is stable)
	indexKey := blockIndexKey(index)
	score, err := rc.Do(ctx, rc.B().Zscore().Key(indexKey).Member(blockID.String()).Build()).AsFloat64()
	require.NoError(t, err)
	require.Equal(t, float64(pause4.CreatedAt.UnixMilli()), score)

	// Verify updated metadata reflects the new block composition with boundaries from pause2 to pause4
	metadataKey := blockMetadataKey(index)
	exists, err := rc.Do(ctx, rc.B().Hexists().Key(metadataKey).Field(blockID.String()).Build()).AsBool()
	require.NoError(t, err)
	require.True(t, exists)

	metadataBytes, err := rc.Do(ctx, rc.B().Hget().Key(metadataKey).Field(blockID.String()).Build()).AsBytes()
	require.NoError(t, err)

	var metadata blockMetadata
	err = json.Unmarshal(metadataBytes, &metadata)
	require.NoError(t, err)
	require.Equal(t, 3, metadata.Len)
	require.Equal(t, pause2.CreatedAt.UnixMilli(), metadata.Timeranges[0])
	require.Equal(t, pause4.CreatedAt.UnixMilli(), metadata.Timeranges[1])

	// Now test deleting all remaining pauses to verify complete block deletion
	err = store.Delete(ctx, index, *pause2)
	require.NoError(t, err)

	err = store.Delete(ctx, index, *pause3)
	require.NoError(t, err)

	err = store.Delete(ctx, index, *pause4)
	require.NoError(t, err)

	// Wait for async compaction to complete and verify block is completely removed
	blobKey := store.(*blockstore).BlockKey(index, blockID)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		// Block should be removed from index
		_, err := rc.Do(ctx, rc.B().Zscore().Key(indexKey).Member(blockID.String()).Build()).AsFloat64()
		assert.True(t, rueidis.IsRedisNil(err), "expected Redis nil error when block is removed from index")

		// Block metadata should be removed
		metadataExists, err := rc.Do(ctx, rc.B().Hexists().Key(metadataKey).Field(blockID.String()).Build()).AsBool()
		assert.NoError(t, err)
		assert.False(t, metadataExists)

		// Delete tracking should be cleaned up
		deleteKey := blockDeleteKey(index, blockID)
		deleteExists, err := rc.Do(ctx, rc.B().Exists().Key(deleteKey).Build()).AsBool()
		assert.NoError(t, err)
		assert.False(t, deleteExists)

		// Block should be removed from blob storage
		blobExists, err := bucket.Exists(ctx, blobKey)
		assert.NoError(t, err)
		assert.False(t, blobExists)
	}, 5*time.Second, 20*time.Millisecond)

	// Verify no blocks exist for this index
	blocks, err = store.BlocksSince(ctx, index, time.Time{})
	require.NoError(t, err)
	require.Len(t, blocks, 0)

	rc.Close()
}

func TestCompactionFailsBoundaryCheck(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	// Create pause client
	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	now := time.Now()
	pause1 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now,
	}
	pause2 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(time.Second),
	}
	// Last two pauses have the same timestamp - this will cause boundary error
	pause3 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(2 * time.Second),
	}
	pause4 := &state.Pause{
		ID:        uuid.New(),
		CreatedAt: now.Add(2 * time.Second), // Same as pause3
	}

	mockBufferer := &mockBufferer{
		pauses: []*state.Pause{pause1, pause2, pause3, pause4},
	}

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              mockBufferer,
		Leaser:                leaser,
		BlockSize:             4,
		CompactionGarbageRatio:       0.5,
		CompactionSample:      1.0,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	index := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.boundary",
	}
	ctx := context.Background()

	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	blocks, err := store.BlocksSince(ctx, index, time.Time{})
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	blockID := blocks[0]

	// Delete first two pauses, leaving pause3 and pause4 with same timestamp
	err = store.Delete(ctx, index, *pause1)
	require.NoError(t, err)

	err = store.Delete(ctx, index, *pause2)
	require.NoError(t, err)

	// Trigger compaction manually to ensure it runs
	err = store.(*blockstore).compact(ctx, index)
	require.NoError(t, err)

	// Verify that compaction failed and block still exists with original pauses
	// because blockMetadata() should return an error for same timestamps
	block, err := store.ReadBlock(ctx, index, blockID)
	require.NoError(t, err)
	require.Len(t, block.Pauses, 4, "block should still have all original pauses when compaction fails")

	// Verify delete tracking still exists since compaction failed
	deleteKey := blockDeleteKey(index, blockID)
	deleteCount, err := rc.Do(ctx, rc.B().Scard().Key(deleteKey).Build()).AsInt64()
	require.NoError(t, err)
	require.Equal(t, int64(2), deleteCount, "delete tracking should remain when compaction fails")

	// Now delete one more pause, leaving only pause4 (1 pause left)
	// This should also fail compaction due to boundary check needing at least 2 pauses
	err = store.Delete(ctx, index, *pause3)
	require.NoError(t, err)

	// Trigger compaction again
	err = store.(*blockstore).compact(ctx, index)
	require.NoError(t, err)

	// Verify that compaction failed again since we can't generate metadata with only 1 pause
	block, err = store.ReadBlock(ctx, index, blockID)
	require.NoError(t, err)
	require.Len(t, block.Pauses, 4, "block should still have all original pauses when single pause compaction fails")

	// Verify delete tracking now shows 3 deletions
	deleteCount, err = rc.Do(ctx, rc.B().Scard().Key(deleteKey).Build()).AsInt64()
	require.NoError(t, err)
	require.Equal(t, int64(3), deleteCount, "delete tracking should show 3 deletions after third delete")

	rc.Close()
}

func TestPauseByIDAfterFlush(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	sm, err := redis_state.New(
		context.Background(),
		redis_state.WithUnshardedClient(unshardedClient),
	)
	require.NoError(t, err)

	now := time.Now()
	workspaceID := uuid.New()
	eventName := "test.event"

	testPause := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Identifier: state.PauseIdentifier{
			RunID:      ulid.Make(),
			AccountID:  workspaceID,
			FunctionID: uuid.New(),
		},
		Event:     &eventName,
		Expires:   state.Time(now.Add(time.Hour)),
		CreatedAt: now,
	}

	pause2 := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Identifier: state.PauseIdentifier{
			RunID:      ulid.Make(),
			AccountID:  workspaceID,
			FunctionID: uuid.New(),
		},
		Event:     &eventName,
		Expires:   state.Time(now.Add(time.Hour)),
		CreatedAt: now.Add(time.Second),
	}

	_, err = sm.SavePause(context.Background(), testPause)
	require.NoError(t, err)
	_, err = sm.SavePause(context.Background(), pause2)
	require.NoError(t, err)

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	bufferer := StateBufferer(sm)
	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              bufferer,
		Leaser:                leaser,
		BlockSize:             2,
		CompactionGarbageRatio:       0.5,
		CompactionSample:      0.1,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	index := Index{
		WorkspaceID: workspaceID,
		EventName:   eventName,
	}
	ctx := context.Background()

	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// Verify block was created
	blocks, err := store.BlocksSince(ctx, index, time.Time{})
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	// Wait for pauses to be deleted from buffer after flush
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		bufLen, err := bufferer.BufferLen(ctx, index)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), bufLen, "buffer should be empty after flushing")
	}, 5*time.Second, 200*time.Millisecond)

	// Should find pause after flush
	foundPause, err := store.PauseByID(ctx, index, testPause.ID)
	require.NoError(t, err)
	require.Equal(t, testPause.ID, foundPause.ID)
	require.Equal(t, testPause.CreatedAt.UnixMilli(), foundPause.CreatedAt.UnixMilli())

	t.Run("with empty event name", func(t *testing.T) {
		emptyEventIndex := Index{
			WorkspaceID: workspaceID,
			EventName:   "",
		}

		foundPause, err := store.PauseByID(ctx, emptyEventIndex, testPause.ID)
		require.NoError(t, err)
		require.Equal(t, testPause.ID, foundPause.ID)
	})

	// Mark pause as deleted
	err = store.Delete(ctx, index, testPause)
	require.NoError(t, err)

	// Should not find pause after delete
	_, err = store.PauseByID(ctx, index, testPause.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, state.ErrPauseNotFound)

	_, err = store.PauseByID(ctx, index, uuid.New())
	require.Error(t, err)
	require.ErrorIs(t, err, state.ErrPauseNotFound)
}

func TestCompactionCleansUpBlockIndexWhenAllPausesDeleted(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})

	mgr, err := redis_state.New(
		context.Background(),
		redis_state.WithUnshardedClient(unshardedClient),
		redis_state.WithShardedClient(shardedClient),
	)
	require.NoError(t, err)

	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              redisAdapter{rsm: mgr},
		Leaser:                leaser,
		BlockSize:             2,
		CompactionGarbageRatio:       0.5,
		CompactionSample:      1.0,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	ctx := context.Background()
	workspaceID := uuid.New()
	eventName := "test.event"

	index := Index{
		WorkspaceID: workspaceID,
		EventName:   eventName,
	}

	pause1 := &state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Identifier: state.PauseIdentifier{
			RunID:      ulid.Make(),
			FunctionID: uuid.New(),
			AccountID:  uuid.New(),
		},
		Event:     &eventName,
		Expires:   state.Time(time.Now().Add(time.Hour)),
		CreatedAt: time.Now(),
	}

	pause2 := &state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Identifier: state.PauseIdentifier{
			RunID:      ulid.Make(),
			FunctionID: uuid.New(),
			AccountID:  uuid.New(),
		},
		Event:     &eventName,
		Expires:   state.Time(time.Now().Add(time.Hour)),
		CreatedAt: time.Now().Add(time.Minute),
	}

	_, err = mgr.SavePause(ctx, *pause1)
	require.NoError(t, err)
	_, err = mgr.SavePause(ctx, *pause2)
	require.NoError(t, err)

	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// require.EventuallyWithT(t, func(t *assert.CollectT) {
	// 	keys, err := rc.Do(ctx, rc.B().Keys().Pattern("*:pause-block:*").Build()).AsStrSlice()
	// 	assert.NoError(t, err)
	// 	assert.Equal(t, 2, len(keys), "Expected 2 pause-block key after flush, but found: %v", keys)
	// }, 5*time.Second, 100*time.Millisecond)

	err = store.Delete(ctx, index, *pause1)
	require.NoError(t, err)
	err = store.Delete(ctx, index, *pause2)
	require.NoError(t, err)

	// Expire TTLs
	r.FastForward(20 * time.Minute)

	for _, key := range r.Keys() {
		assert.NotContains(t, key, "pause-block")
	}
}

func TestCompactionCleansUpBlockIndexWhenSomePausesDeleted(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	leaser := redisBlockLeaser{
		rc:       rc,
		prefix:   "test",
		duration: 5 * time.Second,
	}

	pauseClient := redis_state.NewPauseClient(rc, redis_state.StateDefaultKey)

	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
	})

	mgr, err := redis_state.New(
		context.Background(),
		redis_state.WithUnshardedClient(unshardedClient),
		redis_state.WithShardedClient(shardedClient),
	)
	require.NoError(t, err)

	store, err := NewBlockstore(BlockstoreOpts{
		PauseClient:           pauseClient,
		Bucket:                bucket,
		Bufferer:              redisAdapter{rsm: mgr},
		Leaser:                leaser,
		BlockSize:             3,
		CompactionGarbageRatio:       0.33,
		CompactionSample:      1.0,
		CompactionLeaser:      leaser,
		DeleteAfterFlush:      func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
	})
	require.NoError(t, err)

	ctx := context.Background()
	workspaceID := uuid.New()
	eventName := "test.event"

	index := Index{
		WorkspaceID: workspaceID,
		EventName:   eventName,
	}

	pause1 := &state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Identifier: state.PauseIdentifier{
			RunID:      ulid.Make(),
			FunctionID: uuid.New(),
			AccountID:  uuid.New(),
		},
		Event:     &eventName,
		Expires:   state.Time(time.Now().Add(time.Hour)),
		CreatedAt: time.Now(),
	}

	pause2 := &state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Identifier: state.PauseIdentifier{
			RunID:      ulid.Make(),
			FunctionID: uuid.New(),
			AccountID:  uuid.New(),
		},
		Event:     &eventName,
		Expires:   state.Time(time.Now().Add(time.Hour)),
		CreatedAt: time.Now().Add(time.Minute),
	}

	pause3 := &state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Identifier: state.PauseIdentifier{
			RunID:      ulid.Make(),
			FunctionID: uuid.New(),
			AccountID:  uuid.New(),
		},
		Event:     &eventName,
		Expires:   state.Time(time.Now().Add(time.Hour)),
		CreatedAt: time.Now().Add(2 * time.Minute),
	}

	_, err = mgr.SavePause(ctx, *pause1)
	require.NoError(t, err)
	_, err = mgr.SavePause(ctx, *pause2)
	require.NoError(t, err)
	_, err = mgr.SavePause(ctx, *pause3)
	require.NoError(t, err)

	err = store.FlushIndexBlock(ctx, index)
	require.NoError(t, err)

	// Wait for pause deletions after flushing to finish
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		keys, err := rc.Do(ctx, rc.B().Keys().Pattern("*:pause-block:*").Build()).AsStrSlice()
		assert.NoError(t, err)
		assert.Equal(t, 3, len(keys), "Expected 3 pause-block key after flush, but found: %v", keys)
	}, 5*time.Second, 100*time.Millisecond)

	err = store.Delete(ctx, index, *pause1)
	require.NoError(t, err)
	err = store.Delete(ctx, index, *pause2)
	require.NoError(t, err)

	// Expire TTLs
	r.FastForward(20 * time.Minute)

	var pauseIdxs []string
	for _, key := range r.Keys() {
		if strings.Contains(key, "pause-block") {
			pauseIdxs = append(pauseIdxs, key)
		}
	}

	assert.Len(t, pauseIdxs, 1)

	remainingKey := pauseIdxs[0]
	expectedKey := pauseClient.KeyGenerator().PauseBlockIndex(ctx, pause3.ID)
	assert.Equal(t, expectedKey, remainingKey, "Remaining pause-block key should be for pause3")
}
