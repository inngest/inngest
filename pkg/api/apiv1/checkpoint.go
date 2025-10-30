package apiv1

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngestgo"
	"github.com/karlseguin/ccache/v2"
	"github.com/oklog/ulid/v2"
)

var CheckpointRoutePrefixes = []string{
	"/http/runs",  // old prefix in private beta
	"/checkpoint", // public prefix
}

var (
	CheckpointOutputWaitMax = time.Minute * 5

	// CheckpointPollInterval is the duration between each poll for HTTP responses
	CheckpointPollInterval = time.Second * 2
)

// CheckpointAPI represents an API implementation for the checkpointing implementations.
type CheckpointAPI interface {
	// http.Handler allows the checkpoint API to be mounted.
	http.Handler

	// CheckpointNewRun attempts to start a new run, including any steps that may have already
	// been ran in the sync based function.
	CheckpointNewRun(w http.ResponseWriter, r *http.Request)
	// CheckpointSteps checkpoints steps to an already existing sync run.
	CheckpointSteps(w http.ResponseWriter, r *http.Request)
	// CheckpointAsyncSteps checkpoints steps to an already existing async run.  This allows async
	// functions to continue execution immediately after steps.
	CheckpointAsyncSteps(w http.ResponseWriter, r *http.Request)
	// Output returns run output given a JWT which has access to the given {env, runID} claimset.
	Output(w http.ResponseWriter, r *http.Request)
}

// RunOutputReader represents any implementation that fetches run outputs.
type RunOutputReader interface {
	// RunOutput fetches run outputs given an environment ID and a run ID.
	RunOutput(ctx context.Context, envID uuid.UUID, runID ulid.ULID) ([]byte, error)
}

// CheckpointAPIOpts represents options for the checkpoint API.
type CheckpointAPIOpts struct {
	// CheckpointMetrics records metrics for checkpoints.
	CheckpointMetrics CheckpointMetricsProvider
	// RunOutputReader is the reader used to fetch run outputs for checkpoint APIs.
	RunOutputReader RunOutputReader
	// RunJWTSecret is the secret for signing run claim JWTs, allowing sync APIs
	// to redirect to an API endpoint that fetches outputs for a specific run.
	RunJWTSecret []byte
}

// checkpointAPI is the base implementation.
type checkpointAPI struct {
	Opts

	chi.Router

	// upserted tracks fn IDs and their associated config in memory once upserted, allowing
	// us to prevent DB queries from hitting the DB each time a sync fn begins.
	upserted *ccache.Cache
	// runClaimsSecret is the secret for creating run claims JWTs
	runClaimsSecret []byte
	// outputReader allows us to read run output for a given env / run ID
	outputReader RunOutputReader
	// m records metrics, eg. for runs and so on.
	m CheckpointMetricsProvider
}

func NewCheckpointAPI(o Opts) CheckpointAPI {
	metrics := o.CheckpointOpts.CheckpointMetrics
	if metrics == nil {
		metrics = nilCheckpointMetrics{}
	}

	api := checkpointAPI{
		Router:          chi.NewRouter(),
		Opts:            o,
		upserted:        ccache.New(ccache.Configure().MaxSize(10_000)),
		runClaimsSecret: o.CheckpointOpts.RunJWTSecret,
		outputReader:    o.CheckpointOpts.RunOutputReader,
		m:               metrics,
	}

	api.Post("/", api.CheckpointNewRun)
	api.Post("/{runID}/steps", api.CheckpointSteps)
	api.Post("/{runID}/async", api.CheckpointAsyncSteps)
	api.HandleFunc("/{runID}/output", api.Output)

	return api
}

