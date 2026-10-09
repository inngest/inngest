package apiv1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

type otelEnabledTraceReader struct {
	cqrs.TraceReader
}

func (otelEnabledTraceReader) OtelTracesEnabled(context.Context, uuid.UUID) (bool, error) {
	return true, nil
}

func TestTracesOverCapSetsRetryAfter(t *testing.T) {
	router := router{API: &API{opts: Opts{
		AuthFinder:  apiv1auth.NilAuthFinder,
		TraceReader: otelEnabledTraceReader{},
		ExtendedTraceCapCheck: func(context.Context, uuid.UUID, int64) ExtendedTraceCapDecision {
			return ExtendedTraceCapDecision{OverCap: true}
		},
	}}}

	req := httptest.NewRequest(http.MethodPost, "/v1/traces/userland", nil)
	rec := httptest.NewRecorder()
	router.traces(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "60", rec.Header().Get("Retry-After"))
}
