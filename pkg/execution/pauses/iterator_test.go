package pauses

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestDualIter(t *testing.T) {
	// Create test index
	idx := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.event",
	}

	// Create test pauses for buffer
	bufferPauses := []*state.Pause{
		{ID: uuid.New()},
		{ID: uuid.New()},
	}

	// Create test blocks
	blockIDs := []ulid.ULID{
		ulid.Make(),
		ulid.Make(),
	}

	// Create mock buffer iterator
	bufferIter := &mockPauseIterator{
		pauses: bufferPauses,
	}

	// Create mock block reader
	blockReader := &mockBlockReader{
		blocks: map[ulid.ULID]*Block{
			blockIDs[0]: {
				Pauses: []*state.Pause{
					{ID: uuid.New()},
					{ID: uuid.New()},
				},
			},
			blockIDs[1]: {
				Pauses: []*state.Pause{
					{ID: uuid.New()},
					{ID: uuid.New()},
				},
			},
		},
	}

	// Create dual iterator
	iter := newDualIter(idx, bufferIter, blockReader, blockIDs)

	// Test Count
	expectedCount := len(bufferPauses) + (len(blockIDs) * DefaultPausesPerBlock)
	require.Equal(t, expectedCount, iter.Count())

	// Test iteration through buffer
	ctx := context.Background()
	var seenPauses []uuid.UUID

	// First two pauses should come from buffer
	for i := 0; i < 2; i++ {
		require.True(t, iter.Next(ctx))
		pause := iter.Val(ctx)
		require.NotNil(t, pause)
		seenPauses = append(seenPauses, pause.ID)
	}

	// Next pauses should come from blocks
	for i := 0; i < 4; i++ {
		require.True(t, iter.Next(ctx))
		pause := iter.Val(ctx)
		require.NotNil(t, pause)
		seenPauses = append(seenPauses, pause.ID)
	}

	// No more pauses
	require.False(t, iter.Next(ctx))
	require.Nil(t, iter.Val(ctx))
	require.NoError(t, iter.Error())

	// Verify we saw all pauses
	require.Len(t, seenPauses, 6)
}

func TestDualIterConcurrentFetching(t *testing.T) {
	idx := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.event",
	}

	// Create many blocks to test concurrent fetching
	blockIDs := make([]ulid.ULID, DefaultConcurrentBlockFetches*2)
	for i := range blockIDs {
		blockIDs[i] = ulid.Make()
	}

	// Create mock block reader with delays to test concurrency
	blockReader := &mockBlockReader{
		blocks: make(map[ulid.ULID]*Block),
		delay:  10 * time.Millisecond,
	}

	// Add a single pause to each block
	for _, id := range blockIDs {
		blockReader.blocks[id] = &Block{
			Pauses: []*state.Pause{{ID: uuid.New()}},
		}
	}

	// Create dual iterator with empty buffer
	iter := newDualIter(idx, &mockPauseIterator{}, blockReader, blockIDs)

	// Test iteration
	ctx := context.Background()
	var seenPauses []uuid.UUID

	// Should be able to iterate through all blocks
	for i := 0; i < len(blockIDs); i++ {
		require.True(t, iter.Next(ctx))
		pause := iter.Val(ctx)
		require.NotNil(t, pause)
		seenPauses = append(seenPauses, pause.ID)
	}

	// No more pauses
	require.False(t, iter.Next(ctx))
	require.Nil(t, iter.Val(ctx))
	require.NoError(t, iter.Error())

	// Verify we saw all pauses
	require.Len(t, seenPauses, len(blockIDs))
}

func TestDualIterErrorHandling(t *testing.T) {
	idx := Index{
		WorkspaceID: uuid.New(),
		EventName:   "test.event",
	}

	// Create test block with error
	blockID := ulid.Make()
	expectedErr := errors.New("test error")

	blockReader := &mockBlockReader{
		blocks: make(map[ulid.ULID]*Block),
		err:    expectedErr,
	}

	// Create dual iterator with empty buffer
	iter := newDualIter(idx, &mockPauseIterator{}, blockReader, []ulid.ULID{blockID})

	// Test iteration
	ctx := context.Background()

	// Should return false on Next() when there's an error
	require.False(t, iter.Next(ctx))
	require.Equal(t, expectedErr, iter.Error())
}

// mockBlockReader implements BlockReader for testing
type mockBlockReader struct {
	blocks map[ulid.ULID]*Block
	delay  time.Duration
	err    error
}

func (m *mockBlockReader) ReadBlock(ctx context.Context, idx Index, blockID ulid.ULID) (*Block, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return nil, m.err
	}
	if block, ok := m.blocks[blockID]; ok {
		return block, nil
	}
	return nil, nil
}

func (m *mockBlockReader) BlocksSince(ctx context.Context, idx Index, since time.Time) ([]ulid.ULID, error) {
	// Not needed for this test
	return nil, nil
}

func (m *mockBlockReader) LastBlockMetadata(ctx context.Context, index Index) (*blockMetadata, error) {
	// Not needed for this test
	return nil, nil
}

func (m *mockBlockReader) IndexExists(ctx context.Context, i Index) (bool, error) {
	return len(m.blocks) > 0, nil
}

func (m *mockBlockReader) PauseByID(ctx context.Context, index Index, pauseID uuid.UUID) (*state.Pause, error) {
	for _, block := range m.blocks {
		for _, pause := range block.Pauses {
			if pause.ID == pauseID {
				return pause, nil
			}
		}
	}
	return nil, nil
}

func (m *mockBlockReader) GetBlockMetadata(ctx context.Context, index Index) (map[string]*blockMetadata, error) {
	return nil, nil // Not needed for this test
}

func (m *mockBlockReader) GetBlockDeleteCount(ctx context.Context, index Index, blockID ulid.ULID) (int64, error) {
	return 0, nil // Not needed for this test
}

func (m *mockBlockReader) GetBlockPauseIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error) {
	return nil, 0, nil // Not needed for this test
}

func (m *mockBlockReader) GetBlockDeletedIDs(ctx context.Context, index Index, blockID ulid.ULID) ([]string, int64, error) {
	return nil, 0, nil // Not needed for this test
}