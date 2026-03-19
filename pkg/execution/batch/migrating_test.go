package batch

import (
	"context"
	"crypto/rand"
	"testing"

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

type migratingTestEnv struct {
	t *testing.T

	rCurrent *miniredis.Miniredis
	rNext    *miniredis.Miniredis

	rcCurrent rueidis.Client
	rcNext    rueidis.Client

	bcCurrent *redis_state.BatchClient
	bcNext    *redis_state.BatchClient

	bmCurrent BatchManager
	bmNext    BatchManager

	mode    MigrationMode
	migBM   BatchManager
}

func newMigratingTestEnv(t *testing.T) *migratingTestEnv {
	t.Helper()

	rCurrent := miniredis.RunT(t)
	rNext := miniredis.RunT(t)

	rcCurrent, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{rCurrent.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { rcCurrent.Close() })

	rcNext, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{rNext.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { rcNext.Close() })

	bcCurrent := redis_state.NewBatchClient(rcCurrent, redis_state.QueueDefaultKey)
	bcNext := redis_state.NewBatchClient(rcNext, redis_state.QueueDefaultKey)

	bmCurrent := NewRedisBatchManager(bcCurrent, nil, WithoutBuffer())
	bmNext := NewRedisBatchManager(bcNext, nil, WithoutBuffer())

	env := &migratingTestEnv{
		t:         t,
		rCurrent:  rCurrent,
		rNext:     rNext,
		rcCurrent: rcCurrent,
		rcNext:    rcNext,
		bcCurrent: bcCurrent,
		bcNext:    bcNext,
		bmCurrent: bmCurrent,
		bmNext:    bmNext,
		mode:      MigrationModeCurrentOnly,
	}

	env.migBM = NewMigratingBatchManager(bmCurrent, bmNext, func(_ context.Context) MigrationMode {
		return env.mode
	})

	return env
}

func newFunction(fnID uuid.UUID) inngest.Function {
	return inngest.Function{
		ID: fnID,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
		},
	}
}

func appendItem(ctx context.Context, t *testing.T, bm BatchManager, fnID uuid.UUID, fn inngest.Function) (*BatchAppendResult, BatchItem) {
	t.Helper()
	bi := BatchItem{
		AccountID:  uuid.New(),
		FunctionID: fnID,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event: event.Event{
			Name: "test/event",
			Data: map[string]any{"hello": "world"},
		},
	}
	res, err := bm.Append(ctx, bi, fn)
	require.NoError(t, err)
	return res, bi
}

func TestMigratingIntegration_NilNextReturnsCurrent(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	current := NewRedisBatchManager(bc, nil, WithoutBuffer())

	result := NewMigratingBatchManager(current, nil, func(_ context.Context) MigrationMode {
		return MigrationModeCurrentOnly
	})
	require.Equal(t, current, result)
}

func TestMigratingIntegration_NilModeReturnsCurrent(t *testing.T) {
	r1 := miniredis.RunT(t)
	r2 := miniredis.RunT(t)

	rc1, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r1.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc1.Close()

	rc2, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r2.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc2.Close()

	bc1 := redis_state.NewBatchClient(rc1, redis_state.QueueDefaultKey)
	bc2 := redis_state.NewBatchClient(rc2, redis_state.QueueDefaultKey)

	current := NewRedisBatchManager(bc1, nil, WithoutBuffer())
	next := NewRedisBatchManager(bc2, nil, WithoutBuffer())

	result := NewMigratingBatchManager(current, next, nil)
	require.Equal(t, current, result)
}

