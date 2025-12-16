package checkpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util/interval"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCheckpointSyncSteps_ThreeStepRuns asserts that checkpointing three separate steps attempts
// to save state and traces with the right data (via mock providers).
func TestCheckpointSyncSteps_ThreeStepRuns(t *testing.T) {
	ctx := context.Background()
	require := require.New(t)

	// We'll be checkpointing 3 step run ops.
	now := time.Now()
	ops := make([]state.GeneratorOpcode, 3)
	for i := range 3 {
		ops[i] = state.GeneratorOpcode{
			ID:     fmt.Sprintf("step-%d", i+1),
			Op:     enums.OpcodeStepRun,
			Data:   json.RawMessage(fmt.Sprintf(`{"result": "step %d output"}`, i+1)),
			Name:   fmt.Sprintf("Step %d", i+1),
			Timing: interval.New(now.Add(time.Duration(i*100)*time.Millisecond), now.Add(time.Duration((i+1)*100)*time.Millisecond)),
		}
	}

	// Setup test data and mocks
	mocks, testData := setupSyncCheckpointTest(t, ops...)

	//
	// Create mock assertions prior to checkpointing.
	//

	// Expect SaveStep to be called for each step when checkpointing.
	for _, op := range ops {
		switch op.Op {
		case enums.OpcodeStepRun:
			expectedData := map[string]any{
				"data": json.RawMessage(op.Data),
			}
			expectedOutputBytes, _ := json.Marshal(expectedData)
			mocks.state.On("SaveStep", ctx, testData.metadata.ID, op.ID, expectedOutputBytes).Return(false, nil)
		}
	}

	// Expect CreateSpan to be called for each step
	mocks.tracer.
		On(
			"CreateSpan",
			mock.AnythingOfType("*context.valueCtx"),
			meta.SpanNameStep,
			mock.AnythingOfType("*tracing.CreateSpanOptions"),
		).
		Return(&meta.SpanReference{}, nil).
		Times(3)

	// Expect OnStepFinished to be called for each step
	mocks.metrics.
		On(
			"OnStepFinished",
			ctx,
			mock.AnythingOfType("checkpoint.MetricCardinality"),
			enums.StepStatusCompleted,
		).
		Times(3)

	//
	// Execute the sync checkpoint
	//
	err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
	require.NoError(err)

	//
	// Other Assertions
	//

	// Verify traces are created correctly
	require.Len(mocks.tracer.createdSpans, 3, "Expected exactly 3 spans to be created")
	for i, capture := range mocks.tracer.createdSpans {
		require.Equal(meta.SpanNameStep, capture.name, "Span %d should have correct name", i+1)
		require.NotNil(capture.options, "Span %d should have options", i+1)
		require.NotNil(capture.options.StartTime, "Span %d should have start time", i+1)
		require.NotNil(capture.options.EndTime, "Span %d should have end time", i+1)
		require.NotNil(capture.attributes, "Span %d should have attributes", i+1)

		// Assert that the completed attribute is set in tracing.
		require.NotNil(capture.attributes.Get(meta.Attrs.DynamicStatus.Key()))
		require.EqualValues("Completed", capture.attributes.Get(meta.Attrs.DynamicStatus.Key()).(*enums.StepStatus).String())
	}

	// Verify all mocks were satisfied
	mocks.state.AssertExpectations(t)
	mocks.tracer.AssertExpectations(t)
	mocks.queue.AssertExpectations(t)
	mocks.executor.AssertExpectations(t)
}

// TestCheckpointSyncSteps_WithStepAndSleep asserts that checkpointing a step and sleep enqueues a new job
func TestCheckpointSyncSteps_WithStepAndSleep(t *testing.T) {
	ctx := context.Background()
	require := require.New(t)

	// We'll be checkpointing a step and a sleep op.
	now := time.Now()
	ops := []state.GeneratorOpcode{
		{
			ID:     "step-1",
			Op:     enums.OpcodeStepRun,
			Data:   json.RawMessage(`{"result": "step 1 output"}`),
			Name:   "Step 1",
			Timing: interval.New(now, now.Add(100*time.Millisecond)),
		},
		{
			ID:   "sleep-1",
			Op:   enums.OpcodeSleep,
			Data: json.RawMessage(`{"until": "` + now.Add(5*time.Minute).Format(time.RFC3339) + `"}`),
			Name: "Sleep 1",
		},
	}

	// Setup test data and mocks
	mocks, testData := setupSyncCheckpointTest(t, ops...)

	//
	// Create mock assertions prior to checkpointing.
	//

	// Expect SaveStep to be called for the step run
	expectedData := map[string]any{
		"data": json.RawMessage(`{"result": "step 1 output"}`),
	}
	expectedOutputBytes, _ := json.Marshal(expectedData)
	mocks.state.On("SaveStep", ctx, testData.metadata.ID, "step-1", expectedOutputBytes).Return(false, nil)

	// Expect CreateSpan to be called for the step
	mocks.tracer.
		On(
			"CreateSpan",
			mock.AnythingOfType("*context.valueCtx"),
			meta.SpanNameStep,
			mock.AnythingOfType("*tracing.CreateSpanOptions"),
		).
		Return(&meta.SpanReference{}, nil).
		Once()

	// Expect OnStepFinished to be called for the step
	mocks.metrics.
		On(
			"OnStepFinished",
			ctx,
			mock.AnythingOfType("checkpoint.MetricCardinality"),
			enums.StepStatusCompleted,
		).
		Once()

	// Expect HandleGenerator to be called for the sleep opcode, which should enqueue a job
	mocks.executor.
		On(
			"HandleGenerator",
			ctx,
			mock.AnythingOfType("*checkpoint.checkpointRunContext"),
			mock.MatchedBy(func(op state.GeneratorOpcode) bool {
				return op.ID == "sleep-1" && op.Op == enums.OpcodeSleep
			}),
		).
		Return(nil)

	//
	// Execute the sync checkpoint
	//
	err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
	require.NoError(err)

	//
	// Assertions
	//

	// Verify traces are created correctly (only for the step, not the sleep)
	require.Len(mocks.tracer.createdSpans, 1, "Expected exactly 1 span to be created for the step")
	capture := mocks.tracer.createdSpans[0]
	require.Equal(meta.SpanNameStep, capture.name, "Span should have correct name")
	require.NotNil(capture.options, "Span should have options")

	// Verify HandleGenerator was called for the sleep
	mocks.executor.AssertCalled(t, "HandleGenerator", ctx, mock.AnythingOfType("*checkpoint.checkpointRunContext"), mock.MatchedBy(func(op state.GeneratorOpcode) bool {
		return op.ID == "sleep-1" && op.Op == enums.OpcodeSleep
	}))

	// Verify all mocks were satisfied
	mocks.state.AssertExpectations(t)
	mocks.tracer.AssertExpectations(t)
	mocks.queue.AssertExpectations(t)
	mocks.executor.AssertExpectations(t)
}

