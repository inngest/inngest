package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

type scheduledBatchData struct {
	cqrs.Manager
	fn inngest.Function
}

func (d scheduledBatchData) Functions(context.Context) ([]inngest.Function, error) {
	return []inngest.Function{d.fn}, nil
}

type scheduledBatchExecutor struct {
	execution.Executor
	scheduled []batch.ScheduleBatchPayload
}

func (e *scheduledBatchExecutor) RetrieveAndScheduleBatch(_ context.Context, _ inngest.Function, p batch.ScheduleBatchPayload, _ *execution.BatchExecOpts) error {
	e.scheduled = append(e.scheduled, p)
	return nil
}

func TestHandleScheduledBatch(t *testing.T) {
	tests := []struct {
		name string
		// started marks the batch started before the handler runs, as a
		// previous attempt that failed after StartExecution would.
		started       bool
		absent        bool
		wantScheduled bool
	}{
		{name: "ready batch is scheduled", wantScheduled: true},
		{name: "started batch is scheduled again on retry", started: true, wantScheduled: true},
		{name: "absent batch is not scheduled", absent: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)

			bc := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)
			bm := batch.NewRedisBatchManager(bc, nil, batch.WithoutBuffer())
			fn := inngest.Function{
				ID:         uuid.New(),
				EventBatch: &inngest.EventBatchConfig{MaxSize: 10, Timeout: "60s"},
			}
			accountID := uuid.New()

			batchID := ulid.Make()
			batchPointer := bc.KeyGenerator().BatchPointer(ctx, fn.ID)
			if !tt.absent {
				res, err := bm.Append(ctx, batch.BatchItem{AccountID: accountID, FunctionID: fn.ID, EventID: ulid.Make()}, fn)
				require.NoError(t, err)
				batchID = ulid.MustParse(res.BatchID)
				batchPointer = res.BatchPointerKey
			}
			if tt.started {
				status, err := bm.StartExecution(ctx, accountID, fn.ID, batchID, batchPointer)
				require.NoError(t, err)
				require.Equal(t, enums.BatchStatusReady.String(), status)
			}

			exec := &scheduledBatchExecutor{}
			s := &svc{
				data:    scheduledBatchData{fn: fn},
				exec:    exec,
				batcher: bm,
			}

			payload, err := json.Marshal(batch.ScheduleBatchOpts{
				ScheduleBatchPayload: batch.ScheduleBatchPayload{
					BatchID:      batchID,
					BatchPointer: batchPointer,
					AccountID:    accountID,
					FunctionID:   fn.ID,
				},
			})
			require.NoError(t, err)

			err = s.handleScheduledBatch(ctx, queue.Item{
				Identifier: state.Identifier{AccountID: accountID, WorkflowID: fn.ID},
				Payload:    json.RawMessage(payload),
			})
			require.NoError(t, err)

			if !tt.wantScheduled {
				require.Empty(t, exec.scheduled)
				return
			}
			require.Len(t, exec.scheduled, 1)
			require.Equal(t, batchID, exec.scheduled[0].BatchID)
			require.Equal(t, accountID, exec.scheduled[0].AccountID)
		})
	}
}
