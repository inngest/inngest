package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/inngest"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHTTPGateway_FunctionsPauseResume(t *testing.T) {
	for _, action := range []struct {
		name, method string
		paused       bool
	}{
		{"pause", apiv2.V2_PauseFunctions_FullMethodName, true},
		{"resume", apiv2.V2_ResumeFunctions_FullMethodName, false},
	} {
		t.Run(action.name, func(t *testing.T) {
			for _, unknownApp := range []bool{false, true} {
				appID := "my app"
				if unknownApp {
					appID = "missing"
				}
				t.Run(appID, func(t *testing.T) {
					ids := []string{"nope", "echo", "archived", "lookup-error", "write-error", "scope-miss", "scheduled", "hello%2Fworld", "hello/world"}
					functions := &mockFunctionProvider{}
					state := map[uuid.UUID]time.Time{}
					for _, id := range ids {
						fn := inngest.DeployedFunction{ID: uuid.New(), Slug: "my app-" + id, Function: inngest.Function{Name: id, Slug: id}}
						if !action.paused {
							state[fn.ID] = time.Now().Add(-time.Hour)
						}
						if id == "archived" {
							fn.ArchivedAt = time.Now().Add(-time.Hour)
						}
						var lookupErr error
						if id == "lookup-error" {
							lookupErr = errors.New("database unavailable")
						}
						if unknownApp || id == "nope" {
							lookupErr = ErrFunctionNotFound
						}
						functions.On("GetFunctionByApp", mock.Anything, appID, id).Return(fn, lookupErr).Twice()
					}
					limiter := &mockRateLimitProvider{}
					limiter.On("CheckRateLimit", mock.Anything, action.method).Return(RateLimitResult{}).Twice()
					var applied []string
					pauser := functionPauserFunc(func(_ context.Context, fn inngest.DeployedFunction) (inngest.DeployedFunction, error) {
						id := fn.Function.Slug
						applied = append(applied, id)
						switch id {
						case "write-error":
							return fn, errors.New("queue unavailable")
						case "scope-miss":
							return fn, ErrFunctionNotFound
						case "scheduled":
							return fn, ErrFunctionPauseScheduled
						}
						if action.paused {
							if state[fn.ID].IsZero() {
								state[fn.ID] = time.Now().Add(-time.Second)
							}
						} else {
							state[fn.ID] = time.Time{}
						}
						fn.PausedAt = state[fn.ID]
						return fn, nil
					})
					planCalls := 0
					config := functionConfigProviderFunc(func(_ context.Context, fn inngest.DeployedFunction) int {
						planCalls++
						require.Equal(t, "echo", fn.Function.Slug)
						return 37
					})
					handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Functions: functions, FunctionPauser: pauser, RateLimitProvider: limiter, FunctionConfig: config}, HTTPHandlerOptions{})
					require.NoError(t, err)
					body, err := json.Marshal(map[string]any{"functionIds": ids, "appId": "ignored-body-app"})
					require.NoError(t, err)
					var first []map[string]json.RawMessage
					for repeat := 0; repeat < 2; repeat++ {
						rec := httptest.NewRecorder()
						pathApp := "my%20app"
						if unknownApp {
							pathApp = "missing"
						}
						handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/"+pathApp+"/functions/"+action.name, strings.NewReader(string(body))))
						require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
						var response struct {
							Data     []map[string]json.RawMessage `json:"data"`
							Metadata struct {
								FetchedAt time.Time `json:"fetchedAt"`
							} `json:"metadata"`
						}
						require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
						require.Len(t, response.Data, len(ids))
						require.WithinDuration(t, time.Now(), response.Metadata.FetchedAt, time.Second)
						for i, item := range response.Data {
							var id string
							require.NoError(t, json.Unmarshal(item["functionId"], &id))
							require.Equal(t, ids[i], id)
							code, message := "", ""
							switch id {
							case "nope", "archived", "scope-miss":
								code, message = "not_found", "Function not found"
							case "lookup-error":
								code, message = "internal_error", "Unable to fetch function"
							case "write-error":
								code, message = "internal_error", "Unable to "+action.name+" function"
							case "scheduled":
								code, message = "function_pause_scheduled", "A scheduled pause cannot be resumed before it takes effect"
							}
							if unknownApp {
								code, message = "not_found", "Function not found"
							}
							if code != "" {
								require.NotContains(t, item, "function")
								var failure apiv2base.ErrorItem
								require.NoError(t, json.Unmarshal(item["error"], &failure))
								require.Equal(t, apiv2base.ErrorItem{Code: code, Message: message}, failure)
								continue
							}
							require.NotContains(t, item, "error")
							var fn struct {
								ID            string `json:"id"`
								IsPaused      bool   `json:"isPaused"`
								Configuration struct {
									Concurrency []struct {
										Limit struct {
											Value int `json:"value"`
										} `json:"limit"`
									} `json:"concurrency"`
								} `json:"configuration"`
								App struct {
									ID string `json:"id"`
								} `json:"app"`
							}
							require.NoError(t, json.Unmarshal(item["function"], &fn))
							require.Equal(t, id, fn.ID)
							require.Equal(t, "my app", fn.App.ID)
							require.Equal(t, action.paused, fn.IsPaused)
							require.Len(t, fn.Configuration.Concurrency, 1)
							require.Equal(t, 37, fn.Configuration.Concurrency[0].Limit.Value)
						}
						if repeat == 0 {
							first = response.Data
						} else {
							require.Equal(t, first, response.Data)
						}
					}
					if unknownApp {
						require.Empty(t, applied)
						require.Zero(t, planCalls)
					} else {
						require.Equal(t, []string{"echo", "write-error", "scope-miss", "scheduled", "hello%2Fworld", "hello/world", "echo", "write-error", "scope-miss", "scheduled", "hello%2Fworld", "hello/world"}, applied)
						require.Equal(t, 2, planCalls)
					}
					functions.AssertExpectations(t)
					limiter.AssertExpectations(t)
				})
			}
		})
	}
}

