package pauses

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob/memblob"
)

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

	// Create a mock bufferer that returns some test pauses
	mockBufferer := &mockBufferer{
		pauses: []*state.Pause{
			{
				ID: uuid.New(),
			},
		},
	}

	// Create block store
	store, err := NewBlockstore(BlockstoreOpts{
		RC:               rc,
		Bucket:           bucket,
		Bufferer:         mockBufferer,
		Leaser:           leaser,
		BlockSize:        1, // Small block size for testing
		CompactionLimit:  1,
		CompactionSample: 0.1,
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
	require.Len(t, block.Pauses, 1)
	require.Equal(t, mockBufferer.pauses[0].ID, block.Pauses[0].ID)
}

// mockBufferer implements the Bufferer interface for testing
type mockBufferer struct {
	pauses []*state.Pause
}

func (m *mockBufferer) Write(ctx context.Context, index Index, pauses ...*state.Pause) (int, error) {
	// For testing purposes, we'll just append the pauses to our mock buffer
	startLen := len(m.pauses)
	m.pauses = append(m.pauses, pauses...)
	return len(m.pauses) - startLen, nil
}

func (m *mockBufferer) PausesSince(ctx context.Context, index Index, since time.Time) (state.PauseIterator, error) {
	return &mockPauseIterator{pauses: m.pauses}, nil
}

func (m *mockBufferer) PauseTimestamp(ctx context.Context, pause state.Pause) (time.Time, error) {
	return time.Now(), nil
}

func (m *mockBufferer) Delete(ctx context.Context, index Index, pause state.Pause) error {
	// For testing purposes, we'll just remove the pause from our mock buffer
	for i, p := range m.pauses {
		if p.ID == pause.ID {
			m.pauses = append(m.pauses[:i], m.pauses[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("pause not found")
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
