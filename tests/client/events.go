package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

const eventsPath = "/test/events"

type LifecycleEventKind string

const (
	KindFunctionScheduled LifecycleEventKind = "function.scheduled"
	KindFunctionStarted   LifecycleEventKind = "function.started"
	KindFunctionFinished  LifecycleEventKind = "function.finished"
	KindFunctionCancelled LifecycleEventKind = "function.cancelled"
	KindStepFinished      LifecycleEventKind = "step.finished"
)

type LifecycleEvent struct {
	Kind       LifecycleEventKind `json:"kind"`
	RunID      string             `json:"run_id"`
	FunctionID string             `json:"function_id"`
	StepID     string             `json:"step_id,omitempty"`
	Status     string             `json:"status,omitempty"`
	Error      string             `json:"error,omitempty"`
	Time       time.Time          `json:"ts"`
}

// EventStream is a live SSE subscription to dev server lifecycle events. Open
// it before triggering work; the connection buffers events server-side, so
// subscribing first guarantees you don't miss the events the test will assert
// on.
type EventStream struct {
	t      *testing.T
	cancel context.CancelFunc

	mu     sync.Mutex
	events []LifecycleEvent
	cond   *sync.Cond
	closed bool
	err    error
}

// SubscribeEvents opens the SSE stream and returns once the server has
// acknowledged the subscription (so events fired after this call cannot be
// missed). The stream is automatically closed at test cleanup.
func (c *Client) SubscribeEvents(ctx context.Context, t *testing.T) *EventStream {
	t.Helper()

	streamCtx, cancel := context.WithCancel(ctx)

	req, err := http.NewRequestWithContext(streamCtx, http.MethodGet, c.APIHost+eventsPath, nil)
	if err != nil {
		cancel()
		t.Fatalf("build events request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.Client.Do(req)
	if err != nil {
		cancel()
		t.Fatalf("dial events stream: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("events stream returned %s", resp.Status)
	}

	s := &EventStream{t: t, cancel: cancel}
	s.cond = sync.NewCond(&s.mu)

	ready := make(chan struct{})
	go s.read(resp, ready)

	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timed out waiting for events stream to open")
	}

	t.Cleanup(s.Close)
	return s
}

func (s *EventStream) read(resp *http.Response, ready chan struct{}) {
	defer resp.Body.Close()
	defer func() {
		s.mu.Lock()
		s.closed = true
		s.cond.Broadcast()
		s.mu.Unlock()
	}()

	br := bufio.NewReader(resp.Body)
	signaled := false
	var data bytes.Buffer

	for {
		line, err := br.ReadBytes('\n')
		if err != nil {
			s.mu.Lock()
			if !errors.Is(err, context.Canceled) {
				s.err = err
			}
			s.mu.Unlock()
			return
		}

		// SSE comment line — used by server as the "ready" sentinel.
		if !signaled && bytes.HasPrefix(line, []byte(":")) {
			close(ready)
			signaled = true
			continue
		}
		if bytes.HasPrefix(line, []byte(":")) {
			continue
		}

		// Blank line ends an event.
		if line[0] == '\n' || (len(line) >= 2 && line[0] == '\r' && line[1] == '\n') {
			if data.Len() == 0 {
				continue
			}
			var ev LifecycleEvent
			if err := json.Unmarshal(data.Bytes(), &ev); err == nil {
				s.mu.Lock()
				s.events = append(s.events, ev)
				s.cond.Broadcast()
				s.mu.Unlock()
			}
			data.Reset()
			continue
		}

		if rest, ok := bytes.CutPrefix(line, []byte("data: ")); ok {
			data.Write(bytes.TrimRight(rest, "\r\n"))
		}
	}
}

func (s *EventStream) Close() {
	s.cancel()
}

// Wait blocks until match returns true for some buffered event, or timeout
// elapses. Returns the matching event. Fails the test on timeout.
func (s *EventStream) Wait(timeout time.Duration, desc string, match func(LifecycleEvent) bool) LifecycleEvent {
	s.t.Helper()
	deadline := time.Now().Add(timeout)

	timer := time.AfterFunc(timeout, func() {
		s.mu.Lock()
		s.cond.Broadcast()
		s.mu.Unlock()
	})
	defer timer.Stop()

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := 0
	for {
		for ; idx < len(s.events); idx++ {
			if match(s.events[idx]) {
				return s.events[idx]
			}
		}
		if s.closed {
			s.t.Fatalf("event stream closed before %s (err: %v)", desc, s.err)
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("timeout waiting for %s after %s; saw %d events: %s",
				desc, timeout, len(s.events), summarize(s.events))
			return LifecycleEvent{}
		}
		s.cond.Wait()
	}
}

// WaitForFinished waits for an OnFunctionFinished event for runID.
func (s *EventStream) WaitForFinished(runID string, timeout time.Duration) LifecycleEvent {
	return s.Wait(timeout, "function.finished for run "+runID, func(e LifecycleEvent) bool {
		return e.Kind == KindFunctionFinished && e.RunID == runID
	})
}

// WaitForCancelled waits for an OnFunctionCancelled event for runID.
func (s *EventStream) WaitForCancelled(runID string, timeout time.Duration) LifecycleEvent {
	return s.Wait(timeout, "function.cancelled for run "+runID, func(e LifecycleEvent) bool {
		return e.Kind == KindFunctionCancelled && e.RunID == runID
	})
}

// WaitForAnyCancelled waits for any OnFunctionCancelled event. Useful for
// tests that don't surface the runID directly (e.g. tests driven through the
// proxy harness rather than the Go SDK).
func (s *EventStream) WaitForAnyCancelled(timeout time.Duration) LifecycleEvent {
	return s.Wait(timeout, "any function.cancelled", func(e LifecycleEvent) bool {
		return e.Kind == KindFunctionCancelled
	})
}

// WaitForFunctionStartedCount blocks until at least n function.started events
// have been observed for the given functionID prefix (matches the function
// slug). Returns the events seen.
func (s *EventStream) WaitForFunctionStartedCount(n int, timeout time.Duration) []LifecycleEvent {
	s.t.Helper()
	deadline := time.Now().Add(timeout)

	timer := time.AfterFunc(timeout, func() {
		s.mu.Lock()
		s.cond.Broadcast()
		s.mu.Unlock()
	})
	defer timer.Stop()

	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		var started []LifecycleEvent
		for _, e := range s.events {
			if e.Kind == KindFunctionStarted {
				started = append(started, e)
			}
		}
		if len(started) >= n {
			return started
		}
		if s.closed {
			s.t.Fatalf("event stream closed; got %d/%d function.started events", len(started), n)
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("timeout waiting for %d function.started events after %s; got %d: %s",
				n, timeout, len(started), summarize(s.events))
			return nil
		}
		s.cond.Wait()
	}
}

func summarize(evs []LifecycleEvent) string {
	if len(evs) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(evs))
	limit := 20
	for i, e := range evs {
		if i >= limit {
			parts = append(parts, fmt.Sprintf("...+%d more", len(evs)-limit))
			break
		}
		parts = append(parts, string(e.Kind)+"("+e.RunID+")")
	}
	return strings.Join(parts, ", ")
}
