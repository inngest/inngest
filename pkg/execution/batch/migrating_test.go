package batch

import (
	"context"
	"crypto/rand"
	"strings"
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

	writeToNext        bool
	accountWriteToNext map[uuid.UUID]bool
	migBM              BatchManager
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

	bmCurrent := NewRedisBatchManager(bcCurrent, nil, WithoutBuffer(), WithStoreName("current"))
	bmNext := NewRedisBatchManager(bcNext, nil, WithoutBuffer(), WithStoreName("next"))

	env := &migratingTestEnv{
		t:                  t,
		rCurrent:           rCurrent,
		rNext:              rNext,
		rcCurrent:          rcCurrent,
		rcNext:             rcNext,
		bcCurrent:          bcCurrent,
		bcNext:             bcNext,
		bmCurrent:          bmCurrent,
		bmNext:             bmNext,
		accountWriteToNext: map[uuid.UUID]bool{},
	}

	env.migBM, err = NewMigratingBatchManager(bmCurrent, bmNext, func(_ context.Context, accountID uuid.UUID) bool {
		if writeToNext, ok := env.accountWriteToNext[accountID]; ok {
			return writeToNext
		}
		return env.writeToNext
	})
	require.NoError(t, err)

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

	result, err := NewMigratingBatchManager(current, nil, func(_ context.Context, _ uuid.UUID) bool {
		return false
	})
	require.NoError(t, err)
	require.Equal(t, current, result)
}

func TestMigratingIntegration_NilCurrentErrors(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	next := NewRedisBatchManager(redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey), nil, WithoutBuffer())

	result, err := NewMigratingBatchManager(nil, next, nil)
	require.Error(t, err)
	require.Nil(t, result)
}

func TestMigratingIntegration_NilWriteToNextWritesToCurrent(t *testing.T) {
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

	ctx := context.Background()
	migBM, err := NewMigratingBatchManager(current, next, nil)
	require.NoError(t, err)
	require.NotEqual(t, current, migBM)

	fnID := uuid.New()
	fn := newFunction(fnID)

	res, _ := appendItem(ctx, t, migBM, fnID, fn)
	require.True(t, r1.Exists(bc1.KeyGenerator().Batch(ctx, fnID, ulid.MustParse(res.BatchID))))
	require.Equal(t, 0, len(r2.Keys()))

	// Reads still fall back to next.
	nextRes, nextBI := appendItem(ctx, t, next, fnID, fn)
	items, err := migBM.RetrieveItems(ctx, uuid.Nil, fnID, ulid.MustParse(nextRes.BatchID))
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, nextBI.EventID, items[0].EventID)
}

