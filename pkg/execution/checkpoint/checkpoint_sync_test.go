package checkpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/apiresult"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	sv1 "github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/util/interval"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckpointSyncSteps(t *testing.T) {
	t.Run("three step runs", func(t *testing.T) {
		// Checkpointing three separate steps attempts to save state and
		// traces with the right data (via mock providers).
		ctx := context.Background()
		require := require.New(t)

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

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		// Expect UpdateMetadata to be called with ForceStepPlan=true since we have >1 steps (parallel mode)
		mocks.state.On("UpdateMetadata", ctx, testData.metadata.ID, mock.MatchedBy(func(config state.MutableConfig) bool {
			return config.ForceStepPlan == true
		})).Return(nil)

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

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

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

		mocks.state.AssertExpectations(t)
		mocks.tracer.AssertExpectations(t)
		mocks.queue.AssertExpectations(t)
		mocks.executor.AssertExpectations(t)
	})

	t.Run("with step and sleep", func(t *testing.T) {
		// Checkpointing a step and a sleep enqueues a new job.
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
			},
			{
				ID:   "sleep-1",
				Op:   enums.OpcodeSleep,
				Data: json.RawMessage(`{"until": "` + now.Add(5*time.Minute).Format(time.RFC3339) + `"}`),
				Name: "Sleep 1",
			},
		}

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		// Expect UpdateMetadata to be called with ForceStepPlan=true since we have >1 steps (parallel mode)
		mocks.state.On("UpdateMetadata", ctx, testData.metadata.ID, mock.MatchedBy(func(config state.MutableConfig) bool {
			return config.ForceStepPlan == true
		})).Return(nil)

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

		// Expect UpdateSpan to be called when the async opcode (sleep) is encountered,
		// which triggers the mode change tracking for Durable Endpoint runs
		mocks.tracer.
			On(
				"UpdateSpan",
				ctx,
				mock.AnythingOfType("*tracing.UpdateSpanOptions"),
			).
			Return(nil).
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

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

		// Verify traces are created correctly (only for the step, not the sleep)
		require.Len(mocks.tracer.createdSpans, 1, "Expected exactly 1 span to be created for the step")
		capture := mocks.tracer.createdSpans[0]
		require.Equal(meta.SpanNameStep, capture.name, "Span should have correct name")
		require.NotNil(capture.options, "Span should have options")

		// Verify HandleGenerator was called for the sleep
		mocks.executor.AssertCalled(t, "HandleGenerator", ctx, mock.AnythingOfType("*checkpoint.checkpointRunContext"), mock.MatchedBy(func(op state.GeneratorOpcode) bool {
			return op.ID == "sleep-1" && op.Op == enums.OpcodeSleep
		}))

		// Verify UpdateSpan was called with DurableEndpointModeChangedAt attribute for mode change tracking
		require.Len(mocks.tracer.updatedSpans, 1, "Expected exactly 1 span update for mode change tracking")
		updateCapture := mocks.tracer.updatedSpans[0]
		require.NotNil(updateCapture.attributes, "Updated span should have attributes")
		modeChangedAt := updateCapture.attributes.Get(meta.Attrs.DurableEndpointModeChangedAt.Key())
		require.NotNil(modeChangedAt, "DurableEndpointModeChangedAt attribute should be set")

		mocks.state.AssertExpectations(t)
		mocks.tracer.AssertExpectations(t)
		mocks.queue.AssertExpectations(t)
		mocks.executor.AssertExpectations(t)
	})

	t.Run("defer add", func(t *testing.T) {
		// A sync checkpoint containing an OpcodeDeferAdd persists a Defer
		// record with DeferStatusAfterRun, matching the executor's
		// non-checkpoint handleGeneratorDeferAdd path.
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

		mocks, testData := setupSyncCheckpointTest(t, op)

		mocks.state.On("SaveDefer", ctx, testData.metadata.ID, mock.MatchedBy(func(d state.Defer) bool {
			return d.FnSlug == "onDefer-score" &&
				d.HashedID == "step-defer" &&
				d.ScheduleStatus == enums.DeferStatusAfterRun &&
				string(d.Input) == `{"user_id":"u_123"}`
		})).Return(nil)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

		// No discovery step should be enqueued (the SDK is driving the run).
		mocks.queue.AssertNotCalled(t, "Enqueue")
		// DeferAdd is a sync opcode — no async mode transition should fire.
		mocks.tracer.AssertNotCalled(t, "UpdateSpan")

		mocks.state.AssertExpectations(t)
		mocks.tracer.AssertExpectations(t)
		mocks.queue.AssertExpectations(t)
		mocks.executor.AssertExpectations(t)
	})

	t.Run("defer add bundled with RunComplete", func(t *testing.T) {
		// [DeferAdd, RunComplete] in a single batch persists the Defer (so
		// Finalize's LoadDefers can read it before state deletion) and
		// invokes Finalize. ForceStepPlan must NOT trigger because the
		// batch has only one non-lazy op.
		ctx := context.Background()
		require := require.New(t)

		ops := []state.GeneratorOpcode{
			{
				ID: "step-defer",
				Op: enums.OpcodeDeferAdd,
				Opts: map[string]any{
					"fn_slug": "onDefer-score",
					"input":   map[string]any{},
				},
			},
			{
				ID:   "run-complete",
				Op:   enums.OpcodeRunComplete,
				Data: json.RawMessage(`{"data": {"status_code": 200}}`),
			},
		}

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		// SaveDefer must run before Finalize (which deletes state) so the
		// Defer record is readable by Finalize's LoadDefers call.
		mocks.state.On("SaveDefer", ctx, testData.metadata.ID, mock.MatchedBy(func(d state.Defer) bool {
			return d.HashedID == "step-defer" && d.FnSlug == "onDefer-score"
		})).Return(nil)

		// Finalize must be called for this run with the RunComplete response type.
		mocks.executor.On("Finalize", ctx, mock.MatchedBy(func(opts execution.FinalizeOpts) bool {
			return opts.Metadata.ID == testData.metadata.ID &&
				opts.Response.Type == execution.FinalizeResponseAPI
		})).Return(nil)

		// Registered so the async goroutine in checkpoint.go (`go MetricsProvider.OnFnFinished`)
		// doesn't panic on an unmocked call. Not asserted: the goroutine races against
		// the test's return.
		mocks.metrics.On("OnFnFinished", ctx, mock.AnythingOfType("checkpoint.MetricCardinality"), enums.RunStatusCompleted)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

		mocks.state.AssertNotCalled(t, "UpdateMetadata", mock.Anything, mock.Anything, mock.Anything)

		mocks.state.AssertExpectations(t)
		mocks.tracer.AssertExpectations(t)
		mocks.queue.AssertExpectations(t)
		mocks.executor.AssertExpectations(t)
	})

	t.Run("defer add reordered before RunComplete still saves", func(t *testing.T) {
		// Ordering invariant: DeferAdd/DeferAbort must drain before
		// RunComplete even when the SDK delivers them in the opposite
		// order. Without the priority reorder, RunComplete's Finalize
		// would delete state before SaveDefer ran, silently dropping the
		// deferred run.
		ctx := context.Background()
		r := require.New(t)

		ops := []state.GeneratorOpcode{
			{
				ID:   "run-complete",
				Op:   enums.OpcodeRunComplete,
				Data: json.RawMessage(`{"data": {"status_code": 200}}`),
			},
			{
				ID: "step-defer",
				Op: enums.OpcodeDeferAdd,
				Opts: map[string]any{
					"fn_slug": "onDefer-score",
					"input":   map[string]any{},
				},
			},
		}

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		var saveDeferAt, finalizeAt int
		var calls int

		mocks.state.On("SaveDefer", ctx, testData.metadata.ID, mock.MatchedBy(func(d state.Defer) bool {
			return d.HashedID == "step-defer" && d.FnSlug == "onDefer-score"
		})).Run(func(args mock.Arguments) {
			calls++
			saveDeferAt = calls
		}).Return(nil)

		mocks.executor.On("Finalize", ctx, mock.MatchedBy(func(opts execution.FinalizeOpts) bool {
			return opts.Metadata.ID == testData.metadata.ID &&
				opts.Response.Type == execution.FinalizeResponseAPI
		})).Run(func(args mock.Arguments) {
			calls++
			finalizeAt = calls
		}).Return(nil)

		mocks.metrics.On("OnFnFinished", ctx, mock.AnythingOfType("checkpoint.MetricCardinality"), enums.RunStatusCompleted)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		r.NoError(err)

		r.NotZero(saveDeferAt, "SaveDefer must be called")
		r.NotZero(finalizeAt, "Finalize must be called")
		r.Less(saveDeferAt, finalizeAt, "SaveDefer must run before Finalize so LoadDefers can read the record")

		mocks.state.AssertExpectations(t)
		mocks.tracer.AssertExpectations(t)
		mocks.queue.AssertExpectations(t)
		mocks.executor.AssertExpectations(t)
	})

	t.Run("defer abort", func(t *testing.T) {
		// A sync-checkpointed OpcodeDeferAbort flips the target defer to
		// Aborted via SetDeferStatus.
		ctx := context.Background()
		require := require.New(t)

		op := state.GeneratorOpcode{
			ID: "step-abort",
			Op: enums.OpcodeDeferAbort,
			Opts: map[string]any{
				"target_hashed_id": "step-defer",
			},
		}

		mocks, testData := setupSyncCheckpointTest(t, op)

		mocks.state.On("SetDeferStatus", ctx, testData.metadata.ID, "step-defer", enums.DeferStatusAborted).Return(nil)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

		mocks.queue.AssertNotCalled(t, "Enqueue")
		mocks.tracer.AssertNotCalled(t, "UpdateSpan")

		mocks.state.AssertExpectations(t)
		mocks.tracer.AssertExpectations(t)
		mocks.queue.AssertExpectations(t)
		mocks.executor.AssertExpectations(t)
	})

	t.Run("defer abort missing target soft-fails", func(t *testing.T) {
		// Aborting a hashedID that doesn't exist (e.g. SDK-bug
		// `[DeferAbort, DeferAdd]` ordering, or aborting an ID never
		// added in this run) is logged and skipped without failing the
		// parent run.
		ctx := context.Background()
		require := require.New(t)

		op := state.GeneratorOpcode{
			ID: "step-abort",
			Op: enums.OpcodeDeferAbort,
			Opts: map[string]any{
				"target_hashed_id": "never-added",
			},
		}

		mocks, testData := setupSyncCheckpointTest(t, op)

		mocks.state.
			On("SetDeferStatus", ctx, testData.metadata.ID, "never-added", enums.DeferStatusAborted).
			Return(fmt.Errorf("defer not found for hashedID %q", "never-added"))

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err, "missing-target DeferAbort must NOT fail the parent run; soft-fail with log")

		mocks.state.AssertExpectations(t)
		mocks.tracer.AssertExpectations(t)
		mocks.queue.AssertExpectations(t)
		mocks.executor.AssertExpectations(t)
	})

	t.Run("defer abort missing target_hashed_id soft-fails", func(t *testing.T) {
		// A DeferAbort without target_hashed_id is logged and skipped
		// without failing the parent run. SetDeferStatus must not be
		// called because validation fails first.
		ctx := context.Background()
		require := require.New(t)

		op := state.GeneratorOpcode{
			ID:   "step-abort",
			Op:   enums.OpcodeDeferAbort,
			Opts: map[string]any{},
		}

		mocks, testData := setupSyncCheckpointTest(t, op)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err, "invalid DeferAbort must NOT fail the parent run; soft-fail with log")

		mocks.state.AssertNotCalled(t, "SetDeferStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		mocks.state.AssertExpectations(t)
		mocks.tracer.AssertExpectations(t)
		mocks.queue.AssertExpectations(t)
		mocks.executor.AssertExpectations(t)
	})

	t.Run("step output too large", func(t *testing.T) {
		// A single step whose output exceeds MaxStepOutputSize causes
		// CheckpointSyncSteps to return ErrStepOutputTooLarge without
		// attempting to save state.
		ctx := context.Background()
		require := require.New(t)

		// A JSON string of MaxStepOutputSize 'x' characters: when wrapped in
		// {"data": ...} the total output exceeds the 4 MiB per-step limit.
		largeData := json.RawMessage(`"` + strings.Repeat("x", consts.MaxStepOutputSize) + `"`)
		op := state.GeneratorOpcode{
			ID:   "big-step",
			Op:   enums.OpcodeStepRun,
			Data: largeData,
			Name: "Big Step",
		}

		_, testData := setupSyncCheckpointTest(t, op)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.ErrorIs(err, sv1.ErrStepOutputTooLarge)
	})

	t.Run("cumulative state overflow", func(t *testing.T) {
		// When accumulated state already equals the 32 MiB limit, any
		// additional step output causes CheckpointSyncSteps to return
		// ErrStateOverflowed before saving state.
		ctx := context.Background()
		require := require.New(t)

		op := state.GeneratorOpcode{
			ID:   "step-1",
			Op:   enums.OpcodeStepRun,
			Data: json.RawMessage(`"small"`),
			Name: "Step 1",
		}

		_, testData := setupSyncCheckpointTest(t, op)
		testData.syncCheckpoint.Metadata.Metrics.StateSize = consts.DefaultMaxStateSizeLimit

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.ErrorIs(err, sv1.ErrStateOverflowed)
	})
}

