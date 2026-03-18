package logger

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"
)

var evtCtxKeyVal = evtCtxKey{}

type evtCtxKey struct{}

type eventstore struct {
	// Name is the name for the event store container
	Name string
	// Events represent all events in the event store
	Events EventList
	// T represents the time the event store was created or was reset
	T time.Time

	mu sync.Mutex
}

func (e *eventstore) Append(evt Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Events = append(e.Events, evt)
}

func (e *eventstore) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Preserve the underlying array capacity for reuse.
	e.Events = e.Events[:0]
	e.T = time.Now()
}

// Event represents a log event for wide logs, used in req end logging
type Event struct {
	// Name is a custom event name, optional.
	Name string `json:"name"`
	// Fn is the calling function that creatd the event.  This is auto-generated via the runtime
	// package for any caller of [AddEvent] or [TrackFn]
	Fn string `json:"fn"`
	// File is the calling file && line that creatd the event.  This is auto-generated via the runtime
	// package for any caller of [AddEvent] or [TrackFn]
	// File string `json:"file,omitempty"`
	// Start is the start time of the event:  the time the event occurred.
	Start time.Time `json:"start,omitempty,omitzero"`
	// Duration is the duration for the overall function, or the duration for the event.
	// Logged in microseconds.
	Duration time.Duration `json:"d,omitempty,omitzero"`
	// Metadata includes any info you want in the event.
	Metadata map[string]any `json:"metadata,omitempty,omitzero"`
}

// LogValue implements slog.LogValuer so that each Event renders as a structured
// group when passed to slog.Any.
func (e Event) LogValue() slog.Value {
	var attrs []slog.Attr
	if e.Name != "" {
		attrs = append(attrs, slog.String("name", e.Name))
	}
	if e.Fn != "" {
		fn := e.Fn
		if i := strings.LastIndex(fn, "/"); i >= 0 {
			fn = fn[i+1:]
		}
		attrs = append(attrs, slog.String("fn", fn))
	}
	// if e.File != "" {
	// 	attrs = append(attrs, slog.String("file", e.File))
	// }
	if !e.Start.IsZero() {
		attrs = append(attrs, slog.Time("start", e.Start))
	}
	if e.Duration > 0 {
		attrs = append(attrs, slog.Duration("d", e.Duration))
	}
	for k, v := range e.Metadata {
		attrs = append(attrs, slog.Any(k, v))
	}
	return slog.GroupValue(attrs...)
}

// EventList is a slice of Event that implements slog.LogValuer so that
// slog.Any("events", eventList) produces numbered groups ("0", "1", …).
type EventList []Event

// LogValue implements slog.LogValuer for EventList.
func (el EventList) LogValue() slog.Value {
	attrs := make([]slog.Attr, len(el))
	for i, e := range el {
		attrs[i] = slog.Any(fmt.Sprintf("%d", i), e)
	}
	return slog.GroupValue(attrs...)
}

// NewEventStore creates a new event store in context for wide logging.
//
// This collects all events added via `TrackFn` and `AddEvent` for logging via
// [Logger.LogEvents]
func NewEventStore(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, evtCtxKeyVal, &eventstore{Name: name, T: time.Now()})
}

// ResetEventStore clears events in the event store from context.
func ResetEventStore(ctx context.Context) {
	es, ok := ctx.Value(evtCtxKeyVal).(*eventstore)
	if !ok || es == nil {
		return
	}
	es.Reset()
}

// AddEvent tracks an event for future logging in the current event store.
func AddEvent(ctx context.Context, e Event) {
	es, ok := ctx.Value(evtCtxKeyVal).(*eventstore)
	if !ok || es == nil {
		return
	}

	if e.Fn == "" {
		pc, _, _, _ := runtime.Caller(1)
		e.Fn = runtime.FuncForPC(pc).Name()
	}

	es.Append(e)
}

// TrackFn tracks a method as an event.  This also tracks method duratins.  Usage:
//
//	defer logging.TrackFn(ctx)
func TrackFn(ctx context.Context, metadata map[string]any) func() {
	pc, _, _, _ := runtime.Caller(1)
	evt := Event{
		Fn:       runtime.FuncForPC(pc).Name(),
		Start:    time.Now(),
		Metadata: metadata,
	}

	return func() {
		evt.Duration = time.Since(evt.Start)
		AddEvent(ctx, evt)
	}
}
