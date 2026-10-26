package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestRunJobsRankPagination(t *testing.T) {
	r, client := initRedis(t)
	t.Cleanup(client.Close)
	q, shard := newQueue(t, client)
	kg := NewQueueClient(client, QueueDefaultKey).KeyGenerator()
	scope := osqueue.Scope{AccountID: uuid.New(), EnvID: uuid.New(), FunctionID: uuid.New()}
	run := ulid.Make()
	for i := range 1101 {
		id := fmt.Sprintf("job-%04d", i)
		item := osqueue.QueueItem{ID: id, FunctionID: scope.FunctionID, Data: osqueue.Item{Identifier: state.Identifier{RunID: run, AccountID: scope.AccountID, WorkspaceID: scope.EnvID}}}
		data, err := json.Marshal(item)
		require.NoError(t, err)
		r.HSet(kg.QueueItem(), id, string(data))
		r.ZAdd(kg.RunIndex(run), float64(i), id)
	}
	for _, tc := range []struct {
		offset, limit int64
		count         int
	}{{0, 10, 10}, {10, 10, 10}, {1000, 1000, 101}, {1101, 10, 0}} {
		jobs, err := q.RunJobs(context.Background(), shard.Name(), scope, run, tc.limit, tc.offset)
		require.NoError(t, err)
		require.Len(t, jobs, tc.count)
		for i, job := range jobs {
			require.Equal(t, fmt.Sprintf("job-%04d", int(tc.offset)+i), job.JobID)
		}
	}
	_, err := q.RunJobs(context.Background(), consts.DefaultQueueShardName, scope, run, 10, -1)
	require.Error(t, err)
}
