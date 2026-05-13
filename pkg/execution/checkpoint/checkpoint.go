package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/apiresult"
	"github.com/inngest/inngest/pkg/execution/defers"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/executor/queueref"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/flags"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/oklog/ulid/v2"
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

const pkgName = "checkpoint"

var ErrStaleDispatch = errors.New("stale dispatch")

// Disallow dispatch validation if the queue item is younger than this duration.
// This is to reduce the number of validations, which in turn reduces load on
// the queue.
//
// We chose 10 seconds somewhat arbitrarily. We want a value that will not
// exceed timeout durations on our users' cloud providers, and some serverless
// providers have a 10 second timeout.
const dispatchValidationSkipDuration = 10 * time.Second

type queueItemLoader interface {
	LoadQueueItem(ctx context.Context, shardName string, itemID string) (*queue.QueueItem, error)
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
	// BackoffFunc computes the retry time for a given attempt number.
	// If nil, defaults to backoff.DefaultBackoff.
	BackoffFunc backoff.BackoffFunc
	// AllowStepMetadata controls whether step metadata is allowed for a given account.
	AllowStepMetadata executor.AllowStepMetadata
	// AllowAsyncDispatchValidation gates the dispatch validator per account.
	AllowAsyncDispatchValidation flags.BoolFlag
}

func New(o Opts) Checkpointer {
	if o.MetricsProvider == nil {
		o.MetricsProvider = nilCheckpointMetrics{}
	}
	if o.BackoffFunc == nil {
		o.BackoffFunc = backoff.GetLinearBackoffFunc(5 * time.Second)
	}

	return checkpointer{o}
}

type checkpointer struct {
	Opts
}

func (c checkpointer) Metrics() MetricsProvider {
	return c.MetricsProvider
}