func TestSyncStepMetadata(t *testing.T) {
	t.Run("creates spans on success", func(t *testing.T) {
		// A sync step with valid metadata entries creates both the step
		// span and metadata spans when AllowStepMetadata returns true.
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

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		// Replace checkpointer with one that has AllowStepMetadata enabled
		testData.checkpointer = New(Opts{
			State:           mocks.state,
			TracerProvider:  mocks.tracer,
			Queue:           mocks.queue,
			MetricsProvider: mocks.metrics,
			Executor:        mocks.executor,
			FnReader:        mocks.fnReader,
			AllowStepMetadata: executor.AllowStepMetadata(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		})

		expectedData := map[string]any{"data": json.RawMessage(`{"result": "step 1 output"}`)}
		expectedOutputBytes, _ := json.Marshal(expectedData)
		mocks.state.On("SaveStep", ctx, testData.metadata.ID, "step-1", expectedOutputBytes).Return(false, nil)

		mocks.tracer.
			On("CreateSpan", mock.Anything, mock.Anything, mock.AnythingOfType("*tracing.CreateSpanOptions")).
			Return(&meta.SpanReference{}, nil)

		mocks.metrics.On("OnStepFinished", ctx, mock.AnythingOfType("checkpoint.MetricCardinality"), enums.StepStatusCompleted)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

		// Both step and metadata spans should be created.
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
	})

	t.Run("creates spans on step error", func(t *testing.T) {
		// A sync step error with metadata entries creates both the step
		// span and metadata spans.
		ctx := context.Background()
		require := require.New(t)

		now := time.Now()
		ops := []state.GeneratorOpcode{
			{
				ID:   "step-err-1",
				Op:   enums.OpcodeStepError,
				Data: json.RawMessage(`{"error": {"message": "something failed"}}`),
				Name: "Step Error 1",
				Error: &state.UserError{
					Name:    "Error",
					Message: "something failed",
				},
				Timing: interval.New(now, now.Add(100*time.Millisecond)),
				Metadata: []metadata.ScopedUpdate{
					{
						Scope: enums.MetadataScopeStep,
						Update: metadata.Update{
							RawUpdate: metadata.RawUpdate{
								Kind:   "userland.error-context",
								Op:     enums.MetadataOpcodeMerge,
								Values: metadata.Values{"err_detail": json.RawMessage(`"detail"`)},
							},
						},
					},
				},
			},
		}

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		testData.checkpointer = New(Opts{
			State:           mocks.state,
			TracerProvider:  mocks.tracer,
			Queue:           mocks.queue,
			MetricsProvider: mocks.metrics,
			Executor:        mocks.executor,
			FnReader:        mocks.fnReader,
			AllowStepMetadata: executor.AllowStepMetadata(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		})

		mocks.tracer.
			On("CreateSpan", mock.Anything, mock.Anything, mock.AnythingOfType("*tracing.CreateSpanOptions")).
			Return(&meta.SpanReference{}, nil)

		mocks.executor.
			On("HandleGenerator", ctx, mock.AnythingOfType("*checkpoint.checkpointRunContext"), mock.MatchedBy(func(op state.GeneratorOpcode) bool {
				return op.ID == "step-err-1" && op.Op == enums.OpcodeStepError
			})).
			Return(nil)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

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
	})

	t.Run("no metadata when flag disabled", func(t *testing.T) {
		// No metadata spans are created when AllowStepMetadata returns
		// false, even if the opcode contains metadata entries.
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

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		testData.checkpointer = New(Opts{
			State:           mocks.state,
			TracerProvider:  mocks.tracer,
			Queue:           mocks.queue,
			MetricsProvider: mocks.metrics,
			Executor:        mocks.executor,
			FnReader:        mocks.fnReader,
			AllowStepMetadata: executor.AllowStepMetadata(func(ctx context.Context, acctID uuid.UUID) bool {
				return false
			}),
		})

		expectedData := map[string]any{"data": json.RawMessage(`{"result": "step 1 output"}`)}
		expectedOutputBytes, _ := json.Marshal(expectedData)
		mocks.state.On("SaveStep", ctx, testData.metadata.ID, "step-1", expectedOutputBytes).Return(false, nil)

		mocks.tracer.
			On("CreateSpan", mock.Anything, mock.Anything, mock.AnythingOfType("*tracing.CreateSpanOptions")).
			Return(&meta.SpanReference{}, nil)

		mocks.metrics.On("OnStepFinished", ctx, mock.AnythingOfType("checkpoint.MetricCardinality"), enums.StepStatusCompleted)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

		require.Len(mocks.tracer.createdSpans, 1, "Expected only 1 step span, no metadata span")
		require.Equal(meta.SpanNameStep, mocks.tracer.createdSpans[0].name)
	})

	t.Run("invalid metadata skipped", func(t *testing.T) {
		// Invalid metadata entries (failing Validate()) are skipped
		// silently without causing the checkpoint call to return an
		// error.
		ctx := context.Background()
		require := require.New(t)

		// Kind exceeding MaxKindLength to trigger Validate() failure.
		invalidKind := metadata.Kind(strings.Repeat("x", metadata.MaxKindLength+1))

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
								Kind:   invalidKind,
								Op:     enums.MetadataOpcodeMerge,
								Values: metadata.Values{"key": json.RawMessage(`"value"`)},
							},
						},
					},
				},
			},
		}

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		testData.checkpointer = New(Opts{
			State:           mocks.state,
			TracerProvider:  mocks.tracer,
			Queue:           mocks.queue,
			MetricsProvider: mocks.metrics,
			Executor:        mocks.executor,
			FnReader:        mocks.fnReader,
			AllowStepMetadata: executor.AllowStepMetadata(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		})

		expectedData := map[string]any{"data": json.RawMessage(`{"result": "step 1 output"}`)}
		expectedOutputBytes, _ := json.Marshal(expectedData)
		mocks.state.On("SaveStep", ctx, testData.metadata.ID, "step-1", expectedOutputBytes).Return(false, nil)

		mocks.tracer.
			On("CreateSpan", mock.Anything, mock.Anything, mock.AnythingOfType("*tracing.CreateSpanOptions")).
			Return(&meta.SpanReference{}, nil)

		mocks.metrics.On("OnStepFinished", ctx, mock.AnythingOfType("checkpoint.MetricCardinality"), enums.StepStatusCompleted)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err, "Invalid metadata should not cause an error")

		require.Len(mocks.tracer.createdSpans, 1, "Expected only 1 step span, invalid metadata skipped")
		require.Equal(meta.SpanNameStep, mocks.tracer.createdSpans[0].name)
	})

	t.Run("with experiment opts emits metadata", func(t *testing.T) {
		// When a step opcode's opts carry the experiment context the SDK
		// spreads in group.experiment() variant callbacks, the checkpoint
		// path emits an inngest.experiment metadata span even though the
		// opcode has no explicit Metadata entries.
		ctx := context.Background()
		require := require.New(t)

		now := time.Now()
		ops := []state.GeneratorOpcode{
			{
				ID:     "variant-step-1",
				Op:     enums.OpcodeStepRun,
				Data:   json.RawMessage(`{"result": "variant output"}`),
				Name:   "Variant Step",
				Timing: interval.New(now, now.Add(100*time.Millisecond)),
				// No explicit Metadata entries — the executor must emit
				// the experiment metadata from opts alone.
				Opts: map[string]any{
					"experimentName":    "checkout-flow",
					"variant":           "express",
					"selectionStrategy": "weighted",
				},
			},
		}

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		testData.checkpointer = New(Opts{
			State:           mocks.state,
			TracerProvider:  mocks.tracer,
			Queue:           mocks.queue,
			MetricsProvider: mocks.metrics,
			Executor:        mocks.executor,
			FnReader:        mocks.fnReader,
			AllowStepMetadata: executor.AllowStepMetadata(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		})

		expectedData := map[string]any{"data": json.RawMessage(`{"result": "variant output"}`)}
		expectedOutputBytes, _ := json.Marshal(expectedData)
		mocks.state.On("SaveStep", ctx, testData.metadata.ID, "variant-step-1", expectedOutputBytes).Return(false, nil)

		mocks.tracer.
			On("CreateSpan", mock.Anything, mock.Anything, mock.AnythingOfType("*tracing.CreateSpanOptions")).
			Return(&meta.SpanReference{}, nil)

		mocks.metrics.On("OnStepFinished", ctx, mock.AnythingOfType("checkpoint.MetricCardinality"), enums.StepStatusCompleted)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

		// 1 step span + 1 experiment metadata span.
		require.Len(mocks.tracer.createdSpans, 2, "expected step + experiment metadata span")
		var metaSpans int
		for _, s := range mocks.tracer.createdSpans {
			if s.name == meta.SpanNameMetadata {
				metaSpans++
			}
		}
		require.Equal(1, metaSpans, "expected exactly one metadata span for experiment opts")
	})

	t.Run("non-variant opts emit no experiment metadata", func(t *testing.T) {
		// Regular (non-variant) step opts do not trigger a spurious
		// experiment metadata span.
		ctx := context.Background()
		require := require.New(t)

		now := time.Now()
		ops := []state.GeneratorOpcode{
			{
				ID:     "regular-step",
				Op:     enums.OpcodeStepRun,
				Data:   json.RawMessage(`{"result": "ok"}`),
				Name:   "Regular Step",
				Timing: interval.New(now, now.Add(100*time.Millisecond)),
				Opts: map[string]any{
					"type":  "step",
					"input": []any{},
				},
			},
		}

		mocks, testData := setupSyncCheckpointTest(t, ops...)

		testData.checkpointer = New(Opts{
			State:           mocks.state,
			TracerProvider:  mocks.tracer,
			Queue:           mocks.queue,
			MetricsProvider: mocks.metrics,
			Executor:        mocks.executor,
			FnReader:        mocks.fnReader,
			AllowStepMetadata: executor.AllowStepMetadata(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		})

		expectedData := map[string]any{"data": json.RawMessage(`{"result": "ok"}`)}
		expectedOutputBytes, _ := json.Marshal(expectedData)
		mocks.state.On("SaveStep", ctx, testData.metadata.ID, "regular-step", expectedOutputBytes).Return(false, nil)

		mocks.tracer.
			On("CreateSpan", mock.Anything, mock.Anything, mock.AnythingOfType("*tracing.CreateSpanOptions")).
			Return(&meta.SpanReference{}, nil)

		mocks.metrics.On("OnStepFinished", ctx, mock.AnythingOfType("checkpoint.MetricCardinality"), enums.StepStatusCompleted)

		err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
		require.NoError(err)

		// Only the step span — no metadata span.
		require.Len(mocks.tracer.createdSpans, 1, "expected only the step span")
		require.Equal(meta.SpanNameStep, mocks.tracer.createdSpans[0].name)
	})
}

// TestCheckpointSyncSteps_RunComplete asserts that an OpcodeRunComplete op:
//   - is forwarded to Executor.Finalize with the correct StatusCode/Headers/Body;
//   - fires Executor.RunFunctionFinishedLifecycle with the same StatusCode so the
//     legacy OTel pipeline emits a terminal status
func TestCheckpointSyncSteps_RunComplete(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		wantErr    string
	}{
		{name: "success", statusCode: 200, wantErr: ""},
		{name: "server error", statusCode: 500, wantErr: "invalid status code: 500"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			require := require.New(t)

			body := `{"hello":"world"}`
			headers := map[string]string{"content-type": "application/json"}

			wire, err := json.Marshal(apiresult.APIResult{
				StatusCode: tc.statusCode,
				Headers:    headers,
				Body:       body,
			})
			require.NoError(err)

			ops := []state.GeneratorOpcode{{
				ID:   "run-complete-1",
				Op:   enums.OpcodeRunComplete,
				Data: wire,
			}}

			mocks, testData := setupSyncCheckpointTest(t, ops...)

			// matchAPIResult validates the StatusCode/Headers/Body that the
			// checkpoint path forwards into Finalize and the lifecycle. This
			// is the assertion the original bug needed: without it, a
			// zero-valued APIResult passes silently.
			matchAPIResult := func(want apiresult.APIResult) func(apiresult.APIResult) bool {
				return func(got apiresult.APIResult) bool {
					return got.StatusCode == want.StatusCode &&
						got.Body == want.Body &&
						got.Headers["content-type"] == want.Headers["content-type"]
				}
			}
			expected := apiresult.APIResult{
				StatusCode: tc.statusCode,
				Headers:    headers,
				Body:       body,
			}

			mocks.executor.On("Finalize", ctx, mock.MatchedBy(func(opts execution.FinalizeOpts) bool {
				return opts.Response.Type == execution.FinalizeResponseAPI &&
					matchAPIResult(expected)(opts.Response.APIResponse)
			})).Return(nil).Once()

			mocks.executor.On(
				"RunFunctionFinishedLifecycle",
				ctx,
				testData.metadata,
				mock.AnythingOfType("queue.Item"),
				mock.AnythingOfType("[]json.RawMessage"),
				mock.MatchedBy(func(resp state.DriverResponse) bool {
					if resp.StatusCode != tc.statusCode {
						return false
					}
					out, ok := resp.Output.(string)
					if !ok || out != body {
						return false
					}
					if tc.wantErr == "" {
						return resp.Err == nil
					}
					return resp.Err != nil && *resp.Err == tc.wantErr
				}),
			).Once()

			// OnFnFinished is dispatched in a goroutine; signal via a channel
			fnFinished := make(chan struct{}, 1)
			mocks.metrics.On(
				"OnFnFinished",
				mock.Anything,
				mock.AnythingOfType("checkpoint.MetricCardinality"),
				enums.RunStatusCompleted,
			).Run(func(args mock.Arguments) {
				fnFinished <- struct{}{}
			}).Once()

			err = testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
			require.NoError(err)

			select {
			case <-fnFinished:
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for OnFnFinished")
			}

			mocks.executor.AssertExpectations(t)
			mocks.metrics.AssertExpectations(t)
		})
	}
}

