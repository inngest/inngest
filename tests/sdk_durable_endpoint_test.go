package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/tests/client"
	"github.com/stretchr/testify/require"
)

// TestDurableEndpoint_SyncResponseRecorded verifies the end-to-end Durable
// Endpoint flow. Specifically:
func TestDurableEndpoint_SyncResponseRecorded(t *testing.T) {
	cli := client.New(t)
	ctx := context.Background()

	start := time.Now().Add(-2 * time.Second)

	// 1. Hit the JS endpoint directly. The SDK runs the user handler in
	// sync mode, sends an OpcodeRunComplete to /checkpoint, and returns
	// the user's Response to us.
	resp, body := mustRequest(t, http.MethodPost, "http://127.0.0.1:3000/api/durable/sync", nil)
	require.Equal(t, 200, resp.StatusCode, "status from JS endpoint should match handler")
	require.Equal(t, `{"hello":"world"}`, string(body), "body from JS endpoint should match handler")
	require.Contains(t, resp.Header.Get("content-type"), "application/json")

	// 2. Wait for the run to be recorded with COMPLETED status.
	runID := waitForRecentRun(t, cli, ctx, "POST /api/durable/sync", start, "COMPLETED", 15*time.Second)

	run := cli.Run(ctx, runID)
	require.Equal(t, "COMPLETED", run.Status)

	// 3. When the user handler returns a `Response`, the JS SDK delivers it
	// directly to the original caller and checkpoints `null` for the body to the
	// Inngest server (the body is already on its way out). So the recorded body
	// is the four-character string "null".
	require.Equal(t, "null", run.Output,
		"run output should be the SDK-supplied body as a raw string, not base64")
}

// TestDurableEndpoint_SyncToAsyncResponseRecorded verifies a Durable
// Endpoint that suspends on a step (sync→async transition):
func TestDurableEndpoint_SyncToAsyncResponseRecorded(t *testing.T) {
	cli := client.New(t)
	ctx := context.Background()

	// 1. Hit the JS endpoint. Async durable endpoints return a 302
	// pointing at the server's poll URL; do NOT auto-follow because we
	// want to inspect the redirect target.
	noRedirect := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:3000/api/durable/async", nil)
	require.NoError(t, err)
	resp, err := noRedirect.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode,
		"async durable endpoint should 302 to the server poll URL")

	pollURL := resp.Header.Get("Location")
	require.NotEmpty(t, pollURL, "302 must include a Location header")
	parsed, err := url.Parse(pollURL)
	require.NoError(t, err)
	require.Contains(t, parsed.Path, "/output",
		"redirect should point at /v1/http/runs/{runID}/output")
	require.NotEmpty(t, parsed.Query().Get("token"),
		"polling URL should carry a JWT token")

	runID := extractRunIDFromPollURL(t, parsed)
	require.NotEmpty(t, runID)

	// 2. Poll the redirect URL for the final response.
	final, finalBody := mustRequest(t, http.MethodGet, pollURL, nil)
	require.Equal(t, 200, final.StatusCode)
	require.Contains(t, final.Header.Get("content-type"), "application/json")

	// In async mode the SDK JSON-stringifies the user's body before
	// checkpointing (so the wire body is a JSON-encoded string of the
	// user's body).
	var decoded string
	require.NoError(t, json.Unmarshal(finalBody, &decoded),
		"polled body should be a JSON string (got=%q)", string(finalBody))
	require.Equal(t, `{"hello":"async"}`, decoded,
		"polled body should round-trip to the user-handler response")

	// 3. The run is recorded as COMPLETED.
	run := cli.WaitForRunStatus(ctx, t, "COMPLETED", runID, client.WaitForRunStatusOpts{Timeout: 10 * time.Second})
	require.Equal(t, "COMPLETED", run.Status)
}

// extractRunIDFromPollURL pulls the {runID} segment out of a path shaped
// like `/v1/http/runs/{runID}/output`.
func extractRunIDFromPollURL(t *testing.T, u *url.URL) string {
	t.Helper()
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, p := range parts {
		if p == "runs" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	t.Fatalf("could not extract runID from path %s", u.Path)
	return ""
}

func mustRequest(t *testing.T, method, url string, body []byte) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, respBody
}

// waitForRecentRun polls the runs list for a run with the expected status
// queued after `start`, scoped to the function with the given slug. We can't
// use the runID from the JS SDK's sync wire response (it isn't surfaced), so
// we filter by function ID + time window to avoid grabbing another test's run.
func waitForRecentRun(t *testing.T, cli *client.Client, ctx context.Context, fnSlug string, start time.Time, status string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		fnID, ok := lookupFunctionID(ctx, cli, fnSlug)
		if !ok {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		edges, _, _ := cli.FunctionRuns(ctx, client.FunctionRunOpt{
			Items:       10,
			Status:      []string{status},
			Start:       start,
			End:         time.Now().Add(time.Minute),
			FunctionIDs: []uuid.UUID{fnID},
		})
		if len(edges) > 0 {
			return edges[0].Node.ID
		}
		time.Sleep(200 * time.Millisecond)
	}
	require.FailNowf(t, "no run found", "no run with status %s for fn %q appeared within %s", status, fnSlug, timeout)
	return ""
}

// lookupFunctionID resolves a function slug to its UUID via the GraphQL
// functions list. Returns ok=false if the function isn't registered yet
func lookupFunctionID(ctx context.Context, cli *client.Client, slug string) (id uuid.UUID, ok bool) {
	fns, err := cli.Functions(ctx)
	if err != nil {
		return uuid.Nil, false
	}
	for _, fn := range fns {
		if fn.Name == slug {
			id, err := uuid.Parse(fn.ID)
			if err != nil {
				return uuid.Nil, false
			}
			return id, true
		}
	}
	return uuid.Nil, false
}
