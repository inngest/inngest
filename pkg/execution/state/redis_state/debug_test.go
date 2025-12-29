package redis_state

import (
	"context"
	"fmt"
	"testing"
	"time"

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
		num       int
		interval  time.Duration
		expected  PartitionInspectionResult
		keyQueues bool
	}{
		{
			name: "simple",
			num:  5,
			expected: PartitionInspectionResult{
				Ready:  5,
				Future: 5,
			},
		},
		{
			name:     "with interval",
			num:      5,
			interval: time.Second,
			expected: PartitionInspectionResult{
				Ready:  5,
				Future: 5,
			},
		},
		{
			name: "with key queues",
			num:  10,
			expected: PartitionInspectionResult{
				Backlogs: 1,
			},
			keyQueues: true,
		},
		{
			name:     "with key queues interval",
			num:      10,
			interval: time.Minute,
			expected: PartitionInspectionResult{
				Backlogs: 1,
			},
			keyQueues: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

			q := NewQueue(
				defaultShard,
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
					return tc.keyQueues
				}),
				WithClock(clock),
			)

			for i := range tc.num {
				at := clock.Now()
				at = at.Add(time.Duration(i) * tc.interval)

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

			// fmt.Printf("RESULT: %#v\n", res)
			require.Equal(t, tc.expected.Paused, res.Paused)
			require.Equal(t, tc.expected.AccountActive, res.AccountActive)
			require.Equal(t, tc.expected.AccountInProgress, res.AccountInProgress)
			require.Equal(t, tc.expected.Ready, res.Ready)
			require.Equal(t, tc.expected.InProgress, res.InProgress)
			require.Equal(t, tc.expected.Active, res.Active)
			require.Equal(t, tc.expected.Future, res.Future)
			require.Equal(t, tc.expected.Backlogs, res.Backlogs)
		})
	}
}
