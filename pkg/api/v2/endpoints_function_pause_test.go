package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type functionPauserFunc func(context.Context, inngest.DeployedFunction) (inngest.DeployedFunction, error)

func (f functionPauserFunc) PauseFunction(ctx context.Context, fn inngest.DeployedFunction) (inngest.DeployedFunction, error) {
	return f(ctx, fn)
}

func TestHTTPGateway_PauseFunction(t *testing.T) {
	for _, tc := range []struct {
		name          string
		appID         string
		functionID    string
		lookupErr     error
		pauseErr      error
		alreadyPaused bool
		noPauser      bool
		limited       bool
		status        int
		code          string
	}{
		{name: "paused", status: 200},
		{name: "already paused", alreadyPaused: true, status: 200},
		{name: "unknown app", appID: "missing", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
		{name: "unknown function", functionID: "missing", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
		{name: "lookup failure", lookupErr: errors.New("database unavailable"), status: 500, code: "internal_error"},
		{name: "no provider", noPauser: true, status: 501, code: "not_implemented"},
		{name: "provider failure", pauseErr: errors.New("queue unavailable"), status: 500, code: "internal_error"},
		{name: "rate limited", limited: true, status: 429, code: "rate_limited"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			appID, functionID := tc.appID, tc.functionID
			if appID == "" {
				appID = "app"
			}
			if functionID == "" {
				functionID = "fn"
			}
			fn := inngest.DeployedFunction{
				ID: uuid.New(), AccountID: uuid.New(), EnvironmentID: uuid.New(), AppID: uuid.New(),
				Slug: "app-fn", Function: inngest.Function{Name: "Function", Slug: "fn"},
			}
			pausedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			if tc.alreadyPaused {
				fn.PausedAt = pausedAt
			}
			functions := &mockFunctionProvider{}
			if !tc.noPauser && !tc.limited {
				functions.On("GetFunctionByApp", mock.Anything, appID, functionID).Return(fn, tc.lookupErr).Once()
			}
			t.Cleanup(func() { functions.AssertExpectations(t) })
			calls := 0
			limiter := &testEventRateLimiter{limited: tc.limited}
			opts := ServiceOptions{Functions: functions, RateLimitProvider: limiter}
			if !tc.noPauser {
				opts.FunctionPauser = functionPauserFunc(func(_ context.Context, resolved inngest.DeployedFunction) (inngest.DeployedFunction, error) {
					calls++
					require.Equal(t, fn, resolved, "write must use the scoped lookup result")
					if resolved.PausedAt.IsZero() {
						resolved.PausedAt = pausedAt
					}
					return resolved, tc.pauseErr
				})
			}
			handler, err := newTestHTTPHandler(t.Context(), opts, HTTPHandlerOptions{})
			require.NoError(t, err)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/v2/apps/"+appID+"/functions/"+functionID+"/pause", nil))
			require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
			require.Equal(t, apiv2.V2_PauseFunction_FullMethodName, limiter.method)
			if tc.lookupErr != nil || tc.noPauser || tc.limited {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
			if tc.code != "" {
				var body errorResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
				require.Len(t, body.Errors, 1)
				require.Equal(t, tc.code, body.Errors[0].Code)
				return
			}
			var body struct {
				Data struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					IsPaused bool   `json:"isPaused"`
					App      struct {
						ID string `json:"id"`
					} `json:"app"`
				} `json:"data"`
				Metadata struct {
					FetchedAt time.Time `json:"fetchedAt"`
				} `json:"metadata"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			require.Equal(t, "fn", body.Data.ID)
			require.Equal(t, "app", body.Data.App.ID)
			require.Equal(t, "Function", body.Data.Name)
			require.True(t, body.Data.IsPaused)
			require.WithinDuration(t, time.Now(), body.Metadata.FetchedAt, time.Second)
		})
	}
}

func TestService_PauseFunctionMissingIDs(t *testing.T) {
	for _, req := range []*apiv2.PauseFunctionRequest{{AppId: "app"}, {FunctionId: "fn"}} {
		_, err := NewService(ServiceOptions{}).PauseFunction(t.Context(), req)
		require.ErrorContains(t, err, `"code":"missing_field"`)
	}
}
