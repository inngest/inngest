package checkpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/apiresult"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/executor/queueref"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
)

type Checkpointer interface {
	AsyncCheckpointer
	SyncCheckpointer

	Metrics() MetricsProvider
}

type SyncCheckpointer interface {
	// CheckpointSyncSteps checkpoints steps for a sync function (HTTP-based APIs).
	CheckpointSyncSteps(context.Context, SyncCheckpoint) error
}

type AsyncCheckpointer interface {
	// CheckpointAsyncSteps checkpoints steps for an async function.
	CheckpointAsyncSteps(context.Context, AsyncCheckpoint) error
}

type Opts struct {
	// State allows loading and mutating state from various checkpointing APIs.
	State state.RunService
	// FnReader reads functions from a backing store.
	FnReader cqrs.FunctionReader
	// Executor is required to cancel and manage function executions.
	Executor execution.Executor
	// TracerProvider is used to create spans within the APIv1 endpoints and allows the checkpointing API to write traces.
	TracerProvider tracing.TracerProvider
	// Queue allows the checkppinting API to continue by enqueueing new queue items.
	Queue queue.Queue
	// MetricsProvider reports usage metrics.
	MetricsProvider MetricsProvider
}

func New(o Opts) Checkpointer {
	if o.MetricsProvider == nil {
		o.MetricsProvider = nilCheckpointMetrics{}
	}

	return checkpointer{o}
}

type checkpointer struct {
	Opts
}

func (c checkpointer) Metrics() MetricsProvider {
	return c.MetricsProvider
}

