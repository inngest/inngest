package redis_state

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestPartitionByID(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	defaultShard := QueueShard{
		Kind:        string(enums.QueueShardKindRedis),
		RedisClient: NewQueueClient(rc, QueueDefaultKey),
		Name:        consts.DefaultQueueShardName,
	}
	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	testcases := []struct {
		name      string
		keyQueues bool
	}{
		{
			name: "sample",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

			q := NewQueue(
				defaultShard,
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
					return tc.keyQueues
				}),
				WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
					return false
				}),
				WithClock(clock),
			)

			for i := range 5 {
				at := clock.Now()

				item := osqueue.QueueItem{
					ID:          fmt.Sprintf("test%d", i),
					FunctionID:  fnID,
					WorkspaceID: wsID,
					Data: osqueue.Item{
						WorkspaceID: wsID,
						Kind:        osqueue.KindEdge,
						Identifier: state.Identifier{
							AccountID:       acctId,
							WorkspaceID:     wsID,
							WorkflowID:      fnID,
							WorkflowVersion: 1,
						},
					},
				}

				_, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			res, err := q.PartitionByID(ctx, defaultShard, fnID.String())
			require.NoError(t, err)

			fmt.Printf("RESULT: %#v\n", res)
		})
	}
}