// CheckpointNewRun creates new runs from API-based functions.  These functions do NOT
// start via events;  insteasd, they start directly when your own API is hit.
//
// This checkpointing API is specifically responsible for creating new runs in the state
// store (allwoing replay), and for organizing o11y around the run (traces, metrics, and
// so on).
//
// In the future, this will manage flow control for API-based runs.
func (a checkpointAPI) CheckpointNewRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "Unauthorized"))
		return
	}

	input := &CheckpointNewRunRequest{}
	if err = json.NewDecoder(io.LimitReader(r.Body, consts.MaxStepInputSize)).Decode(input); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid request body"))
		return
	}

	evt := event.InternalEvent{
		ID:          ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:   auth.AccountID(),
		WorkspaceID: auth.WorkspaceID(),
		Event:       runEvent(*input),
	}

	// Publish the event in a goroutine to lower latency in the API.  This is, while extremely important for
	// o11y, actually not required to have the function continue to execute.
	go func() {
		if err := a.EventPublisher.Publish(ctx, evt); err != nil {
			logger.StdlibLogger(ctx).Error("erorr publishing sync checkpoint event", "error", err)
		}
	}()

	// We do upsertions of apps and functions in a goroutine in order to improve
	// the overall speed of the API endpoint.  The app and function IDs are deterministic
	// such that every new run from the same API endpoint produces the same IDs; therefore,
	// if this fails the next API request will upsert these and we will continue to make
	// the apps and runs once again.
	go a.upsertSyncData(ctx, auth, input)

	appID := input.AppID(auth.WorkspaceID())
	fn := input.Fn(appID)

	// Create a new run.  Note that this is currently of type API, and is a sync function.
	// Because of this, it has no job in the queue.
	//
	// We do this by inserting into the state store and adding a trace.  Note that API functions
	// SHOULD automatically have a timeout after 60 minutes;  we should auomatically ensure
	// that functions are marked as FAILED if we do not get a call to finalize them.
	md, err := a.Executor.Schedule(ctx, execution.ScheduleRequest{
		RunID:       &input.RunID,
		Function:    fn,
		AccountID:   auth.AccountID(),
		WorkspaceID: auth.WorkspaceID(),
		AppID:       input.AppID(auth.WorkspaceID()),
		RunMode:     enums.RunModeSync,
		Events:      []event.TrackedEvent{evt},
		URL:         input.URL(),
	})

	switch err {
	case nil:

		go a.m.OnFnScheduled(ctx, CheckpointMetrics{
			AccountID: auth.AccountID(),
			EnvID:     auth.WorkspaceID(),
			AppID:     appID,
			FnID:      fn.ID,
		})

		var jwt string
		if len(input.Steps) > 0 {
			jwt = a.checkpointSyncSteps(ctx, checkpointSyncSteps{
				RunID:     input.RunID,
				FnID:      fn.ID,
				AppID:     appID,
				AccountID: auth.AccountID(),
				EnvID:     auth.WorkspaceID(),
				Steps:     input.Steps,

				md: md,
			}, w)
		}

		_ = WriteResponse(w, CheckpointNewRunResponse{
			RunID: md.ID.RunID.String(),
			FnID:  fn.ID,
			AppID: appID,
			Token: jwt,
		})
		return
	case state.ErrIdentifierExists:
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(http.StatusConflict, "Run already exists"))
		return
	case executor.ErrFunctionRateLimited:
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, http.StatusTooManyRequests, "Rate limits exceeded"))
		return
	default:
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, http.StatusInternalServerError, "Failed to schedule run"))
		return
	}
}

// CheckpointSteps is called from the SDK to checkpoint steps in the background.  The SDK
// executes steps sequentially, then checkpoints once either an error is hit, the API finishes,
// or we need to transform the sync API function into an async queue-backed function (eg.
// step.waitForEvent, which cannot be resolved in the original API request easily.)
//
// This updates state and o11y around the executing steps.
func (a checkpointAPI) CheckpointSteps(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "Unauthorized"))
		return
	}

	// checkpoint those steps by writing to state.
	input := checkpointSyncSteps{}

	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "invalid request body: %s", err))
		return
	}

	input.AccountID = auth.AccountID()
	input.EnvID = auth.WorkspaceID()

	a.checkpointSyncSteps(ctx, input, w)
}

