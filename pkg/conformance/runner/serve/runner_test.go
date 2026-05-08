package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/conformance"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestRunnerRunBasicInvoke(t *testing.T) {
	t.Parallel()

	fixture := newServeFixture(t)
	defer fixture.Close()

	registry := conformance.DefaultRegistry()
	plan, err := (conformance.Selection{
		Transport: conformance.TransportServe,
		Cases:     []string{"serve-introspection", "basic-invoke"},
	}).Resolve(registry)
	require.NoError(t, err)

	runner := NewRunner(nil)
	report, err := runner.Run(context.Background(), plan, fixture.Runtime())
	require.NoError(t, err)
	require.Equal(t, conformance.CompatibilityFull, report.Compatibility)
	require.Len(t, report.Cases, 2)
	for _, result := range report.Cases {
		require.Equal(t, conformance.CaseStatusPassed, result.Status)
	}
}

func TestRunnerDoctorDetectsMissingFixtureFunctions(t *testing.T) {
	t.Parallel()

	fixture := newServeFixture(t)
	defer fixture.Close()
	fixture.functions = fixture.functions[:0]

	registry := conformance.DefaultRegistry()
	plan, err := (conformance.Selection{
		Transport: conformance.TransportServe,
		Cases:     []string{"basic-invoke"},
	}).Resolve(registry)
	require.NoError(t, err)

	runner := NewRunner(nil)
	checks, err := runner.Doctor(context.Background(), plan, fixture.Runtime())
	require.NoError(t, err)
	require.False(t, checks[len(checks)-1].Passed)
	require.Contains(t, checks[len(checks)-1].Message, "missing required fixture function")
}

type serveFixture struct {
	t *testing.T

	functions []sdk.SDKFunction

	sdkServer *httptest.Server
	devServer *httptest.Server

	registeredURL string
}

func newServeFixture(t *testing.T) *serveFixture {
	t.Helper()

	fixture := &serveFixture{t: t}
	fixture.functions = []sdk.SDKFunction{
		{
			Name:     "Simple function",
			Slug:     "test-suite-simple-fn",
			Triggers: []inngest.Trigger{eventTrigger("tests/function.test")},
			Steps: map[string]sdk.SDKStep{
				"step": {
					ID:   "step",
					Name: "step",
					Runtime: map[string]any{
						"url": "http://sdk.invalid/api/inngest",
					},
				},
			},
		},
	}

	fixture.sdkServer = httptest.NewServer(http.HandlerFunc(fixturesSDKHandler(fixture)))
	fixture.devServer = httptest.NewServer(http.HandlerFunc(fixturesDevHandler(fixture)))

	for idx := range fixture.functions {
		for stepID, step := range fixture.functions[idx].Steps {
			step.Runtime["url"] = fixture.sdkServer.URL + "/api/inngest"
			fixture.functions[idx].Steps[stepID] = step
		}
	}

	return fixture
}

func (f *serveFixture) Close() {
	f.sdkServer.Close()
	f.devServer.Close()
}

func (f *serveFixture) Runtime() conformance.RuntimeConfig {
	cfg := conformance.Config{
		Transport: conformance.TransportServe,
		SDK: conformance.SDKConfig{
			URL: f.sdkServer.URL + "/api/inngest",
		},
		Dev: conformance.DevConfig{
			URL:        f.devServer.URL,
			SigningKey: "7468697320697320612074657374206b6579",
			EventKey:   "test",
		},
	}

	runtime, err := cfg.Runtime()
	require.NoError(f.t, err)
	return runtime
}

func fixturesSDKHandler(f *serveFixture) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/inngest":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(f.t, json.NewEncoder(w).Encode(map[string]any{
				"schema_version": "2024-05-24",
				"function_count": len(f.functions),
				"mode":           "dev",
			}))
			return
		case r.Method == http.MethodPut && r.URL.Path == "/api/inngest":
			var syncReq struct {
				URL string `json:"url"`
			}
			require.NoError(f.t, json.NewDecoder(r.Body).Decode(&syncReq))
			require.NotEmpty(f.t, syncReq.URL)

			functions := make([]sdk.SDKFunction, len(f.functions))
			copy(functions, f.functions)
			for idx := range functions {
				for stepID, step := range functions[idx].Steps {
					stepURL, err := url.Parse(syncReq.URL)
					require.NoError(f.t, err)
					values := stepURL.Query()
					values.Set("fnId", functions[idx].Slug)
					values.Set("step", "step")
					stepURL.RawQuery = values.Encode()
					step.Runtime["url"] = stepURL.String()
					functions[idx].Steps[stepID] = step
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("x-inngest-sync-kind", "in_band")
			require.NoError(f.t, json.NewEncoder(w).Encode(map[string]any{
				"app_id":       "conformance-test",
				"functions":    functions,
				"sdk_language": "go",
				"sdk_version":  "test",
				"url":          syncReq.URL,
			}))
			return
		case r.Method == http.MethodPost && r.URL.Path == "/api/inngest":
			body, err := io.ReadAll(r.Body)
			require.NoError(f.t, err)
			defer r.Body.Close()

			req := executorRequest{}
			require.NoError(f.t, json.Unmarshal(body, &req))

			switch req.Event.Name {
			case "tests/function.test":
				w.Header().Set("Content-Type", "application/json")
				require.NoError(f.t, json.NewEncoder(w).Encode(map[string]any{
					"name": "tests/function.test",
					"body": "ok",
				}))
				return
			default:
				http.Error(w, fmt.Sprintf("unexpected event %q", req.Event.Name), http.StatusBadRequest)
				return
			}
		default:
			http.NotFound(w, r)
		}
	}
}

func fixturesDevHandler(f *serveFixture) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/fn/register":
			var req sdk.RegisterRequest
			require.NoError(f.t, json.NewDecoder(r.Body).Decode(&req))
			f.registeredURL = req.URL
			w.WriteHeader(http.StatusOK)
			return
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/e/"):
			var event inngestgo.Event
			require.NoError(f.t, json.NewDecoder(r.Body).Decode(&event))

			payload, err := json.Marshal(executorRequest{
				Event: event,
				Steps: map[string]any{},
				Ctx: driver.SDKRequestContext{
					StepID: "step",
					Stack: &driver.FunctionStack{
						Current: 0,
						Stack:   []string{},
					},
				},
				Version: 1,
			})
			require.NoError(f.t, err)

			req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, f.registeredURL+"?stepId=step", strings.NewReader(string(payload)))
			require.NoError(f.t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(f.t, err)
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(f.t, err)

			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write(respBody)
			return
		default:
			http.NotFound(w, r)
		}
	}
}
