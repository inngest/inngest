package apiv1

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// runTraceReader stubs only GetRun. Every other method of cqrs.TraceReader
// keeps its zero value via the embedded interface.
type runTraceReader struct {
	cqrs.TraceReader
	runErr error
}

func (r *runTraceReader) GetRun(context.Context, ulid.ULID, uuid.UUID, uuid.UUID) (*cqrs.FunctionRun, error) {
	return nil, r.runErr
}

// newRunsRouter builds a router whose only backing store returns runErr from
// GetRun. A run that does not exist surfaces as sql.ErrNoRows, exactly as it
// does on Postgres and SQLite, where the generated query uses
// QueryRowContext(...).Scan(...) and the error is passed through unwrapped.
func newRunsRouter(runErr error) router {
	return router{API: &API{opts: Opts{
		RateLimited: noopRateChecker,
		AuthFinder:  apiv1auth.NilAuthFinder,
		TraceReader: &runTraceReader{runErr: runErr},
	}}}
}

// runsRequest builds a GET /v1/runs/{runID} request. The chi route context
// has to be injected by hand because the handlers read the run ID from it.
func runsRequest(t *testing.T, runID ulid.ULID) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String(), nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("runID", runID.String())
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestGetFunctionRunUnknownRunReturnsNotFound(t *testing.T) {
	runID := ulid.Make()
	router := newRunsRouter(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	router.GetFunctionRun(rec, runsRequest(t, runID))

	require.Equal(t, http.StatusNotFound, rec.Code)

	var response publicerr.Error
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, http.StatusNotFound, response.Status)
}

func TestGetFunctionRunLoadFailureReturnsInternalServerError(t *testing.T) {
	runID := ulid.Make()
	router := newRunsRouter(errors.New("trace backend unavailable"))

	rec := httptest.NewRecorder()
	router.GetFunctionRun(rec, runsRequest(t, runID))

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var response publicerr.Error
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, http.StatusInternalServerError, response.Status)
}

func TestGetFunctionRunJobsUnknownRunReturnsNotFound(t *testing.T) {
	runID := ulid.Make()
	router := newRunsRouter(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	router.GetFunctionRunJobs(rec, runsRequest(t, runID))

	require.Equal(t, http.StatusNotFound, rec.Code)

	var response publicerr.Error
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, http.StatusNotFound, response.Status)
}

func TestGetFunctionRunJobsLoadFailureReturnsInternalServerError(t *testing.T) {
	runID := ulid.Make()
	router := newRunsRouter(errors.New("trace backend unavailable"))

	rec := httptest.NewRecorder()
	router.GetFunctionRunJobs(rec, runsRequest(t, runID))

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var response publicerr.Error
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, http.StatusInternalServerError, response.Status)
}