func sanitizeLogValue(v string) string {
	v = strings.ReplaceAll(v, "\n", "")
	v = strings.ReplaceAll(v, "\r", "")
	return v
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

	// >1 non-lazy steps means parallel mode — see enums.OpcodeIsLazy.
	nonLazyCount := 0
	for _, s := range input.Steps {
		if !enums.OpcodeIsLazy(s.Op) {
			nonLazyCount++
		}
	}
	if nonLazyCount > 1 && !input.Metadata.Config.ForceStepPlan {
		if err := c.State.UpdateMetadata(ctx, input.Metadata.ID, state.MutableConfig{
			ForceStepPlan:  true,
			RequestVersion: input.Metadata.Config.RequestVersion,
			StartedAt:      input.Metadata.Config.StartedAt,
			HasAI:          input.Metadata.Config.HasAI,
		}); err != nil {
			return fmt.Errorf("updating metadata to force step plan: %w", err)
		}
		// Update the local metadata so subsequent operations see the change
		input.Metadata.Config.ForceStepPlan = true
	}

	// Depending on the type of steps, we may end up switching the run from sync to async.  For example,
	// if the opcodes are sleeps, waitForEvents, inferences, etc. we will be resuming the API endpoint
	// at some point in the future.
	onChangeToAsync := sync.OnceFunc(func() { c.updateSpanAsync(ctx, input) })

	// Drain priority opcodes before the rest.
	//
	// NOTE: This assumes that ops are processed sequentially. If they aren't,
	// then priority order would only decrease the chance of a race, but not
	// eliminate it.
	ordered := make([]state.GeneratorOpcode, 0, len(input.Steps))
	for _, op := range input.Steps {
		if enums.OpcodeIsPriority(op.Op) {
			ordered = append(ordered, op)
		}
	}
	for _, op := range input.Steps {
		if !enums.OpcodeIsPriority(op.Op) {
			ordered = append(ordered, op)
		}
	}

	for _, op := range ordered {
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
				stepName := sanitizeLogValue(op.UserDefinedName())
				if errors.Is(err, state.ErrDuplicateResponse) || errors.Is(err, state.ErrIdempotentResponse) {
					// Ignore.
					l.Warn("duplicate checkpoint step", "id", input.Metadata.ID, "name", stepName)
					continue
				}
				if err != nil {
					l.Error("error saving checkpointed step state", "name", stepName, "error", err)
					return fmt.Errorf("failed to save step %s (%s): %w", op.ID, stepName, err)
				}
			}

			// Create a deterministic executor.step span whose ID matches what the SDK
			// generates, so userland spans are correctly parented underneath it.
			max := fn.MaxAttempts()
			stepSpanRef, err := c.TracerProvider.CreateSpan(
				tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
					Identifier:  input.Metadata.ID,
					Attempt:     runCtx.AttemptCount(),
					MaxAttempts: &max,
				}),
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Debug:      &tracing.SpanDebugData{Location: "checkpoint.SyncStep"},
					Seed:       []byte(fmt.Sprintf("%s:%d", op.ID, runCtx.AttemptCount())),
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

			c.processMetadata(ctx, l, input.AccountID, input.Metadata, stepSpanRef, op, "checkpoint.SyncStep.metadata")

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
			stepSpanRef, err := c.TracerProvider.CreateSpan(
				tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
					Identifier:  input.Metadata.ID,
					Attempt:     runCtx.AttemptCount(),
					MaxAttempts: &max,
				}),
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Debug:      &tracing.SpanDebugData{Location: "checkpoint.SyncErr"},
					Seed:       []byte(fmt.Sprintf("%s:%d", op.ID, runCtx.AttemptCount())),
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

			c.processMetadata(ctx, l, input.AccountID, input.Metadata, stepSpanRef, op, "checkpoint.SyncErr.metadata")

			err = c.Executor.HandleGenerator(ctx, runCtx, op)
			if errors.Is(err, executor.ErrHandledStepError) {
				// In the executor, returning an error bubbles up to the queue to requeue.
				jobID := fmt.Sprintf("%s-%s-sync-retry", runCtx.Metadata().IdempotencyKey(), op.ID)
				retryAt := c.BackoffFunc(1)

				// Inject the step span reference into the retry queue item metadata
				// so that execution spans created during retries are properly parented
				// under the step span (instead of being orphaned with parent=0000).
				retryMetadata := make(map[string]any)
				if stepSpanRef != nil {
					if byt, merr := json.Marshal(stepSpanRef); merr == nil {
						retryMetadata[meta.PropagationKey] = string(byt)
					}
				}

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
					Metadata:              retryMetadata,
					ParallelMode:          enums.ParallelModeWait,
				}

				// Continue checking this particular error.
				if err = c.Queue.Enqueue(ctx, nextItem, retryAt, queue.EnqueueOpts{}); err != nil {
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

		case enums.OpcodeDeferAdd:
			if err := defers.SaveFromOp(ctx, c.State, l, input.Metadata.ID, op); err != nil {
				// Log without returning the error: a bad defer must
				// never fail its parent run. We may rethink this as
				// the Defer feature matures.
				l.Error(
					"error handling defer add in checkpoint",
					"error", err,
					"step_id", sanitizeLogValue(op.ID),
					"run_id", input.Metadata.ID.RunID.String(),
				)
			}

		case enums.OpcodeDeferAbort:
			if err := defers.AbortFromOp(ctx, c.State, l, input.Metadata.ID, op); err != nil {
				// Log without returning the error: a bad defer must
				// never fail its parent run. We may rethink this as
				// the Defer feature matures.
				l.Error(
					"error handling defer abort in checkpoint",
					"error", err,
					"step_id", sanitizeLogValue(op.ID),
					"run_id", input.Metadata.ID.RunID.String(),
				)
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

	// Persist cumulative metadata size delta to Redis so subsequent checkpoint
	// requests (potentially on different instances) see the updated total.
	if delta := input.Metadata.Metrics.MetadataSizeDelta(); delta > 0 {
		if err := state.TryIncrementMetadataSize(ctx, c.State, input.Metadata.ID, delta); err != nil {
			l.Warn("error persisting metadata size delta", "error", err, "delta", delta)
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

	if c.AllowAsyncDispatchValidation.Enabled(ctx, input.AccountID) {
		if err := c.validateAsyncDispatch(ctx, input); err != nil {
			return err
		}
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
				return fmt.Errorf("failed to save step %s: %w", op.ID, err)
			}

			stepSpanRef, err := c.TracerProvider.CreateSpan(
				tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
					Identifier: md.ID,
					Attempt:    0,
					// XXX: MaxAttempts isn't stored here, as we don't have that info at this time.,
				}),
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Debug:      &tracing.SpanDebugData{Location: "checkpoint.AsyncStep"},
					Seed:       []byte(fmt.Sprintf("%s:%d", op.ID, 0)),
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

			c.processMetadata(ctx, l, input.AccountID, &md, stepSpanRef, op, "checkpoint.AsyncStep.metadata")

		case enums.OpcodeDeferAdd:
			if err := defers.SaveFromOp(ctx, c.State, l, md.ID, op); err != nil {
				// Log without returning the error: a bad defer must
				// never fail its parent run. We may rethink this as
				// the Defer feature matures.
				l.Error(
					"error handling defer add in checkpoint",
					"error", err,
					"step_id", sanitizeLogValue(op.ID),
					"run_id", md.ID.RunID.String(),
				)
			}

		case enums.OpcodeDeferAbort:
			if err := defers.AbortFromOp(ctx, c.State, l, md.ID, op); err != nil {
				// Log without returning the error: a bad defer must
				// never fail its parent run. We may rethink this as
				// the Defer feature matures.
				l.Error(
					"error handling defer abort in checkpoint",
					"error", err,
					"step_id", sanitizeLogValue(op.ID),
					"run_id", md.ID.RunID.String(),
				)
			}

		default:
			// Return an error
			l.Error("unimplemented checkpoint op", "op", op.Op)
			return fmt.Errorf("cannot checkpoint opcode: %s", op.Op)
		}
	}

	// Persist cumulative metadata size delta to Redis so subsequent checkpoint
	// requests (potentially on different instances) see the updated total.
	if delta := md.Metrics.MetadataSizeDelta(); delta > 0 {
		if err := state.TryIncrementMetadataSize(ctx, c.State, md.ID, delta); err != nil {
			l.Warn("error persisting metadata size delta", "error", err, "delta", delta)
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

func (c checkpointer) validateAsyncDispatch(ctx context.Context, input AsyncCheckpoint) (err error) {
	start := time.Now()
	result := "skipped"
	defer func() {
		if errors.Is(err, ErrStaleDispatch) {
			result = "stale"
		}
		metrics.HistogramCheckpointAsyncDispatchValidationDuration(ctx, time.Since(start), metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"result": result},
		})
	}()

	// Fail open when the SDK didn't echo a request id. Older SDKs predate the
	// fence; rejecting them would break valid checkpoints.
	if input.RequestID == "" {
		return nil
	}

	// Skip the queue-item load when the dispatch is younger than the minimum
	// requeue window. A Requeue is the only path that bumps GenerationID, and
	// it can't fire until the queue lease expires, so a fresh dispatch is
	// provably uncontested. Negative elapsed (future-dated stamp from clock
	// skew or a buggy SDK) falls through to the existing validation.
	if input.RequestStartedAt != 0 {
		elapsed := time.Since(time.UnixMilli(input.RequestStartedAt))
		if elapsed >= 0 && elapsed < dispatchValidationSkipDuration {
			return nil
		}
	}

	parsed, err := ulid.Parse(input.RequestID)
	if err != nil {
		return fmt.Errorf("%w: invalid request id %q: %v", ErrStaleDispatch, input.RequestID, err)
	}

	ref := queueref.Decode(input.QueueItemRef)
	if ref[0] == "" {
		return fmt.Errorf("%w: missing queue item reference", ErrStaleDispatch)
	}

	loader, ok := c.Queue.(queueItemLoader)
	if !ok {
		// Fail open if the queue can't load items (e.g. mock or alt backend);
		// the alternative is rejecting every fenced POST forever.
		logger.StdlibLogger(ctx).Warn("checkpoint: queue does not support dispatch validation; skipping",
			"run_id", input.RunID,
		)
		return nil
	}

	item, err := loader.LoadQueueItem(ctx, ref.ShardID(), ref.JobID())
	if errors.Is(err, queue.ErrQueueItemNotFound) {
		return fmt.Errorf("%w: queue item not found", ErrStaleDispatch)
	}
	if err != nil {
		// Fail open on transient load errors (e.g. Redis timeout). Rejecting
		// here would surface as HTTP 400 and abort an otherwise-valid run.
		logger.StdlibLogger(ctx).Warn("checkpoint: failed to load queue item for dispatch validation; skipping",
			"error", err,
			"run_id", input.RunID,
		)
		return nil
	}
	// Fail open for queue items that pre-date the rollout.
	if item.GenerationID == 0 {
		return nil
	}

	result = "passed"

	// Compare only the entropy: the dispatch timestamp isn't recoverable
	// from the SDK-echoed RequestID and isn't part of the fence.
	if !bytes.Equal(parsed.Entropy(), driver.DispatchRequestIDEntropy(input.RunID, item.GenerationID)) {
		return fmt.Errorf("%w: request id %s does not match queue item generation %d", ErrStaleDispatch, input.RequestID, item.GenerationID)
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

func (c checkpointer) processMetadata(
	ctx context.Context,
	l logger.Logger,
	accountID uuid.UUID,
	md *state.Metadata,
	stepSpanRef *meta.SpanReference,
	op state.GeneratorOpcode,
	location string,
) {
	if !c.AllowStepMetadata.Enabled(ctx, accountID) {
		return
	}

	// Extract experiment metadata from opts and merge into the list of
	// metadata entries to process. The SDK spreads group.experiment()
	// variant context onto a sub-step's opts; emitting the metadata span
	// here (rather than requiring the SDK to call addMetadata()) means
	// end users get experiment observability without an SDK upgrade and
	// keeps the emission consistent across SDK languages.
	metadataEntries := op.Metadata
	if expMd, err := extractors.ExtractExperimentOptsMetadata(op.Opts); err != nil {
		l.Warn("error extracting experiment opts metadata",
			"error", err,
			"run_id", md.ID.RunID,
		)
	} else if expMd != nil {
		values, serializeErr := expMd.Serialize()
		if serializeErr != nil {
			l.Warn("error serializing experiment metadata",
				"error", serializeErr,
				"run_id", md.ID.RunID,
			)
		} else {
			metadataEntries = append(metadataEntries, metadata.ScopedUpdate{
				Scope: enums.MetadataScopeStep,
				Update: metadata.Update{
					RawUpdate: metadata.RawUpdate{
						Kind:   expMd.Kind(),
						Op:     expMd.Op(),
						Values: values,
					},
				},
			})
		}
	}

	for _, spanMd := range metadataEntries {
		if err := spanMd.Validate(); err != nil {
			l.Warn("invalid metadata in checkpoint step",
				"error", err,
				"run_id", md.ID.RunID,
				"metadata_kind", spanMd.Kind(),
			)
			continue
		}

		values, err := spanMd.Serialize()
		if err != nil {
			l.Warn("failed to serialize metadata in checkpoint step",
				"error", err,
				"run_id", md.ID.RunID,
				"metadata_kind", spanMd.Kind(),
			)
			continue
		}

		// Resolve the parent span based on metadata scope, matching the
		// executor's behavior in createMetadataSpan.
		var parent *meta.SpanReference
		switch spanMd.Scope {
		case enums.MetadataScopeRun:
			parent = tracing.RunSpanRefFromMetadata(md)
		case enums.MetadataScopeStep, enums.MetadataScopeStepAttempt:
			// Use the step span created just before this call.
			// Fall back to the run span if the step span was not captured.
			if stepSpanRef != nil {
				parent = stepSpanRef
			} else {
				parent = tracing.RunSpanRefFromMetadata(md)
			}
		default:
			parent = tracing.RunSpanRefFromMetadata(md)
		}

		_, err = tracing.CreateMetadataSpanFromValues(
			ctx,
			c.TracerProvider,
			parent,
			location,
			"checkpoint",
			md,
			spanMd.Kind(),
			spanMd.Op(),
			values,
			spanMd.Scope,
		)
		if err != nil {
			l.Warn("error creating metadata span in checkpoint",
				"error", err,
				"run_id", md.ID.RunID,
				"metadata_kind", spanMd.Kind(),
				"metadata_size", values.Size(),
				"cumulative_metadata_size", md.Metrics.MetadataSize,
			)
		}
	}
}