func TestMigratingIntegration_WriteToCurrent(t *testing.T) {
	ctx := context.Background()
	env := newMigratingTestEnv(t)

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
		items, err := env.migBM.RetrieveItems(ctx, uuid.Nil, fnID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, bi.EventID, items[0].EventID)
	})

	t.Run("GetBatchInfo", func(t *testing.T) {
		infos, err := env.migBM.GetBatchInfo(ctx, fnID, "")
		require.NoError(t, err)
		require.Len(t, infos, 2)
		require.Equal(t, "current", infos[0].Store)
		require.Equal(t, res.BatchID, infos[0].BatchID)
		require.Len(t, infos[0].Items, 1)
		require.Equal(t, "next", infos[1].Store)
		require.Empty(t, infos[1].BatchID)
	})

	t.Run("StartExecution", func(t *testing.T) {
		status, err := env.migBM.StartExecution(ctx, uuid.Nil, fnID, batchID, res.BatchPointerKey)
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

		// RunBatch returns one result per cluster, current first. Only current has a batch to run.
		results, err := env.migBM.RunBatch(ctx, RunBatchOpts{
			FunctionID: newFnID,
		})
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.True(t, results[0].Scheduled)
		require.False(t, results[1].Scheduled)
	})

	t.Run("DeleteBatch", func(t *testing.T) {
		delFnID := uuid.New()
		delFn := newFunction(delFnID)
		appendItem(ctx, t, env.migBM, delFnID, delFn)

		// DeleteBatch returns one result per cluster, current first. Only current has a batch to delete.
		results, err := env.migBM.DeleteBatch(ctx, delFnID, "")
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.True(t, results[0].Deleted)
		require.Equal(t, 1, results[0].ItemCount)
		require.False(t, results[1].Deleted)

		// Verify batch is gone.
		infos, err := env.migBM.GetBatchInfo(ctx, delFnID, "")
		require.NoError(t, err)
		require.Empty(t, infos[0].BatchID)
		require.Empty(t, infos[1].BatchID)
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

		fnID := uuid.New()
		fn := newFunction(fnID)

		// Append directly to current (bypassing migrating manager).
		res, bi := appendItem(ctx, t, env.bmCurrent, fnID, fn)
		batchID := ulid.MustParse(res.BatchID)

		// Read through migrating manager — should find via fallback to current.
		items, err := env.migBM.RetrieveItems(ctx, uuid.Nil, fnID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, bi.EventID, items[0].EventID)
	})

	t.Run("reads fall back to next", func(t *testing.T) {
		env := newMigratingTestEnv(t)

		fnID := uuid.New()
		fn := newFunction(fnID)

		// Append directly to next.
		resNext, biNext := appendItem(ctx, t, env.bmNext, fnID, fn)
		batchID := ulid.MustParse(resNext.BatchID)

		// Read through migrating manager — current has nothing, falls back to next.
		items, err := env.migBM.RetrieveItems(ctx, uuid.Nil, fnID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, biNext.EventID, items[0].EventID)
	})

	t.Run("reads check current first", func(t *testing.T) {
		env := newMigratingTestEnv(t)

		fnID := uuid.New()
		fn := newFunction(fnID)

		// Append to current only.
		resCur, biCur := appendItem(ctx, t, env.bmCurrent, fnID, fn)
		batchID := ulid.MustParse(resCur.BatchID)

		// RetrieveItems through migrating — found on current, the primary.
		items, err := env.migBM.RetrieveItems(ctx, uuid.Nil, fnID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, biCur.EventID, items[0].EventID)

		// GetBatchInfo through migrating reports each cluster: current has the batch.
		infos, err := env.migBM.GetBatchInfo(ctx, fnID, "")
		require.NoError(t, err)
		require.Equal(t, resCur.BatchID, infos[0].BatchID)
		require.Empty(t, infos[1].BatchID)
	})

	t.Run("BulkAppend writes to current", func(t *testing.T) {
		env := newMigratingTestEnv(t)

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

		err := env.migBM.ScheduleExecution(ctx, ScheduleBatchOpts{})
		require.NoError(t, err)
	})

	t.Run("StartExecution finds batch in either cluster", func(t *testing.T) {
		// Data on next — current reports absent, falls back to next.
		env := newMigratingTestEnv(t)

		fnID := uuid.New()
		fn := newFunction(fnID)

		resNext, _ := appendItem(ctx, t, env.bmNext, fnID, fn)
		batchID := ulid.MustParse(resNext.BatchID)

		status, err := env.migBM.StartExecution(ctx, uuid.Nil, fnID, batchID, resNext.BatchPointerKey)
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), status)

		// Data only on current — found on the primary.
		env2 := newMigratingTestEnv(t)

		fnID2 := uuid.New()
		fn2 := newFunction(fnID2)

		resCur, _ := appendItem(ctx, t, env2.bmCurrent, fnID2, fn2)
		batchID2 := ulid.MustParse(resCur.BatchID)

		// next has no keys before the call.
		require.Equal(t, 0, len(env2.rNext.Keys()))

		status2, err := env2.migBM.StartExecution(ctx, uuid.Nil, fnID2, batchID2, resCur.BatchPointerKey)
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), status2)

		// next must still have no keys — it is never consulted when current has the batch.
		require.Equal(t, 0, len(env2.rNext.Keys()))
	})

	t.Run("DeleteBatch finds batch in either cluster", func(t *testing.T) {
		// Data only on next.
		env := newMigratingTestEnv(t)

		fnID := uuid.New()
		fn := newFunction(fnID)
		appendItem(ctx, t, env.bmNext, fnID, fn)

		// One result per cluster, current first. Only next has a batch to delete.
		results, err := env.migBM.DeleteBatch(ctx, fnID, "")
		require.NoError(t, err)
		require.False(t, results[0].Deleted)
		require.True(t, results[1].Deleted)
		require.Equal(t, 1, results[1].ItemCount)

		// Data only on current.
		env2 := newMigratingTestEnv(t)

		fnID2 := uuid.New()
		fn2 := newFunction(fnID2)
		appendItem(ctx, t, env2.bmCurrent, fnID2, fn2)

		// Only current has a batch to delete.
		results2, err := env2.migBM.DeleteBatch(ctx, fnID2, "")
		require.NoError(t, err)
		require.True(t, results2[0].Deleted)
		require.Equal(t, 1, results2[0].ItemCount)
		require.False(t, results2[1].Deleted)
	})

	t.Run("RunBatch finds batch in either cluster", func(t *testing.T) {
		// Data only on next.
		env := newMigratingTestEnv(t)

		fnID := uuid.New()
		fn := newFunction(fnID)
		appendItem(ctx, t, env.bmNext, fnID, fn)

		// One result per cluster, current first. Only next has a batch to run.
		results, err := env.migBM.RunBatch(ctx, RunBatchOpts{FunctionID: fnID})
		require.NoError(t, err)
		require.False(t, results[0].Scheduled)
		require.True(t, results[1].Scheduled)

		// Data only on current.
		env2 := newMigratingTestEnv(t)

		fnID2 := uuid.New()
		fn2 := newFunction(fnID2)
		appendItem(ctx, t, env2.bmCurrent, fnID2, fn2)

		// Only current has a batch to run.
		results2, err := env2.migBM.RunBatch(ctx, RunBatchOpts{FunctionID: fnID2})
		require.NoError(t, err)
		require.True(t, results2[0].Scheduled)
		require.False(t, results2[1].Scheduled)
	})

	t.Run("DeleteKeys hits both clusters", func(t *testing.T) {
		env := newMigratingTestEnv(t)

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
	env.writeToNext = true

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
	items, err := env.migBM.RetrieveItems(ctx, uuid.Nil, fnID, batchID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, bi.EventID, items[0].EventID)

	// GetBatchInfo reports the batch in next.
	infos, err := env.migBM.GetBatchInfo(ctx, fnID, "")
	require.NoError(t, err)
	require.Empty(t, infos[0].BatchID)
	require.Equal(t, res.BatchID, infos[1].BatchID)

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

		status, err := env.migBM.StartExecution(ctx, uuid.Nil, seFnID, seBatchID, seRes.BatchPointerKey)
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), status)
	})

	t.Run("DeleteBatch from next", func(t *testing.T) {
		dbFnID := uuid.New()
		dbFn := newFunction(dbFnID)
		appendItem(ctx, t, env.bmNext, dbFnID, dbFn)

		// One result per cluster, current first. The batch is only in next.
		results, err := env.migBM.DeleteBatch(ctx, dbFnID, "")
		require.NoError(t, err)
		require.False(t, results[0].Deleted)
		require.True(t, results[1].Deleted)
	})

	t.Run("RunBatch from next", func(t *testing.T) {
		rbFnID := uuid.New()
		rbFn := newFunction(rbFnID)
		appendItem(ctx, t, env.bmNext, rbFnID, rbFn)

		// One result per cluster, current first. The batch is only in next.
		results, err := env.migBM.RunBatch(ctx, RunBatchOpts{FunctionID: rbFnID})
		require.NoError(t, err)
		require.False(t, results[0].Scheduled)
		require.True(t, results[1].Scheduled)
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
		items, err := env.migBM.RetrieveItems(ctx, uuid.Nil, curFnID, curBatchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, curBI.EventID, items[0].EventID)
	})
}

