package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func (f functionPauserFunc) ResumeFunction(ctx context.Context, fn inngest.DeployedFunction) (inngest.DeployedFunction, error) {
	return f(ctx, fn)
}

func TestHTTPGateway_FunctionPauseResume(t *testing.T) {
	for _, action := range []struct {
		name   string
		method string
		paused bool
	}{
		{"pause", apiv2.V2_PauseFunction_FullMethodName, true},
		{"resume", apiv2.V2_ResumeFunction_FullMethodName, false},
	} {
		t.Run(action.name, func(t *testing.T) {
			for _, tc := range []struct {
				name           string
				appID          string
				functionID     string
				lookupErr      error
				mutationErr    error
				alreadyApplied bool
				archived       bool
				noPauser       bool
				limited        bool
				status         int
				code           string
			}{
				{name: "changed", status: 200},
				{name: "already applied", alreadyApplied: true, status: 200},
				{name: "archived", archived: true, status: 404, code: "not_found"},
				{name: "unknown app", appID: "missing", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
				{name: "unknown function", functionID: "missing", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
				{name: "out of tenant", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
				{name: "app mismatch", appID: "other", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
				{name: "lookup failure", lookupErr: errors.New("database unavailable"), status: 500, code: "internal_error"},
				{name: "no provider", noPauser: true, status: 501, code: "not_implemented"},
				{name: "scheduled pause", mutationErr: ErrFunctionPauseScheduled, status: 409, code: "function_pause_scheduled"},
				{name: "provider scope miss", mutationErr: ErrFunctionNotFound, status: 404, code: "not_found"},
				{name: "provider failure", mutationErr: errors.New("queue unavailable"), status: 500, code: "internal_error"},
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
					if tc.alreadyApplied == action.paused {
						fn.PausedAt = pausedAt
					}
					if tc.archived {
						fn.ArchivedAt = pausedAt
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
							if action.paused {
								if resolved.PausedAt.IsZero() {
									resolved.PausedAt = pausedAt
								}
							} else {
								resolved.PausedAt = time.Time{}
							}
							if tc.alreadyApplied {
								require.Equal(t, fn, resolved)
							}
							return resolved, tc.mutationErr
						})
					}
					handler, err := newTestHTTPHandler(t.Context(), opts, HTTPHandlerOptions{})
					require.NoError(t, err)
					recorder := httptest.NewRecorder()
					handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v2/apps/"+appID+"/functions/"+functionID+"/"+action.name, nil))
					require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
					require.Equal(t, action.method, limiter.method)
					if tc.lookupErr != nil || tc.noPauser || tc.limited || tc.archived {
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
					require.Equal(t, action.paused, body.Data.IsPaused)
					require.WithinDuration(t, time.Now(), body.Metadata.FetchedAt, time.Second)
				})
			}
		})
	}
}

func TestHTTPGateway_FunctionPauseResumeRejectsInvalidRequests(t *testing.T) {
	for _, action := range []string{"pause", "resume"} {
		t.Run(action, func(t *testing.T) {
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{}, HTTPHandlerOptions{})
			require.NoError(t, err)
			for _, tc := range []struct {
				name    string
				method  string
				body    string
				status  int
				message string
			}{
				// gateway returns 501 for unsupported methods
				{name: "PUT", method: http.MethodPut, status: http.StatusNotImplemented, message: "Method Not Allowed"},
				{name: "JSON body", method: http.MethodPost, body: "{}", status: http.StatusBadRequest},
				{name: "whitespace body", method: http.MethodPost, body: " ", status: http.StatusBadRequest},
			} {
				t.Run(tc.name, func(t *testing.T) {
					rec := httptest.NewRecorder()
					handler.ServeHTTP(rec, httptest.NewRequest(tc.method, "/api/v2/apps/app/functions/fn/"+action, strings.NewReader(tc.body)))
					require.Equal(t, tc.status, rec.Code, rec.Body.String())
					if tc.message != "" {
						require.JSONEq(t, `{"errors":[{"code":"api_error","message":"`+tc.message+`"}]}`, rec.Body.String())
					}
				})
			}
		})
	}
}

func TestService_PauseFunctionMissingIDs(t *testing.T) {
	for _, req := range []*apiv2.PauseFunctionRequest{{AppId: "app"}, {FunctionId: "fn"}} {
		_, err := NewService(ServiceOptions{}).PauseFunction(t.Context(), req)
		require.ErrorContains(t, err, `"code":"missing_field"`)
	}
}

func TestService_ResumeFunctionMissingIDs(t *testing.T) {
	for _, req := range []*apiv2.ResumeFunctionRequest{{AppId: "app"}, {FunctionId: "fn"}} {
		_, err := NewService(ServiceOptions{}).ResumeFunction(t.Context(), req)
		require.ErrorContains(t, err, `"code":"missing_field"`)
	}
}
