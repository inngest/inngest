package batch

import (
	"context"
	"crypto/rand"
	"encoding/json"
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

type recordedBatchDelete struct {
	residencyDuration time.Duration
}

type recordingBatchMetricRecorder struct {
	committedBytes []int64
	deletes        []recordedBatchDelete
}

func (r *recordingBatchMetricRecorder) AccountScopedTags(context.Context, uuid.UUID, uuid.UUID) map[string]any {
	return map[string]any{"backend": "test"}
}

func (r *recordingBatchMetricRecorder) RecordCommit(_ context.Context, _ map[string]any, committedBytes int64) {
	r.committedBytes = append(r.committedBytes, committedBytes)
}

func (r *recordingBatchMetricRecorder) RecordDelete(_ context.Context, residencyDuration time.Duration) {
	r.deletes = append(r.deletes, recordedBatchDelete{
		residencyDuration: residencyDuration,
	})
}

func TestAppendCommittedBytes(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()
	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	recorder := &recordingBatchMetricRecorder{}
	bm := NewRedisBatchManager(
		bc,
		nil,
		WithoutBuffer(),
		WithBatchMetricRecorder(recorder),
	)

	fnID := uuid.New()
	fn := inngest.Function{
		ID: fnID,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
		},
	}
	item := BatchItem{
		AccountID:  uuid.New(),
		FunctionID: fnID,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event:      event.Event{ID: "one", Data: map[string]any{"value": "first"}},
	}
	encoded, err := json.Marshal(item)
	require.NoError(t, err)

	_, err = bm.Append(ctx, item, fn)
	require.NoError(t, err)
	require.Equal(t, []int64{int64(len(encoded))}, recorder.committedBytes)

	secondItem := BatchItem{
		AccountID:  item.AccountID,
		FunctionID: fnID,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event:      event.Event{ID: "two", Data: map[string]any{"value": "second"}},
	}
	secondEncoded, err := json.Marshal(secondItem)
	require.NoError(t, err)
	_, err = bm.Append(ctx, secondItem, fn)
	require.NoError(t, err)
	require.Equal(t, []int64{int64(len(encoded)), int64(len(secondEncoded))}, recorder.committedBytes)

	duplicate, err := bm.Append(ctx, item, fn)
	require.NoError(t, err)
	require.Equal(t, enums.BatchItemExists, duplicate.Status)
	require.Equal(t, []int64{int64(len(encoded)), int64(len(secondEncoded))}, recorder.committedBytes)
}

func TestBatchScriptsAppendToExistingBatch(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()
	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	bm := NewRedisBatchManager(bc, nil, WithoutBuffer())
	accountID := uuid.New()
	functionID := uuid.New()
	batchID := ulid.MustNew(ulid.Now(), rand.Reader)
	fn := inngest.Function{
		ID: functionID,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
		},
	}

	// Seed only the pointer, list, and status written by batching before the
	// metrics fields existed. The new scripts must continue this batch without
	// requiring a metadata migration or creating a replacement batch.
	legacyItem := BatchItem{
		AccountID:  accountID,
		FunctionID: functionID,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event:      event.Event{ID: "legacy"},
	}
	legacyJSON, err := json.Marshal(legacyItem)
	require.NoError(t, err)
	pointerKey := bc.KeyGenerator().BatchPointer(ctx, functionID)
	batchKey := bc.KeyGenerator().Batch(ctx, functionID, batchID)
	metadataKey := bc.KeyGenerator().BatchMetadata(ctx, functionID, batchID)
	require.NoError(t, r.Set(pointerKey, batchID.String()))
	_, err = r.RPush(batchKey, string(legacyJSON))
	require.NoError(t, err)
	r.HSet(metadataKey, "status", enums.BatchStatusPending.String())

	appendItem := BatchItem{
		AccountID:  accountID,
		FunctionID: functionID,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event:      event.Event{ID: "append"},
	}
	appendResult, err := bm.Append(ctx, appendItem, fn)
	require.NoError(t, err)
	require.Equal(t, batchID.String(), appendResult.BatchID)
	require.Equal(t, enums.BatchAppend, appendResult.Status)

	bulkItem := BatchItem{
		AccountID:  accountID,
		FunctionID: functionID,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event:      event.Event{ID: "bulk-append"},
	}
	bulkResult, err := bm.BulkAppend(ctx, []BatchItem{bulkItem}, fn)
	require.NoError(t, err)
	require.Equal(t, batchID.String(), bulkResult.BatchID)
	require.Equal(t, "append", bulkResult.Status)
	require.Equal(t, 1, bulkResult.Committed)
	items, err := bm.RetrieveItems(ctx, functionID, batchID)
	require.NoError(t, err)
	require.Len(t, items, 3)
	require.Equal(t, []string{"legacy", "append", "bulk-append"}, []string{
		items[0].Event.ID,
		items[1].Event.ID,
		items[2].Event.ID,
	})
	pointer, err := r.Get(pointerKey)
	require.NoError(t, err)
	require.Equal(t, batchID.String(), pointer)
}

