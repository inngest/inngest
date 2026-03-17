package logger

import (
	"bytes"
	"context"
	"strings"
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
	require.True(t, strings.HasSuffix(evt.File, "event_test.go:31"), "expected caller file, got %s", evt.File)
}

func TestAddEvent_PresetFnFile(t *testing.T) {
	ctx := NewEventStore(context.Background(), "req")

	AddEvent(ctx, Event{
		Name: "preset",
		Fn:   "custom/pkg.MyFunc",
		File: "custom/file.go:99",
	})

	store := ctx.Value(evtCtxKeyVal).(*eventstore)
	require.Len(t, store.Events, 1)
	require.Equal(t, "custom/pkg.MyFunc", store.Events[0].Fn)
	require.Equal(t, "custom/file.go:99", store.Events[0].File)
}

func TestTrackFnAsEvent(t *testing.T) {
	ctx := NewEventStore(context.Background(), "req")

	func() {
		defer TrackFnAsEvent(ctx)()
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
		defer TrackFnAsEvent(context.Background())()
	}()
}

func TestMultipleEvents(t *testing.T) {
	ctx := NewEventStore(context.Background(), "multi")

	AddEvent(ctx, Event{Name: "first"})

	func() {
		defer TrackFnAsEvent(ctx)()
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

	l.LogEvents(ctx)
	l.Info("request complete")

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
	l.LogEvents(context.Background())
	l.Info("no events")

	output := buf.String()
	require.NotContains(t, output, "events_name")
}

func TestLogEvents_EmptyStore(t *testing.T) {
	ctx := NewEventStore(context.Background(), "empty")

	var buf bytes.Buffer
	l := newLogger(WithLoggerWriter(&buf), WithHandler(JSONHandler))

	// Empty store should be a no-op.
	l.LogEvents(ctx)
	l.Info("no events")

	output := buf.String()
	require.NotContains(t, output, "events_name")
}
