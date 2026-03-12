package loadtest

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
)

// RunMatrix executes all combinations of configs, workloads, and profiles.
// Each scenario runs sequentially for clean measurements.
func RunMatrix(t *testing.T, configs []SystemConfig, workloads []Workload, profiles []LoadProfile) MatrixResult {
	t.Helper()

	result := MatrixResult{
		RunAt:     time.Now(),
		GitCommit: gitCommit(),
	}

	for _, cfg := range configs {
		t.Run(cfg.Name, func(t *testing.T) {
			server := StartServer(t, cfg)

			for _, wl := range workloads {
				t.Run(wl.Name, func(t *testing.T) {
					for _, lp := range profiles {
						t.Run(lp.Name, func(t *testing.T) {
							scenarioName := fmt.Sprintf("%s/%s/%s", cfg.Name, wl.Name, lp.Name)
							sr := runScenario(t, server, wl, lp, scenarioName)
							result.Results = append(result.Results, sr)
							t.Logf("Result: %s — completed=%d/%d p50=%.1fms p95=%.1fms throughput=%.1f/s",
								scenarioName, sr.TotalCompleted, sr.TotalEvents,
								sr.P50E2EMS, sr.P95E2EMS, sr.ThroughputPerSec)
						})
					}
				})
			}
		})
	}

	return result
}

func runScenario(t *testing.T, server *ServerHandle, wl Workload, lp LoadProfile, scenarioName string) ScenarioResult {
	t.Helper()

	// Determine expected completions.
	expectedCompletions := lp.MaxEvents
	if wl.ExpectedCompletions != nil {
		expectedCompletions = wl.ExpectedCompletions(lp.MaxEvents)
	}

	collector := NewCollector(expectedCompletions)

	// Create an SDK client and HTTP server for this workload.
	appID := fmt.Sprintf("loadtest-%s", sanitize(wl.Name))
	client, sdkServer := newSDKClient(t, server, appID)

	// Register the workload's functions.
	wc := newWorkloadClient(client)
	eventName, err := wl.SetupFn(wc, collector)
	if err != nil {
		t.Fatalf("workload setup failed: %v", err)
	}

	// Register functions with the dev server.
	registerFunctions(t, sdkServer)

	// Give the server a moment to process the registration.
	time.Sleep(500 * time.Millisecond)

	// Run the load generator.
	gen := &Generator{
		Client:    client,
		EventName: eventName,
		Profile:   lp,
		Collector: collector,
	}

	start := time.Now()
	ctx := context.Background()
	totalSent, err := gen.Run(ctx)
	if err != nil {
		t.Logf("generator errors: %v", err)
	}

	// Wait for completions with a generous timeout.
	waitTimeout := 60 * time.Second
	if lp.Duration > 0 {
		waitTimeout += lp.Duration
	}
	if err := collector.WaitForAll(waitTimeout); err != nil {
		t.Logf("not all events completed: %v (got %d/%d)", err, collector.CompletedCount(), expectedCompletions)
	}
	elapsed := time.Since(start)

	samples := collector.Samples()
	totalErrors := totalSent - len(samples)
	if totalErrors < 0 {
		totalErrors = 0
	}

	// Clean up the SDK server.
	sdkServer.Close()

	return ComputeResult(scenarioName, samples, totalSent, totalErrors, elapsed)
}

func newSDKClient(t *testing.T, server *ServerHandle, appID string) (inngestgo.Client, *http.Server) {
	t.Helper()
	os.Setenv("INNGEST_DEV", server.BaseURL)

	key := "test"
	opts := inngestgo.ClientOpts{
		AppID:       appID,
		EventKey:    &key,
		Logger:      slog.New(slog.DiscardHandler),
		RegisterURL: inngestgo.StrPtr(fmt.Sprintf("%s/fn/register", server.BaseURL)),
	}

	client, err := inngestgo.NewClient(opts)
	if err != nil {
		t.Fatalf("failed to create SDK client: %v", err)
	}

	// Start an HTTP server for the SDK handler.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	sdkServer := &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		Handler:      client.Serve(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	go func() {
		if err := sdkServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("SDK server error: %v", err)
		}
	}()
	time.Sleep(50 * time.Millisecond)

	// Update the client's URL.
	sdkURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d/", port))
	client.SetURL(sdkURL)

	t.Cleanup(func() {
		sdkServer.Close()
	})

	return client, sdkServer
}

func registerFunctions(t *testing.T, sdkServer *http.Server) {
	t.Helper()
	addr := sdkServer.Addr
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	req, err := http.NewRequest(http.MethodPut, addr, nil)
	if err != nil {
		t.Fatalf("failed to create registration request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to register functions: %v", err)
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("registration returned status %d", resp.StatusCode)
	}
}

func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return '-'
	}, s)
}
