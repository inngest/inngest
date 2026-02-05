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
	"github.com/inngest/inngest/pkg/execution/apiresult"
	"github.com/inngest/inngest/pkg/execution/checkpoint"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
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
	CheckpointMetrics checkpoint.MetricsProvider
	// RunOutputReader is the reader used to fetch run outputs for checkpoint APIs.
	RunOutputReader RunOutputReader
	// RunJWTSecret is the secret for signing run claim JWTs, allowing sync APIs
	// to redirect to an API endpoint that fetches outputs for a specific run.
	RunJWTSecret []byte
}

// checkpointAPI is the base implementation.
type checkpointAPI struct {
	// API opts
	Opts
	chi.Router

	checkpointer checkpoint.Checkpointer

	// upserted tracks fn IDs and their associated config in memory once upserted, allowing
	// us to prevent DB queries from hitting the DB each time a sync fn begins.
	upserted *ccache.Cache
	// runClaimsSecret is the secret for creating run claims JWTs
	runClaimsSecret []byte
	// outputReader allows us to read run output for a given env / run ID
	outputReader RunOutputReader
}

func NewCheckpointAPI(o Opts) CheckpointAPI {
	c := checkpoint.New(checkpoint.Opts{
		State:           o.State,
		FnReader:        o.FunctionReader,
		Executor:        o.Executor,
		TracerProvider:  o.TracerProvider,
		Queue:           o.Queue,
		MetricsProvider: o.CheckpointOpts.CheckpointMetrics,
	})

	api := checkpointAPI{
		Router:          chi.NewRouter(),
		Opts:            o,
		upserted:        ccache.New(ccache.Configure().MaxSize(10_000)),
		runClaimsSecret: o.CheckpointOpts.RunJWTSecret,
		outputReader:    o.CheckpointOpts.RunOutputReader,
		checkpointer:    c,
	}

	api.Post("/", api.CheckpointNewRun)             // sync, API-based fns
	api.Post("/{runID}/steps", api.CheckpointSteps) // sync, API-based fns
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
	ctx := context.WithoutCancel(r.Context())
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

	metrics.IncrExecutorScheduleCount(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"type":   "api_checkpoint",
			"status": executor.ScheduleStatus(err),
		},
	})

	if err != nil {
		switch err {
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

	go a.checkpointer.Metrics().OnFnScheduled(ctx, checkpoint.MetricCardinality{
		AccountID: auth.AccountID(),
		EnvID:     auth.WorkspaceID(),
		AppID:     appID,
		FnID:      fn.ID,
	})

	var jwt string
	if len(input.Steps) > 0 {
		err := a.checkpointer.CheckpointSyncSteps(ctx, checkpoint.SyncCheckpoint{
			RunID:     input.RunID,
			FnID:      fn.ID,
			AppID:     appID,
			AccountID: auth.AccountID(),
			EnvID:     auth.WorkspaceID(),
			Steps:     input.Steps,
			Metadata:  md,
		})
		if err != nil {
			logger.StdlibLogger(ctx).Error("error checkpointing sync steps", "error", err)
		}
		// Create a token that can be used for viewing this particular run.
		claim, err := apiv1auth.CreateRunJWT(
			a.runClaimsSecret,
			md.ID.Tenant.EnvID,
			md.ID.RunID,
		)
		if err != nil {
			logger.StdlibLogger(ctx).Warn("error creating run claim JWT", "error", err)
		}
		jwt = claim

	}

	_ = WriteResponse(w, CheckpointNewRunResponse{
		RunID: md.ID.RunID.String(),
		FnID:  fn.ID,
		AppID: appID,
		Token: jwt,
	})
}

// CheckpointSteps is called from the SDK to checkpoint steps in the background.  The SDK
// executes steps sequentially, then checkpoints once either an error is hit, the API finishes,
// or we need to transform the sync API function into an async queue-backed function (eg.
// step.waitForEvent, which cannot be resolved in the original API request easily.)
//
// This updates state and o11y around the executing steps.
func (a checkpointAPI) CheckpointSteps(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithoutCancel(r.Context())
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

	err = a.checkpointer.CheckpointSyncSteps(ctx, checkpoint.SyncCheckpoint{
		RunID:     input.RunID,
		FnID:      input.FnID,
		AppID:     input.AppID,
		AccountID: auth.AccountID(),
		EnvID:     auth.WorkspaceID(),
		Steps:     input.Steps,
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error checkpointing sync steps", "error", err)
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
func (a checkpointAPI) CheckpointAsyncSteps(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithoutCancel(r.Context())
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

	input.TrackLatency(ctx)

	err = a.checkpointer.CheckpointAsyncSteps(ctx, checkpoint.AsyncCheckpoint{
		RunID:        input.RunID,
		FnID:         input.FnID,
		Steps:        input.Steps,
		QueueItemRef: input.QueueItemRef,
		AccountID:    auth.AccountID(),
		EnvID:        auth.WorkspaceID(),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error checkpointing async steps", "error", err)
		_ = publicerr.WriteHTTP(w, publicerr.Error{
			Message: "Failed to checkpoint steps",
			Data: map[string]any{
				"run_id": input.RunID,
			},
			Status: 400,
		})
	}
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
		if err != nil {
			time.Sleep(CheckpointPollInterval)
			continue
		}

		if len(output) == 0 {
			// XXX: should never happen as the APIResult struct is marshalled.
			return
		}

		// We expect that the output is a wrapped {"data":APIResult} type.
		// Try to parse it and extract the status code, headers, and body.
		// If parsing fails or the status code is invalid, fall back to writing the raw output.
		res := struct {
			Data apiresult.APIResult `json:"data"`
		}{}
		if err := json.Unmarshal(output, &res); err == nil && res.Data.StatusCode > 0 {
			for k, v := range res.Data.Headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(res.Data.StatusCode)
			_, _ = w.Write(res.Data.Body)
			return
		}

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write(output)
		return
	}

	_, _ = w.Write([]byte(`{"status":"running","message":"run did not end within 5 minutes"}`))
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
