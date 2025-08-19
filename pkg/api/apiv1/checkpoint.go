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
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngestgo"
	"github.com/karlseguin/ccache/v2"
	"github.com/oklog/ulid/v2"
)

const CheckpointRoutePrefix = "/http/runs"

type CheckpointAPI interface {
	// http.Handler allows the checkpoint API to be mounted.
	http.Handler

	CheckpointNewRun(w http.ResponseWriter, r *http.Request)
	CheckpointSteps(w http.ResponseWriter, r *http.Request)
	CheckpointResponse(w http.ResponseWriter, r *http.Request)
}

type checkpointAPI struct {
	Opts

	chi.Router

	// upserted tracks fn IDs and their associated config in memory once upserted, allowing
	// us to prevent DB queries from hitting the DB each time a sync fn begins.
	upserted *ccache.Cache
}

func NewCheckpointAPI(o Opts) CheckpointAPI {
	api := checkpointAPI{
		Router:   chi.NewRouter(),
		Opts:     o,
		upserted: ccache.New(ccache.Configure().MaxSize(10_000)),
	}
	api.Post("/", api.CheckpointNewRun)
	api.Post("/{runID}/steps", api.CheckpointSteps)
	api.Post("/{runID}/response", api.CheckpointResponse)

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
	})

	switch err {
	case nil:
		_ = WriteResponse(w, CheckpointNewRunResponse{
			RunID: md.ID.RunID.String(),
			FnID:  fn.ID,
			AppID: appID,
		})
		return
	case state.ErrIdentifierExists:
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(409, "Run already exists"))
		return
	default:
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "Failed to schedule run"))
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
	input := struct {
		RunID ulid.ULID               `json:"run_id"`
		FnID  uuid.UUID               `json:"fn_id"`
		AppID uuid.UUID               `json:"app_id"`
		Steps []state.GeneratorOpcode `json:"steps"`
	}{}

	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "invalid request body: %s", err))
		return
	}

	md, err := a.State.LoadMetadata(r.Context(), sv2.ID{
		RunID:      input.RunID,
		FunctionID: input.FnID,
		Tenant: sv2.Tenant{
			AccountID: auth.AccountID(),
			EnvID:     auth.WorkspaceID(),
			AppID:     input.AppID,
		},
	})
	if errors.Is(err, state.ErrRunNotFound) || errors.Is(err, sv2.ErrMetadataNotFound) {
		// Handle run not found with 404
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 404, "run not found"))
		return
	}
	if err != nil {
		logger.StdlibLogger(ctx).Warn("error loading state for background checkpoint steps", "error", err)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "error loading run state"))
		return
	}

	l := logger.StdlibLogger(ctx).With("run_id", md.ID.RunID)

	// If the opcodes contain a function finished op, we don't need to bother serializing
	// to the state store.  We only care about serializing state if we switch from sync -> async,
	// as the state will be used for resuming functions.
	complete := slices.ContainsFunc(input.Steps, func(s state.GeneratorOpcode) bool {
		return s.Op == enums.OpcodeRunComplete
	})

	// Depending on the type of steps, we may end up switching the run from sync to async.  For example,
	// if the opcodes are sleeps, waitForEvents, inferences, etc. we will be resuming the API endpoint
	// at some point in the future.
	for _, op := range input.Steps {
		attrs := tracing.GeneratorAttrs(&op)

		switch op.Op {
		case enums.OpcodeStepRun, enums.OpcodeStep:
			output, err := op.Output()
			if err != nil {
				l.Error("error fetching checkpoint step output", "error", err)
			}

			if !complete {
				// Checkpointing happens in this API when either the function finishes or we move to
				// async.  Therefore, we onl want to save state if we don't have a complete opcode,
				// as all complete functions will never re-enter.
				if _, err := a.State.SaveStep(ctx, md.ID, op.ID, []byte(output)); err != nil {
					l.Error("error saving checkpointed step state", "error", err)
				}
			}

			_, err = a.TracerProvider.CreateSpan(
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Parent:    md.Config.NewFunctionTrace(),
					StartTime: op.Timing.Start(),
					EndTime:   op.Timing.End(),
					Attributes: attrs.Merge(meta.NewAttrSet(
						meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(op.UserDefinedName())),
						meta.Attr(meta.Attrs.RunID, &input.RunID),
						meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(op.Timing.Start())),
						meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(op.Timing.Start())),
						meta.Attr(meta.Attrs.EndedAt, inngestgo.Ptr(op.Timing.End())),
					)),
				},
			)
			if err != nil {
				// We should never hit a blocker creating a span.  If so, warn loudly.
				l.Error("error saving span for checkpoint op", "error", err)
			}

		case enums.OpcodeRunComplete:
			result := APIResult{}
			if err := json.Unmarshal(op.Data, &result); err != nil {
				l.Error("error unmarshalling api result from sync RunComplete op", "error", err)
			}
			if err := a.finalize(ctx, md, result); err != nil {
				l.Error("error finalizing sync run", "error", err)
			}

		default:
			// Create HTTP client using exechttp.Client
			client := exechttp.Client(exechttp.SecureDialerOpts{})
			httpClient := &client

			// Create the run context with simplified data
			runCtx := &checkpointRunContext{
				md:              md,
				httpClient:      httpClient,
				events:          []json.RawMessage{}, // Empty for checkpoint context
				groupID:         uuid.New().String(),
				attemptCount:    0,
				maxAttempts:     3,                           // Default retry count
				priorityFactor:  nil,                         // Use default priority
				concurrencyKeys: []state.CustomConcurrency{}, // No custom concurrency
				parallelMode:    enums.ParallelModeWait,      // Default to serial
			}

			if err := a.Executor.HandleGenerator(ctx, runCtx, op); err != nil {
				l.Error("error handling generator in checkpoint", "error", err, "opcode", op.Op)
			}
		}
	}

	l.Info("handled sync checkpoint", "ops", len(input.Steps), "complete", complete)
}