func TestMigratingIntegration_CurrentOnly(t *testing.T) {
	ctx := context.Background()
	env := newMigratingTestEnv(t)
	env.mode = MigrationModeCurrentOnly

	fnID := uuid.New()
	fn := newFunction(fnID)

	// Append an item via the migrating manager.
	res, bi := appendItem(ctx, t, env.migBM, fnID, fn)
	require.Equal(t, enums.BatchNew, res.Status)
	require.NotEmpty(t, res.BatchID)

	batchID := ulid.MustParse(res.BatchID)

	// Data should exist in current's Redis.
	require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, fnID, batchID)))
	// Data should NOT exist in next's Redis.
	require.Equal(t, 0, len(env.rNext.Keys()))

	t.Run("RetrieveItems", func(t *testing.T) {
		items, err := env.migBM.RetrieveItems(ctx, fnID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, bi.EventID, items[0].EventID)
	})

	t.Run("GetBatchInfo", func(t *testing.T) {
		info, err := env.migBM.GetBatchInfo(ctx, fnID, "")
		require.NoError(t, err)
		require.Equal(t, res.BatchID, info.BatchID)
		require.Len(t, info.Items, 1)
	})

	t.Run("StartExecution", func(t *testing.T) {
		status, err := env.migBM.StartExecution(ctx, fnID, batchID, res.BatchPointerKey)
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), status)
	})

	t.Run("BulkAppend", func(t *testing.T) {
		baFnID := uuid.New()
		baFn := newFunction(baFnID)
		items := []BatchItem{
			{
				AccountID:  uuid.New(),
				FunctionID: baFnID,
				EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
				Event:      event.Event{Name: "test/event", Data: map[string]any{"i": 0}},
			},
			{
				AccountID:  uuid.New(),
				FunctionID: baFnID,
				EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
				Event:      event.Event{Name: "test/event", Data: map[string]any{"i": 1}},
			},
		}
		bulkRes, err := env.migBM.BulkAppend(ctx, items, baFn)
		require.NoError(t, err)
		require.NotEmpty(t, bulkRes.BatchID)
		require.Equal(t, 2, bulkRes.Committed)

		// Data should be in current only.
		baBatchID := ulid.MustParse(bulkRes.BatchID)
		require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, baFnID, baBatchID)))
		require.Equal(t, 0, len(env.rNext.Keys()))
	})

	t.Run("ScheduleExecution", func(t *testing.T) {
		// With nil queue manager, ScheduleExecution is a no-op (returns nil).
		err := env.migBM.ScheduleExecution(ctx, ScheduleBatchOpts{})
		require.NoError(t, err)
	})

	t.Run("RunBatch", func(t *testing.T) {
		newFnID := uuid.New()
		newFn := newFunction(newFnID)
		appendItem(ctx, t, env.migBM, newFnID, newFn)

		result, err := env.migBM.RunBatch(ctx, RunBatchOpts{
			FunctionID: newFnID,
		})
		require.NoError(t, err)
		require.True(t, result.Scheduled)
	})

	t.Run("DeleteBatch", func(t *testing.T) {
		delFnID := uuid.New()
		delFn := newFunction(delFnID)
		appendItem(ctx, t, env.migBM, delFnID, delFn)

		result, err := env.migBM.DeleteBatch(ctx, delFnID, "")
		require.NoError(t, err)
		require.True(t, result.Deleted)
		require.Equal(t, 1, result.ItemCount)

		// Verify batch is gone.
		info, err := env.migBM.GetBatchInfo(ctx, delFnID, "")
		require.NoError(t, err)
		require.Equal(t, "", info.BatchID)
	})

	t.Run("DeleteKeys", func(t *testing.T) {
		dkFnID := uuid.New()
		dkFn := newFunction(dkFnID)
		dkRes, _ := appendItem(ctx, t, env.migBM, dkFnID, dkFn)
		dkBatchID := ulid.MustParse(dkRes.BatchID)

		require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, dkFnID, dkBatchID)))

		err := env.migBM.DeleteKeys(ctx, dkFnID, dkBatchID)
		require.NoError(t, err)
		require.False(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, dkFnID, dkBatchID)))
	})
}