func TestHTTPGateway_FunctionsPauseResumeRequestErrors(t *testing.T) {
	for _, action := range []struct{ name, method string }{
		{"pause", apiv2.V2_PauseFunctions_FullMethodName},
		{"resume", apiv2.V2_ResumeFunctions_FullMethodName},
	} {
		t.Run(action.name, func(t *testing.T) {
			ids := make([]string, 501)
			for i := range ids {
				ids[i] = fmt.Sprintf("fn-%d", i)
			}
			tooMany, err := json.Marshal(map[string]any{"functionIds": ids})
			require.NoError(t, err)
			for _, tc := range []struct {
				name, body, code               string
				status                         int
				limited, noFunctions, noPauser bool
			}{
				{name: "missing", body: `{}`, status: 400, code: "missing_field"},
				{name: "empty", body: `{"functionIds":[]}`, status: 400, code: "missing_field"},
				{name: "empty id", body: `{"functionIds":["echo","","flaky"]}`, status: 400, code: "missing_field"},
				{name: "duplicate", body: `{"functionIds":["echo","flaky","echo"]}`, status: 400, code: "invalid_field_format"},
				{name: "long id", body: `{"functionIds":["echo","` + strings.Repeat("x", 257) + `"]}`, status: 400, code: "invalid_field_format"},
				{name: "long multibyte id", body: `{"functionIds":["` + strings.Repeat("é", 129) + `"]}`, status: 400, code: "invalid_field_format"},
				{name: "too many", body: string(tooMany), status: 400, code: "invalid_field_format"},
				{name: "unknown field", body: `{"functionIds":["echo"],"extra":true}`, status: 400, code: "invalid_request"},
				{name: "wrong type", body: `{"functionIds":"echo"}`, status: 400, code: "api_error"},
				{name: "limited", body: `{"functionIds":["echo","flaky"]}`, status: 429, code: "rate_limited", limited: true},
				{name: "no functions", body: `{"functionIds":["echo"]}`, status: 501, code: "not_implemented", noFunctions: true},
				{name: "no pauser", body: `{"functionIds":["echo"]}`, status: 501, code: "not_implemented", noPauser: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					limiter := &mockRateLimitProvider{}
					if tc.status != 400 {
						limiter.On("CheckRateLimit", mock.Anything, action.method).Return(RateLimitResult{Limited: tc.limited}).Once()
					}
					functions := &mockFunctionProvider{}
					opts := ServiceOptions{Functions: functions, RateLimitProvider: limiter, FunctionPauser: functionPauserFunc(func(context.Context, inngest.DeployedFunction) (inngest.DeployedFunction, error) {
						t.Fatal("invalid request applied a mutation")
						return inngest.DeployedFunction{}, nil
					})}
					if tc.noFunctions {
						opts.Functions = nil
					}
					if tc.noPauser {
						opts.FunctionPauser = nil
					}
					handler, err := newTestHTTPHandler(t.Context(), opts, HTTPHandlerOptions{})
					require.NoError(t, err)
					rec := httptest.NewRecorder()
					handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/"+action.name, strings.NewReader(tc.body)))
					require.Equal(t, tc.status, rec.Code, rec.Body.String())
					var response errorResponse
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
					require.Len(t, response.Errors, 1)
					require.Equal(t, tc.code, response.Errors[0].Code)
					if tc.code == "missing_field" {
						require.JSONEq(t, `{"errors":[{"code":"missing_field","message":"App ID and function IDs are required"}]}`, rec.Body.String())
					}
					switch tc.name {
					case "too many":
						require.Equal(t, "Function IDs cannot exceed 500", response.Errors[0].Message)
					case "duplicate":
						require.Equal(t, "Function IDs must be unique", response.Errors[0].Message)
					case "long id", "long multibyte id":
						require.Equal(t, "Function IDs cannot exceed 256 bytes", response.Errors[0].Message)
					}
					limiter.AssertExpectations(t)
					functions.AssertExpectations(t)
				})
			}
		})
	}
}