// checkpointSyncSteps handles the checkpointing of new steps.
//
// this accepts all opcodes in the current request, then handles trace pipelines and optional
// state updates in the state store for resumability.
//
// NOTE:  In order to power seamless APIs, we have to detect whether there are async ops in
// the first checkpoint.  If so, we produce a token which allows arbitrary users to request
// access to the run's output;  this token is used when redirecting in the sync fns that started
// the checkpoint.
func (a checkpointAPI) checkpointSyncSteps(ctx context.Context, input checkpointSyncSteps, w http.ResponseWriter) string {
	var jwt string

	if input.md == nil {
		md, err := a.State.LoadMetadata(ctx, sv2.ID{
			RunID:      input.RunID,
			FunctionID: input.FnID,
			Tenant: sv2.Tenant{
				AccountID: input.AccountID,
				EnvID:     input.EnvID,
				AppID:     input.AppID,
			},
		})
		if errors.Is(err, state.ErrRunNotFound) || errors.Is(err, sv2.ErrMetadataNotFound) {
			// Handle run not found with 404
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 404, "run not found"))
			return ""
		}
		if err != nil {
			logger.StdlibLogger(ctx).Error("error loading state for background checkpoint steps", "error", err)
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "error loading run state"))
			return ""
		}
		input.md = &md
	}

	l := logger.StdlibLogger(ctx).With("run_id", input.md.ID.RunID)

	// Load the function config.
	fn, err := a.fn(ctx, input.md.ID.FunctionID)
	if err != nil {
		logger.StdlibLogger(ctx).Warn("error loading fn for background checkpoint steps", "error", err)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "error loading function config"))
		return ""
	}

	runCtx := a.runContext(ctx, *input.md, fn)

	// If the opcodes contain a function finished op, we don't need to bother serializing
	// to the state store.  We only care about serializing state if we switch from sync -> async,
	// as the state will be used for resuming functions.
	complete := slices.ContainsFunc(input.Steps, func(s state.GeneratorOpcode) bool {
		return s.Op == enums.OpcodeRunComplete
	})

	async := slices.ContainsFunc(input.Steps, func(s state.GeneratorOpcode) bool {
		return enums.OpcodeIsAsync(s.Op)
	})

	if async {
		// Create a token that can be used for viewing this particular run.
		claim, err := apiv1auth.CreateRunJWT(
			a.runClaimsSecret,
			input.md.ID.Tenant.EnvID,
			input.md.ID.RunID,
		)
		if err != nil {
			logger.StdlibLogger(ctx).Warn("error creating run claim JWT for async token", "error", err)
		}
		jwt = claim
	}

	// Depending on the type of steps, we may end up switching the run from sync to async.  For example,
	// if the opcodes are sleeps, waitForEvents, inferences, etc. we will be resuming the API endpoint
	// at some point in the future.
	for _, op := range input.Steps {
		attrs := tracing.GeneratorAttrs(&op)
		tracing.AddMetadataTenantAttrs(attrs, input.md.ID)

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
				_, err := a.State.SaveStep(ctx, input.md.ID, op.ID, []byte(output))
				if errors.Is(err, state.ErrDuplicateResponse) || errors.Is(err, state.ErrIdempotentResponse) {
					// Ignore.
					l.Warn("duplicate checkpoint step", "id", input.md.ID)
					continue
				}
				if err != nil {
					l.Error("error saving checkpointed step state", "error", err)
				}
			}

			max := fn.MaxAttempts()
			_, err = a.TracerProvider.CreateSpan(
				tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
					Identifier:  input.md.ID,
					Attempt:     runCtx.AttemptCount(),
					MaxAttempts: &max,
				}),
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Parent:    tracing.RunSpanRefFromMetadata(input.md),
					StartTime: op.Timing.Start(),
					EndTime:   op.Timing.End(),
					Attributes: attrs.Merge(
						meta.NewAttrSet(
							meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(op.UserDefinedName())),
							meta.Attr(meta.Attrs.RunID, &input.RunID),
							meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(op.Timing.Start())),
							meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(op.Timing.Start())),
							meta.Attr(meta.Attrs.EndedAt, inngestgo.Ptr(op.Timing.End())),
						),
					),
				},
			)
			if err != nil {
				// We should never hit a blocker creating a span.  If so, warn loudly.
				l.Error("error saving span for checkpoint op", "error", err)
			}

			go a.m.OnStepFinished(ctx, CheckpointMetrics{
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
			_, err = a.TracerProvider.CreateSpan(
				tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
					Identifier:  input.md.ID,
					Attempt:     runCtx.AttemptCount(),
					MaxAttempts: &max,
				}),
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Parent:    tracing.RunSpanRefFromMetadata(input.md),
					StartTime: op.Timing.Start(),
					EndTime:   op.Timing.End(),
					Attributes: attrs.Merge(
						meta.NewAttrSet(
							meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(op.UserDefinedName())),
							meta.Attr(meta.Attrs.RunID, &input.RunID),
							meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(op.Timing.Start())),
							meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(op.Timing.Start())),
							meta.Attr(meta.Attrs.EndedAt, inngestgo.Ptr(op.Timing.End())),
							meta.Attr(meta.Attrs.DynamicStatus, &status),
						),
					),
				},
			)
			if err != nil {
				// We should never hit a blocker creating a span.  If so, warn loudly.
				l.Error("error saving span for checkpoint step error op", "error", err)
			}

			err := a.Executor.HandleGenerator(ctx, runCtx, op)
			if errors.Is(err, executor.ErrHandledStepError) {
				// In the executor, returning an error bubbles up to the queue to requeue.
				jobID := fmt.Sprintf("%s-%s-sync-retry", runCtx.Metadata().IdempotencyKey(), op.ID)
				now := time.Now()
				nextItem := queue.Item{
					JobID:                 &jobID,
					WorkspaceID:           runCtx.Metadata().ID.Tenant.EnvID,
					Kind:                  queue.KindEdge,
					Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()),
					PriorityFactor:        runCtx.PriorityFactor(),
					CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
					Attempt:               1, // This is now the next attempt.
					MaxAttempts:           runCtx.MaxAttempts(),
					Payload:               queue.PayloadEdge{Edge: inngest.SourceEdge}, // doesn't matter for sync functions.
					Metadata:              make(map[string]any),
					ParallelMode:          enums.ParallelModeWait,
				}

				// Continue checking this particular error.
				if err = a.Opts.Queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{}); err != nil {
					l.Error("error enqueueing step error in checkpoint", "error", err, "opcode", op.Op)
				}
			}
			if err != nil {
				l.Error("error handlign step error in checkpoint", "error", err, "opcode", op.Op)
			}

		case enums.OpcodeRunComplete:
			result := struct {
				Data APIResult `json:"data"`
			}{}
			if err := json.Unmarshal(op.Data, &result); err != nil {
				l.Error("error unmarshalling api result from sync RunComplete op", "error", err)
			}

			go a.m.OnFnFinished(ctx, CheckpointMetrics{
				AccountID: input.AccountID,
				EnvID:     input.EnvID,
				AppID:     input.AppID,
				FnID:      input.FnID,
			}, enums.RunStatusCompleted)

			// Call finalize and process the entire op.
			if err := a.finalize(ctx, *input.md, result.Data); err != nil {
				l.Error("error finalizing sync run", "error", err)
			}

		default:
			if err := a.Executor.HandleGenerator(ctx, runCtx, op); err != nil {
				l.Error("error handling generator in checkpoint", "error", err, "opcode", op.Op)
			}
		}
	}

	l.Info("handled sync checkpoint", "ops", len(input.Steps), "complete", complete)
	return jwt
}

