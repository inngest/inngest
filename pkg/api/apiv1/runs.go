package apiv1

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
)

// GetEventRuns returns function runs given an event ID.
func (a router) GetFunctionRun(w http.ResponseWriter, r *http.Request) {
	if a.opts.RateLimited(r, w, "/v1/runs/{runID}") {
		return
	}

	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	runID, err := ulid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid run ID: %s", chi.URLParam(r, "runID")))
		return
	}

	fr, err := a.opts.TraceReader.GetRun(ctx, runID, auth.AccountID(), auth.WorkspaceID())
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load function run: %s", chi.URLParam(r, "runID")))
		return
	}

	// Update the cache every 3 seconds to prevent stale displays
	_ = WriteCachedResponse(w, fr, 3*time.Second)
}

// CancelFunctionRun cancels a function run.
func (a API) CancelFunctionRun(ctx context.Context, runID ulid.ULID) error {
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		return publicerr.Wrap(err, 401, "No auth found")
	}

	fr, err := a.opts.TraceReader.GetRun(ctx, runID, auth.AccountID(), auth.WorkspaceID())
	if err != nil {
		return publicerr.Wrapf(err, 404, "Unable to load function run: %s", runID)
	}
	if fr.WorkspaceID != auth.WorkspaceID() {
		return publicerr.Wrapf(err, 404, "Unable to load function run: %s", runID)
	}

	id := state.ID{
		RunID:      runID,
		FunctionID: fr.FunctionID,
		Tenant: state.Tenant{
			// TODO: AppID is missing
			EnvID:     auth.WorkspaceID(),
			AccountID: auth.AccountID(),
		},
	}
	if err := a.opts.Executor.Cancel(ctx, id, execution.CancelRequest{}); err != nil {
		return publicerr.Wrapf(err, 500, "Unable to cancel function run: %s", err)
	}
	return nil
}

func (a router) cancelFunctionRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	runID, err := ulid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid run ID: %s", chi.URLParam(r, "runID")))
		return
	}
	if err := a.CancelFunctionRun(ctx, runID); err != nil {
		_ = publicerr.WriteHTTP(w, err)
	}
}

func (a router) GetFunctionRunJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, err := a.opts.AuthFinder(ctx)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "No auth found"))
		return
	}

	runID, err := ulid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid run ID: %s", chi.URLParam(r, "runID")))
		return
	}

	fr, err := a.opts.TraceReader.GetRun(ctx, runID, auth.AccountID(), auth.WorkspaceID())
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load function run: %s", chi.URLParam(r, "runID")))
		return
	}

	shard, err := a.opts.QueueShardSelector(ctx, auth.AccountID(), nil)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Internal server error"))
		return
	}

	jobs, err := a.opts.JobQueueReader.RunJobs(
		ctx,
		shard.Name,
		auth.WorkspaceID(),
		fr.FunctionID,
		runID,
		10,
		0,
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to read run jobs: %s", err))
		return
	}
	if jobs == nil {
		jobs = []queue.JobResponse{}
	}

	_ = WriteCachedResponse(w, jobs, 5*time.Second)
}