// On a duplicate save, the OnStepFinished metric must be suppressed —
// otherwise the same step gets counted twice (once by whichever path
// originally persisted it, once by this checkpoint).
func TestCheckpointSyncSteps_DuplicateSaveSuppressesStepFinishedMetric(t *testing.T) {
	ctx := context.Background()
	require := require.New(t)

	op := state.GeneratorOpcode{
		ID:   "step-1",
		Op:   enums.OpcodeStepRun,
		Data: json.RawMessage(`{"result": "step 1 output"}`),
		Name: "Step 1",
	}

	mocks, testData := setupSyncCheckpointTest(t, op)

	expectedData := map[string]any{"data": json.RawMessage(op.Data)}
	expectedOutputBytes, _ := json.Marshal(expectedData)
	mocks.state.On("SaveStep", ctx, testData.metadata.ID, op.ID, expectedOutputBytes).
		Return(false, state.ErrDuplicateResponse)

	err := testData.checkpointer.CheckpointSyncSteps(ctx, testData.syncCheckpoint)
	require.NoError(err)

	mocks.metrics.AssertNotCalled(t, "OnStepFinished")
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

func (m *mockExecutor) RunFunctionFinishedLifecycle(
	ctx context.Context,
	md state.Metadata,
	item queue.Item,
	evts []json.RawMessage,
	resp state.DriverResponse,
) {
	// Only record if a test has registered an expectation; otherwise no-op.
	// This keeps existing tests (which don't exercise OpcodeRunComplete) from
	// having to declare the call.
	for _, c := range m.ExpectedCalls {
		if c.Method == "RunFunctionFinishedLifecycle" {
			m.Called(ctx, md, item, evts, resp)
			return
		}
	}
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
