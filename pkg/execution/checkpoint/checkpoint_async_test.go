package checkpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/executor/queueref"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
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

// TestAsyncStepWithMetadataCreatesMetadataSpans asserts that async checkpoint with metadata-bearing
// opcodes creates both step and metadata spans when AllowStepMetadata returns true.
func TestAsyncStepWithMetadataCreatesMetadataSpans(t *testing.T) {
	ctx := context.Background()
	require := require.New(t)

	now := time.Now()
	ops := []state.GeneratorOpcode{
		{
			ID:     "step-1",
			Op:     enums.OpcodeStepRun,
			Data:   json.RawMessage(`{"result": "step 1 output"}`),
			Name:   "Step 1",
			Timing: interval.New(now, now.Add(100*time.Millisecond)),
			Metadata: []metadata.ScopedUpdate{
				{
					Scope: enums.MetadataScopeRun,
					Update: metadata.Update{
						RawUpdate: metadata.RawUpdate{
							Kind:   "userland.test",
							Op:     enums.MetadataOpcodeMerge,
							Values: metadata.Values{"key": json.RawMessage(`"value"`)},
						},
					},
				},
			},
		},
	}

	mocks, testData := setupAsyncCheckpointTest(t, ops...)

	// Replace checkpointer with AllowStepMetadata enabled
	testData.checkpointer = New(Opts{
		State:           mocks.state,
		TracerProvider:  mocks.tracer,
		Queue:           mocks.queue,
		MetricsProvider: mocks.metrics,
		AllowStepMetadata: executor.AllowStepMetadata(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
	})

	// Expect SaveStep
	expectedData := map[string]any{"data": json.RawMessage(`{"result": "step 1 output"}`)}
	expectedOutputBytes, _ := json.Marshal(expectedData)
	mocks.state.On("SaveStep", ctx, testData.metadata.ID, "step-1", expectedOutputBytes).Return(false, nil)

	// Expect CreateSpan for both step and metadata spans
	mocks.tracer.
		On("CreateSpan", mock.Anything, mock.Anything, mock.AnythingOfType("*tracing.CreateSpanOptions")).
		Return(&meta.SpanReference{}, nil)

	// Expect queue reset
	mocks.queue.On("ResetAttemptsByJobID", ctx, "shard-1", "job-123").Return(nil)

	err := testData.checkpointer.CheckpointAsyncSteps(ctx, testData.asyncCheckpoint)
	require.NoError(err)

	// Assert that both step and metadata spans were created
	require.Len(mocks.tracer.createdSpans, 2, "Expected 1 step span + 1 metadata span")
	var hasStep, hasMetadata bool
	for _, s := range mocks.tracer.createdSpans {
		if s.name == meta.SpanNameStep {
			hasStep = true
		}
		if s.name == meta.SpanNameMetadata {
			hasMetadata = true
		}
	}
	require.True(hasStep, "Expected a step span")
	require.True(hasMetadata, "Expected a metadata span")
}

// Note: there's intentionally no async equivalent of
// TestCheckpointSyncSteps_DeferAdd_RunCompleteSkipsSaveStep. Async checkpoints
// can't bundle RunComplete — checkpointAsyncSteps's switch returns
// "cannot checkpoint opcode" for OpcodeRunComplete (see checkpoint.go).

// TestCheckpointAsyncSteps_DeferAdd asserts that the async checkpoint path
// handles OpcodeDeferAdd the same way the sync path does: memoize the step
// with null data and persist a Defer record. No discovery step is enqueued;
// the SDK is driving the run.
func TestCheckpointAsyncSteps_DeferAdd(t *testing.T) {
	ctx := context.Background()
	require := require.New(t)

	op := state.GeneratorOpcode{
		ID: "step-defer",
		Op: enums.OpcodeDeferAdd,
		Opts: map[string]any{
			"fn_slug": "onDefer-score",
			"input":   map[string]any{"user_id": "u_123"},
		},
	}

	mocks, testData := setupAsyncCheckpointTest(t, op)

	mocks.state.On("SaveStep", ctx, testData.metadata.ID, "step-defer", []byte("null")).Return(false, nil)
	mocks.state.On("SaveDefer", ctx, testData.metadata.ID, mock.MatchedBy(func(d state.Defer) bool {
		return d.FnSlug == "onDefer-score" &&
			d.HashedID == "step-defer" &&
			d.ScheduleStatus == enums.DeferStatusAfterRun &&
			string(d.Input) == `{"user_id":"u_123"}`
	})).Return(nil)
	mocks.queue.On("ResetAttemptsByJobID", ctx, "shard-1", "job-123").Return(nil)

	err := testData.checkpointer.CheckpointAsyncSteps(ctx, testData.asyncCheckpoint)
	require.NoError(err)

	mocks.state.AssertExpectations(t)
	mocks.tracer.AssertExpectations(t)
	mocks.queue.AssertExpectations(t)
}

// TestCheckpointAsyncSteps_DeferCancel asserts the async cancel path:
// memoize the cancel step and flip the target defer to Cancelled.
func TestCheckpointAsyncSteps_DeferCancel(t *testing.T) {
	ctx := context.Background()
	require := require.New(t)

	op := state.GeneratorOpcode{
		ID: "step-cancel",
		Op: enums.OpcodeDeferCancel,
		Opts: map[string]any{
			"target_hashed_id": "step-defer",
		},
	}

	mocks, testData := setupAsyncCheckpointTest(t, op)

	mocks.state.On("SaveStep", ctx, testData.metadata.ID, "step-cancel", []byte("null")).Return(false, nil)
	mocks.state.On("SetDeferStatus", ctx, testData.metadata.ID, "step-defer", enums.DeferStatusCancelled).Return(nil)
	mocks.queue.On("ResetAttemptsByJobID", ctx, "shard-1", "job-123").Return(nil)

	err := testData.checkpointer.CheckpointAsyncSteps(ctx, testData.asyncCheckpoint)
	require.NoError(err)

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

func (m *mockRunService) SaveDefer(ctx context.Context, id state.ID, d state.Defer) error {
	args := m.Called(ctx, id, d)
	return args.Error(0)
}

func (m *mockRunService) LoadDefers(ctx context.Context, id state.ID) (map[string]state.Defer, error) {
	args := m.Called(ctx, id)
	var defers map[string]state.Defer
	if v := args.Get(0); v != nil {
		defers = v.(map[string]state.Defer)
	}
	return defers, args.Error(1)
}

func (m *mockRunService) SetDeferStatus(ctx context.Context, id state.ID, hashedID string, status enums.DeferStatus) error {
	args := m.Called(ctx, id, hashedID, status)
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