func TestService_FunctionsPauseResumeMissingApp(t *testing.T) {
	svc := NewService(ServiceOptions{})
	_, err := svc.PauseFunctions(t.Context(), &apiv2.PauseFunctionsRequest{FunctionIds: []string{"echo"}})
	require.EqualError(t, err, apiv2base.NewError(400, "missing_field", "App ID and function IDs are required").Error())
	_, err = svc.ResumeFunctions(t.Context(), &apiv2.ResumeFunctionsRequest{FunctionIds: []string{"echo"}})
	require.EqualError(t, err, apiv2base.NewError(400, "missing_field", "App ID and function IDs are required").Error())
}

func TestHTTPGateway_FunctionsPauseResumeLimitAndSingleResponse(t *testing.T) {
	for _, action := range []struct{ name, single, bulk string }{
		{"pause", apiv2.V2_PauseFunction_FullMethodName, apiv2.V2_PauseFunctions_FullMethodName},
		{"resume", apiv2.V2_ResumeFunction_FullMethodName, apiv2.V2_ResumeFunctions_FullMethodName},
	} {
		t.Run(action.name, func(t *testing.T) {
			id := "11111111-1111-1111-1111-111111111111"
			fn := inngest.DeployedFunction{ID: uuid.MustParse(id), Slug: "app-echo", Function: inngest.Function{Name: "Echo", Slug: "echo"}}
			functions := &mockFunctionProvider{}
			ids := make([]string, 500)
			for i := range ids {
				ids[i] = fmt.Sprintf("fn-%0253d", i)
			}
			ids[0] = id
			ids[len(ids)-1] = strings.Repeat("é", 128)
			for i, functionID := range ids {
				count := 1
				if i == 0 {
					count = 2
				}
				functions.On("GetFunctionByApp", mock.Anything, "app", functionID).Return(fn, nil).Times(count)
			}
			limiter := &mockRateLimitProvider{}
			limiter.On("CheckRateLimit", mock.Anything, action.single).Return(RateLimitResult{}).Once()
			limiter.On("CheckRateLimit", mock.Anything, action.bulk).Return(RateLimitResult{}).Once()
			pauser := functionPauserFunc(func(_ context.Context, resolved inngest.DeployedFunction) (inngest.DeployedFunction, error) {
				require.Equal(t, fn, resolved)
				if action.name == "pause" {
					resolved.PausedAt = time.Now().Add(-time.Second)
				}
				return resolved, nil
			})
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Functions: functions, FunctionPauser: pauser, RateLimitProvider: limiter}, HTTPHandlerOptions{})
			require.NoError(t, err)
			single := httptest.NewRecorder()
			handler.ServeHTTP(single, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/"+id+"/"+action.name, nil))
			require.Equal(t, http.StatusOK, single.Code, single.Body.String())
			var one struct {
				Data json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(single.Body.Bytes(), &one))
			body, err := json.Marshal(map[string]any{"functionIds": ids})
			require.NoError(t, err)
			bulk := httptest.NewRecorder()
			handler.ServeHTTP(bulk, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/"+action.name, strings.NewReader(string(body))))
			require.Equal(t, http.StatusOK, bulk.Code, bulk.Body.String())
			var many struct {
				Data []struct {
					FunctionID string          `json:"functionId"`
					Function   json.RawMessage `json:"function"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(bulk.Body.Bytes(), &many))
			require.Len(t, many.Data, 500)
			for i, item := range many.Data {
				require.Equal(t, ids[i], item.FunctionID)
				require.JSONEq(t, string(one.Data), string(item.Function))
			}
			functions.AssertExpectations(t)
			limiter.AssertExpectations(t)
		})
	}
}

type functionConfigProviderFunc func(context.Context, inngest.DeployedFunction) int

func (f functionConfigProviderFunc) PlanConcurrencyLimit(ctx context.Context, fn inngest.DeployedFunction) int {
	return f(ctx, fn)
}

func TestService_FunctionsPauseResumeStopsOnCancel(t *testing.T) {
	for _, action := range []struct{ name, method string }{
		{"pause", apiv2.V2_PauseFunctions_FullMethodName},
		{"resume", apiv2.V2_ResumeFunctions_FullMethodName},
	} {
		for _, before := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/before=%t", action.name, before), func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				fn := inngest.DeployedFunction{ID: uuid.New(), Function: inngest.Function{Slug: "echo"}}
				functions := &mockFunctionProvider{}
				if before {
					cancel()
				} else {
					functions.On("GetFunctionByApp", mock.Anything, "app", "echo").Return(fn, nil).Once()
				}
				limiter := &mockRateLimitProvider{}
				limiter.On("CheckRateLimit", mock.Anything, action.method).Return(RateLimitResult{}).Once()
				var applied []string
				svc := NewService(ServiceOptions{
					Functions: functions, RateLimitProvider: limiter,
					FunctionPauser: functionPauserFunc(func(_ context.Context, resolved inngest.DeployedFunction) (inngest.DeployedFunction, error) {
						require.Equal(t, fn, resolved)
						applied = append(applied, resolved.Function.Slug)
						cancel()
						return resolved, nil
					}),
				})
				ids := []string{"echo", "flaky"}
				var err error
				if action.name == "pause" {
					_, err = svc.PauseFunctions(ctx, &apiv2.PauseFunctionsRequest{AppId: "app", FunctionIds: ids})
				} else {
					_, err = svc.ResumeFunctions(ctx, &apiv2.ResumeFunctionsRequest{AppId: "app", FunctionIds: ids})
				}
				require.ErrorIs(t, err, context.Canceled)
				if before {
					require.Empty(t, applied)
				} else {
					require.Equal(t, []string{"echo"}, applied)
				}
				functions.AssertExpectations(t)
				limiter.AssertExpectations(t)
			})
		}
	}
}

func TestHTTPGateway_SingleFunctionPauseResumeDecodesPath(t *testing.T) {
	for _, action := range []string{"pause", "resume"} {
		t.Run(action, func(t *testing.T) {
			fn := inngest.DeployedFunction{ID: uuid.New(), Slug: "my app-hello/world", Function: inngest.Function{Slug: "hello/world"}}
			functions := &mockFunctionProvider{}
			functions.On("GetFunctionByApp", mock.Anything, "my app", "hello/world").Return(fn, nil).Once()
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{
				Functions: functions,
				FunctionPauser: functionPauserFunc(func(_ context.Context, resolved inngest.DeployedFunction) (inngest.DeployedFunction, error) {
					require.Equal(t, fn, resolved)
					return resolved, nil
				}),
			}, HTTPHandlerOptions{})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/my%2520app/functions/hello%252Fworld/"+action, nil))
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var response struct {
				Data struct {
					ID  string `json:"id"`
					App struct {
						ID string `json:"id"`
					} `json:"app"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			require.Equal(t, "hello/world", response.Data.ID)
			require.Equal(t, "my app", response.Data.App.ID)
			functions.AssertExpectations(t)
		})
	}
}
