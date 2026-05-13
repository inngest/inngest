//go:build e2e_loki

// Package loki contains end-to-end tests for the spans-as-logs side pipeline.
//
// These tests boot a Grafana LGTM container (Loki + Tempo + Grafana, all in
// one), spawn a real `inngest dev` subprocess pointed at the container's
// OTLP/HTTP logs endpoint via INNGEST_OTEL_LOGS_ENDPOINT, register an SDK
// function in-process, send an event, wait for the run to finish, and then
// assert via Loki's HTTP query API that the lifecycle hooks emitted the
// expected log records.
//
// Run with:
//
//	go test -tags=e2e_loki -v -timeout=300s ./tests/loki/...
//
// Requires Docker. First run pulls grafana/otel-lgtm (~500MB) and builds the
// inngest binary.
package loki

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/telemetry/exporters"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
	grafanalgtm "github.com/testcontainers/testcontainers-go/modules/grafana-lgtm"
)

func TestSpansAsLogs_E2E(t *testing.T) {
	bootCtx, bootCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer bootCancel()

	container, err := grafanalgtm.Run(bootCtx, "grafana/otel-lgtm:0.8.1")
	require.NoError(t, err, "start grafana-lgtm")
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	otlpEndpoint, err := container.OtlpHttpEndpoint(bootCtx)
	require.NoError(t, err)
	lokiEndpoint, err := container.LokiEndpoint(bootCtx)
	require.NoError(t, err)
	t.Logf("OTLP HTTP: %s | Loki: %s", otlpEndpoint, lokiEndpoint)

	devPort := freePort(t)
	dev := startDevServer(t, devPort, otlpEndpoint)
	t.Cleanup(dev.stop)

	c := client.New(t)
	c.APIHost = fmt.Sprintf("http://127.0.0.1:%d", devPort)

	t.Run("success_path", func(t *testing.T) {
		evt := fmt.Sprintf("loki/test.success.%d", time.Now().UnixNano())
		appID := fmt.Sprintf("loki-success-%d", time.Now().UnixNano())

		var capturedRunID string
		client, server, register := newSDKApp(t, devPort, appID)
		defer server.Close()

		_, err := inngestgo.CreateFunction(
			client,
			inngestgo.FunctionOpts{ID: "fn-success"},
			inngestgo.EventTrigger(evt, nil),
			func(ctx context.Context, in inngestgo.Input[any]) (any, error) {
				capturedRunID = in.InputCtx.RunID
				_, err := step.Run(ctx, "do-work", func(ctx context.Context) (string, error) {
					return "ok", nil
				})
				return "done", err
			},
		)
		require.NoError(t, err)
		register()

		_, err = client.Send(context.Background(), inngestgo.Event{Name: evt, Data: map[string]any{"hi": "world"}})
		require.NoError(t, err)

		require.Eventually(t, func() bool { return capturedRunID != "" }, 30*time.Second, 100*time.Millisecond,
			"function never executed")
		c.WaitForRunStatus(context.Background(), t, "COMPLETED", capturedRunID,
			client_WaitOpts{Timeout: 60 * time.Second}.toClient())

		records, _ := queryLoki(t, lokiEndpoint,
			fmt.Sprintf(`{service_name="tracing"} | json | sdk_run_id=%q`, capturedRunID),
			3, 60*time.Second,
		)
		assertLogTypes(t, records, map[string]int{
			exporters.InngestLogTypeRunStarted:    1,
			exporters.InngestLogTypeStepEnded:     1,
			exporters.InngestLogTypeFunctionEnded: 1,
		})
		// All severities should be INFO on the success path.
		for _, rec := range records {
			if got := stringFrom(rec, "severity_text"); got != "INFO" {
				t.Fatalf("record severity_text=%q; want INFO. body: %#v", got, rec)
			}
		}
	})

	t.Run("failure_path", func(t *testing.T) {
		evt := fmt.Sprintf("loki/test.failure.%d", time.Now().UnixNano())
		appID := fmt.Sprintf("loki-failure-%d", time.Now().UnixNano())

		var capturedRunID string
		client, server, register := newSDKApp(t, devPort, appID)
		defer server.Close()

		_, err := inngestgo.CreateFunction(
			client,
			inngestgo.FunctionOpts{
				ID:      "fn-failure",
				Retries: inngestgo.IntPtr(0),
			},
			inngestgo.EventTrigger(evt, nil),
			func(ctx context.Context, in inngestgo.Input[any]) (any, error) {
				capturedRunID = in.InputCtx.RunID
				return nil, errors.New("boom")
			},
		)
		require.NoError(t, err)
		register()

		_, err = client.Send(context.Background(), inngestgo.Event{Name: evt, Data: map[string]any{}})
		require.NoError(t, err)

		require.Eventually(t, func() bool { return capturedRunID != "" }, 30*time.Second, 100*time.Millisecond,
			"function never executed")
		c.WaitForRunStatus(context.Background(), t, "FAILED", capturedRunID,
			client_WaitOpts{Timeout: 60 * time.Second}.toClient())

		records, _ := queryLoki(t, lokiEndpoint,
			fmt.Sprintf(`{service_name="tracing"} | json | sdk_run_id=%q | inngest_log_type="function.ended"`, capturedRunID),
			1, 60*time.Second,
		)
		if got := stringFrom(records[0], "severity_text"); got != "ERROR" {
			t.Fatalf("function.ended severity_text=%q; want ERROR. body: %#v", got, records[0])
		}
	})

	t.Run("durable_step_resumed", func(t *testing.T) {
		evt := fmt.Sprintf("loki/test.durable.%d", time.Now().UnixNano())
		resumeEvt := fmt.Sprintf("loki/test.durable.resume.%d", time.Now().UnixNano())
		appID := fmt.Sprintf("loki-durable-%d", time.Now().UnixNano())

		var capturedRunID string
		client, server, register := newSDKApp(t, devPort, appID)
		defer server.Close()

		_, err := inngestgo.CreateFunction(
			client,
			inngestgo.FunctionOpts{ID: "fn-durable"},
			inngestgo.EventTrigger(evt, nil),
			func(ctx context.Context, in inngestgo.Input[any]) (any, error) {
				capturedRunID = in.InputCtx.RunID
				_, err := step.WaitForEvent[any](ctx, "wait-resume", step.WaitForEventOpts{
					Event:   resumeEvt,
					Timeout: 30 * time.Second,
				})
				return "done", err
			},
		)
		require.NoError(t, err)
		register()

		_, err = client.Send(context.Background(), inngestgo.Event{Name: evt, Data: map[string]any{}})
		require.NoError(t, err)
		require.Eventually(t, func() bool { return capturedRunID != "" }, 30*time.Second, 100*time.Millisecond,
			"function never executed")

		// Give the wait a moment to register before we resolve it.
		time.Sleep(1 * time.Second)
		_, err = client.Send(context.Background(), inngestgo.Event{Name: resumeEvt, Data: map[string]any{}})
		require.NoError(t, err)

		c.WaitForRunStatus(context.Background(), t, "COMPLETED", capturedRunID,
			client_WaitOpts{Timeout: 60 * time.Second}.toClient())

		// Look for a step.ended record whose opcode is WaitForEvent — that's
		// how the runtime surfaces a resolved durable wait.
		records, _ := queryLoki(t, lokiEndpoint,
			fmt.Sprintf(`{service_name="tracing"} | json | sdk_run_id=%q | inngest_log_type="step.ended"`,
				capturedRunID),
			1, 60*time.Second,
		)
		var foundWait bool
		for _, rec := range records {
			if stringFrom(rec, "sys.step.opcode") == "WaitForEvent" {
				foundWait = true
				break
			}
		}
		if !foundWait {
			t.Fatalf("no step.ended record with sys.step.opcode=WaitForEvent for run %s.\nrecords: %#v",
				capturedRunID, records)
		}
	})
}

