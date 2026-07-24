package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWaitForRunStatus_RetriesTransientErrors verifies that WaitForRunStatus
// retries through transient GQL "not found" errors and eventually returns the
// run once it becomes available.
func TestWaitForRunStatus_RetriesTransientErrors(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")

		if n <= 2 {
			// First 2 calls return a GQL "not found" error (transient).
			resp := map[string]any{
				"data":   nil,
				"errors": []map[string]any{{"message": "function run not found"}},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Subsequent calls return a valid completed run.
		resp := map[string]any{
			"data": map[string]any{
				"functionRun": map[string]any{
					"output": `"done"`,
					"status": "COMPLETED",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{
		Client:  srv.Client(),
		T:       t,
		APIHost: srv.URL,
	}

	ctx := context.Background()
	run := c.WaitForRunStatus(ctx, t, "COMPLETED", "run-123", WaitForRunStatusOpts{
		Timeout: 5 * time.Second,
	})

	assert.Equal(t, "COMPLETED", run.Status)
	assert.Equal(t, `"done"`, run.Output)
	assert.GreaterOrEqual(t, int(callCount.Load()), 3, "expected at least 3 calls (2 transient errors + 1 success)")
}

// TestWaitForRunStatus_PermanentError verifies that WaitForRunStatus eventually
// fails (via require.Failf) when the GQL endpoint never returns a valid response.
func TestWaitForRunStatus_PermanentError(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"data":   nil,
			"errors": []map[string]any{{"message": "function run not found"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{
		Client:  srv.Client(),
		T:       t,
		APIHost: srv.URL,
	}

	// Use a mock testing.T to capture the Failf call without killing the real test.
	mockT := &mockTestingT{}

	ctx := context.Background()
	c.WaitForRunStatus(ctx, mockT, "COMPLETED", "run-456", WaitForRunStatusOpts{
		Timeout: 500 * time.Millisecond,
	})

	require.True(t, mockT.failed, "expected WaitForRunStatus to fail on permanent errors")
	assert.Contains(t, mockT.failMsg, "run-456", "failure message should contain the runID")
	assert.Contains(t, mockT.failMsg, "COMPLETED", "failure message should mention expected status")
	assert.Contains(t, mockT.failMsg, "history:", "failure message should contain status history")
	assert.Contains(t, mockT.failMsg, "end", "failure message history should end with 'end'")
	assert.GreaterOrEqual(t, int(callCount.Load()), 2, "expected multiple retry attempts before failing")
}

// TestWaitForRunStatus_HistoryShowsTransitions verifies that the timeout message
// includes the status-transition history with timestamps.
func TestWaitForRunStatus_HistoryShowsTransitions(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")

		// Simulate QUEUED -> RUNNING but never COMPLETED.
		status := "RUNNING"
		if n <= 2 {
			status = "QUEUED"
		}

		resp := map[string]any{
			"data": map[string]any{
				"functionRun": map[string]any{
					"output": `""`,
					"status": status,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{
		Client:  srv.Client(),
		T:       t,
		APIHost: srv.URL,
	}

	mockT := &mockTestingT{}

	ctx := context.Background()
	c.WaitForRunStatus(ctx, mockT, "COMPLETED", "run-789", WaitForRunStatusOpts{
		Timeout: 500 * time.Millisecond,
	})

	require.True(t, mockT.failed, "expected WaitForRunStatus to fail")
	assert.Contains(t, mockT.failMsg, "history:")
	assert.Contains(t, mockT.failMsg, "QUEUED")
	assert.Contains(t, mockT.failMsg, "RUNNING")
	assert.Contains(t, mockT.failMsg, "end")
}

// TestRunOrError_EmptyRunID verifies that RunOrError returns an error for empty runIDs.
func TestRunOrError_EmptyRunID(t *testing.T) {
	c := &Client{
		Client:  &http.Client{},
		T:       t,
		APIHost: "http://localhost:0", // won't be contacted
	}

	_, err := c.RunOrError(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runID cannot be empty")
}

// mockTestingT captures require.Failf calls without killing the test process.
type mockTestingT struct {
	failed  bool
	failMsg string
}

func (m *mockTestingT) Errorf(format string, args ...interface{}) {
	m.failed = true
	m.failMsg = fmt.Sprintf(format, args...)
}

func (m *mockTestingT) FailNow() {
	m.failed = true
}

func (m *mockTestingT) Helper() {}
