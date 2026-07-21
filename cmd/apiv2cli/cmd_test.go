package apiv2cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestDiscoverEndpointsFromProto(t *testing.T) {
	endpoints := discoverEndpoints()
	require.NotEmpty(t, endpoints)

	byName := map[string]endpoint{}
	for _, ep := range endpoints {
		byName[ep.name] = ep
	}

	require.NotContains(t, byName, "-schema-only")
	require.NotContains(t, byName, "create-partner-account")
	require.NotContains(t, byName, "fetch-partner-accounts")
	require.NotContains(t, byName, "fetch-account")
	require.NotContains(t, byName, "list-webhooks")
	require.Contains(t, byName, "get-account")
	require.Contains(t, byName, "get-webhooks")
	require.Equal(t, http.MethodPost, byName["invoke-function"].method)
	require.Equal(t, "/apps/{app_id}/functions/{function_id}/invoke", byName["invoke-function"].path)
	require.Equal(t, []string{"app_id", "function_id"}, byName["invoke-function"].pathParams)
	require.Equal(t, http.MethodGet, byName["get-function-trace"].method)
	require.Equal(t, "/runs/{run_id}/trace", byName["get-function-trace"].path)
	require.NotContains(t, byName, "get-runs")
	require.Equal(t, "/runs", byName["get-function-runs"].path)
	require.Empty(t, byName["get-function-runs"].pathParams)
}

func TestEndpointCommandNameNormalizesReadVerbs(t *testing.T) {
	require.Equal(t, "get-account", endpointCommandName("FetchAccount"))
	require.Equal(t, "get-webhooks", endpointCommandName("ListWebhooks"))
	require.Equal(t, "get-function-run", endpointCommandName("GetFunctionRun"))
	require.Equal(t, "get-function-runs", endpointCommandName("ListRuns"))
	require.Equal(t, "create-env", endpointCommandName("CreateEnv"))
}

func TestCommandHelpUsesTopLevelAPIAndBetaLabel(t *testing.T) {
	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--help",
	})

	require.NoError(t, err)
	require.Contains(t, out.String(), "inngest api [target/auth flags] <endpoint> [endpoint flags]")
	require.Contains(t, out.String(), "Call Inngest REST API v2 endpoints (beta)")
	require.Contains(t, out.String(), "Beta: this command is under active development and may change.")
	require.NotContains(t, out.String(), "inngest alpha api [target/auth flags]")
}

func TestMovedCommandTellsUsersToUseTopLevelAPI(t *testing.T) {
	cmd := MovedCommand()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
	})

	require.NoError(t, err)
	require.Contains(t, out.String(), "The alpha api command has moved. Use `inngest api` instead.")
}

func TestEndpointCommandsIncludeOperationAndInheritedFlagHelp(t *testing.T) {
	var invoke *cli.Command
	for _, cmd := range endpointCommands() {
		if cmd.Name == "invoke-function" {
			invoke = cmd
			break
		}
	}

	require.NotNil(t, invoke)
	require.Equal(t, "Invoke function", invoke.Usage)
	require.Contains(t, invoke.Description, "Endpoint: POST /apps/{app_id}/functions/{function_id}/invoke")
	require.Contains(t, invoke.Description, "--prod")
	require.Contains(t, invoke.Description, "INNGEST_API_KEY")
	require.Contains(t, invoke.Description, "INNGEST_ENV")
	require.Contains(t, invoke.Description, "/v2")
}

func TestCommandTelemetryContextIncludesEndpointAndFlagNamesOnly(t *testing.T) {
	var getEventRuns endpoint
	for _, ep := range discoverEndpoints() {
		if ep.name == "get-event-runs" {
			getEventRuns = ep
			break
		}
	}
	require.NotEmpty(t, getEventRuns.name)

	var endpointCommand *cli.Command
	for _, command := range endpointCommands() {
		if command.Name == getEventRuns.name {
			endpointCommand = command
			break
		}
	}
	require.NotNil(t, endpointCommand)

	app := Command()
	endpointCommand.Action = func(_ context.Context, cmd *cli.Command) error {
		require.Equal(t, map[string]any{
			"endpoint": "get-event-runs",
			"flags":    []string{"include-output", "limit", "prod"},
		}, commandTelemetryContext(cmd, getEventRuns))
		return nil
	}
	app.Commands = []*cli.Command{endpointCommand}

	err := app.Run(context.Background(), []string{
		"api",
		"--prod",
		"get-event-runs",
		"01KTCTWSZJEKAFEDA4F9GYHFQW",
		"--include-output",
		"--limit", "5",
	})

	require.NoError(t, err)
}

