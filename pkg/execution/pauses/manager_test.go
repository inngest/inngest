package pauses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
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
		PauseClient:            pauseClient,
		Bucket:                 bucket,
		Bufferer:               mockBufferer,
		Leaser:                 leaser,
		BlockSize:              lowBlockSize, // Very low limit to ensure flushing happens quickly
		CompactionGarbageRatio: 0.33,
		CompactionSample:       1.0, // Always compact for testing
		CompactionLeaser:       leaser,
		DeleteAfterFlush:       func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
		EnableBlockCompaction:  func(ctx context.Context, workspaceID uuid.UUID) bool { return true },
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

	baseTime := time.Now()

	// Test 1: Write fewer pauses than the block size limit - should not trigger flush
	pauses := createTestPauses(2, baseTime) // Less than lowBlockSize
	count, err := manager.Write(ctx, index, pauses...)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.EqualValues(t, 0, inProcessFlusher.counter, "No flush should happen when below limit")

	// Test 2: Write more pauses to exceed the block size - should trigger flush
	morePauses := createTestPauses(2, baseTime.Add(10*time.Second)) // This will make total 4 pauses, exceeding lowBlockSize
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
	// Redis iterators set a context.Canceled error when it's done iterating
	// so we want to match that behavior
	require.ErrorIs(t, iter.Error(), context.Canceled)
	assert.Equal(t, 4, pauseCount, "Should retrieve all pauses from buffer and blocks")

	// Existing blocks must remain deletable after both feature flags are disabled.
	manager.blockFlushEnabled = func(context.Context, uuid.UUID) bool { return false }
	manager.blockStoreEnabled = func(context.Context, uuid.UUID) bool { return false }
	pauseToDelete := pauses[0]
	err = manager.Delete(ctx, index, *pauseToDelete)
	require.NoError(t, err)

	// Verify the pause was deleted by trying to access it
	manager.blockStoreEnabled = alwaysEnabled
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
	require.False(t, found)
}

func TestConsumePause(t *testing.T) {
	ctx := context.Background()

	// Setup manager with mock components
	mockBufferer := &mockBufferer{}
	mockBlockStore := &mockBlockStore{}
	mockFlusher := &mockSimpleFlusher{}
	mockRunService := &mockRunService{}

	manager := NewManager(mockBufferer, mockBlockStore, WithFlusher(mockFlusher), WithBlockFlushEnabled(alwaysEnabled), WithBlockStoreEnabled(alwaysEnabled))

	eventName := "test.event"
	workspaceID := uuid.New()

	// Create a pause with an event
	pause := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Event:       &eventName,
	}

	// Test consuming a pause
	result, cleanup, err := manager.ConsumePause(ctx, mockRunService, pause, state.ConsumePauseOpts{
		Data: "test-data",
	})
	require.NoError(t, err)
	require.NoError(t, cleanup())
	assert.Equal(t, true, result.DidConsume)
	assert.True(t, mockRunService.consumeCalled, "ConsumePause should be called on RunService")
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

		assert.Equal(t, 1, mockBlockStore.deleteByIDCalled, "Existing blocks must remain deletable when flags are disabled")
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

