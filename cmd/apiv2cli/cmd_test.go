package apiv2cli

import (
	"bytes"
	"context"
	"net"
	"strconv"
	"sync"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	require.Equal(t, "POST", byName["invoke-function"].httpMethod)
	require.Equal(t, "/apps/{app_id}/functions/{function_id}/invoke", byName["invoke-function"].path)
	require.Equal(t, "/api.v2.V2/InvokeFunction", byName["invoke-function"].fullMethod)
	require.Equal(t, "InvokeFunction", byName["invoke-function"].methodName)
	require.Equal(t, "GET", byName["get-function-trace"].httpMethod)
	require.Equal(t, "/runs/{run_id}/trace", byName["get-function-trace"].path)
	require.Equal(t, "/api.v2.V2/GetFunctionTrace", byName["get-function-trace"].fullMethod)
}

func TestEndpointCommandNameNormalizesReadVerbs(t *testing.T) {
	require.Equal(t, "get-account", endpointCommandName("FetchAccount"))
	require.Equal(t, "get-webhooks", endpointCommandName("ListWebhooks"))
	require.Equal(t, "get-function-run", endpointCommandName("GetFunctionRun"))
	require.Equal(t, "create-env", endpointCommandName("CreateEnv"))
}

// recordingServer captures the last request and metadata it received so tests
// can assert on the request shape and auth/env headers without poking at the
// real implementation.
type recordingServer struct {
	apiv2.UnimplementedV2Server

	mu         sync.Mutex
	lastInvoke *apiv2.InvokeFunctionRequest
	lastTrace  *apiv2.GetFunctionTraceRequest
	lastHealth *apiv2.HealthRequest
	lastMD     metadata.MD
	invokeResp *apiv2.InvokeFunctionResponse
	traceResp  *apiv2.GetFunctionTraceResponse
	healthResp *apiv2.HealthResponse
}

func (s *recordingServer) capture(ctx context.Context) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		s.lastMD = md.Copy()
	}
}

func (s *recordingServer) InvokeFunction(ctx context.Context, req *apiv2.InvokeFunctionRequest) (*apiv2.InvokeFunctionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastInvoke = req
	s.capture(ctx)
	return s.invokeResp, nil
}

func (s *recordingServer) GetFunctionTrace(ctx context.Context, req *apiv2.GetFunctionTraceRequest) (*apiv2.GetFunctionTraceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastTrace = req
	s.capture(ctx)
	return s.traceResp, nil
}

func (s *recordingServer) Health(ctx context.Context, req *apiv2.HealthRequest) (*apiv2.HealthResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastHealth = req
	s.capture(ctx)
	return s.healthResp, nil
}

func startRecordingServer(t *testing.T, svc *recordingServer) (string, int) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer()
	apiv2.RegisterV2Server(srv, svc)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	host, portStr, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	return host, port
}

func TestCommandCallsGeneratedEndpoint(t *testing.T) {
	svc := &recordingServer{
		invokeResp: &apiv2.InvokeFunctionResponse{
			Data: &apiv2.InvokeFunctionData{RunId: "01J00000000000000000000000"},
		},
	}
	host, port := startRecordingServer(t, svc)

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", host,
		"--api-port", strconv.Itoa(port),
		"--signing-key", "signkey-test-abc",
		"--env", "branch-a",
		"invoke-function",
		"--app-id", "my app",
		"--function-id", "hello/world",
		"--data", `{"message":"hi"}`,
		"--idempotency-key", "idem-1",
	})

	require.NoError(t, err)
	require.NotNil(t, svc.lastInvoke)
	require.Equal(t, "my app", svc.lastInvoke.GetAppId())
	require.Equal(t, "hello/world", svc.lastInvoke.GetFunctionId())
	require.Equal(t, "idem-1", svc.lastInvoke.GetIdempotencyKey())
	require.Equal(t, "hi", svc.lastInvoke.GetData().GetFields()["message"].GetStringValue())

	require.Equal(t, []string{"Bearer signkey-test-abc"}, svc.lastMD.Get("authorization"))
	require.Equal(t, []string{"branch-a"}, svc.lastMD.Get("x-inngest-env"))
	require.Contains(t, out.String(), `01J00000000000000000000000`)
}

func TestCommandSetsOptionalFlagOnRequest(t *testing.T) {
	svc := &recordingServer{traceResp: &apiv2.GetFunctionTraceResponse{}}
	host, port := startRecordingServer(t, svc)

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", host,
		"--api-port", strconv.Itoa(port),
		"get-function-trace",
		"--run-id", "01J00000000000000000000000",
		"--include-output",
	})

	require.NoError(t, err)
	require.NotNil(t, svc.lastTrace)
	require.Equal(t, "01J00000000000000000000000", svc.lastTrace.GetRunId())
	require.True(t, svc.lastTrace.GetIncludeOutput())
}

func TestCommandUsesAPIPortForLoopbackHost(t *testing.T) {
	svc := &recordingServer{healthResp: &apiv2.HealthResponse{}}
	_, port := startRecordingServer(t, svc)

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", "127.0.0.1",
		"--api-port", strconv.Itoa(port),
		"health",
	})

	require.NoError(t, err)
	require.NotNil(t, svc.lastHealth)
}