func TestBatchSizeLimit(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	// make the size limit crazy small (10 bytes) for verification purposes
	bm := NewRedisBatchManager(bc, nil, WithoutBuffer(), WithRedisBatchSizeLimit(10))

	accountId := uuid.New()
	fnId := uuid.New()

	res, err := bm.Append(context.Background(), BatchItem{
		AccountID:  accountId,
		FunctionID: fnId,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event: event.Event{
			ID: "test-event",
			Data: map[string]any{
				"hello": "world",
				"yolo":  "yoloyoloyoloyoloyoloyoloyoloyoloyoloyoloyoloyolo",
			},
		},
		Version: 0,
	}, inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchMaxSize, res.Status)
}

func TestBatchAppendIdempotence(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	bm := NewRedisBatchManager(bc, nil, WithoutBuffer())

	accountId := uuid.New()
	fnId := uuid.New()
	function := inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
		},
	}
	bi := BatchItem{
		AccountID:  accountId,
		FunctionID: fnId,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event: event.Event{
			ID: "test-event",
			Data: map[string]any{
				"hello": "world",
			},
		},
		Version: 0,
	}

	// add event to a batch, batch is currently empty, should return status New
	res, err := bm.Append(context.Background(), bi, function)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchNew, res.Status)

	// add same event again to a batch, duplicate first event in a batch should also return status New
	res, err = bm.Append(context.Background(), bi, function)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchNew, res.Status)

	// add a second event to a batch, should be appended to the same batch
	bi.EventID = ulid.MustNew(ulid.Now(), rand.Reader)
	res, err = bm.Append(context.Background(), bi, function)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchAppend, res.Status)

	// add the same second event to the same batch, should return BatchItemExists.
	res, err = bm.Append(context.Background(), bi, function)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchItemExists, res.Status)

}

// When the same event is appended to different batches, we would end up processing the duplicate event a second time in the second batch.
// Per-event idempotence keys span across batches within their TTL window, so duplicates are rejected even after batch rotation.
func TestBatchAppendIdempotenceDifferentBatches(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	bm := NewRedisBatchManager(bc, nil, WithoutBuffer()) // Test direct append behavior

	accountId := uuid.New()
	fnId := uuid.New()
	function := inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
		},
	}
	bi := BatchItem{
		AccountID:  accountId,
		FunctionID: fnId,
		Event: event.Event{
			ID: "test-event",
			Data: map[string]any{
				"hello": "world",
			},
		},
		Version: 0,
	}

	var lastBatchID string
	for i := range 10 {
		// append a new event to the batch
		bi.EventID = ulid.MustNew(ulid.Now(), rand.Reader)

		res, err := bm.Append(context.Background(), bi, function)
		require.NoError(t, err)
		require.NotEmpty(t, res.BatchID)
		require.NotEmpty(t, res.BatchPointerKey)
		switch i {
		case 0:
			require.Equal(t, enums.BatchNew, res.Status)
		case 9:
			require.Equal(t, enums.BatchFull, res.Status)
		default:
			require.Equal(t, enums.BatchAppend, res.Status)
		}
		lastBatchID = res.BatchID
	}

	// Append the last batchitem again. This should be rejected from the next batch.
	res, err := bm.Append(context.Background(), bi, function)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEqual(t, res.BatchID, lastBatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchItemExists, res.Status)
}