func TestMigratingIntegration_DualRead(t *testing.T) {
	ctx := context.Background()

	t.Run("writes go to current", func(t *testing.T) {
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		fnID := uuid.New()
		fn := newFunction(fnID)

		res, _ := appendItem(ctx, t, env.migBM, fnID, fn)
		batchID := ulid.MustParse(res.BatchID)

		// Data in current.
		require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, fnID, batchID)))
		// NOT in next.
		require.Equal(t, 0, len(env.rNext.Keys()))
	})

	t.Run("reads find data on current via fallback", func(t *testing.T) {
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		fnID := uuid.New()
		fn := newFunction(fnID)

		// Append directly to current (bypassing migrating manager).
		res, bi := appendItem(ctx, t, env.bmCurrent, fnID, fn)
		batchID := ulid.MustParse(res.BatchID)

		// Read through migrating manager — should find via fallback to current.
		items, err := env.migBM.RetrieveItems(ctx, fnID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, bi.EventID, items[0].EventID)
	})

	t.Run("reads prefer next", func(t *testing.T) {
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		fnID := uuid.New()
		fn := newFunction(fnID)

		// Append directly to next.
		resNext, biNext := appendItem(ctx, t, env.bmNext, fnID, fn)
		batchID := ulid.MustParse(resNext.BatchID)

		// Read through migrating manager — should return next's data.
		items, err := env.migBM.RetrieveItems(ctx, fnID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, biNext.EventID, items[0].EventID)
	})

	t.Run("reads fall back when next is empty", func(t *testing.T) {
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		fnID := uuid.New()
		fn := newFunction(fnID)

		// Append to current only.
		resCur, biCur := appendItem(ctx, t, env.bmCurrent, fnID, fn)
		batchID := ulid.MustParse(resCur.BatchID)

		// RetrieveItems through migrating — next has nothing, falls back to current.
		items, err := env.migBM.RetrieveItems(ctx, fnID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, biCur.EventID, items[0].EventID)

		// GetBatchInfo through migrating — next has nothing, falls back to current.
		info, err := env.migBM.GetBatchInfo(ctx, fnID, "")
		require.NoError(t, err)
		require.Equal(t, resCur.BatchID, info.BatchID)
	})

	t.Run("BulkAppend writes to current", func(t *testing.T) {
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		fnID := uuid.New()
		fn := newFunction(fnID)
		items := []BatchItem{
			{
				AccountID:  uuid.New(),
				FunctionID: fnID,
				EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
				Event:      event.Event{Name: "test/event", Data: map[string]any{"i": 0}},
			},
		}
		bulkRes, err := env.migBM.BulkAppend(ctx, items, fn)
		require.NoError(t, err)
		require.NotEmpty(t, bulkRes.BatchID)

		batchID := ulid.MustParse(bulkRes.BatchID)
		require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, fnID, batchID)))
		require.Equal(t, 0, len(env.rNext.Keys()))
	})

	t.Run("ScheduleExecution writes to current", func(t *testing.T) {
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		err := env.migBM.ScheduleExecution(ctx, ScheduleBatchOpts{})
		require.NoError(t, err)
	})

	t.Run("StartExecution prefers next then falls back", func(t *testing.T) {
		// Data on next — returns from next.
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		fnID := uuid.New()
		fn := newFunction(fnID)

		resNext, _ := appendItem(ctx, t, env.bmNext, fnID, fn)
		batchID := ulid.MustParse(resNext.BatchID)

		status, err := env.migBM.StartExecution(ctx, fnID, batchID, resNext.BatchPointerKey)
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), status)

		// Data only on current — falls back.
		env2 := newMigratingTestEnv(t)
		env2.mode = MigrationModeDualRead

		fnID2 := uuid.New()
		fn2 := newFunction(fnID2)

		resCur, _ := appendItem(ctx, t, env2.bmCurrent, fnID2, fn2)
		batchID2 := ulid.MustParse(resCur.BatchID)

		status2, err := env2.migBM.StartExecution(ctx, fnID2, batchID2, resCur.BatchPointerKey)
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), status2)
	})

	t.Run("DeleteBatch prefers next then falls back", func(t *testing.T) {
		// Data on next — deletes from next.
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		fnID := uuid.New()
		fn := newFunction(fnID)
		appendItem(ctx, t, env.bmNext, fnID, fn)

		result, err := env.migBM.DeleteBatch(ctx, fnID, "")
		require.NoError(t, err)
		require.True(t, result.Deleted)
		require.Equal(t, 1, result.ItemCount)

		// Data only on current — falls back.
		env2 := newMigratingTestEnv(t)
		env2.mode = MigrationModeDualRead

		fnID2 := uuid.New()
		fn2 := newFunction(fnID2)
		appendItem(ctx, t, env2.bmCurrent, fnID2, fn2)

		result2, err := env2.migBM.DeleteBatch(ctx, fnID2, "")
		require.NoError(t, err)
		require.True(t, result2.Deleted)
		require.Equal(t, 1, result2.ItemCount)
	})

	t.Run("RunBatch prefers next then falls back", func(t *testing.T) {
		// Data on next — runs from next.
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		fnID := uuid.New()
		fn := newFunction(fnID)
		appendItem(ctx, t, env.bmNext, fnID, fn)

		result, err := env.migBM.RunBatch(ctx, RunBatchOpts{FunctionID: fnID})
		require.NoError(t, err)
		require.True(t, result.Scheduled)

		// Data only on current — falls back.
		env2 := newMigratingTestEnv(t)
		env2.mode = MigrationModeDualRead

		fnID2 := uuid.New()
		fn2 := newFunction(fnID2)
		appendItem(ctx, t, env2.bmCurrent, fnID2, fn2)

		result2, err := env2.migBM.RunBatch(ctx, RunBatchOpts{FunctionID: fnID2})
		require.NoError(t, err)
		require.True(t, result2.Scheduled)
	})

	t.Run("DeleteKeys hits both clusters", func(t *testing.T) {
		env := newMigratingTestEnv(t)
		env.mode = MigrationModeDualRead

		fnID := uuid.New()
		fn := newFunction(fnID)

		// Append to current directly.
		resCur, _ := appendItem(ctx, t, env.bmCurrent, fnID, fn)
		batchIDCur := ulid.MustParse(resCur.BatchID)

		// Append to next directly (different batch but same fnID — use different event).
		// We need to use the same batchID for DeleteKeys to remove from both.
		// Since batches are created independently, we'll verify keys exist in each, then delete.
		resNext, _ := appendItem(ctx, t, env.bmNext, fnID, fn)
		batchIDNext := ulid.MustParse(resNext.BatchID)

		require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, fnID, batchIDCur)))
		require.True(t, env.rNext.Exists(env.bcNext.KeyGenerator().Batch(ctx, fnID, batchIDNext)))

		// DeleteKeys for current's batchID — hits both clusters.
		err := env.migBM.DeleteKeys(ctx, fnID, batchIDCur)
		require.NoError(t, err)
		require.False(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, fnID, batchIDCur)))

		// DeleteKeys for next's batchID — hits both clusters.
		err = env.migBM.DeleteKeys(ctx, fnID, batchIDNext)
		require.NoError(t, err)
		require.False(t, env.rNext.Exists(env.bcNext.KeyGenerator().Batch(ctx, fnID, batchIDNext)))
	})
}

