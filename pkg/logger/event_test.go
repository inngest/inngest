package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewEventStore(t *testing.T) {
	ctx := NewEventStore(context.Background(), "test-request")

	store, ok := ctx.Value(evtCtxKeyVal).(*eventstore)
	require.True(t, ok)
	require.NotNil(t, store)
	require.Equal(t, "test-request", store.Name)
	require.Empty(t, store.Events)
}

func TestAddEvent_NoStore(t *testing.T) {
	// Should not panic on a bare context.
	AddEvent(context.Background(), Event{Name: "orphan"})
}

func TestAddEvent_AppendsEvent(t *testing.T) {
	ctx := NewEventStore(context.Background(), "req")

	AddEvent(ctx, Event{Name: "something-happened"})

	store := ctx.Value(evtCtxKeyVal).(*eventstore)
	require.Len(t, store.Events, 1)

	evt := store.Events[0]
	require.Equal(t, "something-happened", evt.Name)
	// Fn and File are auto-populated by runtime.Caller.
	require.NotEmpty(t, evt.Fn)
	require.True(t, strings.Contains(evt.Fn, "TestAddEvent_AppendsEvent"))
}

func TestAddEvent_PresetFnFile(t *testing.T) {
	ctx := NewEventStore(context.Background(), "req")

	AddEvent(ctx, Event{
		Name: "preset",
		Fn:   "custom/pkg.MyFunc",
	})

	store := ctx.Value(evtCtxKeyVal).(*eventstore)
	require.Len(t, store.Events, 1)
	require.Equal(t, "custom/pkg.MyFunc", store.Events[0].Fn)
}

func TestTrackFnAsEvent(t *testing.T) {
	ctx := NewEventStore(context.Background(), "req")

	func() {
		defer TrackFn(ctx, nil)()
		time.Sleep(5 * time.Millisecond)
	}()

	store := ctx.Value(evtCtxKeyVal).(*eventstore)
	require.Len(t, store.Events, 1)

	evt := store.Events[0]
	require.NotEmpty(t, evt.Fn)
	require.True(t, strings.Contains(evt.Fn, "TestTrackFnAsEvent"))
	require.False(t, evt.Start.IsZero())
	require.Greater(t, evt.Duration, time.Duration(0))
}

func TestTrackFnAsEvent_NoStore(t *testing.T) {
	// Should not panic on a bare context.
	func() {
		defer TrackFn(context.Background(), nil)()
	}()
}

func TestMultipleEvents(t *testing.T) {
	ctx := NewEventStore(context.Background(), "multi")

	AddEvent(ctx, Event{Name: "first"})

	func() {
		defer TrackFn(ctx, nil)()
	}()

	AddEvent(ctx, Event{Name: "third"})

	store := ctx.Value(evtCtxKeyVal).(*eventstore)
	require.Len(t, store.Events, 3)
	require.Equal(t, "first", store.Events[0].Name)
	// TrackFnAsEvent doesn't set Name, so it's empty.
	require.Equal(t, "", store.Events[1].Name)
	require.Equal(t, "third", store.Events[2].Name)
}

func TestLogEvents(t *testing.T) {
	ctx := NewEventStore(context.Background(), "log-test")
	AddEvent(ctx, Event{Name: "evt1", Metadata: map[string]any{"key": "val"}})
	AddEvent(ctx, Event{Name: "evt2"})

	var buf bytes.Buffer
	l := newLogger(WithLoggerWriter(&buf), WithHandler(JSONHandler))

	l.LogEvents(ctx).Info("request complete")

	output := buf.String()
	require.Contains(t, output, "events_name")
	require.Contains(t, output, "log-test")
	require.Contains(t, output, "evt1")
	require.Contains(t, output, "evt2")
}

func TestLogEvents_NoStore(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(WithLoggerWriter(&buf), WithHandler(JSONHandler))

	// Should not panic and should not add events attrs.
	l.LogEvents(context.Background()).Info("no events")

	output := buf.String()
	require.NotContains(t, output, "events_name")
}

func TestLogEvents_EmptyStore(t *testing.T) {
	ctx := NewEventStore(context.Background(), "empty")

	var buf bytes.Buffer
	l := newLogger(WithLoggerWriter(&buf), WithHandler(JSONHandler))

	// Empty store should be a no-op.
	l.LogEvents(ctx).Info("no events")

	output := buf.String()
	require.NotContains(t, output, "events_name")
}

