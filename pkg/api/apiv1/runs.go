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
	wsID := a.opts.WorkspaceFinder(ctx)

	runID, err := ulid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid run ID: %s", chi.URLParam(r, "runID")))
		return
	}

	fr, err := a.opts.FunctionRunReader.GetFunctionRun(ctx, wsID, runID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load function run: %s", chi.URLParam(r, "runID")))
		return
	}

	if fr.Result == nil {
		finish, err := a.opts.FunctionRunReader.GetFunctionRunFinishesByRunIDs(ctx, []ulid.ULID{runID})
		if err == nil && len(finish) == 1 {
			fr.Result = finish[0]
		}
	}

	_ = json.NewEncoder(w).Encode(fr)
}

// CancelFunctionRun cancels a function run.
func (a api) CancelFunctionRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID := a.opts.WorkspaceFinder(ctx)

	runID, err := ulid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid run ID: %s", chi.URLParam(r, "runID")))
		return
	}

	fr, err := a.opts.FunctionRunReader.GetFunctionRun(ctx, wsID, runID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 404, "Unable to load function run: %s", chi.URLParam(r, "runID")))
		return
	}

	if fr.WorkspaceID != wsID {
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
	wsID := a.opts.WorkspaceFinder(ctx)

	runID, err := ulid.Parse(chi.URLParam(r, "runID"))
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 400, "Invalid run ID: %s", chi.URLParam(r, "runID")))
		return
	}

	fr, err := a.opts.FunctionRunReader.GetFunctionRun(ctx, wsID, runID)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to load function run: %s", chi.URLParam(r, "runID")))
		return
	}

	jobs, err := a.opts.JobQueueReader.RunJobs(ctx, wsID, fr.FunctionID, runID, 10, 0)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrapf(err, 500, "Unable to read run jobs: %s", err))
		return
	}
	if jobs == nil {
		jobs = []queue.JobResponse{}
	}
	_ = json.NewEncoder(w).Encode(jobs)
}
