package batch

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// mockBatchManager is a test double that records calls and returns configured responses.
type mockBatchManager struct {
	name string

	appendResult    *BatchAppendResult
	appendErr       error
	appendCalls     int

	bulkAppendResult *BulkAppendResult
	bulkAppendErr    error
	bulkAppendCalls  int

	startResult     string
	startErr        error
	startCalls      int

	retrieveResult  []BatchItem
	retrieveErr     error
	retrieveCalls   int

	scheduleErr     error
	scheduleCalls   int

	deleteKeysErr   error
	deleteKeysCalls int

	getBatchInfoResult *BatchInfo
	getBatchInfoErr    error
	getBatchInfoCalls  int

	deleteBatchResult *DeleteBatchResult
	deleteBatchErr    error
	deleteBatchCalls  int

	runBatchResult *RunBatchResult
	runBatchErr    error
	runBatchCalls  int

	closeCalls int
	closeErr   error
}

func (m *mockBatchManager) Append(_ context.Context, _ BatchItem, _ inngest.Function) (*BatchAppendResult, error) {
	m.appendCalls++
	return m.appendResult, m.appendErr
}

func (m *mockBatchManager) BulkAppend(_ context.Context, _ []BatchItem, _ inngest.Function) (*BulkAppendResult, error) {
	m.bulkAppendCalls++
	return m.bulkAppendResult, m.bulkAppendErr
}

func (m *mockBatchManager) StartExecution(_ context.Context, _ uuid.UUID, _ ulid.ULID, _ string) (string, error) {
	m.startCalls++
	return m.startResult, m.startErr
}

func (m *mockBatchManager) RetrieveItems(_ context.Context, _ uuid.UUID, _ ulid.ULID) ([]BatchItem, error) {
	m.retrieveCalls++
	return m.retrieveResult, m.retrieveErr
}

func (m *mockBatchManager) ScheduleExecution(_ context.Context, _ ScheduleBatchOpts) error {
	m.scheduleCalls++
	return m.scheduleErr
}

func (m *mockBatchManager) DeleteKeys(_ context.Context, _ uuid.UUID, _ ulid.ULID) error {
	m.deleteKeysCalls++
	return m.deleteKeysErr
}

func (m *mockBatchManager) GetBatchInfo(_ context.Context, _ uuid.UUID, _ string) (*BatchInfo, error) {
	m.getBatchInfoCalls++
	return m.getBatchInfoResult, m.getBatchInfoErr
}

func (m *mockBatchManager) DeleteBatch(_ context.Context, _ uuid.UUID, _ string) (*DeleteBatchResult, error) {
	m.deleteBatchCalls++
	return m.deleteBatchResult, m.deleteBatchErr
}

func (m *mockBatchManager) RunBatch(_ context.Context, _ RunBatchOpts) (*RunBatchResult, error) {
	m.runBatchCalls++
	return m.runBatchResult, m.runBatchErr
}

func (m *mockBatchManager) Close() error {
	m.closeCalls++
	return m.closeErr
}

func staticMode(mode MigrationMode) MigrationModeFunc {
	return func(_ context.Context) MigrationMode { return mode }
}

func TestMigratingBatchManager_NilNextReturnsCurrent(t *testing.T) {
	current := &mockBatchManager{name: "current"}
	result := NewMigratingBatchManager(current, nil, staticMode(MigrationModeCurrentOnly))
	require.Equal(t, current, result, "nil next should return current directly")
}

func TestMigratingBatchManager_NilModeReturnsCurrent(t *testing.T) {
	current := &mockBatchManager{name: "current"}
	next := &mockBatchManager{name: "next"}
	result := NewMigratingBatchManager(current, next, nil)
	require.Equal(t, current, result, "nil mode should return current directly")
}