func TestMigratingIntegration_FlagFlip(t *testing.T) {
	ctx := context.Background()
	env := newMigratingTestEnv(t)

	// Start writing to current.
	fnID1 := uuid.New()
	fn1 := newFunction(fnID1)

	res1, bi1 := appendItem(ctx, t, env.migBM, fnID1, fn1)
	batchID1 := ulid.MustParse(res1.BatchID)

	// Data is in current.
	require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, fnID1, batchID1)))
	require.Equal(t, 0, len(env.rNext.Keys()))

	// Flip the account to write to next.
	env.writeToNext = true
	fnID2 := uuid.New()
	fn2 := newFunction(fnID2)

	res2, bi2 := appendItem(ctx, t, env.migBM, fnID2, fn2)
	batchID2 := ulid.MustParse(res2.BatchID)

	// New data is in next.
	require.True(t, env.rNext.Exists(env.bcNext.KeyGenerator().Batch(ctx, fnID2, batchID2)))

	// Reads find data from both clusters.
	items1, err := env.migBM.RetrieveItems(ctx, uuid.Nil, fnID1, batchID1)
	require.NoError(t, err)
	require.Len(t, items1, 1)
	require.Equal(t, bi1.EventID, items1[0].EventID)

	items2, err := env.migBM.RetrieveItems(ctx, uuid.Nil, fnID2, batchID2)
	require.NoError(t, err)
	require.Len(t, items2, 1)
	require.Equal(t, bi2.EventID, items2[0].EventID)
}