func (a checkpointAPI) CheckpointAsyncSteps(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "Unauthorized"))
		return
	}

	// checkpoint those steps by writing to state.
	input := checkpointAsyncSteps{}
	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "invalid request body: %s", err))
		return
	}
	input.AccountID = auth.AccountID()
	input.EnvID = auth.WorkspaceID()

	l := logger.StdlibLogger(ctx).With(
		"run_id", input.RunID,
		"account_id", input.AccountID,
		"env_id", input.EnvID,
	)

	md, err := a.State.LoadMetadata(ctx, sv2.ID{
		RunID:      input.RunID,
		FunctionID: input.FnID,
		Tenant: sv2.Tenant{
			AccountID: input.AccountID,
			EnvID:     input.EnvID,
		},
	})
	if errors.Is(err, state.ErrRunNotFound) || errors.Is(err, sv2.ErrMetadataNotFound) {
		// Handle run not found with 404
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 404, "run not found"))
		return
	}
	if err != nil {
		l.Error("error loading state for background checkpoint steps", "error", err)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "error loading run state"))
		return
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
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "cannot checkpoint async steps"))
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

			_, err = a.State.SaveStep(ctx, md.ID, op.ID, []byte(output))
			if errors.Is(err, state.ErrDuplicateResponse) || errors.Is(err, state.ErrIdempotentResponse) {
				// Ignore.
				l.Warn("duplicate checkpoint step", "id", md.ID)
				continue
			}
			if err != nil {
				l.Error("error saving checkpointed step state", "error", err)
			}

			// TODO: add trace CTX data

			_, err = a.TracerProvider.CreateSpan(
				ctx,
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Parent:    md.Config.NewFunctionTrace(),
					StartTime: op.Timing.Start(),
					EndTime:   op.Timing.End(),
					Attributes: attrs.Merge(
						meta.NewAttrSet(
							meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(op.UserDefinedName())),
							meta.Attr(meta.Attrs.RunID, &input.RunID),
							meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(op.Timing.Start())),
							meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(op.Timing.Start())),
							meta.Attr(meta.Attrs.EndedAt, inngestgo.Ptr(op.Timing.End())),
						),
					),
				},
			)
			if err != nil {
				// We should never hit a blocker creating a span.  If so, warn loudly.
				l.Error("error saving span for checkpoint op", "error", err)
			}
		default:
			// Return an error
			l.Error("unimplemented checkpoint op", "op", op.Op)
			_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "cannot checkpoint opcode: %s", op.Op))
		}
	}

	// TODO: Reset the queue item!!!
}