func TestMigratingBatchManager_CurrentOnly(t *testing.T) {
	ctx := context.Background()
	current := &mockBatchManager{
		appendResult:    &BatchAppendResult{Status: enums.BatchNew, BatchID: "current-batch"},
		bulkAppendResult: &BulkAppendResult{Status: "new", BatchID: "current-batch"},
		startResult:     enums.BatchStatusReady.String(),
		retrieveResult:  []BatchItem{{FunctionID: uuid.New()}},
		getBatchInfoResult: &BatchInfo{BatchID: "current-batch"},
		deleteBatchResult:  &DeleteBatchResult{Deleted: true, BatchID: "current-batch"},
		runBatchResult:     &RunBatchResult{Scheduled: true, BatchID: "current-batch"},
	}
	next := &mockBatchManager{name: "next"}

	m := NewMigratingBatchManager(current, next, staticMode(MigrationModeCurrentOnly))

	t.Run("Append", func(t *testing.T) {
		res, err := m.Append(ctx, BatchItem{}, inngest.Function{})
		require.NoError(t, err)
		require.Equal(t, "current-batch", res.BatchID)
		require.Equal(t, 1, current.appendCalls)
		require.Equal(t, 0, next.appendCalls)
	})

	t.Run("BulkAppend", func(t *testing.T) {
		res, err := m.BulkAppend(ctx, []BatchItem{{}}, inngest.Function{})
		require.NoError(t, err)
		require.Equal(t, "current-batch", res.BatchID)
		require.Equal(t, 1, current.bulkAppendCalls)
		require.Equal(t, 0, next.bulkAppendCalls)
	})

	t.Run("StartExecution", func(t *testing.T) {
		res, err := m.StartExecution(ctx, uuid.New(), ulid.Make(), "ptr")
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), res)
		require.Equal(t, 1, current.startCalls)
		require.Equal(t, 0, next.startCalls)
	})

	t.Run("RetrieveItems", func(t *testing.T) {
		items, err := m.RetrieveItems(ctx, uuid.New(), ulid.Make())
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, 1, current.retrieveCalls)
		require.Equal(t, 0, next.retrieveCalls)
	})

	t.Run("ScheduleExecution", func(t *testing.T) {
		err := m.ScheduleExecution(ctx, ScheduleBatchOpts{})
		require.NoError(t, err)
		require.Equal(t, 1, current.scheduleCalls)
		require.Equal(t, 0, next.scheduleCalls)
	})

	t.Run("DeleteKeys", func(t *testing.T) {
		err := m.DeleteKeys(ctx, uuid.New(), ulid.Make())
		require.NoError(t, err)
		require.Equal(t, 1, current.deleteKeysCalls)
		require.Equal(t, 0, next.deleteKeysCalls)
	})

	t.Run("GetBatchInfo", func(t *testing.T) {
		info, err := m.GetBatchInfo(ctx, uuid.New(), "key")
		require.NoError(t, err)
		require.Equal(t, "current-batch", info.BatchID)
		require.Equal(t, 1, current.getBatchInfoCalls)
		require.Equal(t, 0, next.getBatchInfoCalls)
	})

	t.Run("DeleteBatch", func(t *testing.T) {
		res, err := m.DeleteBatch(ctx, uuid.New(), "key")
		require.NoError(t, err)
		require.True(t, res.Deleted)
		require.Equal(t, 1, current.deleteBatchCalls)
		require.Equal(t, 0, next.deleteBatchCalls)
	})

	t.Run("RunBatch", func(t *testing.T) {
		res, err := m.RunBatch(ctx, RunBatchOpts{})
		require.NoError(t, err)
		require.True(t, res.Scheduled)
		require.Equal(t, 1, current.runBatchCalls)
		require.Equal(t, 0, next.runBatchCalls)
	})
}

func TestMigratingBatchManager_DualRead_WritesToCurrent(t *testing.T) {
	ctx := context.Background()
	current := &mockBatchManager{
		appendResult:     &BatchAppendResult{Status: enums.BatchNew, BatchID: "current-batch"},
		bulkAppendResult: &BulkAppendResult{Status: "new", BatchID: "current-batch"},
	}
	next := &mockBatchManager{name: "next"}

	m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

	t.Run("Append writes to current", func(t *testing.T) {
		res, err := m.Append(ctx, BatchItem{}, inngest.Function{})
		require.NoError(t, err)
		require.Equal(t, "current-batch", res.BatchID)
		require.Equal(t, 1, current.appendCalls)
		require.Equal(t, 0, next.appendCalls)
	})

	t.Run("BulkAppend writes to current", func(t *testing.T) {
		res, err := m.BulkAppend(ctx, []BatchItem{{}}, inngest.Function{})
		require.NoError(t, err)
		require.Equal(t, "current-batch", res.BatchID)
		require.Equal(t, 1, current.bulkAppendCalls)
		require.Equal(t, 0, next.bulkAppendCalls)
	})

	t.Run("ScheduleExecution writes to current", func(t *testing.T) {
		err := m.ScheduleExecution(ctx, ScheduleBatchOpts{})
		require.NoError(t, err)
		require.Equal(t, 1, current.scheduleCalls)
		require.Equal(t, 0, next.scheduleCalls)
	})
}

