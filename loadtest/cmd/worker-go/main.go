// Command worker-go is the SDK worker subprocess spawned by the harness.
// One worker process hosts one Inngest "app" and all functions for the
// configured shapes.
//
// Wire contract (so a TS worker can slot in later):
//   - stdin: one JSON object of type config.WorkerConfig
//   - stdout/stderr: human-readable logs, captured by the harness and
//     shown in the UI when the run fails
//   - lifecycle: worker emits a telemetry frame with phase="ready" once it
//     has successfully registered with the target Inngest server
//   - shutdown: SIGTERM → drain the telemetry ring → exit 0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/inngest/inngest/loadtest/internal/config"
	"github.com/inngest/inngest/loadtest/internal/shapes"
	"github.com/inngest/inngest/loadtest/internal/telemetry"
	"github.com/inngest/inngestgo"
)

func main() {
	// stderr is captured by the harness and surfaced in the UI; keep it
	// structured enough that a user can skim and identify where we died.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("[worker] ")
	if err := run(); err != nil {
		log.Printf("FATAL: %v", err)
		os.Exit(1)
	}
}

func run() error {
	var cfg config.WorkerConfig
	if err := json.NewDecoder(os.Stdin).Decode(&cfg); err != nil {
		return fmt.Errorf("decode stdin config: %w", err)
	}
	if cfg.AppID == "" || cfg.Target.URL == "" || cfg.TelemetrySocket == "" {
		return fmt.Errorf("incomplete config: appID/target.url/telemetrySocket required")
	}
	log.Printf("config: worker=%s app=%s target=%s mode=%s hasEventKey=%t hasSigningKey=%t",
		cfg.WorkerID, cfg.AppID, cfg.Target.URL, cfg.Target.Mode,
		cfg.Target.EventKey != nil, cfg.Target.SigningKey != nil)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	log.Printf("dialing telemetry socket: %s", cfg.TelemetrySocket)
	tc, err := telemetry.Dial(ctx, cfg.TelemetrySocket, cfg.WorkerID, 4096)
	if err != nil {
		return fmt.Errorf("dial telemetry: %w", err)
	}
	defer tc.Close()

	registerURL := cfg.Target.URL + "/fn/register"
	eventKey := cfg.Target.EventKey
	if eventKey == nil {
		k := "test"
		eventKey = &k
	}

	// Mode selects the signature-verification posture in the SDK handler.
	//   - ModeDev: set INNGEST_DEV; SDK short-circuits ValidateRequestSignature
	//     to true (inngestgo signature.go:118). Required for `inngest dev`,
	//     which may send unsigned or differently-signed invoke requests.
	//   - ModeSelfHosted: leave INNGEST_DEV unset; SDK performs strict
	//     verification using the supplied SigningKey, which must match what
	//     `inngest start --signing-key` was given.
	switch cfg.Target.Mode {
	case config.ModeSelfHosted:
		log.Printf("mode=selfhosted: strict signature verification (SigningKey required)")
	default:
		_ = os.Setenv("INNGEST_DEV", cfg.Target.URL)
		log.Printf("mode=dev: signature verification skipped (INNGEST_DEV=%s)", cfg.Target.URL)
	}

	log.Printf("creating inngestgo client (registerURL=%s)", registerURL)
	client, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:           cfg.AppID,
		EventKey:        eventKey,
		SigningKey:      cfg.Target.SigningKey,
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		RegisterURL:     strPtr(registerURL),
		EventAPIBaseURL: strPtr(cfg.Target.URL),
		APIBaseURL:      strPtr(cfg.Target.URL),
	})
	if err != nil {
		return fmt.Errorf("inngestgo.NewClient: %w", err)
	}

	log.Printf("registering %d shape(s): %v", len(cfg.Shapes), cfg.Shapes)
	if err := shapes.Register(client, tc, cfg.AppID, cfg.Shapes); err != nil {
		return fmt.Errorf("register shapes: %w", err)
	}

	port := cfg.HTTPPort
	lis, err := listenPort(port)
	if err != nil {
		return fmt.Errorf("listen %d: %w", port, err)
	}
	actualPort := lis.Addr().(*net.TCPAddr).Port
	selfURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d/", actualPort))
	client.SetURL(selfURL)
	log.Printf("SDK HTTP handler listening on %s", selfURL)

	srv := &http.Server{
		Handler:      client.Serve(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(lis) }()

	// Give the listener a moment to be ready before poking it.
	time.Sleep(50 * time.Millisecond)

	// Trigger SDK self-registration by PUT-ing the worker's own URL. The
	// SDK handles the PUT by POSTing function metadata to the registerURL.
	log.Printf("triggering SDK self-registration via PUT %s", selfURL)
	if err := triggerRegister(selfURL.String()); err != nil {
		return fmt.Errorf("trigger register (target=%s eventKey=%s): %w", cfg.Target.URL, safeKey(eventKey), err)
	}
	log.Printf("registration complete; target accepted the sync")

	// Signal ready.
	tc.Emit(telemetry.PhaseReady, "", "", "", 0)
	log.Printf("signalled ready to harness")

	select {
	case <-ctx.Done():
		log.Printf("received shutdown signal")
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http serve: %w", err)
		}
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
	tc.Close()
	log.Printf("shutdown complete")
	return nil
}

func listenPort(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
}

func triggerRegister(selfURL string) error {
	req, err := http.NewRequest(http.MethodPut, selfURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("register returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func strPtr(s string) *string { return &s }

// safeKey masks the middle of an event key so logs don't leak secrets but
// still show enough to tell "oh that's not the key I set".
func safeKey(p *string) string {
	if p == nil {
		return "<nil>"
	}
	s := *p
	if len(s) <= 6 {
		return s
	}
	return s[:3] + "…" + s[len(s)-3:]
}
