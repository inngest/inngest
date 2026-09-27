package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"
)

type exchange struct {
	RequestBody     json.RawMessage   `json:"request_body"`
	ResponseStatus  int               `json:"response_status"`
	ResponseHeaders map[string]string `json:"response_headers"`
	ResponseBody    json.RawMessage   `json:"response_body"`
}

type runResult struct {
	Output string `json:"output"`
	Status string `json:"status"`
}

type report struct {
	SchemaVersion string          `json:"schema_version"`
	Scenario      string          `json:"scenario"`
	RunID         string          `json:"run_id"`
	Exchange      exchange        `json:"exchange"`
	Run           runResult       `json:"run"`
	Output        json.RawMessage `json:"output"`
}

func main() {
	sdkTarget := flag.String("sdk-target", "http://127.0.0.1:3132", "SDK fixture origin")
	devServer := flag.String("dev-server", "http://127.0.0.1:8288", "Inngest dev server origin")
	listen := flag.String("listen", "127.0.0.1:3140", "recording proxy address")
	output := flag.String("output", "", "write report to this path; defaults to stdout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	target, err := url.Parse(*sdkTarget)
	if err != nil {
		fatalf("parse SDK target: %v", err)
	}
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		fatalf("listen for recording proxy: %v", err)
	}
	defer listener.Close()

	exchanges := make(chan exchange, 8)
	proxy := recordingProxy(target, exchanges)
	server := &http.Server{Handler: proxy, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fatalf("serve recording proxy: %v", err)
		}
	}()
	defer server.Shutdown(context.Background())

	proxyURL := "http://" + listener.Addr().String() + "/api/inngest"
	if err := register(ctx, proxyURL); err != nil {
		fatalf("register fixture through dev server: %v", err)
	}
	if err := sendEvent(ctx, *devServer); err != nil {
		fatalf("send trigger event: %v", err)
	}

	observed, err := waitForInvocation(ctx, exchanges)
	if err != nil {
		fatalf("wait for SDK invocation: %v", err)
	}
	runID, err := requestRunID(observed.RequestBody)
	if err != nil {
		fatalf("read run ID from real executor request: %v", err)
	}
	run, err := waitForRun(ctx, *devServer, runID)
	if err != nil {
		fatalf("wait for durable run: %v", err)
	}

	var actual json.RawMessage
	if err := json.Unmarshal([]byte(run.Output), &actual); err != nil {
		fatalf("decode durable output: %v", err)
	}
	want := json.RawMessage(`{"kind":"basic-completion","value":"devserver-spike"}`)
	if !jsonEqual(actual, want) {
		fatalf("durable output = %s, want %s", actual, want)
	}

	result := report{
		SchemaVersion: "0.1",
		Scenario:      "devserver.basic-completion",
		RunID:         runID,
		Exchange:      observed,
		Run:           run,
		Output:        actual,
	}
	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fatalf("encode report: %v", err)
	}
	payload = append(payload, '\n')
	if *output == "" {
		_, _ = os.Stdout.Write(payload)
		return
	}
	if err := os.WriteFile(*output, payload, 0o644); err != nil {
		fatalf("write report: %v", err)
	}
	fmt.Printf("PASS devserver.basic-completion (%s)\n", runID)
}

func recordingProxy(target *url.URL, exchanges chan<- exchange) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.Request.Method != http.MethodPost || resp.Request.URL.Path != "/api/inngest" {
			return nil
		}
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body = io.NopCloser(bytes.NewReader(responseBody))
		requestBody, _ := resp.Request.Context().Value(requestBodyKey{}).([]byte)
		select {
		case exchanges <- exchange{
			RequestBody:     append(json.RawMessage(nil), requestBody...),
			ResponseStatus:  resp.StatusCode,
			ResponseHeaders: selectedHeaders(resp.Header),
			ResponseBody:    append(json.RawMessage(nil), responseBody...),
		}:
		default:
		}
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/inngest" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			r = r.WithContext(context.WithValue(r.Context(), requestBodyKey{}, body))
		}
		proxy.ServeHTTP(w, r)
	})
}

type requestBodyKey struct{}

func register(ctx context.Context, endpoint string) error {
	body, err := json.Marshal(map[string]string{"url": endpoint})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inngest-Server-Kind", "dev")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	return nil
}

func sendEvent(ctx context.Context, devServer string) error {
	body := []byte(`{"name":"conformance/basic-completion","data":{"value":"devserver-spike"}}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(devServer, "/")+"/e/test", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	return nil
}

func waitForInvocation(ctx context.Context, exchanges <-chan exchange) (exchange, error) {
	for {
		select {
		case <-ctx.Done():
			return exchange{}, ctx.Err()
		case current := <-exchanges:
			var request struct {
				Event struct {
					Name string `json:"name"`
				} `json:"event"`
			}
			if json.Unmarshal(current.RequestBody, &request) == nil && request.Event.Name == "conformance/basic-completion" {
				return current, nil
			}
		}
	}
}

func requestRunID(body []byte) (string, error) {
	var request struct {
		Context struct {
			RunID string `json:"run_id"`
		} `json:"ctx"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		return "", err
	}
	if request.Context.RunID == "" {
		return "", fmt.Errorf("ctx.run_id is empty")
	}
	return request.Context.RunID, nil
}

func waitForRun(ctx context.Context, devServer, runID string) (runResult, error) {
	query := `query GetRun($runID: ID!) { functionRun(query: { functionRunId: $runID }) { output status } }`
	requestBody, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]string{"runID": runID},
	})
	if err != nil {
		return runResult{}, err
	}
	for {
		select {
		case <-ctx.Done():
			return runResult{}, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(devServer, "/")+"/v0/gql", bytes.NewReader(requestBody))
		if err != nil {
			return runResult{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		var payload struct {
			Data struct {
				FunctionRun runResult `json:"functionRun"`
			} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		if err == nil {
			switch payload.Data.FunctionRun.Status {
			case "COMPLETED":
				return payload.Data.FunctionRun, nil
			case "FAILED", "CANCELLED":
				return runResult{}, fmt.Errorf("run ended with status %s: %s", payload.Data.FunctionRun.Status, payload.Data.FunctionRun.Output)
			}
		}
	}
}

func selectedHeaders(header http.Header) map[string]string {
	result := map[string]string{}
	for _, key := range []string{"Content-Type", "X-Inngest-No-Retry", "X-Inngest-Req-Version"} {
		if value := header.Get(key); value != "" {
			result[strings.ToLower(key)] = value
		}
	}
	return result
}

func jsonEqual(left, right []byte) bool {
	var leftValue, rightValue any
	return json.Unmarshal(left, &leftValue) == nil &&
		json.Unmarshal(right, &rightValue) == nil &&
		reflect.DeepEqual(leftValue, rightValue)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