//
//
// Testing utils.
//
//

// setupSyncCheckpointTest creates new mocks for sync checkpoint testing
func setupSyncCheckpointTest(t *testing.T, ops ...state.GeneratorOpcode) (*testSyncMocks, *testSyncData) {
	ctx := context.Background()

	// Create mock dependencies
	mocks := &testSyncMocks{
		state:    &mockRunService{},
		tracer:   &mockTracerProvider{},
		queue:    &mockQueue{},
		metrics:  &mockMetricsProvider{},
		executor: &mockExecutor{},
		fnReader: &mockFnReader{},
	}

	// Create test IDs
	runID := ulid.MustNew(ulid.Now(), nil)
	fnID := uuid.New()
	accountID := uuid.New()
	envID := uuid.New()
	appID := uuid.New()

	// Create test metadata
	testMetadata := state.Metadata{
		ID: state.ID{
			RunID:      runID,
			FunctionID: fnID,
			Tenant: state.Tenant{
				AccountID: accountID,
				EnvID:     envID,
				AppID:     appID,
			},
		},
	}

	// Create sync checkpoint input
	syncCheckpoint := SyncCheckpoint{
		RunID:     runID,
		FnID:      fnID,
		AppID:     appID,
		Steps:     ops,
		AccountID: accountID,
		EnvID:     envID,
		Metadata:  &testMetadata,
	}

	// Setup mock expectations
	mocks.fnReader.On("GetFunctionByInternalUUID", ctx, fnID).Return(&mockConfigFunction{}, nil)

	// LoadMetadata should NOT be called since syncCheckpoint.Metadata is already set
	mocks.state.AssertNotCalled(t, "LoadMetadata")

	// Create checkpointer
	checkpointer := New(Opts{
		State:           mocks.state,
		TracerProvider:  mocks.tracer,
		Queue:           mocks.queue,
		MetricsProvider: mocks.metrics,
		Executor:        mocks.executor,
		FnReader:        mocks.fnReader,
	})

	return mocks, &testSyncData{
		metadata:       testMetadata,
		stepOpcodes:    ops,
		syncCheckpoint: syncCheckpoint,
		checkpointer:   checkpointer,
	}
}

// Additional mock implementations for sync tests

type testSyncMocks struct {
	state    *mockRunService
	tracer   *mockTracerProvider
	queue    *mockQueue
	metrics  *mockMetricsProvider
	executor *mockExecutor
	fnReader *mockFnReader
}

type testSyncData struct {
	metadata       state.Metadata
	stepOpcodes    []state.GeneratorOpcode
	syncCheckpoint SyncCheckpoint
	checkpointer   Checkpointer
}

// mockExecutor mocks the executor interface
type mockExecutor struct {
	execution.Executor
	mock.Mock
}

func (m *mockExecutor) HandleGenerator(ctx context.Context, runCtx execution.RunContext, op state.GeneratorOpcode) error {
	args := m.Called(ctx, runCtx, op)
	return args.Error(0)
}

func (m *mockExecutor) Finalize(ctx context.Context, opts execution.FinalizeOpts) error {
	args := m.Called(ctx, opts)
	return args.Error(0)
}

// mockFnReader mocks the function reader interface
type mockFnReader struct {
	cqrs.FunctionReader
	mock.Mock
}

func (m *mockFnReader) GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*cqrs.Function, error) {
	args := m.Called(ctx, fnID)
	_ = args
	return &cqrs.Function{
		Config: json.RawMessage(`{}`),
	}, nil
}

// mockConfigFunction mocks the config function interface
type mockConfigFunction struct{}

func (m *mockConfigFunction) InngestFunction() (interface{}, error) {
	return &mockInngestFunction{}, nil
}

// mockInngestFunction mocks the inngest function interface
type mockInngestFunction struct{}

func (m *mockInngestFunction) MaxAttempts() int {
	return 3
}
