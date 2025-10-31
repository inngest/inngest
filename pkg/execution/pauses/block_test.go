package pauses

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
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

	// Create block store
	store, err := NewBlockstore(BlockstoreOpts{
		RC:               rc,
		Bucket:           bucket,
		Bufferer:         mockBufferer,
		Leaser:           leaser,
		BlockSize:        2, // Small block size for testing
		CompactionLimit:  1,
		CompactionSample: 0.1,
		DeleteAfterFlush: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
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

	// Create block store
	store := &blockstore{
		rc:        rc,
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

func (m *mockBufferer) Delete(ctx context.Context, index Index, pause state.Pause) error {
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

func (m *mockBuffererSameTimestamp) Delete(ctx context.Context, index Index, pause state.Pause) error {
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

	store, err := NewBlockstore(BlockstoreOpts{
		RC:               rc,
		Bucket:           bucket,
		Bufferer:         mockBufferer,
		Leaser:           leaser,
		BlockSize:        2,
		CompactionLimit:  1,
		CompactionSample: 0.1,
		DeleteAfterFlush: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
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

	store, err := NewBlockstore(BlockstoreOpts{
		RC:               rc,
		Bucket:           bucket,
		Bufferer:         mockBufferer,
		Leaser:           leaser,
		BlockSize:        2,
		CompactionLimit:  3,
		CompactionSample: 1.0,
		DeleteAfterFlush: func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
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
		err := store.Delete(ctx, index, *pause1)
		require.NoError(t, err)

		deleteKey := blockDeleteKey(index)
		exists, err := rc.Do(ctx, rc.B().Sismember().Key(deleteKey).Member(pause1.ID.String()).Build()).AsBool()
		require.NoError(t, err)
		require.True(t, exists)
	})

	t.Run("delete without CreatedAt falls back to bufferer", func(t *testing.T) {
		pauseWithoutTime := state.Pause{
			ID:          uuid.New(),
			WorkspaceID: index.WorkspaceID,
		}

		err := store.Delete(ctx, index, pauseWithoutTime)
		require.NoError(t, err)

		deleteKey := blockDeleteKey(index)
		exists, err := rc.Do(ctx, rc.B().Sismember().Key(deleteKey).Member(pauseWithoutTime.ID.String()).Build()).AsBool()
		require.NoError(t, err)
		require.True(t, exists)
	})

	t.Run("delete nonexistent pause returns without error", func(t *testing.T) {
		futureTime := now.Add(10 * time.Minute)
		futurePause := state.Pause{
			ID:        uuid.New(),
			CreatedAt: futureTime,
		}

		err := store.Delete(ctx, index, futurePause)
		require.NoError(t, err)
	})

}