func TestMigratingIntegration_PerAccountWriteToNext(t *testing.T) {
	ctx := context.Background()
	env := newMigratingTestEnv(t)

	currentAccountID := uuid.New()
	nextAccountID := uuid.New()
	env.accountWriteToNext[currentAccountID] = false
	env.accountWriteToNext[nextAccountID] = true

	currentFnID := uuid.New()
	currentFn := newFunction(currentFnID)
	currentItem := BatchItem{
		AccountID:  currentAccountID,
		FunctionID: currentFnID,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event:      event.Event{Name: "test/current"},
	}
	currentResult, err := env.migBM.Append(ctx, currentItem, currentFn)
	require.NoError(t, err)
	currentBatchID := ulid.MustParse(currentResult.BatchID)

	nextFnID := uuid.New()
	nextFn := newFunction(nextFnID)
	nextItem := BatchItem{
		AccountID:  nextAccountID,
		FunctionID: nextFnID,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event:      event.Event{Name: "test/next"},
	}
	nextResult, err := env.migBM.Append(ctx, nextItem, nextFn)
	require.NoError(t, err)
	nextBatchID := ulid.MustParse(nextResult.BatchID)

	require.True(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, currentFnID, currentBatchID)))
	require.False(t, env.rNext.Exists(env.bcNext.KeyGenerator().Batch(ctx, currentFnID, currentBatchID)))
	require.True(t, env.rNext.Exists(env.bcNext.KeyGenerator().Batch(ctx, nextFnID, nextBatchID)))
	require.False(t, env.rCurrent.Exists(env.bcCurrent.KeyGenerator().Batch(ctx, nextFnID, nextBatchID)))

	currentItems, err := env.migBM.RetrieveItems(ctx, currentAccountID, currentFnID, currentBatchID)
	require.NoError(t, err)
	require.Len(t, currentItems, 1)

	nextItems, err := env.migBM.RetrieveItems(ctx, nextAccountID, nextFnID, nextBatchID)
	require.NoError(t, err)
	require.Len(t, nextItems, 1)
}

func TestMigratingIntegration_PrimaryErrorsDoNotFallBack(t *testing.T) {
	ctx := context.Background()

	t.Run("primary cluster error", func(t *testing.T) {
		env := newMigratingTestEnv(t)
		fnID := uuid.New()
		fn := newFunction(fnID)

		// The batch lives in next, but current is the primary and is down.
		res, _ := appendItem(ctx, t, env.bmNext, fnID, fn)
		batchID := ulid.MustParse(res.BatchID)
		env.rCurrent.SetError("current unavailable")

		_, err := env.migBM.RetrieveItems(ctx, uuid.Nil, fnID, batchID)
		require.Error(t, err)

		_, err = env.migBM.StartExecution(ctx, uuid.Nil, fnID, batchID, res.BatchPointerKey)
		require.Error(t, err)

		// next was never consulted, so the batch was not started.
		env.rCurrent.SetError("")
		status, err := env.migBM.StartExecution(ctx, uuid.Nil, fnID, batchID, res.BatchPointerKey)
		require.NoError(t, err)
		require.Equal(t, enums.BatchStatusReady.String(), status)
	})
}