// CheckpointSyncSteps handles the checkpointing of new steps via sync, HTTP-based functions
// that are treated as API endpoints.
//
// This accepts all opcodes in the current request, then handles trace pipelines and optional
// state updates in the state store for resumability.
func (c checkpointer) CheckpointSyncSteps(ctx context.Context, input SyncCheckpoint) error {
	if input.Metadata == nil {
		md, err := c.State.LoadMetadata(ctx, input.ID())
		if errors.Is(err, state.ErrRunNotFound) || errors.Is(err, state.ErrMetadataNotFound) {
			// Handle run not found with 404
			return err
		}
		if err != nil {
			logger.StdlibLogger(ctx).Error("error loading state for background checkpoint steps", "error", err)
			return err
		}
		input.Metadata = &md
	}

	l := logger.StdlibLogger(ctx).With("run_id", input.Metadata.ID.RunID)

	// Load the function config.
	fn, err := c.fn(ctx, input.Metadata.ID.FunctionID)
	if err != nil {
		logger.StdlibLogger(ctx).Warn("error loading fn for background checkpoint steps", "error", err)
		return err
	}

	runCtx := c.runContext(*input.Metadata, fn)

	// If the opcodes contain a function finished op, we don't need to bother serializing
	// to the state store.  We only care about serializing state if we switch from sync -> async,
	// as the state will be used for resuming functions.
	complete := slices.ContainsFunc(input.Steps, func(s state.GeneratorOpcode) bool {
		return s.Op == enums.OpcodeRunComplete
	})

	// Depending on the type of steps, we may end up switching the run from sync to async.  For example,
	// if the opcodes are sleeps, waitForEvents, inferences, etc. we will be resuming the API endpoint
	// at some point in the future.
	onChangeToAsync := sync.OnceFunc(func() { c.updateSpanAsync(ctx, input) })

	for _, op := range input.Steps {
		attrs := tracing.GeneratorAttrs(&op)
		tracing.AddMetadataTenantAttrs(attrs, input.Metadata.ID)

		switch op.Op {
		case enums.OpcodeStepRun, enums.OpcodeStep:
			// Steps are checkpointed after they execute.  We only need to store traces here, then
			// continune; we do not need to handle anything within the executor.

			output, err := op.Output()
			if err != nil {
				l.Error("error fetching checkpoint step output", "error", err)
			}

			if !complete {
				// Checkpointing happens in this API when either the function finishes or we move to
				// async.  Therefore, we onl want to save state if we don't have a complete opcode,
				// as all complete functions will never re-enter.
				_, err := c.State.SaveStep(ctx, input.Metadata.ID, op.ID, []byte(output))
				if errors.Is(err, state.ErrDuplicateResponse) || errors.Is(err, state.ErrIdempotentResponse) {
					// Ignore.
					l.Warn("duplicate checkpoint step", "id", input.Metadata.ID)
					continue
				}
				if err != nil {
					l.Error("error saving checkpointed step state", "error", err)
				}
			}

			max := fn.MaxAttempts()
			_, err = c.TracerProvider.CreateSpan(
				tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
					Identifier:  input.Metadata.ID,
					Attempt:     runCtx.AttemptCount(),
					MaxAttempts: &max,
				}),
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Debug:      &tracing.SpanDebugData{Location: "checkpoint.SyncStep"},
					Parent:     tracing.RunSpanRefFromMetadata(input.Metadata),
					StartTime:  op.Timing.Start(),
					EndTime:    op.Timing.End(),
					Attributes: stepRunAttrs(attrs, op, input.RunID),
				},
			)
			if err != nil {
				// We should never hit a blocker creating a span.  If so, warn loudly.
				l.Error("error saving span for checkpoint op", "error", err)
			}

			go c.MetricsProvider.OnStepFinished(ctx, MetricCardinality{
				AccountID: input.AccountID,
				EnvID:     input.EnvID,
				AppID:     input.AppID,
				FnID:      input.FnID,
			}, enums.StepStatusCompleted)

		case enums.OpcodeStepError, enums.OpcodeStepFailed:
			// StepErrors are unique.  Firstly, we must always store traces.  However, if
			// we retry the step, we move from sync -> async, requiring jobs to be scheduled.
			//
			// If steps only have one attempt, however, we can assume that the SDK handles
			// step errors and continues
			status := enums.StepStatusErrored
			max := fn.MaxAttempts()
			_, err = c.TracerProvider.CreateSpan(
				tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
					Identifier:  input.Metadata.ID,
					Attempt:     runCtx.AttemptCount(),
					MaxAttempts: &max,
				}),
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Debug:      &tracing.SpanDebugData{Location: "checkpoint.SyncErr"},
					Parent:     tracing.RunSpanRefFromMetadata(input.Metadata),
					StartTime:  op.Timing.Start(),
					EndTime:    op.Timing.End(),
					Attributes: stepErrorAttrs(attrs, op, input.RunID, status),
				},
			)
			if err != nil {
				// We should never hit a blocker creating a span.  If so, warn loudly.
				l.Error("error saving span for checkpoint step error op", "error", err)
			}

			err := c.Executor.HandleGenerator(ctx, runCtx, op)
			if errors.Is(err, executor.ErrHandledStepError) {
				// In the executor, returning an error bubbles up to the queue to requeue.
				jobID := fmt.Sprintf("%s-%s-sync-retry", runCtx.Metadata().IdempotencyKey(), op.ID)
				now := time.Now()
				nextItem := queue.Item{
					JobID:                 &jobID,
					WorkspaceID:           runCtx.Metadata().ID.Tenant.EnvID,
					Kind:                  queue.KindEdge,
					Identifier:            state.V1FromMetadata(*runCtx.Metadata()),
					PriorityFactor:        runCtx.PriorityFactor(),
					CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
					Attempt:               1, // This is now the next attempt.
					MaxAttempts:           runCtx.MaxAttempts(),
					Payload:               queue.PayloadEdge{Edge: inngest.SourceEdge}, // doesn't matter for sync functions.
					Metadata:              make(map[string]any),
					ParallelMode:          enums.ParallelModeWait,
				}

				// Continue checking this particular error.
				if err = c.Queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{}); err != nil {
					l.Error("error enqueueing step error in checkpoint", "error", err, "opcode", op.Op)
				}
			}
			if err != nil {
				l.Error("error handlign step error in checkpoint", "error", err, "opcode", op.Op)
			}

		case enums.OpcodeRunComplete:
			result := struct {
				Data apiresult.APIResult `json:"data"`
			}{}
			if err := json.Unmarshal(op.Data, &result); err != nil {
				l.Error("error unmarshalling api result from sync RunComplete op", "error", err)
			}

			go c.MetricsProvider.OnFnFinished(ctx, MetricCardinality{
				AccountID: input.AccountID,
				EnvID:     input.EnvID,
				AppID:     input.AppID,
				FnID:      input.FnID,
			}, enums.RunStatusCompleted)

			// Call finalize and process the entire op.
			if err := c.finalize(ctx, *input.Metadata, result.Data); err != nil {
				l.Error("error finalizing sync run", "error", err)
			}

		default:
			// This is an async opcode (sleep, waitForEvent, invoke, etc.) that causes
			// the run to transition from sync to async mode. Track this on the run span
			// only once per checkpoint.
			onChangeToAsync()

			if err := c.Executor.HandleGenerator(ctx, runCtx, op); err != nil {
				l.Error("error handling generator in checkpoint", "error", err, "opcode", op.Op)
			}
		}
	}

	l.Info("handled sync checkpoint", "ops", len(input.Steps), "complete", complete)
	return nil
}

