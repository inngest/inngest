package checkpoint

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const instanceIDAttr = "_inngest.connect.instance_id"

// instanceIDsOnStepSpans returns the connect instance ID of every exported
// executor.step span.  a span without the attribute maps to "".
func instanceIDsOnStepSpans(t *testing.T, exp *tracetest.InMemoryExporter) []string {
	t.Helper()

	ids := []string{}
	for _, s := range exp.GetSpans() {
		if s.Name != "executor.step" {
			continue
		}
		id := ""
		for _, kv := range s.Attributes {
			if string(kv.Key) == instanceIDAttr {
				id = kv.Value.AsString()
			}
		}
		ids = append(ids, id)
	}
	return ids
}

func TestCheckpointStepSpansCarryConnectInstanceID(t *testing.T) {
	stepRun := state.GeneratorOpcode{
		ID:   "step-1",
		Op:   enums.OpcodeStepRun,
		Data: json.RawMessage(`{"result": "ok"}`),
		Name: "Step 1",
	}

	for _, instanceID := range []string{"worker-1", ""} {
		t.Run("sync step run with instance ID "+instanceID, func(t *testing.T) {
			ctx := context.Background()
			mocks, testData := setupSyncCheckpointTest(t, stepRun)

			mocks.state.On("SaveStep", ctx, testData.metadata.ID, stepRun.ID, mock.Anything).Return(false, nil)
			mocks.metrics.On("OnStepFinished", ctx, mock.Anything, enums.StepStatusCompleted).Maybe()
			mocks.executor.On("RunStepRunFinishedLifecycle", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			exp := tracetest.NewInMemoryExporter()
			c := New(Opts{
				State:           mocks.state,
				TracerProvider:  tracing.NewOtelTracerProvider(exp, time.Millisecond),
				Queue:           mocks.queue,
				MetricsProvider: mocks.metrics,
				Executor:        mocks.executor,
				FnReader:        mocks.fnReader,
			})

			input := testData.syncCheckpoint
			input.InstanceID = instanceID
			require.NoError(t, c.CheckpointSyncSteps(ctx, input))

			require.Equal(t, []string{instanceID}, instanceIDsOnStepSpans(t, exp))
		})

		t.Run("async step run with instance ID "+instanceID, func(t *testing.T) {
			ctx := context.Background()
			mocks, testData := setupAsyncCheckpointTestWithValidation(t, false, stepRun)

			mocks.state.On("SaveStep", ctx, testData.metadata.ID, stepRun.ID, mock.Anything).Return(false, nil)
			mocks.queue.On("ResetAttemptsByJobID", ctx, "shard-1", testData.scope, "job-123").Return(nil)
			mocks.executor.On("RunStepRunFinishedLifecycle", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			exp := tracetest.NewInMemoryExporter()
			c := New(Opts{
				State:           mocks.state,
				TracerProvider:  tracing.NewOtelTracerProvider(exp, time.Millisecond),
				Queue:           mocks.queue,
				MetricsProvider: mocks.metrics,
				Executor:        mocks.executor,
				AllowAsyncDispatchValidation: func(ctx context.Context, acctID uuid.UUID) bool {
					return false
				},
			})

			input := testData.asyncCheckpoint
			input.InstanceID = instanceID
			require.NoError(t, c.CheckpointAsyncSteps(ctx, input))

			require.Equal(t, []string{instanceID}, instanceIDsOnStepSpans(t, exp))
		})
	}
}