func createTestPauses(count int, baseTime time.Time) []*state.Pause {
	pauses := make([]*state.Pause, count)
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

type mockBlockStore struct {
	deleteCalled     int
	deleteByIDCalled int
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

func (m *mockBlockStore) CleanBlock(ctx context.Context, index Index, blockID ulid.ULID) error {
	return nil
}

type mockRunService struct {
	consumeCalled bool
}

func (m *mockRunService) ConsumePause(ctx context.Context, p state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, error) {
	m.consumeCalled = true
	return state.ConsumePauseResult{DidConsume: true}, nil
}

func (m *mockRunService) Create(ctx context.Context, s statev2.CreateState) (statev2.State, error) {
	return statev2.State{}, nil
}

func (m *mockRunService) Delete(ctx context.Context, id statev2.ID, opts ...statev2.DeleteOption) error {
	return nil
}

func (m *mockRunService) Migrate(ctx context.Context, s statev2.MigrateState) error {
	return nil
}

func (m *mockRunService) LookupIdempotency(ctx context.Context, id statev2.ID, key string) (*statev2.IdempotencyEntry, error) {
	return nil, nil
}

func (m *mockRunService) Exists(ctx context.Context, id statev2.ID) (bool, error) {
	return false, nil
}

func (m *mockRunService) UpdateMetadata(ctx context.Context, id statev2.ID, config statev2.MutableConfig) error {
	return nil
}

func (m *mockRunService) SaveStep(ctx context.Context, id statev2.ID, stepID string, data []byte) (bool, error) {
	return false, nil
}

func (m *mockRunService) SavePending(ctx context.Context, id statev2.ID, pending []string) error {
	return nil
}

func (m *mockRunService) LoadMetadata(ctx context.Context, id statev2.ID, _ ...statev2.LoadMetadataOption) (statev2.Metadata, error) {
	return statev2.Metadata{}, nil
}

func (m *mockRunService) LoadEvents(ctx context.Context, id statev2.ID) ([]json.RawMessage, error) {
	return nil, nil
}

func (m *mockRunService) LoadSteps(ctx context.Context, id statev2.ID) (map[string]json.RawMessage, error) {
	return nil, nil
}

func (m *mockRunService) LoadStepInputs(ctx context.Context, id statev2.ID) (map[string]json.RawMessage, error) {
	return nil, nil
}

func (m *mockRunService) LoadStepsWithIDs(ctx context.Context, id statev2.ID, stepIDs []string) (map[string]json.RawMessage, error) {
	return nil, nil
}

func (m *mockRunService) LoadStack(ctx context.Context, id statev2.ID) ([]string, error) {
	return nil, nil
}

func (m *mockRunService) LoadPending(ctx context.Context, id statev2.ID) ([]string, error) {
	return nil, nil
}

func (m *mockRunService) LoadState(ctx context.Context, id statev2.ID) (statev2.State, error) {
	return statev2.State{}, nil
}

func (m *mockRunService) SaveDefer(ctx context.Context, id statev2.ID, d statev2.Defer) error {
	return nil
}
func (m *mockRunService) SetDeferStatus(ctx context.Context, id statev2.ID, hashedID string, status enums.DeferStatus) error {
	return nil
}
func (m *mockRunService) SaveRejectedDefer(ctx context.Context, id statev2.ID, fnSlug string, hashedID string) error {
	return nil
}
func (m *mockRunService) SaveInboxMessage(ctx context.Context, id statev2.ID, msg statev2.InboxMessage) ([]statev2.InboxWake, error) {
	return nil, nil
}
func (m *mockRunService) RecordInboxWait(ctx context.Context, id statev2.ID, wait statev2.InboxWait) (*statev2.InboxWake, error) {
	return nil, nil
}
func (m *mockRunService) TimeoutInboxWait(ctx context.Context, id statev2.ID, waitID string, data []byte) (*statev2.InboxWake, error) {
	return nil, nil
}
func (m *mockRunService) CloseInbox(ctx context.Context, id statev2.ID, closing statev2.InboxClose) ([]statev2.InboxWake, bool, error) {
	return nil, false, nil
}

func (m *mockRunService) DeleteInboxWait(ctx context.Context, id statev2.ID, waitID string) error {
	return nil
}

func (m *mockRunService) LoadInboxWaits(ctx context.Context, id statev2.ID) (statev2.InboxWaitSet, error) {
	return statev2.InboxWaitSet{}, nil
}

func (m *mockRunService) LoadDefers(ctx context.Context, id statev2.ID) (map[string]statev2.Defer, error) {
	return nil, nil
}
func (m *mockRunService) LoadDefersMeta(ctx context.Context, id statev2.ID) (map[string]statev2.DeferMeta, error) {
	return nil, nil
}

type pauseLookupBuffer struct {
	Bufferer
	pause *state.Pause
	err   error
}

func (b *pauseLookupBuffer) PauseByID(context.Context, Index, uuid.UUID) (*state.Pause, error) {
	return b.pause, b.err
}

type pauseLookupBlockStore struct {
	BlockStore
	pause *state.Pause
	calls int
}

func (b *pauseLookupBlockStore) PauseByID(context.Context, Index, uuid.UUID) (*state.Pause, error) {
	b.calls++
	return b.pause, nil
}

func TestPauseByIDBufferErrors(t *testing.T) {
	ctx := context.Background()
	p := &state.Pause{ID: uuid.New(), WorkspaceID: uuid.New()}
	storeErr := errors.New("buffer unavailable")
	for _, tt := range []struct {
		name       string
		pause      *state.Pause
		err        error
		wantErr    error
		wantBlocks bool
	}{
		{name: "buffer hit", pause: p},
		{name: "buffer miss", err: state.ErrPauseNotFound, wantBlocks: true},
		{name: "wrapped buffer miss", err: fmt.Errorf("lookup: %w", state.ErrPauseNotFound), wantBlocks: true},
		{name: "nil buffer result", wantBlocks: true},
		{name: "buffer failure", err: storeErr, wantErr: storeErr},
		{name: "partial buffer result with failure", pause: p, err: storeErr, wantErr: storeErr},
		{name: "cancellation", err: context.Canceled, wantErr: context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			buffer := &pauseLookupBuffer{pause: tt.pause, err: tt.err}
			blocks := &pauseLookupBlockStore{pause: p}
			mgr := NewManager(buffer, blocks, WithBlockStoreEnabled(alwaysEnabled))
			got, err := mgr.PauseByID(ctx, Index{WorkspaceID: p.WorkspaceID}, p.ID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Same(t, p, got)
			}
			require.Equal(t, tt.wantBlocks, blocks.calls == 1)
		})
	}
}
