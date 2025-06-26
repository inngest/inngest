package apiv1

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
)

type CheckpointOpts struct {
	AppCreator      cqrs.AppCreator
	FunctionCreator cqrs.FunctionCreator
}

type checkpointAPI struct {
	Opts
	CheckpointOpts
}

func (a checkpointAPI) CheckpointNewRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	input := &CheckpointNewRunRequest{}
	err := json.NewDecoder(io.LimitReader(r.Body, consts.MaxStepInputSize)).Decode(input)
	if err != nil {
		// TODO: Return error
		return
	}

	// Create the app, if it doesn't exist.
	app, err := a.AppCreator.UpsertApp(ctx, cqrs.UpsertAppParams{})
	if err != nil {
		// TODO: Return error
		return
	}

	// TODO: Create the function, if it doesn't exist.
	fn, err := a.FunctionCreator.InsertFunction(ctx, cqrs.InsertFunctionParams{
		AppID:     app.ID,
		Name:      input.FnSlug(),
		Slug:      input.FnSlug(),
		Config:    input.FnConfig(),
		CreatedAt: time.UnixMilli(input.Event.Timestamp),
	})
	if err != nil {
		// TODO: Return error
		return
	}

	// Create a new run.  Note that this is currently of type API, and is a sync function.
	// Because of this, it has no job in the queue.
	//
	// We do this by inserting into the state store and adding a trace.  Note that API functions
	// SHOULD automatically have a timeout after 60 minutes.
	//
	// TODO: Schedule as sync.
	smv2, err := a.Executor.Schedule(ctx, execution.ScheduleRequest{})
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