func TestEndpointFlagsUseProtoFieldDescriptions(t *testing.T) {
	var invoke endpoint
	for _, ep := range discoverEndpoints() {
		if ep.name == "invoke-function" {
			invoke = ep
			break
		}
	}
	require.NotEmpty(t, invoke.name)

	flags := endpointFlags(invoke)
	byName := map[string]cli.Flag{}
	for _, flag := range flags {
		byName[flag.Names()[0]] = flag
	}

	require.Contains(t, byName["data"].String(), "JSON object containing the input data for the function")
	require.Contains(t, byName["function-id"].String(), "The ID of the function to invoke")
	require.Contains(t, byName["idempotency-key"].String(), "Optional idempotency key")
}

func TestEndpointDescriptionReferencesValidFlags(t *testing.T) {
	var invoke endpoint
	for _, ep := range discoverEndpoints() {
		if ep.name == "invoke-function" {
			invoke = ep
			break
		}
	}
	require.NotEmpty(t, invoke.name)

	valid := map[string]bool{}
	for _, flag := range commonFlags() {
		for _, name := range flag.Names() {
			valid["--"+name] = true
		}
	}

	desc := endpointDescription(invoke)
	for _, match := range regexp.MustCompile(`--[a-z][a-z0-9-]*`).FindAllString(desc, -1) {
		require.True(t, valid[match],
			"endpointDescription references %s, which is not defined in commonFlags(); keep the inherited-flag block in sync", match)
	}
}

func TestCommandCallsGeneratedEndpoint(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotEnv string
	var gotContentType string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotEnv = r.Header.Get("X-Inngest-Env")
		gotContentType = r.Header.Get("Content-Type")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"runId":"01J00000000000000000000000"}}`))
	}))
	defer srv.Close()

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"invoke-function",
		"--api-host", srv.URL,
		"--signing-key", "signkey-test-abc",
		"--env", "branch-a",
		"--app-id", "my app",
		"--function-id", "hello/world",
		"--data", `{"message":"hi"}`,
		"--idempotency-key", "idem-1",
	})

	require.NoError(t, err)
	require.Equal(t, "/api/v2/apps/my%20app/functions/hello%2Fworld/invoke", gotPath)
	require.Equal(t, "Bearer signkey-test-abc", gotAuth)
	require.Equal(t, "branch-a", gotEnv)
	require.Equal(t, "application/json", gotContentType)
	require.Equal(t, map[string]any{
		"data":           map[string]any{"message": "hi"},
		"idempotencyKey": "idem-1",
	}, gotBody)
	require.Contains(t, out.String(), `"runId": "01J00000000000000000000000"`)
}

func TestCommandUsesQueryParamsForGetEndpoint(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{},"metadata":{}}`))
	}))
	defer srv.Close()

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", srv.URL,
		"get-function-trace",
		"--run-id", "01J00000000000000000000000",
		"--include-output",
	})

	require.NoError(t, err)
	require.Equal(t, "includeOutput=true", gotQuery)
}

