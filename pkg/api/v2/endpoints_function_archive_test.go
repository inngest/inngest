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

type functionArchiverFunc func(context.Context, inngest.DeployedFunction) (inngest.DeployedFunction, error)

func (f functionArchiverFunc) ArchiveFunction(ctx context.Context, fn inngest.DeployedFunction) (inngest.DeployedFunction, error) {
	return f(ctx, fn)
}

func (f functionArchiverFunc) UnarchiveFunction(ctx context.Context, fn inngest.DeployedFunction) (inngest.DeployedFunction, error) {
	return f(ctx, fn)
}

func TestHTTPGateway_FunctionArchiveUnarchive(t *testing.T) {
	for _, action := range []struct {
		name     string
		method   string
		archived bool
	}{
		{"archive", apiv2.V2_ArchiveFunction_FullMethodName, true},
		{"unarchive", apiv2.V2_UnarchiveFunction_FullMethodName, false},
	} {
		t.Run(action.name, func(t *testing.T) {
			for _, tc := range []struct {
				name           string
				appID          string
				functionID     string
				lookupErr      error
				mutationErr    error
				alreadyApplied bool
				noArchiver     bool
				limited        bool
				status         int
				code           string
			}{
				{name: "changed", status: 200},
				{name: "already applied", alreadyApplied: true, status: 200},
				{name: "unknown app", appID: "missing", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
				{name: "unknown function", functionID: "missing", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
				{name: "out of tenant", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
				{name: "app mismatch", appID: "other", lookupErr: ErrFunctionNotFound, status: 404, code: "not_found"},
				{name: "lookup failure", lookupErr: errors.New("database unavailable"), status: 500, code: "internal_error"},
				{name: "no provider", noArchiver: true, status: 501, code: "not_implemented"},
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
					archivedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
					if tc.alreadyApplied == action.archived {
						fn.ArchivedAt = archivedAt
					}
					functions := &mockFunctionProvider{}
					if !tc.noArchiver && !tc.limited {
						functions.On("GetFunctionByApp", mock.Anything, appID, functionID).Return(fn, tc.lookupErr).Once()
					}
					t.Cleanup(func() { functions.AssertExpectations(t) })
					calls := 0
					limiter := &testEventRateLimiter{limited: tc.limited}
					opts := ServiceOptions{Functions: functions, RateLimitProvider: limiter}
					if !tc.noArchiver {
						opts.FunctionArchiver = functionArchiverFunc(func(_ context.Context, resolved inngest.DeployedFunction) (inngest.DeployedFunction, error) {
							calls++
							require.Equal(t, fn, resolved, "write must use the scoped lookup result")
							if action.archived {
								if resolved.ArchivedAt.IsZero() {
									resolved.ArchivedAt = archivedAt
								}
							} else {
								resolved.ArchivedAt = time.Time{}
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
					if tc.lookupErr != nil || tc.noArchiver || tc.limited {
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
							ID         string `json:"id"`
							Name       string `json:"name"`
							IsArchived *bool  `json:"isArchived"`
							App        struct {
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
					require.NotNil(t, body.Data.IsArchived)
					require.Equal(t, action.archived, *body.Data.IsArchived)
					require.WithinDuration(t, time.Now(), body.Metadata.FetchedAt, time.Second)
				})
			}
		})
	}
}

func TestHTTPGateway_FunctionArchiveUnarchiveRejectsInvalidRequests(t *testing.T) {
	for _, action := range []string{"archive", "unarchive"} {
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

func TestService_ArchiveFunctionMissingIDs(t *testing.T) {
	for _, req := range []*apiv2.ArchiveFunctionRequest{{AppId: "app"}, {FunctionId: "fn"}} {
		_, err := NewService(ServiceOptions{}).ArchiveFunction(t.Context(), req)
		require.ErrorContains(t, err, `"code":"missing_field"`)
	}
}

func TestService_UnarchiveFunctionMissingIDs(t *testing.T) {
	for _, req := range []*apiv2.UnarchiveFunctionRequest{{AppId: "app"}, {FunctionId: "fn"}} {
		_, err := NewService(ServiceOptions{}).UnarchiveFunction(t.Context(), req)
		require.ErrorContains(t, err, `"code":"missing_field"`)
	}
}

type archiveFunctionStore struct {
	FunctionProvider
	fn inngest.DeployedFunction
}

func (s *archiveFunctionStore) GetFunctionByApp(_ context.Context, appID, functionID string) (inngest.DeployedFunction, error) {
	if appID != "app" || functionID != "fn" {
		return inngest.DeployedFunction{}, ErrFunctionNotFound
	}
	return s.fn, nil
}

func (s *archiveFunctionStore) ArchiveFunction(_ context.Context, fn inngest.DeployedFunction) (inngest.DeployedFunction, error) {
	if fn.ArchivedAt.IsZero() {
		fn.ArchivedAt = time.Now()
	}
	s.fn = fn
	return fn, nil
}

func (s *archiveFunctionStore) UnarchiveFunction(_ context.Context, fn inngest.DeployedFunction) (inngest.DeployedFunction, error) {
	fn.ArchivedAt = time.Time{}
	s.fn = fn
	return fn, nil
}

func TestHTTPGateway_FunctionArchiveRoundTrip(t *testing.T) {
	pausedAt := time.Now().Add(-time.Hour)
	store := &archiveFunctionStore{fn: inngest.DeployedFunction{
		ID: uuid.New(), AppName: "app", Slug: "app-fn", PausedAt: pausedAt,
		Function: inngest.Function{Slug: "fn", Name: "Function"},
	}}
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Functions: store, FunctionArchiver: store}, HTTPHandlerOptions{})
	require.NoError(t, err)
	for _, action := range []string{"archive", "unarchive"} {
		var first map[string]any
		var state inngest.DeployedFunction
		for i := 0; i < 2; i++ {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v2/apps/app/functions/fn/"+action, nil))
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var response struct{ Data map[string]any }
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			require.Equal(t, action == "archive", response.Data["isArchived"])
			require.Equal(t, true, response.Data["isPaused"])
			require.Equal(t, pausedAt, store.fn.PausedAt)
			if i == 0 {
				first, state = response.Data, store.fn
			} else {
				require.Equal(t, first, response.Data)
				require.Equal(t, state, store.fn)
			}
		}
	}
}
