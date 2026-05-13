package testapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
)

type EventKind string

const (
	KindFunctionScheduled EventKind = "function.scheduled"
	KindFunctionStarted   EventKind = "function.started"
	KindFunctionFinished  EventKind = "function.finished"
	KindFunctionCancelled EventKind = "function.cancelled"
	KindStepFinished      EventKind = "step.finished"
)

type LifecycleEvent struct {
	Kind       EventKind `json:"kind"`
	RunID      string    `json:"run_id"`
	FunctionID string    `json:"function_id"`
	StepID     string    `json:"step_id,omitempty"`
	Status     string    `json:"status,omitempty"`
	Error      string    `json:"error,omitempty"`
	Time       time.Time `json:"ts"`
}

// Hub is an in-memory pubsub for lifecycle events. Subscribers each get their
// own buffered channel. If a subscriber falls behind by more than the buffer
// size, events for that subscriber are dropped (publish is non-blocking).
type Hub struct {
	mu   sync.RWMutex
	subs map[*subscriber]struct{}
}

type subscriber struct {
	ch     chan LifecycleEvent
	closed atomic.Bool
}

func NewHub() *Hub {
	return &Hub{subs: map[*subscriber]struct{}{}}
}

// Subscribe returns a receive channel and a cancel func. Buf controls the
// per-subscriber buffer; events are dropped when the buffer is full.
func (h *Hub) Subscribe(buf int) (<-chan LifecycleEvent, func()) {
	if buf <= 0 {
		buf = 1024
	}
	s := &subscriber{ch: make(chan LifecycleEvent, buf)}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		delete(h.subs, s)
		h.mu.Unlock()
		if s.closed.CompareAndSwap(false, true) {
			close(s.ch)
		}
	}
	return s.ch, cancel
}

func (h *Hub) Publish(e LifecycleEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for s := range h.subs {
		select {
		case s.ch <- e:
		default:
		}
	}
}

// Listener implements execution.LifecycleListener and publishes a subset of
// lifecycle hooks to the hub. Only the hooks needed by tests are populated;
// the rest are inherited from NoopLifecyceListener.
type Listener struct {
	execution.NoopLifecyceListener

	Hub *Hub
}

var _ execution.LifecycleListener = (*Listener)(nil)

func (l *Listener) OnFunctionScheduled(_ context.Context, md statev2.Metadata, _ queue.Item, _ []event.TrackedEvent) {
	if l.Hub == nil {
		return
	}
	l.Hub.Publish(LifecycleEvent{
		Kind:       KindFunctionScheduled,
		RunID:      md.ID.RunID.String(),
		FunctionID: md.ID.FunctionID.String(),
		Time:       time.Now(),
	})
}

func (l *Listener) OnFunctionStarted(_ context.Context, md statev2.Metadata, _ queue.Item, _ []json.RawMessage) {
	if l.Hub == nil {
		return
	}
	l.Hub.Publish(LifecycleEvent{
		Kind:       KindFunctionStarted,
		RunID:      md.ID.RunID.String(),
		FunctionID: md.ID.FunctionID.String(),
		Time:       time.Now(),
	})
}

func (l *Listener) OnFunctionFinished(_ context.Context, md statev2.Metadata, _ queue.Item, _ []json.RawMessage, resp statev1.DriverResponse) {
	if l.Hub == nil {
		return
	}
	status := "completed"
	errStr := ""
	if resp.Err != nil {
		status = "failed"
		errStr = *resp.Err
	}
	l.Hub.Publish(LifecycleEvent{
		Kind:       KindFunctionFinished,
		RunID:      md.ID.RunID.String(),
		FunctionID: md.ID.FunctionID.String(),
		Status:     status,
		Error:      errStr,
		Time:       time.Now(),
	})
}

func (l *Listener) OnFunctionCancelled(_ context.Context, md statev2.Metadata, _ execution.CancelRequest, _ []json.RawMessage) {
	if l.Hub == nil {
		return
	}
	l.Hub.Publish(LifecycleEvent{
		Kind:       KindFunctionCancelled,
		RunID:      md.ID.RunID.String(),
		FunctionID: md.ID.FunctionID.String(),
		Status:     "cancelled",
		Time:       time.Now(),
	})
}

func (l *Listener) OnStepFinished(_ context.Context, md statev2.Metadata, _ queue.Item, edge inngest.Edge, resp *statev1.DriverResponse, fnErr error) {
	if l.Hub == nil {
		return
	}
	status := "completed"
	errStr := ""
	if resp != nil && resp.Err != nil {
		status = "failed"
		errStr = *resp.Err
	}
	if fnErr != nil {
		status = "failed"
		if errStr == "" {
			errStr = fnErr.Error()
		}
	}
	l.Hub.Publish(LifecycleEvent{
		Kind:       KindStepFinished,
		RunID:      md.ID.RunID.String(),
		FunctionID: md.ID.FunctionID.String(),
		StepID:     edge.Incoming,
		Status:     status,
		Error:      errStr,
		Time:       time.Now(),
	})
}

// streamEvents serves an SSE stream of lifecycle events.
func (t *TestAPI) streamEvents(w http.ResponseWriter, r *http.Request) {
	if t.options.Hub == nil {
		http.Error(w, "events hub not configured", http.StatusServiceUnavailable)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch, cancel := t.options.Hub.Subscribe(2048)
	defer cancel()

	_, _ = fmt.Fprint(w, ": ready\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			b, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