func TestCommandAcceptsRFC3339TimestampQueryFlags(t *testing.T) {
	var gotQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"metadata":{},"page":{}}`))
	}))
	defer srv.Close()

	cmd := Command()
	cmd.Writer = &bytes.Buffer{}

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", srv.URL,
		"get-function-runs",
		"--status", "FAILED",
		"--from", "2026-07-20T00:00:00-04:00",
		"--until", "2026-07-20T23:59:59.999999999-04:00",
		"--time-field", "queuedAt",
		"--order", "DESC",
		"--limit", "20",
	})

	require.NoError(t, err)
	require.Equal(t, []string{"FAILED"}, gotQuery["status"])
	require.Equal(t, "2026-07-20T00:00:00-04:00", gotQuery.Get("from"))
	require.Equal(t, "2026-07-20T23:59:59.999999999-04:00", gotQuery.Get("until"))
	require.Equal(t, "queuedAt", gotQuery.Get("timeField"))
	require.Equal(t, "DESC", gotQuery.Get("order"))
	require.Equal(t, "20", gotQuery.Get("limit"))
}

func TestParseTimestamp(t *testing.T) {
	t.Run("plain RFC 3339", func(t *testing.T) {
		value, err := parseTimestamp("2026-07-20T00:00:00-04:00")

		require.NoError(t, err)
		require.Equal(t, "2026-07-20T00:00:00-04:00", value)
	})

	t.Run("JSON string", func(t *testing.T) {
		value, err := parseTimestamp(`"2026-07-20T00:00:00-04:00"`)

		require.NoError(t, err)
		require.Equal(t, "2026-07-20T00:00:00-04:00", value)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := parseTimestamp("yesterday")

		require.Error(t, err)
	})
}

func TestCommandAcceptsPositionalPathParams(t *testing.T) {
	var gotPath string
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{},"metadata":{}}`))
	}))
	defer srv.Close()

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", srv.URL,
		"get-function-trace",
		"01J00000000000000000000000",
		"--include-output",
	})

	require.NoError(t, err)
	require.Equal(t, "/api/v2/runs/01J00000000000000000000000/trace", gotPath)
	require.Equal(t, "includeOutput=true", gotQuery)
}

func TestCommandAcceptsMultiplePositionalPathParams(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"runId":"01J00000000000000000000000"}}`))
	}))
	defer srv.Close()

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", srv.URL,
		"invoke-function",
		"my app",
		"hello/world",
		"--data", `{"message":"hi"}`,
	})

	require.NoError(t, err)
	require.Equal(t, "/api/v2/apps/my%20app/functions/hello%2Fworld/invoke", gotPath)
	require.Equal(t, map[string]any{
		"data": map[string]any{"message": "hi"},
	}, gotBody)
}

func TestCommandFlagOverridesPositionalPathParam(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{},"metadata":{}}`))
	}))
	defer srv.Close()

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", srv.URL,
		"get-function-trace",
		"positional-id",
		"--run-id", "flag-id",
	})

	require.NoError(t, err)
	require.Equal(t, "/api/v2/runs/flag-id/trace", gotPath)
}

func TestCommandMissingPathParamReportsBothInputs(t *testing.T) {
	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", "http://localhost:1",
		"get-function-trace",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required --run-id or positional argument <run-id>")
}

func TestCommandRejectsExtraPositionalArgs(t *testing.T) {
	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", "http://localhost:1",
		"get-function-trace",
		"01J00000000000000000000000",
		"extra-junk",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected positional argument(s): extra-junk")
}