func TestBatchCleanup(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	bm := NewRedisBatchManager(bc, nil, WithoutBuffer()) // Test direct append behavior

	accountId := uuid.New()
	fnId := uuid.New()

	res, err := bm.Append(context.Background(), BatchItem{
		AccountID:  accountId,
		FunctionID: fnId,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event: event.Event{
			ID: "test-event",
		},
		Version: 0,
	}, inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchNew, res.Status)

	require.True(t, r.Exists(bc.KeyGenerator().Batch(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.True(t, r.Exists(bc.KeyGenerator().BatchMetadata(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.True(t, r.Exists(bc.KeyGenerator().BatchPointer(context.Background(), fnId)))
	// Per-event idempotence key exists instead of a single sorted set
	require.Equal(t, 4, len(r.Keys()))

	err = bm.DeleteKeys(context.Background(), fnId, ulid.MustParse(res.BatchID))
	require.NoError(t, err)

	require.False(t, r.Exists(bc.KeyGenerator().Batch(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.False(t, r.Exists(bc.KeyGenerator().BatchMetadata(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.True(t, r.Exists(bc.KeyGenerator().BatchPointer(context.Background(), fnId)))
	// Per-event idem key + batch pointer remain
	require.Equal(t, 2, len(r.Keys()))
}

func TestGetBatchInfo(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	bm := NewRedisBatchManager(bc, nil)

	accountId := uuid.New()
	workspaceId := uuid.New()
	appId := uuid.New()
	fnId := uuid.New()

	t.Run("no batch exists returns empty info", func(t *testing.T) {
		info, err := bm.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Equal(t, "", info.BatchID)
		require.Empty(t, info.Items)
		require.Equal(t, "none", info.Status)
	})

	t.Run("batch with default key", func(t *testing.T) {
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		eventID := ulid.MustNew(ulid.Now(), rand.Reader)
		bi := BatchItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      fnId,
			FunctionVersion: 1,
			EventID:         eventID,
			Event: event.Event{
				Name: "test/event",
				Data: map[string]any{"foo": "bar"},
			},
		}

		res, err := bm.Append(context.Background(), bi, fn)
		require.NoError(t, err)
		require.NotEmpty(t, res.BatchID)

		// Query with empty batch key (should use default)
		info, err := bm.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Equal(t, res.BatchID, info.BatchID)
		require.Len(t, info.Items, 1)
		require.Equal(t, eventID, info.Items[0].EventID)

		// Query with explicit "default" key should return same result
		info2, err := bm.GetBatchInfo(context.Background(), fnId, "default")
		require.NoError(t, err)
		require.Equal(t, res.BatchID, info2.BatchID)
		require.Len(t, info2.Items, 1)
	})

	t.Run("batch with custom key expression", func(t *testing.T) {
		customFnId := uuid.New()
		customBatchKey := "user-123"

		fn := inngest.Function{
			ID: customFnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
				Key:     strPtr("event.data.user_id"),
			},
		}

		eventID := ulid.MustNew(ulid.Now(), rand.Reader)
		bi := BatchItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      customFnId,
			FunctionVersion: 1,
			EventID:         eventID,
			Event: event.Event{
				Name: "test/event",
				Data: map[string]any{"user_id": customBatchKey},
			},
		}

		res, err := bm.Append(context.Background(), bi, fn)
		require.NoError(t, err)
		require.NotEmpty(t, res.BatchID)

		// Query with the custom batch key
		info, err := bm.GetBatchInfo(context.Background(), customFnId, customBatchKey)
		require.NoError(t, err)
		require.Equal(t, res.BatchID, info.BatchID)
		require.Len(t, info.Items, 1)
		require.Equal(t, eventID, info.Items[0].EventID)

		// Query with default key should NOT find this batch
		info2, err := bm.GetBatchInfo(context.Background(), customFnId, "default")
		require.NoError(t, err)
		require.Equal(t, "", info2.BatchID)
		require.Empty(t, info2.Items)
	})

	t.Run("batch with multiple items", func(t *testing.T) {
		multiFnId := uuid.New()
		fn := inngest.Function{
			ID: multiFnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		var eventIDs []ulid.ULID
		for i := 0; i < 3; i++ {
			eventID := ulid.MustNew(ulid.Now(), rand.Reader)
			eventIDs = append(eventIDs, eventID)
			bi := BatchItem{
				AccountID:       accountId,
				WorkspaceID:     workspaceId,
				AppID:           appId,
				FunctionID:      multiFnId,
				FunctionVersion: 1,
				EventID:         eventID,
				Event: event.Event{
					Name: "test/event",
					Data: map[string]any{"index": i},
				},
			}
			_, err := bm.Append(context.Background(), bi, fn)
			require.NoError(t, err)
		}

		info, err := bm.GetBatchInfo(context.Background(), multiFnId, "")
		require.NoError(t, err)
		require.NotEmpty(t, info.BatchID)
		require.Len(t, info.Items, 3)

		// Verify all event IDs are present
		foundIDs := make(map[string]bool)
		for _, item := range info.Items {
			foundIDs[item.EventID.String()] = true
		}
		for _, expectedID := range eventIDs {
			require.True(t, foundIDs[expectedID.String()], "expected event ID %s not found", expectedID)
		}
	})

	t.Run("non-existent function returns empty", func(t *testing.T) {
		nonExistentFnId := uuid.New()
		info, err := bm.GetBatchInfo(context.Background(), nonExistentFnId, "")
		require.NoError(t, err)
		require.Equal(t, "", info.BatchID)
		require.Empty(t, info.Items)
		require.Equal(t, "none", info.Status)
	})
}

func strPtr(s string) *string {
	return &s
}

// TestPerEventIdempotenceKeys verifies that per-event SET keys are used for dedup
// instead of the legacy sorted set, and that they expire independently.
func TestPerEventIdempotenceKeys(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	bm := NewRedisBatchManager(bc, nil, WithoutBuffer(), WithRedisBatchIdempotenceSetTTL(10))

	fnId := uuid.New()
	fn := inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 100,
			Timeout: "60s",
		},
	}

	eventID := ulid.MustNew(ulid.Now(), rand.Reader)
	bi := BatchItem{
		AccountID:  uuid.New(),
		FunctionID: fnId,
		EventID:    eventID,
		Event:      event.Event{ID: "test"},
	}

	// Append first event
	res, err := bm.Append(context.Background(), bi, fn)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.Equal(t, enums.BatchNew, res.Status)

	// Verify per-event idem key was created (not the legacy sorted set)
	prefix := bc.KeyGenerator().QueuePrefix(context.Background(), fnId)
	idemKey := prefix + ":batch_idem:" + eventID.String()
	require.True(t, r.Exists(idemKey), "per-event idem key should exist")
	require.False(t, r.Exists(bc.KeyGenerator().BatchIdempotenceKey(context.Background(), fnId)), "legacy sorted set should NOT be created")

	// Same event should be rejected as duplicate
	res, err = bm.Append(context.Background(), bi, fn)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.Equal(t, enums.BatchNew, res.Status) // size == 1, so returns "new"

	// Second event should work fine
	bi2 := bi
	bi2.EventID = ulid.MustNew(ulid.Now(), rand.Reader)
	res, err = bm.Append(context.Background(), bi2, fn)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.Equal(t, enums.BatchAppend, res.Status)

	// Per-event keys should expire independently
	r.FastForward(11 * time.Second)
	require.False(t, r.Exists(idemKey), "per-event idem key should have expired")

	// After expiry, same event ID can be appended again (TTL window passed)
	res, err = bm.Append(context.Background(), bi, fn)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.Equal(t, enums.BatchAppend, res.Status)
}

// TestPerEventIdempotenceBulkAppend verifies per-event dedup works in the bulk append path.
func TestPerEventIdempotenceBulkAppend(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	mgr := NewRedisBatchManager(bc, nil, WithoutBuffer())
	bm := mgr.(*redisBatchManager)

	fnId := uuid.New()
	fn := inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 100,
			Timeout: "60s",
		},
	}

	event1ID := ulid.MustNew(ulid.Now(), rand.Reader)
	event2ID := ulid.MustNew(ulid.Now(), rand.Reader)
	event3ID := ulid.MustNew(ulid.Now(), rand.Reader)

	items := []BatchItem{
		{AccountID: uuid.New(), FunctionID: fnId, EventID: event1ID, Event: event.Event{ID: "e1"}},
		{AccountID: uuid.New(), FunctionID: fnId, EventID: event2ID, Event: event.Event{ID: "e2"}},
		{AccountID: uuid.New(), FunctionID: fnId, EventID: event3ID, Event: event.Event{ID: "e3"}},
	}
	var committedBytes int64
	for _, item := range items {
		encoded, err := json.Marshal(item)
		require.NoError(t, err)
		committedBytes += int64(len(encoded))
	}

	// Bulk append 3 events
	res, err := bm.BulkAppend(context.Background(), items, fn)
	require.NoError(t, err)
	require.Equal(t, 3, res.Committed)
	require.Equal(t, 0, res.Duplicates)
	require.Equal(t, committedBytes, res.CommittedBytes)

	// Bulk append same 3 events again — all should be duplicates
	res, err = bm.BulkAppend(context.Background(), items, fn)
	require.NoError(t, err)
	require.Equal(t, "itemexists", res.Status)
	require.Equal(t, 0, res.Committed)
	require.Equal(t, 3, res.Duplicates)
	require.Zero(t, res.CommittedBytes)

	// Bulk append mix of new and duplicate
	event4ID := ulid.MustNew(ulid.Now(), rand.Reader)
	mixedItems := []BatchItem{
		{AccountID: uuid.New(), FunctionID: fnId, EventID: event1ID, Event: event.Event{ID: "e1"}}, // dup
		{AccountID: uuid.New(), FunctionID: fnId, EventID: event4ID, Event: event.Event{ID: "e4"}}, // new
	}
	res, err = bm.BulkAppend(context.Background(), mixedItems, fn)
	require.NoError(t, err)
	require.Equal(t, 1, res.Committed)
	require.Equal(t, 1, res.Duplicates)
	encodedEvent4, err := json.Marshal(mixedItems[1])
	require.NoError(t, err)
	require.Equal(t, int64(len(encodedEvent4)), res.CommittedBytes)
}

func TestBulkAppendPayloadAccountingAcrossOverflow(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()
	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	bm := NewRedisBatchManager(bc, nil, WithoutBuffer())
	fnID := uuid.New()
	fn := inngest.Function{ID: fnID, EventBatch: &inngest.EventBatchConfig{MaxSize: 2, Timeout: "60s"}}
	accountID := uuid.New()
	items := []BatchItem{
		{AccountID: accountID, FunctionID: fnID, EventID: ulid.MustNew(ulid.Now(), rand.Reader), Event: event.Event{ID: "one"}},
		{AccountID: accountID, FunctionID: fnID, EventID: ulid.MustNew(ulid.Now(), rand.Reader), Event: event.Event{ID: "two"}},
		{AccountID: accountID, FunctionID: fnID, EventID: ulid.MustNew(ulid.Now(), rand.Reader), Event: event.Event{ID: "three"}},
	}
	encodedBytes := make([]int, len(items))
	for i, item := range items {
		encoded, err := json.Marshal(item)
		require.NoError(t, err)
		encodedBytes[i] = len(encoded)
	}

	result, err := bm.BulkAppend(ctx, items, fn)
	require.NoError(t, err)
	require.Equal(t, "overflow", result.Status)
	require.Equal(t, 3, result.Committed)
	require.Equal(t, int64(encodedBytes[0]+encodedBytes[1]+encodedBytes[2]), result.CommittedBytes)
}

func TestDeleteKeysRecordsResidency(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	defer rc.Close()

	ctx := context.Background()
	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	recorder := &recordingBatchMetricRecorder{}
	bm := NewRedisBatchManager(bc, nil, WithoutBuffer(), WithBatchMetricRecorder(recorder))
	functionID := uuid.New()
	batchID := ulid.MustNew(ulid.Timestamp(time.Now().Add(-2*time.Minute)), rand.Reader)
	batchKey := bc.KeyGenerator().Batch(ctx, functionID, batchID)
	metadataKey := bc.KeyGenerator().BatchMetadata(ctx, functionID, batchID)

	_, err = r.RPush(batchKey, "payload")
	require.NoError(t, err)
	r.HSet(metadataKey, "status", enums.BatchStatusPending.String())

	require.NoError(t, bm.DeleteKeys(ctx, functionID, batchID))
	require.Len(t, recorder.deletes, 1)
	require.InDelta(t, (2 * time.Minute).Milliseconds(), recorder.deletes[0].residencyDuration.Milliseconds(), 2_000)

	// Idempotent retries do not record removal or residency a second time.
	require.NoError(t, bm.DeleteKeys(ctx, functionID, batchID))
	require.Len(t, recorder.deletes, 1)
}

func TestBatchCleanupIdempotenceKeyExpires(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	// Set a 5s TTL on per-event idem keys to ensure they expire after inactivity.
	// Disable buffer to test direct append behavior.
	bm := NewRedisBatchManager(bc, nil, WithoutBuffer(), WithRedisBatchIdempotenceSetTTL(5))

	accountId := uuid.New()
	fnId := uuid.New()

	res, err := bm.Append(context.Background(), BatchItem{
		AccountID:  accountId,
		FunctionID: fnId,
		EventID:    ulid.MustNew(ulid.Now(), rand.Reader),
		Event: event.Event{
			ID: "test-event",
		},
		Version: 0,
	}, inngest.Function{
		ID: fnId,
		EventBatch: &inngest.EventBatchConfig{
			MaxSize: 10,
			Timeout: "60s",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchNew, res.Status)

	require.True(t, r.Exists(bc.KeyGenerator().Batch(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.True(t, r.Exists(bc.KeyGenerator().BatchMetadata(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.True(t, r.Exists(bc.KeyGenerator().BatchPointer(context.Background(), fnId)))
	// Per-event idem key: batch list + metadata + pointer + idem key = 4
	require.Equal(t, 4, len(r.Keys()))

	// DeleteKeys removes batch list and metadata but not the per-event idem key (it has its own TTL).
	err = bm.DeleteKeys(context.Background(), fnId, ulid.MustParse(res.BatchID))
	require.NoError(t, err)
	require.False(t, r.Exists(bc.KeyGenerator().Batch(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.False(t, r.Exists(bc.KeyGenerator().BatchMetadata(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.True(t, r.Exists(bc.KeyGenerator().BatchPointer(context.Background(), fnId)))
	// pointer + per-event idem key remain
	require.Equal(t, 2, len(r.Keys()))

	// Per-event idem key TTL is 5s, should expire after that.
	r.FastForward(6 * time.Second)
	// Only the batch pointer remains (no TTL on it)
	require.Equal(t, 1, len(r.Keys()))
}

func TestDeleteBatch(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
	bm := NewRedisBatchManager(bc, nil)

	accountId := uuid.New()
	workspaceId := uuid.New()
	appId := uuid.New()
	fnId := uuid.New()

	t.Run("delete non-existent batch returns deleted=false", func(t *testing.T) {
		nonExistentFnId := uuid.New()
		result, err := bm.DeleteBatch(context.Background(), nonExistentFnId, "")
		require.NoError(t, err)
		require.False(t, result.Deleted)
		require.Equal(t, "", result.BatchID)
		require.Equal(t, 0, result.ItemCount)
	})

	t.Run("delete existing batch with default key", func(t *testing.T) {
		fn := inngest.Function{
			ID: fnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		// Add two items to the batch
		for i := 0; i < 2; i++ {
			eventID := ulid.MustNew(ulid.Now(), rand.Reader)
			bi := BatchItem{
				AccountID:       accountId,
				WorkspaceID:     workspaceId,
				AppID:           appId,
				FunctionID:      fnId,
				FunctionVersion: 1,
				EventID:         eventID,
				Event: event.Event{
					Name: "test/event",
					Data: map[string]any{"index": i},
				},
			}
			_, err := bm.Append(context.Background(), bi, fn)
			require.NoError(t, err)
		}

		// Verify batch exists
		info, err := bm.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.NotEmpty(t, info.BatchID)
		require.Len(t, info.Items, 2)
		batchID := info.BatchID

		// Delete the batch
		result, err := bm.DeleteBatch(context.Background(), fnId, "")
		require.NoError(t, err)
		require.True(t, result.Deleted)
		require.Equal(t, batchID, result.BatchID)
		require.Equal(t, 2, result.ItemCount)

		// Verify batch no longer exists
		infoAfter, err := bm.GetBatchInfo(context.Background(), fnId, "")
		require.NoError(t, err)
		require.Equal(t, "", infoAfter.BatchID)
		require.Empty(t, infoAfter.Items)
	})

	t.Run("delete batch with custom key", func(t *testing.T) {
		customFnId := uuid.New()
		customBatchKey := "tenant-456"

		fn := inngest.Function{
			ID: customFnId,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
				Key:     strPtr("event.data.tenant_id"),
			},
		}

		eventID := ulid.MustNew(ulid.Now(), rand.Reader)
		bi := BatchItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      customFnId,
			FunctionVersion: 1,
			EventID:         eventID,
			Event: event.Event{
				Name: "test/event",
				Data: map[string]any{"tenant_id": customBatchKey},
			},
		}

		_, err := bm.Append(context.Background(), bi, fn)
		require.NoError(t, err)

		// Verify batch exists
		info, err := bm.GetBatchInfo(context.Background(), customFnId, customBatchKey)
		require.NoError(t, err)
		require.NotEmpty(t, info.BatchID)
		batchID := info.BatchID

		// Delete using the custom key
		result, err := bm.DeleteBatch(context.Background(), customFnId, customBatchKey)
		require.NoError(t, err)
		require.True(t, result.Deleted)
		require.Equal(t, batchID, result.BatchID)
		require.Equal(t, 1, result.ItemCount)

		// Verify batch no longer exists
		infoAfter, err := bm.GetBatchInfo(context.Background(), customFnId, customBatchKey)
		require.NoError(t, err)
		require.Equal(t, "", infoAfter.BatchID)
	})
}

func TestRunBatch(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)

	accountId := uuid.New()
	workspaceId := uuid.New()
	appId := uuid.New()

	t.Run("run non-existent batch returns scheduled=false", func(t *testing.T) {
		bm := NewRedisBatchManager(bc, nil)
		nonExistentFnId := uuid.New()

		result, err := bm.RunBatch(context.Background(), RunBatchOpts{
			FunctionID:  nonExistentFnId,
			BatchKey:    "",
			AccountID:   accountId,
			WorkspaceID: workspaceId,
			AppID:       appId,
		})
		require.NoError(t, err)
		require.False(t, result.Scheduled)
		require.Equal(t, "", result.BatchID)
		require.Equal(t, 0, result.ItemCount)
	})
}