func TestMigratingBatchManager_DualRead_ReadsNextFirst(t *testing.T) {
	ctx := context.Background()

	t.Run("StartExecution found on next", func(t *testing.T) {
		current := &mockBatchManager{startResult: enums.BatchStatusReady.String()}
		next := &mockBatchManager{startResult: enums.BatchStatusReady.String()}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		res, err := m.StartExecution(ctx, uuid.New(), ulid.Make(), "ptr")
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), res)
		require.Equal(t, 1, next.startCalls)
		require.Equal(t, 0, current.startCalls)
	})

	t.Run("StartExecution absent on next falls back to current", func(t *testing.T) {
		current := &mockBatchManager{startResult: enums.BatchStatusReady.String()}
		next := &mockBatchManager{startResult: enums.BatchStatusAbsent.String()}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		res, err := m.StartExecution(ctx, uuid.New(), ulid.Make(), "ptr")
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), res)
		require.Equal(t, 1, next.startCalls)
		require.Equal(t, 1, current.startCalls)
	})

	t.Run("StartExecution error on next falls back to current", func(t *testing.T) {
		current := &mockBatchManager{startResult: enums.BatchStatusReady.String()}
		next := &mockBatchManager{startErr: errors.New("connection refused")}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		res, err := m.StartExecution(ctx, uuid.New(), ulid.Make(), "ptr")
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), res)
		require.Equal(t, 1, next.startCalls)
		require.Equal(t, 1, current.startCalls)
	})

	t.Run("RetrieveItems found on next", func(t *testing.T) {
		item := BatchItem{FunctionID: uuid.New()}
		current := &mockBatchManager{retrieveResult: []BatchItem{{FunctionID: uuid.New()}}}
		next := &mockBatchManager{retrieveResult: []BatchItem{item}}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		items, err := m.RetrieveItems(ctx, uuid.New(), ulid.Make())
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, item.FunctionID, items[0].FunctionID)
		require.Equal(t, 1, next.retrieveCalls)
		require.Equal(t, 0, current.retrieveCalls)
	})

	t.Run("RetrieveItems empty on next falls back to current", func(t *testing.T) {
		item := BatchItem{FunctionID: uuid.New()}
		current := &mockBatchManager{retrieveResult: []BatchItem{item}}
		next := &mockBatchManager{retrieveResult: nil}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		items, err := m.RetrieveItems(ctx, uuid.New(), ulid.Make())
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, item.FunctionID, items[0].FunctionID)
		require.Equal(t, 1, next.retrieveCalls)
		require.Equal(t, 1, current.retrieveCalls)
	})

	t.Run("RetrieveItems error on next falls back to current", func(t *testing.T) {
		item := BatchItem{FunctionID: uuid.New()}
		current := &mockBatchManager{retrieveResult: []BatchItem{item}}
		next := &mockBatchManager{retrieveErr: errors.New("timeout")}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		items, err := m.RetrieveItems(ctx, uuid.New(), ulid.Make())
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, 1, next.retrieveCalls)
		require.Equal(t, 1, current.retrieveCalls)
	})

	t.Run("GetBatchInfo found on next", func(t *testing.T) {
		current := &mockBatchManager{getBatchInfoResult: &BatchInfo{BatchID: "current"}}
		next := &mockBatchManager{getBatchInfoResult: &BatchInfo{BatchID: "next"}}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		info, err := m.GetBatchInfo(ctx, uuid.New(), "key")
		require.NoError(t, err)
		require.Equal(t, "next", info.BatchID)
		require.Equal(t, 1, next.getBatchInfoCalls)
		require.Equal(t, 0, current.getBatchInfoCalls)
	})

	t.Run("GetBatchInfo empty BatchID on next falls back to current", func(t *testing.T) {
		current := &mockBatchManager{getBatchInfoResult: &BatchInfo{BatchID: "current"}}
		next := &mockBatchManager{getBatchInfoResult: &BatchInfo{BatchID: ""}}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		info, err := m.GetBatchInfo(ctx, uuid.New(), "key")
		require.NoError(t, err)
		require.Equal(t, "current", info.BatchID)
		require.Equal(t, 1, next.getBatchInfoCalls)
		require.Equal(t, 1, current.getBatchInfoCalls)
	})

	t.Run("DeleteBatch found on next", func(t *testing.T) {
		current := &mockBatchManager{deleteBatchResult: &DeleteBatchResult{Deleted: true, BatchID: "current"}}
		next := &mockBatchManager{deleteBatchResult: &DeleteBatchResult{Deleted: true, BatchID: "next"}}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		res, err := m.DeleteBatch(ctx, uuid.New(), "key")
		require.NoError(t, err)
		require.Equal(t, "next", res.BatchID)
		require.Equal(t, 1, next.deleteBatchCalls)
		require.Equal(t, 0, current.deleteBatchCalls)
	})

	t.Run("DeleteBatch not deleted on next falls back to current", func(t *testing.T) {
		current := &mockBatchManager{deleteBatchResult: &DeleteBatchResult{Deleted: true, BatchID: "current"}}
		next := &mockBatchManager{deleteBatchResult: &DeleteBatchResult{Deleted: false}}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		res, err := m.DeleteBatch(ctx, uuid.New(), "key")
		require.NoError(t, err)
		require.Equal(t, "current", res.BatchID)
		require.Equal(t, 1, next.deleteBatchCalls)
		require.Equal(t, 1, current.deleteBatchCalls)
	})

	t.Run("RunBatch found on next", func(t *testing.T) {
		current := &mockBatchManager{runBatchResult: &RunBatchResult{Scheduled: true, BatchID: "current"}}
		next := &mockBatchManager{runBatchResult: &RunBatchResult{Scheduled: true, BatchID: "next"}}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		res, err := m.RunBatch(ctx, RunBatchOpts{})
		require.NoError(t, err)
		require.Equal(t, "next", res.BatchID)
		require.Equal(t, 1, next.runBatchCalls)
		require.Equal(t, 0, current.runBatchCalls)
	})

	t.Run("RunBatch not scheduled on next falls back to current", func(t *testing.T) {
		current := &mockBatchManager{runBatchResult: &RunBatchResult{Scheduled: true, BatchID: "current"}}
		next := &mockBatchManager{runBatchResult: &RunBatchResult{Scheduled: false}}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

		res, err := m.RunBatch(ctx, RunBatchOpts{})
		require.NoError(t, err)
		require.Equal(t, "current", res.BatchID)
		require.Equal(t, 1, next.runBatchCalls)
		require.Equal(t, 1, current.runBatchCalls)
	})
}

