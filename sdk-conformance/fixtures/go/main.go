package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
)

const (
	appID       = "sdk-conformance"
	servePath   = "/api/inngest"
	cloudPath   = "/api/inngest/cloud"
	defaultPort = 3000
	sdkCommit   = "ff799d9758ba"
)

type fixture struct {
	ID         string `json:"id"`
	Behavior   string `json:"behavior"`
	FunctionID string `json:"function_id"`
	Event      string `json:"event"`
}

var fixtures = []fixture{
	{ID: "basic-completion", Behavior: "basic_completion", FunctionID: "sdk-conformance-basic-completion", Event: "conformance/basic-completion"},
	{ID: "step-run", Behavior: "step_run", FunctionID: "sdk-conformance-step-run", Event: "conformance/step-run"},
	{ID: "step-sleep", Behavior: "step_sleep", FunctionID: "sdk-conformance-step-sleep", Event: "conformance/step-sleep"},
	{ID: "wait-for-event", Behavior: "wait_for_event", FunctionID: "sdk-conformance-wait-for-event", Event: "conformance/wait-for-event"},
	{ID: "function-error", Behavior: "function_error", FunctionID: "sdk-conformance-function-error", Event: "conformance/function-error"},
	{ID: "step-error", Behavior: "step_error", FunctionID: "sdk-conformance-step-error", Event: "conformance/step-error"},
}

func main() {
	port, err := numericPort()
	if err != nil {
		log.Fatal(err)
	}

	dev := true
	signingKey := envOr("INNGEST_SIGNING_KEY", "7468697320697320612074657374206b6579")
	signingKeyFallback := envOr("INNGEST_SIGNING_KEY_FALLBACK", "66616c6c6261636b20636f6e666f726d616e6365")
	client, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:      appID,
		EventKey:   stringPointer(envOr("INNGEST_EVENT_KEY", "test")),
		SigningKey: stringPointer(signingKey),
		Dev:        &dev,
	})
	if err != nil {
		log.Fatalf("create Inngest client: %v", err)
	}
	registerFunctions(client)
	cloud := false
	cloudClient, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:              appID,
		EventKey:           stringPointer(envOr("INNGEST_EVENT_KEY", "test")),
		SigningKey:         stringPointer(signingKey),
		SigningKeyFallback: stringPointer(signingKeyFallback),
		Dev:                &cloud,
	})
	if err != nil {
		log.Fatalf("create cloud-mode Inngest client: %v", err)
	}
	registerFunctions(cloudClient)

	mux := http.NewServeMux()
	mux.Handle(servePath, client.Serve())
	mux.Handle(cloudPath, cloudClient.Serve())
	mux.HandleFunc("/__conformance", conformanceHandler)
	mux.HandleFunc("/__conformance/reset", resetHandler)

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	log.Printf("Go SDK conformance fixture listening on http://%s", addr)
	if err := http.ListenAndServe(addr, mux); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func conformanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schema_version": "0.1",
		"sdk": map[string]string{
			"language": "go",
			"version":  inngestgo.SDKVersion,
			"commit":   sdkCommit,
		},
		"profiles":         []string{"serve.http.v1", "serve.execution.v1"},
		"capabilities":     []string{"serve.introspection", "function.basic_completion", "step.run", "step.sleep", "step.wait_for_event", "error.function", "error.step"},
		"serve_path":       servePath,
		"cloud_serve_path": cloudPath,
		"fixtures":         fixtures,
	})
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// All fixture behavior is stateless. Keep this endpoint so the harness can
	// reset every SDK fixture uniformly as stateful cases are added.
	w.WriteHeader(http.StatusNoContent)
}

func registerFunctions(client inngestgo.Client) {
	create(client, "basic-completion", "conformance/basic-completion", func(_ context.Context, input inngestgo.Input[map[string]any]) (any, error) {
		return map[string]any{"kind": "basic-completion", "value": input.Event.Data["value"]}, nil
	})
	create(client, "step-run", "conformance/step-run", func(ctx context.Context, _ inngestgo.Input[map[string]any]) (any, error) {
		result, err := step.Run(ctx, "deterministic-step", func(context.Context) (map[string]string, error) {
			return map[string]string{"value": "step-ran"}, nil
		})
		return map[string]any{"kind": "step-run", "value": result["value"]}, err
	})
	create(client, "step-sleep", "conformance/step-sleep", func(ctx context.Context, _ inngestgo.Input[map[string]any]) (any, error) {
		step.Sleep(ctx, "deterministic-sleep", time.Second)
		return map[string]any{"kind": "step-sleep", "slept": true}, nil
	})
	create(client, "wait-for-event", "conformance/wait-for-event", func(ctx context.Context, _ inngestgo.Input[map[string]any]) (any, error) {
		event, err := step.WaitForEvent[map[string]any](ctx, "deterministic-wait", step.WaitForEventOpts{
			Event:   "conformance/wait-for-event.resume",
			Timeout: time.Hour,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"kind": "wait-for-event", "received": event["data"]}, nil
	})
	create(client, "function-error", "conformance/function-error", func(context.Context, inngestgo.Input[map[string]any]) (any, error) {
		return nil, errors.New("conformance function error")
	})
	create(client, "step-error", "conformance/step-error", func(ctx context.Context, _ inngestgo.Input[map[string]any]) (any, error) {
		_, err := step.Run(ctx, "deterministic-error", func(context.Context) (any, error) {
			return nil, errors.New("conformance step error")
		})
		return nil, err
	})
}

func create(client inngestgo.Client, id, event string, fn func(context.Context, inngestgo.Input[map[string]any]) (any, error)) {
	if _, err := inngestgo.CreateFunction(client, inngestgo.FunctionOpts{ID: id}, inngestgo.EventTrigger(event, nil), fn); err != nil {
		log.Fatalf("create function %q: %v", id, err)
	}
}

func numericPort() (int, error) {
	value := envOr("PORT", strconv.Itoa(defaultPort))
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("PORT must be a numeric port between 1 and 65535, got %q", value)
	}
	return port, nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func stringPointer(value string) *string { return &value }

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
