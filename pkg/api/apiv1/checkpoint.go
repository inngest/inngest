package apiv1

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
)

type CheckpointOpts struct {
	AppCreator      cqrs.AppCreator
	FunctionCreator cqrs.FunctionCreator
	EventPublisher  event.Publisher
}

type checkpointAPI struct {
	Opts
	CheckpointOpts
}

func (a checkpointAPI) CheckpointNewRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.AuthFinder(ctx)
	if err != nil {
		// TODO: Return HTTP error
		return
	}

	input := &CheckpointNewRunRequest{}
	if err = json.NewDecoder(io.LimitReader(r.Body, consts.MaxStepInputSize)).Decode(input); err != nil {
		// TODO: Return HTTP error
		return
	}

	// Create the app, if it doesn't exist.
	app, err := a.AppCreator.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:     input.AppID(auth.WorkspaceID()),
		Name:   input.AppSlug(),
		Url:    input.AppURL(),
		Method: enums.AppMethodAPI.String(),
	})
	if err != nil {
		// TODO: Return HTTP error
		return
	}

	// TODO: Create the function, if it doesn't exist.
	fn, err := a.FunctionCreator.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID:        input.FnID(auth.WorkspaceID()),
		AccountID: auth.AccountID(),
		EnvID:     auth.WorkspaceID(),
		AppID:     app.ID,
		Name:      input.FnSlug(),
		Slug:      input.FnSlug(),
		Config:    input.FnConfig(),
		CreatedAt: time.UnixMilli(input.Event.Timestamp),
	})
	if err != nil {
		// TODO: Return HTTP error
		return
	}

	ifn, err := fn.InngestFunction()
	if err != nil {
		// TODO: Return HTTP error
		return
	}

	// TODO: Publish the event in a goroutine.

	// Create a new run.  Note that this is currently of type API, and is a sync function.
	// Because of this, it has no job in the queue.
	//
	// We do this by inserting into the state store and adding a trace.  Note that API functions
	// SHOULD automatically have a timeout after 60 minutes.
	meta, err := a.Executor.Schedule(ctx, execution.ScheduleRequest{
		Function:    *ifn,
		AccountID:   auth.AccountID(),
		WorkspaceID: auth.WorkspaceID(),
		AppID:       app.ID,
		RunMode:     enums.RunModeSync,
		Events:      []event.TrackedEvent{},
	})

	switch err {
	case nil:
		// TODO: Return success
		_ = meta
		return
	case state.ErrIdentifierExists:
		// TODO: If run already exists due to idemptoency, issue error response, allowing the
		// SDK to redirect to the previous run.
		return
	default:
		// TODO: return HTTP error
	}
}

func (a checkpointAPI) CheckpointSteps(w http.ResponseWriter, r *http.Request) {
	// checkpoint those steps by writing to state.

	// Depending on the type of steps, we may end up switching the run from sync to async.  For example,
	// if the opcodes are sleeps, waitForEvents, inferences, etc. we will be resuming the API endpoint
	// at some point in the future.
}

func (a checkpointAPI) CheckpointResponse(w http.ResponseWriter, r *http.Request) {
	// Finalize the run by storing the response
}
