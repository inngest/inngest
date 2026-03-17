package logger

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

var evtCtxKeyVal = evtCtxKey{}

type evtCtxKey struct{}

type eventstore struct {
	Name   string
	Events []Event
	mu     sync.Mutex
}

func (e *eventstore) Append(evt Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Events = append(e.Events, evt)
}

func (e *eventstore) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Events = []Event{}
}

// Event represents a log event for wide logs, used in req end logging
type Event struct {
	// Name is a custom event name, optional.
	Name string `json:"name"`
	// Fn is the calling function that creatd the event.  This is auto-generated via the runtime
	// package for any caller of [AddEvent] or [TrackFnAsEvent]
	Fn string `json:"fn"`
	// File is the calling file && line that creatd the event.  This is auto-generated via the runtime
	// package for any caller of [AddEvent] or [TrackFnAsEvent]
	File string `json:"file"`
	// Start is the start time of the event:  the time the event occurred.
	Start time.Time `json:"start,omitempty,omitzero"`
	// Duration is the duration for the overall function, or the duration for the event.
	// Logged in microseconds.
	Duration time.Duration `json:"d,omitempty,omitzero"`
	// Metadata includes any info you want in the event.
	Metadata map[string]any `json:"metadata,omitempty,omitzero"`
}

// NewEventStore creates a new event store in context for wide logging.
//
// This collects all events added via `TrackFnAsEvent` and `AddEvent` for logging via
// [Logger.LogEvents]
func NewEventStore(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, evtCtxKeyVal, &eventstore{Name: name})
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
		pc, file, line, _ := runtime.Caller(1)
		e.Fn = runtime.FuncForPC(pc).Name()
		e.File = fmt.Sprintf("%s:%d", file, line)
	}

	es.Append(e)
}

// TrackFnAsEvent tracks a method as an event.  This also tracks method duratins.  Usage:
//
//	defer logging.TrackFnAsEvent(ctx)
func TrackFnAsEvent(ctx context.Context) func() {
	pc, file, line, _ := runtime.Caller(1)
	evt := Event{
		Fn:    runtime.FuncForPC(pc).Name(),
		File:  fmt.Sprintf("%s:%d", file, line),
		Start: time.Now(),
	}

	return func() {
		evt.Duration = time.Since(evt.Start)
		AddEvent(ctx, evt)
	}
}