func TestResetEventStore(t *testing.T) {
	ctx := NewEventStore(context.Background(), "reset-test")
	AddEvent(ctx, Event{Name: "before-reset"})

	store := ctx.Value(evtCtxKeyVal).(*eventstore)
	require.Len(t, store.Events, 1)

	ResetEventStore(ctx)
	require.Empty(t, store.Events)

	// After reset, LogEvents should return the original logger (no-op).
	var buf bytes.Buffer
	l := newLogger(WithLoggerWriter(&buf), WithHandler(JSONHandler))

	returned := l.LogEvents(ctx)
	returned.Info("after reset")

	output := buf.String()
	require.NotContains(t, output, "events_name")
	require.Contains(t, output, "after reset")
}

func TestResetEventStore_NoStore(t *testing.T) {
	// Should not panic on a bare context.
	ResetEventStore(context.Background())
}

func TestEventStore_ConcurrentAppend(t *testing.T) {
	ctx := NewEventStore(context.Background(), "concurrent")

	const goroutines = 50
	const eventsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				AddEvent(ctx, Event{Name: "concurrent-event"})
			}
		}()
	}
	wg.Wait()

	store := ctx.Value(evtCtxKeyVal).(*eventstore)
	require.Len(t, store.Events, goroutines*eventsPerGoroutine)
}

func TestEvent_LogValue(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	evt := Event{
		Name:     "test-event",
		Fn:       "pkg.MyFunc",
		Start:    now,
		Duration: 5 * time.Millisecond,
		Metadata: map[string]any{"key": "val"},
	}

	val := evt.LogValue()
	require.Equal(t, slog.KindGroup, val.Kind())

	attrs := val.Group()
	attrMap := make(map[string]slog.Value, len(attrs))
	for _, a := range attrs {
		attrMap[a.Key] = a.Value
	}

	require.Equal(t, "test-event", attrMap["name"].String())
	require.Equal(t, "pkg.MyFunc", attrMap["fn"].String()) // no slash prefix to trim
	require.Equal(t, now, attrMap["start"].Time())
	require.Equal(t, 5*time.Millisecond, attrMap["d"].Duration())
	require.Equal(t, "val", attrMap["key"].Any())
}

func TestEvent_LogValue_ZeroFields(t *testing.T) {
	evt := Event{}

	val := evt.LogValue()
	attrs := val.Group()
	attrMap := make(map[string]slog.Value, len(attrs))
	for _, a := range attrs {
		attrMap[a.Key] = a.Value
	}

	// All zero-value fields should be omitted.
	for _, key := range []string{"name", "fn", "start", "d"} {
		_, has := attrMap[key]
		require.False(t, has, "expected %q to be omitted for zero Event", key)
	}
}

func TestEvent_LogValue_TrimsFn(t *testing.T) {
	evt := Event{
		Fn: "github.com/inngest/inngest/pkg/execution/queue.DurationWithTags[...]",
	}
	val := evt.LogValue()
	attrs := val.Group()
	attrMap := make(map[string]slog.Value, len(attrs))
	for _, a := range attrs {
		attrMap[a.Key] = a.Value
	}
	require.Equal(t, "queue.DurationWithTags[...]", attrMap["fn"].String())
}

func TestEventList_LogValue(t *testing.T) {
	el := EventList{
		{Name: "first", Fn: "a.Fn"},
		{Name: "second", Fn: "b.Fn"},
	}

	val := el.LogValue()
	require.Equal(t, slog.KindGroup, val.Kind())

	attrs := val.Group()
	require.Len(t, attrs, 2)
	require.Equal(t, "0", attrs[0].Key)
	require.Equal(t, "1", attrs[1].Key)

	// Each element should itself be a group with event fields.
	inner := attrs[0].Value.Resolve().Group()
	innerMap := make(map[string]slog.Value, len(inner))
	for _, a := range inner {
		innerMap[a.Key] = a.Value
	}
	require.Equal(t, "first", innerMap["name"].String())
}

func TestLogEvents_StructuredJSON(t *testing.T) {
	ctx := NewEventStore(context.Background(), "json-test")
	AddEvent(ctx, Event{
		Name:     "evt1",
		Fn:       "pkg.Fn",
		Metadata: map[string]any{"k": "v"},
	})

	var buf bytes.Buffer
	l := newLogger(WithLoggerWriter(&buf), WithHandler(JSONHandler))

	l.LogEvents(ctx).Info("done")

	output := buf.String()

	// Parse as JSON to verify structured output.
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &parsed))

	// events should be a nested object, not a Go %v string.
	events, ok := parsed["events"]
	require.True(t, ok, "expected 'events' key in JSON output")

	eventsMap, ok := events.(map[string]any)
	require.True(t, ok, "events should be a JSON object, got: %T", events)

	evt0, ok := eventsMap["0"].(map[string]any)
	require.True(t, ok, "events.0 should be a JSON object")
	require.Equal(t, "evt1", evt0["name"])
	require.Equal(t, "v", evt0["k"])
}
