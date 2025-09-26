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
	bm := NewRedisBatchManager(bc, nil, WithRedisBatchSizeLimit(10))

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
	bm := NewRedisBatchManager(bc, nil)

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

	res, err := bm.Append(context.Background(), bi, function)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchNew, res.Status)

	res, err = bm.Append(context.Background(), bi, function)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchItemExists, res.Status)
}

// When the same event is appended to different batches, we would end up processing the duplicate event a second time in the second batch.
// Currently Idempotency for eventIDs are only tracked within a batch. When a batch is full and scheduled, we lose track of eventIDs already processed.
func TestBatchAppendIdempotenceDifferentBatches(t *testing.T) {
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

	// append the last batchitem again. Since last batch was full, this event goes to a new batch and ends up getting appended to a batch.
	res, err := bm.Append(context.Background(), bi, function)
	require.NoError(t, err)
	require.NotEmpty(t, res.BatchID)
	require.NotEqual(t, res.BatchID, lastBatchID)
	require.NotEmpty(t, res.BatchPointerKey)
	require.Equal(t, enums.BatchNew, res.Status)
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
	bm := NewRedisBatchManager(bc, nil)

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
	require.True(t, r.Exists(bc.KeyGenerator().BatchIdempotenceKey(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.Equal(t, 4, len(r.Keys()))

	err = bm.DeleteKeys(context.Background(), fnId, ulid.MustParse(res.BatchID))
	require.NoError(t, err)

	require.False(t, r.Exists(bc.KeyGenerator().Batch(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.False(t, r.Exists(bc.KeyGenerator().BatchMetadata(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.False(t, r.Exists(bc.KeyGenerator().BatchIdempotenceKey(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.True(t, r.Exists(bc.KeyGenerator().BatchPointer(context.Background(), fnId)))
	require.Equal(t, 1, len(r.Keys()))
}