func TestMigratingBatchManager_DualRead_DeleteKeysBothClusters(t *testing.T) {
	ctx := context.Background()
	current := &mockBatchManager{}
	next := &mockBatchManager{}
	m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

	err := m.DeleteKeys(ctx, uuid.New(), ulid.Make())
	require.NoError(t, err)
	require.Equal(t, 1, current.deleteKeysCalls)
	require.Equal(t, 1, next.deleteKeysCalls)
}

func TestMigratingBatchManager_DualRead_DeleteKeysJoinsErrors(t *testing.T) {
	ctx := context.Background()
	current := &mockBatchManager{deleteKeysErr: errors.New("current err")}
	next := &mockBatchManager{deleteKeysErr: errors.New("next err")}
	m := NewMigratingBatchManager(current, next, staticMode(MigrationModeDualRead))

	err := m.DeleteKeys(ctx, uuid.New(), ulid.Make())
	require.Error(t, err)
	require.Contains(t, err.Error(), "next err")
	require.Contains(t, err.Error(), "current err")
}

func TestMigratingBatchManager_WriteToNext(t *testing.T) {
	ctx := context.Background()

	t.Run("Append writes to next", func(t *testing.T) {
		current := &mockBatchManager{}
		next := &mockBatchManager{
			appendResult: &BatchAppendResult{Status: enums.BatchNew, BatchID: "next-batch"},
		}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeWriteToNext))

		res, err := m.Append(ctx, BatchItem{}, inngest.Function{})
		require.NoError(t, err)
		require.Equal(t, "next-batch", res.BatchID)
		require.Equal(t, 0, current.appendCalls)
		require.Equal(t, 1, next.appendCalls)
	})

	t.Run("BulkAppend writes to next", func(t *testing.T) {
		current := &mockBatchManager{}
		next := &mockBatchManager{
			bulkAppendResult: &BulkAppendResult{Status: "new", BatchID: "next-batch"},
		}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeWriteToNext))

		res, err := m.BulkAppend(ctx, []BatchItem{{}}, inngest.Function{})
		require.NoError(t, err)
		require.Equal(t, "next-batch", res.BatchID)
		require.Equal(t, 0, current.bulkAppendCalls)
		require.Equal(t, 1, next.bulkAppendCalls)
	})

	t.Run("ScheduleExecution writes to next", func(t *testing.T) {
		current := &mockBatchManager{}
		next := &mockBatchManager{}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeWriteToNext))

		err := m.ScheduleExecution(ctx, ScheduleBatchOpts{})
		require.NoError(t, err)
		require.Equal(t, 0, current.scheduleCalls)
		require.Equal(t, 1, next.scheduleCalls)
	})

	t.Run("reads still check next first then current", func(t *testing.T) {
		item := BatchItem{FunctionID: uuid.New()}
		current := &mockBatchManager{retrieveResult: []BatchItem{item}}
		next := &mockBatchManager{retrieveResult: nil}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeWriteToNext))

		items, err := m.RetrieveItems(ctx, uuid.New(), ulid.Make())
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, 1, next.retrieveCalls)
		require.Equal(t, 1, current.retrieveCalls)
	})
}

