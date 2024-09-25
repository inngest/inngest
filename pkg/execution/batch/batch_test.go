package batch

import (
	"context"
	"crypto/rand"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"testing"
)

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
	require.Equal(t, 3, len(r.Keys()))

	err = bm.DeleteKeys(context.Background(), fnId, ulid.MustParse(res.BatchID))
	require.NoError(t, err)

	require.False(t, r.Exists(bc.KeyGenerator().Batch(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.False(t, r.Exists(bc.KeyGenerator().BatchMetadata(context.Background(), fnId, ulid.MustParse(res.BatchID))))
	require.True(t, r.Exists(bc.KeyGenerator().BatchPointer(context.Background(), fnId)))
	require.Equal(t, 1, len(r.Keys()))
}