// ---------- helpers ----------

// devProcess wraps a subprocess running `inngest dev`. It captures stdout/err
// for diagnostics and provides a clean stop.
type devProcess struct {
	cmd     *exec.Cmd
	logPath string
	stopper context.CancelFunc
}

func (d *devProcess) stop() {
	d.stopper()
	_ = d.cmd.Wait()
}

func startDevServer(t *testing.T, port int, otlpEndpoint string) *devProcess {
	t.Helper()

	repoRoot := repoRoot(t)
	binPath := filepath.Join(t.TempDir(), "inngest-dev")

	t.Logf("building inngest binary…")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd")
	buildCmd.Dir = repoRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	require.NoError(t, buildCmd.Run(), "build inngest binary")

	logFile, err := os.CreateTemp(t.TempDir(), "dev-*.log")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, binPath, "dev",
		"--port", fmt.Sprint(port),
		"--no-discovery",
		"--no-poll",
	)
	cmd.Env = append(os.Environ(),
		"INNGEST_OTEL_LOGS_ENDPOINT="+otlpEndpoint,
		// Spans-as-logs ships under the user tracer service name "tracing".
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	t.Logf("starting `inngest dev --port=%d` (logs: %s)", port, logFile.Name())
	require.NoError(t, cmd.Start(), "start dev server")

	// Wait for /health.
	deadline := time.Now().Add(60 * time.Second)
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			t.Logf("dev server healthy on :%d", port)
			return &devProcess{cmd: cmd, logPath: logFile.Name(), stopper: cancel}
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	cancel()
	_ = cmd.Wait()
	logBytes, _ := os.ReadFile(logFile.Name())
	t.Fatalf("dev server failed to become healthy on :%d. logs:\n%s", port, string(logBytes))
	return nil
}