func TestMigratingBatchManager_Close(t *testing.T) {
	t.Run("calls both", func(t *testing.T) {
		current := &mockBatchManager{}
		next := &mockBatchManager{}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeCurrentOnly))

		err := m.Close()
		require.NoError(t, err)
		require.Equal(t, 1, current.closeCalls)
		require.Equal(t, 1, next.closeCalls)
	})

	t.Run("joins errors", func(t *testing.T) {
		current := &mockBatchManager{closeErr: errors.New("current close err")}
		next := &mockBatchManager{closeErr: errors.New("next close err")}
		m := NewMigratingBatchManager(current, next, staticMode(MigrationModeCurrentOnly))

		err := m.Close()
		require.Error(t, err)
		require.Contains(t, err.Error(), "current close err")
		require.Contains(t, err.Error(), "next close err")
	})
}

func TestMigratingBatchManager_DynamicModeSwitch(t *testing.T) {
	ctx := context.Background()

	mode := MigrationModeCurrentOnly
	modeFunc := func(_ context.Context) MigrationMode { return mode }

	current := &mockBatchManager{
		appendResult: &BatchAppendResult{Status: enums.BatchNew, BatchID: "current-batch"},
	}
	next := &mockBatchManager{
		appendResult: &BatchAppendResult{Status: enums.BatchNew, BatchID: "next-batch"},
	}
	m := NewMigratingBatchManager(current, next, modeFunc)

	// CurrentOnly: writes to current
	res, err := m.Append(ctx, BatchItem{}, inngest.Function{})
	require.NoError(t, err)
	require.Equal(t, "current-batch", res.BatchID)
	require.Equal(t, 1, current.appendCalls)
	require.Equal(t, 0, next.appendCalls)

	// Switch to WriteToNext: writes to next
	mode = MigrationModeWriteToNext
	res, err = m.Append(ctx, BatchItem{}, inngest.Function{})
	require.NoError(t, err)
	require.Equal(t, "next-batch", res.BatchID)
	require.Equal(t, 1, current.appendCalls)
	require.Equal(t, 1, next.appendCalls)
}