// Output returns run output given a JWT which has access to the given {env, runID} claimset.
func (a checkpointAPI) Output(w http.ResponseWriter, r *http.Request) {
	// Assert that we have a checkpoint JWT in the query param.
	token := r.URL.Query().Get("token")

	claims, err := apiv1auth.VerifyRunJWT(r.Context(), a.runClaimsSecret, token)
	if err != nil || claims == nil {
		w.WriteHeader(401)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "Unable to find run with auth token"))
		return
	}

	if a.outputReader == nil {
		logger.StdlibLogger(r.Context()).Error("unable to fetch run output in checkpoint API")
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"status":"unknown","message":"unable to fetch run output"}`))
		return
	}

	until := time.Now().Add(CheckpointOutputWaitMax)

	for time.Now().Before(until) {
		output, err := a.outputReader.RunOutput(r.Context(), claims.Env, claims.RunID)

		if err == nil {
			// XXX: (tonyhb) add status code handling here.
			_, _ = w.Write(output)
			return
		}
		time.Sleep(CheckpointPollInterval)
	}

	w.Header().Set("content-type", "application/json")
	_, _ = w.Write([]byte(`{"status":"running","message":"run did not end within 5 minutes"}`))
}

// finalize finishes a run after receiving a RunComplete opcode.  This assumes that all prior
// work has finished, and eg. step.Defer items are not running.
func (a checkpointAPI) finalize(ctx context.Context, md sv2.Metadata, result APIResult) error {
	httpHeader := http.Header{}
	for k, v := range result.Headers {
		httpHeader[k] = []string{v}
	}

	return a.Executor.Finalize(ctx, execution.FinalizeOpts{
		Metadata: md,
		RunMode:  enums.RunModeSync,
		Response: state.DriverResponse{
			Output:     result.Body,
			Header:     httpHeader,
			StatusCode: result.StatusCode,
		},
		Optional: execution.FinalizeOptional{},
	})
}

// upsertSyncData adds apps and functions to the backing datastore the first time
// that an API is called.  This is called every time a sync fn is invoked, and
// we memoize this call via an in-memory map.  Because our API is deployed many times
// over and this is in-memory, we must make these upserts and not insert N times.
func (a checkpointAPI) upsertSyncData(ctx context.Context, auth apiv1auth.V1Auth, input *CheckpointNewRunRequest) {
	envID := auth.WorkspaceID()
	appID := input.AppID(envID)
	fnID := input.FnID(appID)

	config := input.FnConfig(auth.WorkspaceID())

	if item := a.upserted.Get(fnID.String()); item != nil {
		// This item is in the LRU cache and has already been inserted;
		// don't bother to upsert the app and only update the function configuration.
		if util.XXHash(config) == item.Value().(string) {
			return
		}
		_, err := a.FunctionCreator.UpdateFunctionConfig(ctx, cqrs.UpdateFunctionConfigParams{
			Config: config,
			ID:     fnID,
		})
		if err != nil {
			logger.StdlibLogger(ctx).Error("failed to update fn config",
				"error", err,
				"app_id", input.AppID(auth.WorkspaceID()),
			)
		}
		return
	}

	app, err := a.AppCreator.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:     appID,
		Name:   input.AppSlug(),
		Url:    input.AppURL(),
		Method: enums.AppMethodAPI.String(),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to upsert app",
			"error", err,
			"app_id", input.AppID(auth.WorkspaceID()),
		)
		return
	}

	// We may have already added this function, so check if it exists prior to inserting and updating.
	// XXX: It would be good to add an upsert method to the function CQRS layer.
	fn, err := a.FunctionReader.GetFunctionByInternalUUID(ctx, fnID)
	if err == nil && fn != nil {
		if string(fn.Config) == config {
			return // no need to update
		}
		_, err = a.FunctionCreator.UpdateFunctionConfig(ctx, cqrs.UpdateFunctionConfigParams{
			Config:    config,
			ID:        fnID,
			AccountID: auth.AccountID(),
			EnvID:     auth.WorkspaceID(),
			AppID:     app.ID,
		})
		if err != nil {
			logger.StdlibLogger(ctx).Error("failed to update fn config",
				"error", err,
				"app_id", input.AppID(auth.WorkspaceID()),
			)
		}
		return
	}

	_, err = a.FunctionCreator.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID:        fnID,
		AccountID: auth.AccountID(),
		EnvID:     auth.WorkspaceID(),
		AppID:     app.ID,
		Name:      input.FnSlug(),
		Slug:      input.FnSlug(),
		Config:    config,
		CreatedAt: time.UnixMilli(input.Event.Timestamp),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to insert function",
			"error", err,
			"function_id", input.FnID(input.AppID(auth.WorkspaceID())),
			"app_id", app.ID,
		)
		return
	}

	// Hash the config so that we can quickly compare whether we upsert when memoizing
	// this upsert routine.
	a.upserted.Set(fnID.String(), util.XXHash(config), time.Hour*24)

	logger.StdlibLogger(ctx).Debug("upserted fn",
		"error", err,
		"function_id", input.FnID(input.AppID(auth.WorkspaceID())),
		"app_id", app.ID,
	)
}

func (a checkpointAPI) runContext(ctx context.Context, md sv2.Metadata, fn *inngest.Function) execution.RunContext {
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

func (a checkpointAPI) fn(ctx context.Context, fnID uuid.UUID) (*inngest.Function, error) {
	// Load the function config.
	cfn, err := a.Opts.FunctionReader.GetFunctionByInternalUUID(ctx, fnID)
	if err != nil {
		return nil, fmt.Errorf("error loading function: %w", err)
	}
	return cfn.InngestFunction()
}
