package checkpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/executor/queueref"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util/interval"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCheckpointAsyncSteps_ThreeStepRuns asserts that checkpointing three separate steps attempts
// to save state and traces with the right data (via mock providers).
func TestCheckpointAsyncSteps_ThreeStepRuns(t *testing.T) {
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
	mocks, testData := setupAsyncCheckpointTest(t, ops...)

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

	// Expect queue reset to be called with the job ID and shard info.
	//
	// Without this, a failed queue item that becomes successful will have an attempt count > 0 on the next
	// retry.
	mocks.queue.On("ResetAttemptsByJobID", ctx, "shard-1", "job-123").Return(nil)

	//
	// Execute the async checkpoint
	//
	err := testData.checkpointer.CheckpointAsyncSteps(ctx, testData.asyncCheckpoint)
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

	// ASSERTIONS: Verify queue reset is called
	mocks.queue.AssertCalled(t, "ResetAttemptsByJobID", ctx, "shard-1", "job-123")

	// Verify all mocks were satisfied
	mocks.state.AssertExpectations(t)
	mocks.tracer.AssertExpectations(t)
	mocks.queue.AssertExpectations(t)
}

func TestCheckpointAsyncSteps_WithSleepFails(t *testing.T) {
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
	mocks, testData := setupAsyncCheckpointTest(
		t, state.GeneratorOpcode{
			ID: "sleep",
			Op: enums.OpcodeSleep,
			// This will fail on the Op.
		},
		state.GeneratorOpcode{
			ID:   "step-run",
			Op:   enums.OpcodeStepRun,
			Data: json.RawMessage(`{"result": "step rund output"}`),
		})

	// Execute the async checkpoint
	err := testData.checkpointer.CheckpointAsyncSteps(ctx, testData.asyncCheckpoint)
	require.Error(err, "cannot checkpoint async steps")

	// ASSERTIONS: Verify traces are created correctly
	require.Len(mocks.tracer.createdSpans, 0, "Expected exactly 3 spans to be created")

	// ASSERTIONS: Verify SaveStep calls are made correctly
	mocks.state.AssertNotCalled(t, "SaveStep")

	// ASSERTIONS: Verify queue reset is called
	mocks.queue.AssertNotCalled(t, "ResetAttemptsByJobID")

	// Verify all mocks were satisfied
	mocks.state.AssertExpectations(t)
	mocks.tracer.AssertExpectations(t)
	mocks.queue.AssertExpectations(t)
}

//
//
// Testing utils.
//
//

// setupAsyncCheckpointTest creates new mocks, asserting that
func setupAsyncCheckpointTest(t *testing.T, ops ...state.GeneratorOpcode) (*testMocks, *testData) {
	ctx := context.Background()

	// Create mock dependencies
	mocks := &testMocks{
		state:   &mockRunService{},
		tracer:  &mockTracerProvider{},
		queue:   &mockQueue{},
		metrics: &mockMetricsProvider{},
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

	// Create async checkpoint input
	queueRef := queueref.QueueRef{"job-123", "shard-1"} // [jobID, shardID]
	asyncCheckpoint := AsyncCheckpoint{
		RunID:        runID,
		FnID:         fnID,
		Steps:        ops,
		QueueItemRef: queueRef.String(),
		AccountID:    accountID,
		EnvID:        envID,
	}

	// Setup mock expectations
	mocks.state.On("LoadMetadata", ctx, asyncCheckpoint.ID()).Return(testMetadata, nil)

	// Create checkpointer
	checkpointer := New(Opts{
		State:           mocks.state,
		TracerProvider:  mocks.tracer,
		Queue:           mocks.queue,
		MetricsProvider: mocks.metrics,
	})

	return mocks, &testData{
		metadata:        testMetadata,
		stepOpcodes:     ops,
		asyncCheckpoint: asyncCheckpoint,
		checkpointer:    checkpointer,
	}
}

// Mock implementations

// mockRunService mocks the state.RunService interface
type mockRunService struct {
	state.RunService
	mock.Mock
}

func (m *mockRunService) LoadMetadata(ctx context.Context, id state.ID) (state.Metadata, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(state.Metadata), args.Error(1)
}

func (m *mockRunService) SaveStep(ctx context.Context, id state.ID, stepID string, data []byte) (bool, error) {
	args := m.Called(ctx, id, stepID, data)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}
	return false, args.Error(1)
}

func (m *mockRunService) UpdateMetadata(ctx context.Context, id state.ID, config state.MutableConfig) error {
	args := m.Called(ctx, id, config)
	return args.Error(0)
}

// mockTracerProvider mocks the tracing.TracerProvider interface
type mockTracerProvider struct {
	tracing.TracerProvider
	mock.Mock

	createdSpans []*spanCapture
	updatedSpans []*updateCapture
}

type spanCapture struct {
	name       string
	options    *tracing.CreateSpanOptions
	attributes *meta.SerializableAttrs
}

type updateCapture struct {
	options    *tracing.UpdateSpanOptions
	attributes *meta.SerializableAttrs
}

func (m *mockTracerProvider) CreateSpan(ctx context.Context, name string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error) {
	args := m.Called(ctx, name, opts)

	// Capture the span details for verification
	capture := &spanCapture{
		name:       name,
		options:    opts,
		attributes: opts.Attributes,
	}
	m.createdSpans = append(m.createdSpans, capture)

	// Return a mock span reference
	spanRef := &meta.SpanReference{
		TraceParent: "test-trace-parent",
		TraceState:  "test-trace-state",
	}

	return spanRef, args.Error(1)
}

func (m *mockTracerProvider) UpdateSpan(ctx context.Context, opts *tracing.UpdateSpanOptions) error {
	args := m.Called(ctx, opts)

	// Capture the update details for verification
	capture := &updateCapture{
		options:    opts,
		attributes: opts.Attributes,
	}
	m.updatedSpans = append(m.updatedSpans, capture)

	return args.Error(0)
}

// mockQueue mocks the queue.Queue interface
type mockQueue struct {
	queue.Queue
	mock.Mock
}

func (m *mockQueue) ResetAttemptsByJobID(ctx context.Context, shardID, jobID string) error {
	args := m.Called(ctx, shardID, jobID)
	return args.Error(0)
}

func (m *mockQueue) Enqueue(ctx context.Context, item queue.Item, at time.Time, opts queue.EnqueueOpts) error {
	args := m.Called(ctx, item, at, opts)
	return args.Error(0)
}

// mockMetricsProvider mocks the MetricsProvider interface
type mockMetricsProvider struct {
	mock.Mock
}

func (m *mockMetricsProvider) OnFnScheduled(ctx context.Context, mc MetricCardinality) {
	m.Called(ctx, mc)
}

func (m *mockMetricsProvider) OnStepFinished(ctx context.Context, mc MetricCardinality, status enums.StepStatus) {
	m.Called(ctx, mc, status)
}

func (m *mockMetricsProvider) OnFnFinished(ctx context.Context, mc MetricCardinality, status enums.RunStatus) {
	m.Called(ctx, mc, status)
}

// Test helper types and functions

type testMocks struct {
	state   *mockRunService
	tracer  *mockTracerProvider
	queue   *mockQueue
	metrics *mockMetricsProvider
}

type testData struct {
	metadata        state.Metadata
	stepOpcodes     []state.GeneratorOpcode
	asyncCheckpoint AsyncCheckpoint
	checkpointer    Checkpointer
}