func TestMigratingIntegration_WriteToNext(t *testing.T) {
	ctx := context.Background()
	env := newMigratingTestEnv(t)
	env.mode = MigrationModeWriteToNext

	fnID := uuid.New()
	fn := newFunction(fnID)

	// Append via migrating manager — should write to next.
	res, bi := appendItem(ctx, t, env.migBM, fnID, fn)
	require.Equal(t, enums.BatchNew, res.Status)
	batchID := ulid.MustParse(res.BatchID)

	// Data should be in next's Redis.
	require.True(t, env.rNext.Exists(env.bcNext.KeyGenerator().Batch(ctx, fnID, batchID)))
	// Data should NOT be in current's Redis.
	require.Equal(t, 0, len(env.rCurrent.Keys()))

	// Reads still find data from next.
	items, err := env.migBM.RetrieveItems(ctx, fnID, batchID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, bi.EventID, items[0].EventID)

	// GetBatchInfo reads from next.
	info, err := env.migBM.GetBatchInfo(ctx, fnID, "")
	require.NoError(t, err)
	require.Equal(t, res.BatchID, info.BatchID)

	t.Run("BulkAppend writes to next", func(t *testing.T) {
		baFnID := uuid.New()
		baFn := newFunction(baFnID)
		baItems := []BatchItem{
			{
				AccountID:  uuid.New(),
				FunctionID: baFnID,
				EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
				Event:      event.Event{Name: "test/event", Data: map[string]any{"i": 0}},
			},
		}
		bulkRes, err := env.migBM.BulkAppend(ctx, baItems, baFn)
		require.NoError(t, err)
		require.NotEmpty(t, bulkRes.BatchID)

		baBatchID := ulid.MustParse(bulkRes.BatchID)
		require.True(t, env.rNext.Exists(env.bcNext.KeyGenerator().Batch(ctx, baFnID, baBatchID)))
	})

	t.Run("ScheduleExecution writes to next", func(t *testing.T) {
		err := env.migBM.ScheduleExecution(ctx, ScheduleBatchOpts{})
		require.NoError(t, err)
	})

	t.Run("StartExecution prefers next", func(t *testing.T) {
		seFnID := uuid.New()
		seFn := newFunction(seFnID)
		seRes, _ := appendItem(ctx, t, env.bmNext, seFnID, seFn)
		seBatchID := ulid.MustParse(seRes.BatchID)

		status, err := env.migBM.StartExecution(ctx, seFnID, seBatchID, seRes.BatchPointerKey)
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), status)
	})

	t.Run("DeleteBatch from next", func(t *testing.T) {
		dbFnID := uuid.New()
		dbFn := newFunction(dbFnID)
		appendItem(ctx, t, env.bmNext, dbFnID, dbFn)

		result, err := env.migBM.DeleteBatch(ctx, dbFnID, "")
		require.NoError(t, err)
		require.True(t, result.Deleted)
	})

	t.Run("RunBatch from next", func(t *testing.T) {
		rbFnID := uuid.New()
		rbFn := newFunction(rbFnID)
		appendItem(ctx, t, env.bmNext, rbFnID, rbFn)

		result, err := env.migBM.RunBatch(ctx, RunBatchOpts{FunctionID: rbFnID})
		require.NoError(t, err)
		require.True(t, result.Scheduled)
	})

	t.Run("DeleteKeys hits both clusters", func(t *testing.T) {
		dkFnID := uuid.New()
		dkFn := newFunction(dkFnID)

		resCur, _ := appendItem(ctx, t, env.bmCurrent, dkFnID, dkFn)
		curBatchID := ulid.MustParse(resCur.BatchID)
		require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, dkFnID, curBatchID)))

		err := env.migBM.DeleteKeys(ctx, dkFnID, curBatchID)
		require.NoError(t, err)
		require.False(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, dkFnID, curBatchID)))
	})

	t.Run("reads fall back to current when next is empty", func(t *testing.T) {
		curFnID := uuid.New()
		curFn := newFunction(curFnID)

		// Append directly to current.
		curRes, curBI := appendItem(ctx, t, env.bmCurrent, curFnID, curFn)
		curBatchID := ulid.MustParse(curRes.BatchID)

		// RetrieveItems through migrating — next has nothing for this fn, falls back to current.
		items, err := env.migBM.RetrieveItems(ctx, curFnID, curBatchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, curBI.EventID, items[0].EventID)
	})
}