func TestCommandUsesAPIPortForAPIHost(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{},"metadata":{}}`))
	}))
	defer srv.Close()

	srvURL, err := url.Parse(srv.URL)
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(srvURL.Host)
	require.NoError(t, err)
	apiPort, err := strconv.Atoi(port)
	require.NoError(t, err)

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err = cmd.Run(context.Background(), []string{
		"api",
		"--api-host", "127.0.0.1",
		"--api-port", strconv.Itoa(apiPort),
		"health",
	})

	require.NoError(t, err)
	require.Equal(t, "/api/v2/health", gotPath)
}

func TestResolveBaseURLProdUsesCloud(t *testing.T) {
	var baseURL string
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				baseURL, err = resolveBaseURL(ctx, cmd)
				return err
			},
		},
	}

	err := cmd.Run(context.Background(), []string{
		"api",
		"--prod",
		"capture",
	})

	require.NoError(t, err)
	require.Equal(t, cloudAPIURL, baseURL)
}

func TestResolveBaseURLCustomTargetOverridesProd(t *testing.T) {
	var baseURL string
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				baseURL, err = resolveBaseURL(ctx, cmd)
				return err
			},
		},
	}

	err := cmd.Run(context.Background(), []string{
		"api",
		"--prod",
		"--api-host", "http://localhost:1",
		"capture",
	})

	require.NoError(t, err)
	require.Equal(t, "http://localhost:1/api/v2", baseURL)
}

func TestResolveBaseURLAppliesAPIPortToURLHostWithoutPort(t *testing.T) {
	var baseURL string
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				baseURL, err = resolveBaseURL(ctx, cmd)
				return err
			},
		},
	}

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", "http://127.0.0.1",
		"--api-port", "8090",
		"capture",
	})

	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:8090/api/v2", baseURL)
}

func TestResolveBaseURLDoesNotApplyDefaultPortToURLHost(t *testing.T) {
	var baseURL string
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				baseURL, err = resolveBaseURL(ctx, cmd)
				return err
			},
		},
	}

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", "https://inngest.example.com",
		"capture",
	})

	require.NoError(t, err)
	require.Equal(t, "https://inngest.example.com/api/v2", baseURL)
}

func TestResolveBaseURLAPIPortOverridesProd(t *testing.T) {
	var baseURL string
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				baseURL, err = resolveBaseURL(ctx, cmd)
				return err
			},
		},
	}

	err := cmd.Run(context.Background(), []string{
		"api",
		"--prod",
		"--api-port", "9999",
		"capture",
	})

	require.NoError(t, err)
	require.Equal(t, "http://localhost:9999/api/v2", baseURL)
}

func TestResolveBaseURLDefaultsToDevServer(t *testing.T) {
	t.Setenv("INNGEST_API_HOST", "")
	t.Setenv("INNGEST_API_PORT", "")
	t.Setenv("INNGEST_PROD", "")

	var baseURL string
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				baseURL, err = resolveBaseURL(ctx, cmd)
				return err
			},
		},
	}

	err := cmd.Run(context.Background(), []string{
		"api",
		"capture",
	})

	require.NoError(t, err)
	require.Equal(t, defaultDevServerURL, baseURL)
}

func TestCommandPrefersAPIKeyOverSigningKeyEnv(t *testing.T) {
	t.Setenv("INNGEST_API_KEY", "sk-inn-api-test")
	t.Setenv("INNGEST_SIGNING_KEY", "signkey-test")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{},"metadata":{}}`))
	}))
	defer srv.Close()

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"health",
		"--api-host", srv.URL,
	})

	require.NoError(t, err)
	require.Equal(t, "Bearer sk-inn-api-test", gotAuth)
}

func TestNormalizeAPIURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{
			name:     "official cloud origin uses hosted v2 path",
			rawURL:   "https://api.inngest.com",
			expected: "https://api.inngest.com/v2",
		},
		{
			name:     "self hosted origin uses oss api v2 path",
			rawURL:   "https://inngest.example.com",
			expected: "https://inngest.example.com/api/v2",
		},
		{
			name:     "local origin uses oss api v2 path",
			rawURL:   "http://localhost:8288",
			expected: "http://localhost:8288/api/v2",
		},
		{
			name:     "existing v2 path is preserved",
			rawURL:   "https://inngest.example.com/v2",
			expected: "https://inngest.example.com/v2",
		},
		{
			name:     "existing api v2 path is preserved",
			rawURL:   "https://inngest.example.com/api/v2",
			expected: "https://inngest.example.com/api/v2",
		},
		{
			name:     "existing api path is completed",
			rawURL:   "https://inngest.example.com/api",
			expected: "https://inngest.example.com/api/v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := normalizeAPIURL(tt.rawURL)
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestNormalizeAPIHostTarget(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		port     int
		expected string
	}{
		{
			name:     "local host adds server port and uses http",
			rawURL:   "localhost",
			port:     9999,
			expected: "http://localhost:9999/api/v2",
		},
		{
			name:     "non-local host without scheme defaults to https",
			rawURL:   "inngest.example.com",
			port:     8288,
			expected: "https://inngest.example.com/api/v2",
		},
		{
			name:     "host with explicit port preserves port",
			rawURL:   "inngest.example.com:9443",
			port:     8288,
			expected: "https://inngest.example.com:9443/api/v2",
		},
		{
			name:     "origin value uses api v2 path",
			rawURL:   "http://127.0.0.1:9999",
			port:     8288,
			expected: "http://127.0.0.1:9999/api/v2",
		},
		{
			name:     "url host without port uses api port",
			rawURL:   "http://127.0.0.1",
			port:     8090,
			expected: "http://127.0.0.1:8090/api/v2",
		},
		{
			name:     "url host with explicit path and missing port uses api port",
			rawURL:   "http://127.0.0.1/v2",
			port:     8090,
			expected: "http://127.0.0.1:8090/v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := normalizeAPIHostTarget(tt.rawURL, tt.port)
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}
