package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
)

// GetEventRuns returns function runs given an event ID.
func (a api) GetFunctionRun(w http.ResponseWriter, r *http.Request) {
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

	fr, err := a.opts.FunctionRunReader.GetFunctionRun(ctx, auth.AccountID(), auth.WorkspaceID(), runID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load function run: %s", chi.URLParam(r, "runID")))
		return
	}

	_ = json.NewEncoder(w).Encode(fr)
}

// CancelFunctionRun cancels a function run.
func (a api) CancelFunctionRun(w http.ResponseWriter, r *http.Request) {
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

	fr, err := a.opts.FunctionRunReader.GetFunctionRun(
		ctx,
		auth.AccountID(),
		auth.WorkspaceID(),
		runID,
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 404, "Unable to load function run: %s", chi.URLParam(r, "runID")))
		return
	}

	if fr.WorkspaceID != auth.WorkspaceID() {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 404, "Unable to load function run: %s", chi.URLParam(r, "runID")))
		return
	}

	if err := a.opts.Executor.Cancel(ctx, runID, execution.CancelRequest{}); err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to cancel function run: %s", err))
		return
	}
}

func (a api) GetFunctionRunJobs(w http.ResponseWriter, r *http.Request) {
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

	fr, err := a.opts.FunctionRunReader.GetFunctionRun(
		ctx,
		auth.AccountID(),
		auth.WorkspaceID(),
		runID,
	)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load function run: %s", chi.URLParam(r, "runID")))
		return
	}

	jobs, err := a.opts.JobQueueReader.RunJobs(
		ctx,
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
	_ = json.NewEncoder(w).Encode(jobs)
}
