package apiv2cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	require.Equal(t, http.MethodPost, byName["invoke-function"].method)
	require.Equal(t, "/apps/{app_id}/functions/{function_id}/invoke", byName["invoke-function"].path)
	require.Equal(t, []string{"app_id", "function_id"}, byName["invoke-function"].pathParams)
	require.Equal(t, http.MethodGet, byName["get-function-trace"].method)
	require.Equal(t, "/runs/{run_id}/trace", byName["get-function-trace"].path)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := normalizeAPIHostTarget(tt.rawURL, tt.port)
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}