// newSDKApp creates an inngestgo client whose registration target points at
// our random-port dev server. The returned register() walks the registration
// handshake.
func newSDKApp(t *testing.T, devPort int, appID string) (inngestgo.Client, *appHTTPServer, func()) {
	t.Helper()

	devURL := fmt.Sprintf("http://127.0.0.1:%d", devPort)
	t.Setenv("INNGEST_DEV", devURL)

	key := "test"
	sdk, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:       appID,
		EventKey:    &key,
		Logger:      slog.New(slog.DiscardHandler),
		RegisterURL: inngestgo.StrPtr(devURL + "/fn/register"),
	})
	require.NoError(t, err)

	srv := newAppHTTPServer(t, sdk.Serve())
	u, _ := url.Parse(srv.URL())
	sdk.SetURL(u)

	register := func() {
		t.Helper()
		req, err := http.NewRequest(http.MethodPut, srv.URL(), nil)
		require.NoError(t, err)
		req.Close = true
		resp, err := (&http.Client{
			Transport: &http.Transport{DisableKeepAlives: true},
		}).Do(req)
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		require.Equal(t, 200, resp.StatusCode, "PUT to SDK app to trigger registration")
	}
	return sdk, srv, register
}

type appHTTPServer struct {
	server *http.Server
	port   int
}

func newAppHTTPServer(t *testing.T, h http.Handler) *appHTTPServer {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())

	s := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: h}
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("sdk app server: %v", err)
		}
	}()
	time.Sleep(50 * time.Millisecond)
	return &appHTTPServer{server: s, port: port}
}

func (a *appHTTPServer) URL() string  { return fmt.Sprintf("http://127.0.0.1:%d/", a.port) }
func (a *appHTTPServer) Close() error { return a.server.Close() }

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// file = .../tests/loki/e2e_test.go ; root is two dirs up.
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// client_WaitOpts wraps the client.WaitForRunStatusOpts so we can pass it
// without leaking an import cycle.
type client_WaitOpts struct {
	Timeout time.Duration
}

func (c client_WaitOpts) toClient() client.WaitForRunStatusOpts {
	return client.WaitForRunStatusOpts{Timeout: c.Timeout}
}

func assertLogTypes(t *testing.T, records []map[string]any, want map[string]int) {
	t.Helper()
	got := map[string]int{}
	for _, rec := range records {
		got[stringFrom(rec, exporters.InngestLogTypeKey)]++
	}
	for typ, n := range want {
		if got[typ] != n {
			t.Fatalf("log type %q: got %d records, want %d. all records: %#v", typ, got[typ], n, records)
		}
	}
}

func stringFrom(rec map[string]any, key string) string {
	if v, ok := rec[key].(string); ok {
		return v
	}
	return ""
}