func TestMigratingIntegration_DeleteBatchClearsBothClusters(t *testing.T) {
	ctx := context.Background()
	env := newMigratingTestEnv(t)
	fnID := uuid.New()
	fn := newFunction(fnID)

	// Right after a flag flip both clusters can hold an open batch for the same pointer.
	appendItem(ctx, t, env.bmCurrent, fnID, fn)
	appendItem(ctx, t, env.bmCurrent, fnID, fn)
	resNext, _ := appendItem(ctx, t, env.bmNext, fnID, fn)

	// Both clusters hold an open batch, so DeleteBatch deletes each and reports both, current first.
	results, err := env.migBM.DeleteBatch(ctx, fnID, "")
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, DeleteBatchResult{Store: "current", Deleted: true, BatchID: results[0].BatchID, ItemCount: 2}, results[0])
	require.Equal(t, DeleteBatchResult{Store: "next", Deleted: true, BatchID: resNext.BatchID, ItemCount: 1}, results[1])

	infosCur, err := env.bmCurrent.GetBatchInfo(ctx, fnID, "")
	require.NoError(t, err)
	require.Len(t, infosCur, 1)
	infoCur := infosCur[0]
	require.Empty(t, infoCur.BatchID)
	infosNext, err := env.bmNext.GetBatchInfo(ctx, fnID, "")
	require.NoError(t, err)
	require.Len(t, infosNext, 1)
	infoNext := infosNext[0]
	require.Empty(t, infoNext.BatchID)
}

func TestMigratingIntegration_Close(t *testing.T) {
	env := newMigratingTestEnv(t)
	err := env.migBM.Close()
	require.NoError(t, err)
}

func TestMigratingIntegration_DebugAPIsReportBothClusters(t *testing.T) {
	ctx := context.Background()
	env := newMigratingTestEnv(t)
	fnID := uuid.New()
	fn := newFunction(fnID)

	// Right after a flag flip both clusters can hold an open batch for the same pointer.
	resCur, _ := appendItem(ctx, t, env.bmCurrent, fnID, fn)
	appendItem(ctx, t, env.bmCurrent, fnID, fn)
	resNext, _ := appendItem(ctx, t, env.bmNext, fnID, fn)

	infos, err := env.migBM.GetBatchInfo(ctx, fnID, "")
	require.NoError(t, err)
	require.Len(t, infos, 2)
	require.Equal(t, "current", infos[0].Store)
	require.Equal(t, resCur.BatchID, infos[0].BatchID)
	require.Len(t, infos[0].Items, 2)
	require.Equal(t, "next", infos[1].Store)
	require.Equal(t, resNext.BatchID, infos[1].BatchID)
	require.Len(t, infos[1].Items, 1)

	// Both clusters hold an open batch, so RunBatch schedules each and reports both, current first.
	results, err := env.migBM.RunBatch(ctx, RunBatchOpts{FunctionID: fnID})
	require.NoError(t, err)
	require.Equal(t, []RunBatchResult{
		{Store: "current", Scheduled: true, BatchID: resCur.BatchID, ItemCount: 2},
		{Store: "next", Scheduled: true, BatchID: resNext.BatchID, ItemCount: 1},
	}, results)
}

func TestMigratingIntegration_DebugAPIErrorsNameFailingCluster(t *testing.T) {
	tests := []struct {
		name        string
		failCurrent bool
		failNext    bool
	}{
		{name: "next fails", failNext: true},
		{name: "current fails", failCurrent: true},
		{name: "both fail", failCurrent: true, failNext: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			env := newMigratingTestEnv(t)
			fnID := uuid.New()
			if tt.failCurrent {
				env.rCurrent.SetError("unavailable")
			}
			if tt.failNext {
				env.rNext.SetError("unavailable")
			}

			// A failing cluster fails the debug call, and the error names each
			// failing cluster so an operator knows which store to check.
			checkErr := func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, tt.failCurrent, strings.Contains(err.Error(), "batch store current:"))
				require.Equal(t, tt.failNext, strings.Contains(err.Error(), "batch store next:"))
			}

			_, err := env.migBM.GetBatchInfo(ctx, fnID, "")
			checkErr(t, err)
			_, err = env.migBM.RunBatch(ctx, RunBatchOpts{FunctionID: fnID})
			checkErr(t, err)
			_, err = env.migBM.DeleteBatch(ctx, fnID, "")
			checkErr(t, err)
		})
	}
}
