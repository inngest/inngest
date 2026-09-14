package apiv1

import (
	"context"
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

type eventRunsTraceReader struct {
	cqrs.TraceReader
	runs     []*cqrs.FunctionRun
	spansErr error
	gotRunID ulid.ULID
}

func (r *eventRunsTraceReader) GetEventRuns(context.Context, ulid.ULID, uuid.UUID, uuid.UUID) ([]*cqrs.FunctionRun, error) {
	return r.runs, nil
}

func (r *eventRunsTraceReader) GetSpansByRunID(_ context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	r.gotRunID = runID
	return nil, r.spansErr
}

func TestGetEventRunsReturnsInternalServerErrorWhenRunStatusQueryFails(t *testing.T) {
	eventID := ulid.Make()
	runID := ulid.Make()
	traceReader := &eventRunsTraceReader{
		runs:     []*cqrs.FunctionRun{{RunID: runID}},
		spansErr: errors.New("trace backend unavailable"),
	}
	router := router{API: &API{opts: Opts{
		RateLimited: noopRateChecker,
		AuthFinder:  apiv1auth.NilAuthFinder,
		TraceReader: traceReader,
	}}}

	req := httptest.NewRequest(http.MethodGet, "/v1/events/"+eventID.String()+"/runs", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("eventID", eventID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rec := httptest.NewRecorder()

	router.getEventRuns(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, runID, traceReader.gotRunID)

	var response publicerr.Error
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, "Unable to query run status", response.Message)
	require.Equal(t, http.StatusInternalServerError, response.Status)
}
