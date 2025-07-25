package apiv1

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngestgo"
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
}

func NewCheckpointAPI(o Opts) CheckpointAPI {
	api := checkpointAPI{
		Router: chi.NewRouter(),
		Opts:   o,
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
		a.EventPublisher.Publish(ctx, evt)
	}()

	// We do upsertions of apps and functions in a goroutine in order to improve
	// the overall speed of the API endpoint.  The app and function IDs are deterministic
	// such that every new run from the same API endpoint produces the same IDs; therefore,
	// if this fails the next API request will upsert these and we will continue to make
	// the apps and runs once again.
	go a.upsertData(ctx, auth, input)

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
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "error loading state"))
		// TODO: log
		return
	}

	// Depending on the type of steps, we may end up switching the run from sync to async.  For example,
	// if the opcodes are sleeps, waitForEvents, inferences, etc. we will be resuming the API endpoint
	// at some point in the future.
	for _, op := range input.Steps {
		switch op.Op {
		case enums.OpcodeStepRun, enums.OpcodeStep:
			// TODO:Save steps to state store for future retries.

			ref, err := a.TracerProvider.CreateSpan(
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Parent:    md.Config.NewFunctionTrace(),
					StartTime: op.Timing.Start(),
					EndTime:   op.Timing.End(),
					Attributes: meta.NewAttrSet(
						meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(op.UserDefinedName())),
						meta.Attr(meta.Attrs.RunID, &input.RunID),
						meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(op.Timing.Start())),
						meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(op.Timing.Start())),
						meta.Attr(meta.Attrs.EndedAt, inngestgo.Ptr(op.Timing.End())),
					),
				},
			)
			_, _ = ref, err
		default:
			// This is now async.  For now, do NOT allow this.
			panic("TODO")
		}
	}
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
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "error loading state"))
		// TODO: Log
		return
	}

	// TODO: If the run mode is async (due to background steps, or a switch with waits)
	// we need to ensure that we do not finish the run.  Finishing will be managed via the
	// regular async executor.

	err = a.TracerProvider.UpdateSpan(&tracing.UpdateSpanOptions{
		Metadata:   &md,
		TargetSpan: tracing.RunSpanRefFromMetadata(&md),
		EndTime:    md.ID.RunID.Timestamp().Add(input.Result.Duration),
		Status:     enums.StepStatusCompleted, // Optionally set a status for the span
		Attributes: meta.NewAttrSet(),
	})
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 500, "error updating state"))
		// TODO: Log
		return
	}

	_, err = a.State.Delete(ctx, md.ID)
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error deleting state in finalize",
			"error", err,
			"run_id", md.ID.RunID.String(),
		)
	}
}

func (a checkpointAPI) upsertData(ctx context.Context, auth apiv1auth.V1Auth, input *CheckpointNewRunRequest) {
	app, err := a.AppCreator.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:     input.AppID(auth.WorkspaceID()),
		Name:   input.AppSlug(),
		Url:    input.AppURL(),
		Method: enums.AppMethodAPI.String(),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to upsert app",
			"error", err,
			"app_id", input.AppID(auth.WorkspaceID()),
			"app_slug", input.AppSlug(),
		)
		return
	}

	// TODO: Upsert function.

	_, err = a.FunctionCreator.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID:        input.FnID(input.AppID(auth.WorkspaceID())),
		AccountID: auth.AccountID(),
		EnvID:     auth.WorkspaceID(),
		AppID:     app.ID,
		Name:      input.FnSlug(),
		Slug:      input.FnSlug(),
		Config:    input.FnConfig(auth.WorkspaceID()),
		CreatedAt: time.UnixMilli(input.Event.Timestamp),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to insert function",
			"error", err,
			"function_id", input.FnID(input.AppID(auth.WorkspaceID())),
			"function_slug", input.FnSlug(),
			"app_id", app.ID,
		)
		return
	}
}