// CheckpointResponse is called from the SDK when the API responds to the user. This indicates
// that the API-based function has finished computing.
//
// Note that some steps may be deferred to the background via a future `step.defer` API call.
// This implies that the API handler has finished but the function has NOT yet finished, if
// thre are hybrid async and sync steps running.
//
// The following is true:
//
// - If the run mode is Sync, this means that the function has finished
// - If the run mode is Async, we only finish the function once all pending steps have finished
func (a checkpointAPI) CheckpointResponse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "Unauthorized"))
		return
	}

	input := struct {
		RunID  ulid.ULID `json:"run_id"`
		FnID   uuid.UUID `json:"fn_id"`
		AppID  uuid.UUID `json:"app_id"`
		Result APIResult `json:"result"`
	}{}

	// Load the state from the state store.
	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid request body"))
		return
	}

	md, err := a.State.LoadMetadata(r.Context(), sv2.ID{
		RunID:      input.RunID,
		FunctionID: input.FnID,
		Tenant: sv2.Tenant{
			AccountID: auth.AccountID(),
			EnvID:     auth.WorkspaceID(),
			AppID:     input.AppID,
		},
	})
	// TODO: Handle run not found with 404
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error loading state in sync fn finalize",
			"error", err,
			"run_id", md.ID.RunID.String(),
		)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "error loading state"))
		return
	}

	if err := a.finalize(ctx, md, input.Result); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "error finalizing run"))
		return
	}
}

// finalize finishes a run after receiving a RunComplete opcode.  This assumes that all prior
// work has finished, and eg. step.Defer items are not running.
func (a checkpointAPI) finalize(ctx context.Context, md sv2.Metadata, result APIResult) error {
	err := a.TracerProvider.UpdateSpan(&tracing.UpdateSpanOptions{
		Metadata:   &md,
		TargetSpan: tracing.RunSpanRefFromMetadata(&md),
		EndTime:    md.ID.RunID.Timestamp().Add(result.Duration),
		Status:     enums.StepStatusCompleted, // Optionally set a status for the span
		Attributes: meta.NewAttrSet(),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error finalizing sync api span",
			"error", err,
			"run_id", md.ID.RunID.String(),
		)
		return fmt.Errorf("error updating span: %w", err)
	}
	_, err = a.State.Delete(ctx, md.ID)
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error deleting state in finalize",
			"error", err,
			"run_id", md.ID.RunID.String(),
		)
	}
	return nil
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
