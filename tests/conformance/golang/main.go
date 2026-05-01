package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
)

const (
	defaultAddr = "127.0.0.1:3000"
	appID       = "conformance-golang"
)

var (
	// retryStepAttempts tracks the number of times the retry case step has
	// executed across retries. The conformance runner expects the first attempt
	// to fail, then the retried attempt to succeed.
	retryStepAttempts atomic.Int32

	// retryFunctionAttempts tracks the function-level retry path after the step
	// has already succeeded.
	retryFunctionAttempts atomic.Int32
)

func main() {
	addr := getenv("PORT", defaultAddr)
	eventKey := getenv("INNGEST_EVENT_KEY", "test")

	client, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:    appID,
		EventKey: inngestgo.StrPtr(eventKey),
	})
	if err != nil {
		log.Fatalf("create client: %v", err)
	}

	registerFunctions(client)

	serve := client.Serve()
	mux := http.NewServeMux()

	// The conformance runner currently expects a conventional serve endpoint and
	// a separate introspection endpoint. The Go SDK handler already responds to
	// GET requests with introspection data, so we mount the same handler on both
	// routes.
	mux.Handle("/api/inngest", serve)
	mux.Handle("/api/introspect", serve)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("starting Go conformance fixture on http://%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
}

func registerFunctions(client inngestgo.Client) {
	mustCreate(client, inngestgo.FunctionOpts{ID: "test-suite-simple-fn"},
		inngestgo.EventTrigger("tests/function.test", nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			return map[string]any{
				"name": input.Event.Name,
				"body": "ok",
			}, nil
		},
	)

	mustCreate(client, inngestgo.FunctionOpts{ID: "test-suite-step-test"},
		inngestgo.EventTrigger("tests/step.test", nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			first, err := step.Run(ctx, "first step", func(ctx context.Context) (string, error) {
				return "first step", nil
			})
			if err != nil {
				return nil, err
			}

			// The step name is chosen intentionally to match the protocol-level
			// expectations in the serve conformance runner.
			step.Sleep(ctx, "sleep", 2*time.Second)

			_, err = step.Run(ctx, "second step", func(ctx context.Context) (map[string]any, error) {
				return map[string]any{
					"first":  first,
					"second": true,
				}, nil
			})
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"name": input.Event.Name,
				"body": "ok",
			}, nil
		},
	)

	mustCreate(client, inngestgo.FunctionOpts{ID: "test-suite-retry-test"},
		inngestgo.EventTrigger("tests/retry.test", nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			_, err := step.Run(ctx, "first step", func(ctx context.Context) (string, error) {
				attempt := retryStepAttempts.Add(1)
				switch attempt {
				case 1:
					return "", errors.New("broken")
				default:
					return "yes: 2", nil
				}
			})
			if err != nil {
				return nil, err
			}

			// The runner expects the function body itself to fail once after the
			// step has succeeded, then return a normal response on retry.
			if retryFunctionAttempts.Add(1) == 1 {
				return nil, errors.New("broken func")
			}

			return map[string]any{
				"name": input.Event.Name,
				"body": "ok",
			}, nil
		},
	)

	mustCreate(client, inngestgo.FunctionOpts{
		ID: "test-suite-cancel-test",
		Cancel: []inngestgo.ConfigCancel{
			{Event: "cancel/please", If: inngestgo.StrPtr("async.data.request_id == event.data.request_id")},
		},
	},
		inngestgo.EventTrigger("tests/cancel.test", nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			step.Sleep(ctx, "sleep", 10*time.Second)

			_, err := step.Run(ctx, "After the sleep", func(ctx context.Context) (string, error) {
				return "This should be cancelled if a matching cancel event is received", nil
			})
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"name": input.Event.Name,
				"body": "ok",
			}, nil
		},
	)

	mustCreate(client, inngestgo.FunctionOpts{ID: "test-suite-wait-for-event"},
		inngestgo.EventTrigger("tests/wait.test", nil),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			payload, err := step.WaitForEvent[map[string]any](ctx, "wait", step.WaitForEventOpts{
				Event:   "test/resume",
				If:      inngestgo.StrPtr("async.data.resume == true && async.data.id == event.data.id"),
				Timeout: 10 * time.Second,
			})
			if err == step.ErrEventNotReceived {
				return map[string]any{}, nil
			}
			if err != nil {
				return nil, err
			}

			// The runner expects the resumed event payload to be returned under a
			// top-level result field.
			return map[string]any{
				"result": payload,
			}, nil
		},
	)
}

func mustCreate[T any](client inngestgo.Client, opts inngestgo.FunctionOpts, trigger inngestgo.Trigger, fn func(context.Context, inngestgo.Input[T]) (any, error)) {
	if _, err := inngestgo.CreateFunction(client, opts, trigger, fn); err != nil {
		log.Fatalf("create function %s: %v", opts.ID, err)
	}
}

func getenv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