func TestResolveTargetProdUsesCloud(t *testing.T) {
	var (
		gotTarget string
		gotTLS    bool
	)
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				gotTarget, gotTLS, err = resolveTarget(ctx, cmd)
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
	require.Equal(t, cloudGRPCTarget, gotTarget)
	require.True(t, gotTLS)
}

func TestResolveTargetCustomHostOverridesProd(t *testing.T) {
	var gotTarget string
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				gotTarget, _, err = resolveTarget(ctx, cmd)
				return err
			},
		},
	}

	err := cmd.Run(context.Background(), []string{
		"api",
		"--prod",
		"--api-host", "localhost:1",
		"capture",
	})

	require.NoError(t, err)
	require.Equal(t, "localhost:1", gotTarget)
}

func TestResolveTargetAPIPortOverridesProd(t *testing.T) {
	var (
		gotTarget string
		gotTLS    bool
	)
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				gotTarget, gotTLS, err = resolveTarget(ctx, cmd)
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
	require.Equal(t, "localhost:9999", gotTarget)
	require.False(t, gotTLS)
}

func TestResolveTargetDefaultsToDevServer(t *testing.T) {
	t.Setenv("INNGEST_API_HOST", "")
	t.Setenv("INNGEST_API_PORT", "")
	t.Setenv("INNGEST_PROD", "")

	var gotTarget string
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				gotTarget, _, err = resolveTarget(ctx, cmd)
				return err
			},
		},
	}

	err := cmd.Run(context.Background(), []string{
		"api",
		"capture",
	})

	require.NoError(t, err)
	require.Equal(t, "localhost:50051", gotTarget)
}

func TestResolveTargetUsesAPIPortEnv(t *testing.T) {
	t.Setenv("INNGEST_API_HOST", "")
	t.Setenv("INNGEST_API_PORT", "50505")
	t.Setenv("INNGEST_PROD", "")

	var gotTarget string
	cmd := Command()
	cmd.Commands = []*cli.Command{
		{
			Name: "capture",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				var err error
				gotTarget, _, err = resolveTarget(ctx, cmd)
				return err
			},
		},
	}

	err := cmd.Run(context.Background(), []string{
		"api",
		"capture",
	})

	require.NoError(t, err)
	require.Equal(t, "localhost:50505", gotTarget)
}

func TestCommandPrefersAPIKeyOverSigningKeyEnv(t *testing.T) {
	t.Setenv("INNGEST_API_KEY", "sk-inn-api-test")
	t.Setenv("INNGEST_SIGNING_KEY", "signkey-test")

	svc := &recordingServer{healthResp: &apiv2.HealthResponse{}}
	host, port := startRecordingServer(t, svc)

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", host,
		"--api-port", strconv.Itoa(port),
		"health",
	})

	require.NoError(t, err)
	require.Equal(t, []string{"Bearer sk-inn-api-test"}, svc.lastMD.Get("authorization"))
}

func TestBuildTarget(t *testing.T) {
	tests := []struct {
		name       string
		rawHost    string
		port       int
		insecure   bool
		wantTarget string
		wantTLS    bool
	}{
		{
			name:       "local host with explicit port stays insecure",
			rawHost:    "localhost",
			port:       9999,
			wantTarget: "localhost:9999",
			wantTLS:    false,
		},
		{
			name:       "non-local host defaults to TLS gRPC port",
			rawHost:    "inngest.example.com",
			wantTarget: "inngest.example.com:443",
			wantTLS:    true,
		},
		{
			name:       "host:port pair preserves port",
			rawHost:    "inngest.example.com:9443",
			wantTarget: "inngest.example.com:9443",
			wantTLS:    true,
		},
		{
			name:       "url form is parsed to host:port",
			rawHost:    "https://inngest.example.com:9443",
			wantTarget: "inngest.example.com:9443",
			wantTLS:    true,
		},
		{
			name:       "insecure flag disables TLS even for remote host",
			rawHost:    "inngest.example.com",
			port:       1234,
			insecure:   true,
			wantTarget: "inngest.example.com:1234",
			wantTLS:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, useTLS, err := buildTarget(tt.rawHost, tt.port, tt.insecure)
			require.NoError(t, err)
			require.Equal(t, tt.wantTarget, target)
			require.Equal(t, tt.wantTLS, useTLS)
		})
	}
}

func TestRefusePlaintextCredsToRemoteHost(t *testing.T) {
	svc := &recordingServer{healthResp: &apiv2.HealthResponse{}}
	host, port := startRecordingServer(t, svc)
	_ = host

	cmd := Command()
	out := bytes.Buffer{}
	cmd.Writer = &out

	err := cmd.Run(context.Background(), []string{
		"api",
		"--api-host", "inngest.example.com",
		"--api-port", strconv.Itoa(port),
		"--insecure",
		"--signing-key", "signkey-test",
		"health",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to send credentials over plaintext")
}