func (c checkpointer) updateSpanAsync(ctx context.Context, input SyncCheckpoint) {
	l := logger.StdlibLogger(ctx).With("run_id", input.Metadata.ID.RunID)

	modeChangedAt := time.Now()
	runSpanRef := tracing.RunSpanRefFromMetadata(input.Metadata)
	if runSpanRef == nil {
		return
	}
	err := c.TracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
		TargetSpan: runSpanRef,
		Metadata:   input.Metadata,
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DurableEndpointModeChangedAt, &modeChangedAt),
		),
	})
	if err != nil {
		l.Warn("error updating run span with mode change time", "error", err)
	}
}

// CheckpointAsyncSteps is used to checkpoint from background functions (async functions).
//
// In this case, we can assume that a background function is executed via the classic job
// queue means, and that we're executing steps as we receive them.
//
// Note that step opcodes here in the future could be sync or async:  it's theoretically
// valid for an executor to hit the SDK;  the SDK to checkpoint
// StepRun, StepRun, StepWaitForEvent], then return a noop StepNone to the original executor.
//
// For now, though, we assume that this only contains sync steps.
func (c checkpointer) CheckpointAsyncSteps(ctx context.Context, input AsyncCheckpoint) error {
	l := logger.StdlibLogger(ctx).With(
		"run_id", input.RunID,
		"account_id", input.AccountID,
		"env_id", input.EnvID,
	)

	t := time.Now()
	err := c.checkpointAsyncSteps(ctx, input, l)
	if d := time.Since(t); d > time.Second {
		l.Warn("slow async checkpoint", "duration_ms", d.Milliseconds())
	}
	return err
}