func TestMigratingIntegration_DynamicModeSwitch(t *testing.T) {
	ctx := context.Background()
	env := newMigratingTestEnv(t)

	// Start in CurrentOnly mode.
	env.mode = MigrationModeCurrentOnly
	fnID1 := uuid.New()
	fn1 := newFunction(fnID1)

	res1, bi1 := appendItem(ctx, t, env.migBM, fnID1, fn1)
	batchID1 := ulid.MustParse(res1.BatchID)

	// Data is in current.
	require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, fnID1, batchID1)))
	require.Equal(t, 0, len(env.rNext.Keys()))

	// Switch to WriteToNext.
	env.mode = MigrationModeWriteToNext
	fnID2 := uuid.New()
	fn2 := newFunction(fnID2)

	res2, bi2 := appendItem(ctx, t, env.migBM, fnID2, fn2)
	batchID2 := ulid.MustParse(res2.BatchID)

	// New data is in next.
	require.True(t, env.rNext.Exists(env.bcNext.KeyGenerator().Batch(ctx, fnID2, batchID2)))

	// Reads find data from both clusters.
	items1, err := env.migBM.RetrieveItems(ctx, fnID1, batchID1)
	require.NoError(t, err)
	require.Len(t, items1, 1)
	require.Equal(t, bi1.EventID, items1[0].EventID)

	items2, err := env.migBM.RetrieveItems(ctx, fnID2, batchID2)
	require.NoError(t, err)
	require.Len(t, items2, 1)
	require.Equal(t, bi2.EventID, items2[0].EventID)
}

func TestMigratingIntegration_Close(t *testing.T) {
	env := newMigratingTestEnv(t)
	err := env.migBM.Close()
	require.NoError(t, err)
}
