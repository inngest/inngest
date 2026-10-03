package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type locationBatchManager struct {
	batch.BatchManager
	result    *batch.BatchAppendResult
	scheduled batch.ScheduleBatchOpts
}

func (m *locationBatchManager) RetrieveItems(ctx context.Context, _ uuid.UUID, _ ulid.ULID) ([]batch.BatchItem, error) {
	m.scheduled.BatchCluster = batch.BatchCluster(ctx)
	m.scheduled.BatchGeneration = redis_state.BatchGeneration(ctx)
	return nil, nil
}

func (m *locationBatchManager) Append(context.Context, batch.BatchItem, inngest.Function) (*batch.BatchAppendResult, error) {
	return m.result, nil
}

func (m *locationBatchManager) ScheduleExecution(_ context.Context, opts batch.ScheduleBatchOpts) error {
	m.scheduled = opts
	return nil
}

func TestAppendAndScheduleBatchCopiesAppendLocationToTimeoutJob(t *testing.T) {
	batchID := ulid.Make()
	manager := &locationBatchManager{result: &batch.BatchAppendResult{
		Status:          enums.BatchNew,
		BatchID:         batchID.String(),
		BatchPointerKey: "pointer",
		BatchCluster:    "valkey-batching-a",
		BatchGeneration: "01K0T21HZW9DHDZ5P5TQKBN1E6",
	}}
	exec := &executor{batcher: manager}
	fnID := uuid.New()
	err := exec.AppendAndScheduleBatch(context.Background(), inngest.Function{
		ID:         fnID,
		EventBatch: &inngest.EventBatchConfig{Timeout: time.Minute.String()},
	}, batch.BatchItem{
		AccountID: uuid.New(), WorkspaceID: uuid.New(), AppID: uuid.New(),
		FunctionID: fnID, EventID: ulid.Make(),
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "valkey-batching-a", manager.scheduled.BatchCluster)
	require.Equal(t, "01K0T21HZW9DHDZ5P5TQKBN1E6", manager.scheduled.BatchGeneration)
}

func TestRetrieveAndScheduleBatchAppliesPayloadLocation(t *testing.T) {
	manager := &locationBatchManager{}
	exec := &executor{batcher: manager}
	functionID := uuid.New()
	err := exec.RetrieveAndScheduleBatch(context.Background(), inngest.Function{ID: functionID}, batch.ScheduleBatchPayload{
		BatchID:         ulid.Make(),
		BatchCluster:    "valkey-batching-a",
		BatchGeneration: "01K0T21HZW9DHDZ5P5TQKBN1E6",
		FunctionID:      functionID,
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "valkey-batching-a", manager.scheduled.BatchCluster)
	require.Equal(t, "01K0T21HZW9DHDZ5P5TQKBN1E6", manager.scheduled.BatchGeneration)
}