func (c checkpointer) checkpointAsyncSteps(ctx context.Context, input AsyncCheckpoint, l logger.Logger) error {
	md, err := c.State.LoadMetadata(ctx, input.ID())
	if errors.Is(err, state.ErrRunNotFound) || errors.Is(err, state.ErrMetadataNotFound) {
		// Handle run not found with 404
		return err
	}
	if err != nil {
		l.Error("error loading state for background checkpoint steps", "error", err)
		return err
	}

	// NOTE: This should never contain async steps, because checkpointing is only used
	// when sync steps are found.  Here, though, we check to see if there are async steps
	// and track warnings if so.  It could still *technically* work, but is not the paved path
	// that we want, and so is unimplemented.
	async := slices.ContainsFunc(input.Steps, func(s state.GeneratorOpcode) bool {
		return enums.OpcodeIsAsync(s.Op)
	})
	if async {
		l.Error("found async steps in async checkpoint")
		return fmt.Errorf("cannot checkpoint async steps")
	}

	for _, op := range input.Steps {
		attrs := tracing.GeneratorAttrs(&op)
		tracing.AddMetadataTenantAttrs(attrs, md.ID)

		switch op.Op {
		case enums.OpcodeStepRun, enums.OpcodeStep:
			// Checkpointing is also always used while runs are in progress.  These must always
			// be stored in the state store.
			output, err := op.Output()
			if err != nil {
				l.Error("error fetching checkpoint step output", "error", err)
			}

			_, err = c.State.SaveStep(ctx, md.ID, op.ID, []byte(output))
			if errors.Is(err, state.ErrDuplicateResponse) || errors.Is(err, state.ErrIdempotentResponse) {
				// Ignore.
				l.Warn("duplicate checkpoint step", "id", md.ID)
				continue
			}
			if err != nil {
				l.Error("error saving checkpointed step state", "error", err)
			}

			_, err = c.TracerProvider.CreateSpan(
				tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
					Identifier: md.ID,
					Attempt:    0,
					// XXX: MaxAttempts isn't stored here, as we don't have that info at this time.,
				}),
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Debug:      &tracing.SpanDebugData{Location: "checkpoint.AsyncStep"},
					Seed:       []byte(op.ID + op.Timing.String()),
					Parent:     tracing.RunSpanRefFromMetadata(&md),
					StartTime:  op.Timing.Start(),
					EndTime:    op.Timing.End(),
					Attributes: stepRunAttrs(attrs, op, input.RunID),
				},
			)
			if err != nil {
				// We should never hit a blocker creating a span.  If so, warn loudly.
				l.Error("error saving span for checkpoint op", "error", err)
			}
		default:
			// Return an error
			l.Error("unimplemented checkpoint op", "op", op.Op)
			return fmt.Errorf("cannot checkpoint opcode: %s", op.Op)
		}
	}

	// Decode the queue item ID into its shard and job ID.
	ref := queueref.Decode(input.QueueItemRef)
	if ref[0] == "" {
		return nil
	}

	if err := c.Queue.ResetAttemptsByJobID(ctx, ref.ShardID(), ref.JobID()); err != nil {
		l.Error("error resetting queue item attempts", "error", err)
		return err
	}

	return nil
}

func (c checkpointer) runContext(md state.Metadata, fn *inngest.Function) execution.RunContext {
	// Create a run context specifically for each op;  we need this for any
	// async op, such as the step error and what not.
	client := exechttp.Client(exechttp.SecureDialerOpts{})
	httpClient := &client

	// Create the run context with simplified data
	return &checkpointRunContext{
		md:         md,
		httpClient: httpClient,
		events:     []json.RawMessage{}, // Empty for checkpoint context
		groupID:    uuid.New().String(),

		// Sync checkpoints always have a 0 attempt index, as this API
		// endpoint is only for sync functions that have not yet re-entered,
		// ie. first attempts at teps.
		attemptCount: 0,

		maxAttempts:     fn.MaxAttempts(),
		priorityFactor:  nil,                         // Use default priority
		concurrencyKeys: []state.CustomConcurrency{}, // No custom concurrency
		parallelMode:    enums.ParallelModeWait,      // Default to serial
	}
}

// finalize finishes a run after receiving a RunComplete opcode.  This assumes that all prior
// work has finished, and eg. step.Defer items are not running.
func (c checkpointer) finalize(ctx context.Context, md state.Metadata, result apiresult.APIResult) error {
	httpHeader := http.Header{}
	for k, v := range result.Headers {
		httpHeader[k] = []string{v}
	}

	return c.Executor.Finalize(ctx, execution.FinalizeOpts{
		Metadata: md,
		Response: execution.FinalizeResponse{
			Type:        execution.FinalizeResponseAPI,
			APIResponse: result,
		},
		Optional: execution.FinalizeOptional{},
	})
}

func (c checkpointer) fn(ctx context.Context, fnID uuid.UUID) (*inngest.Function, error) {
	// Load the function config.
	cfn, err := c.FnReader.GetFunctionByInternalUUID(ctx, fnID)
	if err != nil {
		return nil, fmt.Errorf("error loading function: %w", err)
	}
	return cfn.InngestFunction()
}
