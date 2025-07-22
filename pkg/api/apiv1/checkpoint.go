package apiv1

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

	evt := runEvent(*input)

	// Publish the event in a goroutine to lower latency in the API.  This is, while extremely important for
	// o11y, actually not required to have the function continue to execute.
	go func() {
		a.EventPublisher.Publish(ctx, event.InternalEvent{
			ID:          ulid.MustNew(ulid.Now(), rand.Reader),
			AccountID:   auth.AccountID(),
			WorkspaceID: auth.WorkspaceID(),
			Event:       evt,
		})
	}()

	// We do upsertions of apps and functions in a goroutine in order to improve
	// the overall speed of the API endpoint.  The app and function IDs are deterministic
	// such that every new run from the same API endpoint produces the same IDs; therefore,
	// if this fails the next API request will upsert these and we will continue to make
	// the apps and runs once again.
	go func() {
		// Create the app, if it doesn't exist.
		app, err := a.AppCreator.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:     input.AppID(auth.WorkspaceID()),
			Name:   input.AppSlug(),
			Url:    input.AppURL(),
			Method: enums.AppMethodAPI.String(),
		})
		if err != nil {
			logger.StdlibLogger(ctx).Error("failed to upsert app",
				"error", err,
				"path", r.URL.Path,
				"app_id", input.AppID(auth.WorkspaceID()),
				"app_slug", input.AppSlug(),
			)
			return
		}

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
				"path", r.URL.Path,
				"function_id", input.FnID(input.AppID(auth.WorkspaceID())),
				"function_slug", input.FnSlug(),
				"app_id", app.ID,
			)
			return
		}
	}()

	appID := input.AppID(auth.WorkspaceID())
	fn := input.Fn(appID)

	// Create a new run.  Note that this is currently of type API, and is a sync function.
	// Because of this, it has no job in the queue.
	//
	// We do this by inserting into the state store and adding a trace.  Note that API functions
	// SHOULD automatically have a timeout after 60 minutes.
	md, err := a.Executor.Schedule(ctx, execution.ScheduleRequest{
		RunID:       &input.RunID,
		Function:    fn,
		AccountID:   auth.AccountID(),
		WorkspaceID: auth.WorkspaceID(),
		AppID:       input.AppID(auth.WorkspaceID()),
		RunMode:     enums.RunModeSync,
		Events: []event.TrackedEvent{
			event.InternalEvent{
				ID:          ulid.MustNew(ulid.Now(), rand.Reader),
				AccountID:   auth.AccountID(),
				WorkspaceID: auth.WorkspaceID(),
				Event:       evt,
			},
		},
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
		// log
		return
	}

	// Depending on the type of steps, we may end up switching the run from sync to async.  For example,
	// if the opcodes are sleeps, waitForEvents, inferences, etc. we will be resuming the API endpoint
	// at some point in the future.
	for _, op := range input.Steps {
		switch op.Op {
		case enums.OpcodeStepRun, enums.OpcodeStep:
			ref, err := a.TracerProvider.CreateSpan(
				meta.SpanNameStep,
				&tracing.CreateSpanOptions{
					Parent: md.Config.NewFunctionTrace(),
					Attributes: meta.NewAttrSet(
						meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(op.UserDefinedName())),
						meta.Attr(meta.Attrs.RunID, &input.RunID),
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

func (a checkpointAPI) CheckpointResponse(w http.ResponseWriter, r *http.Request) {
	// Finalize the run by storing the response
}
