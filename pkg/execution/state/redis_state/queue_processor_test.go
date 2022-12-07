package redis_state

import (
	"context"
	"crypto/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestQueueRunBasic(t *testing.T) {
	r := miniredis.RunT(t)
	q := queue{
		r: redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100}),
		pf: func(ctx context.Context, workflowID uuid.UUID) uint {
			return 4
		},
	}
	ctx := context.Background()

	idA, idB := uuid.New(), uuid.New()
	items := []QueueItem{
		{
			WorkflowID:  idA,
			MaxAttempts: 3,
			Data: osqueue.Item{
				Kind: osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID: idA,
					RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
				},
			},
		},
		{
			WorkflowID:  idB,
			MaxAttempts: 1,
			Data: osqueue.Item{
				Kind: osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID: idB,
					RunID:      ulid.MustNew(ulid.Now(), rand.Reader),
				},
			},
		},
	}

	var handled int32
	go func() {
		q.Run(ctx, func(ctx context.Context, item osqueue.Item) error {
			logger.From(ctx).Debug().Interface("item", item).Msg("received item")
			atomic.AddInt32(&handled, 1)
			return nil
		})
	}()

	for _, item := range items {
		_, err := q.Enqueue(ctx, item, time.Now())
		require.NoError(t, err)
	}

	<-time.After(2 * time.Second)
	require.EqualValues(t, len(items), atomic.LoadInt32(&handled))

	// TODO: Assert queue items have been processed
	// TODO: Assert queue items have been dequeued, and peek is nil for workflows.
	// TODO: Assert metrics are correct.
}
