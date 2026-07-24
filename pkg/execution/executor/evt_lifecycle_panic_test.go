package executor

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/logger"
)

func TestRunEventLifecyclesRecoversFromListenerPanic(t *testing.T) {
	panicking := &panickingEventLifecycle{called: make(chan struct{}, 2)}
	healthy := &recordingEventLifecycle{called: make(chan struct{}, 2)}

	e := &executor{
		log:           logger.From(context.Background()),
		evtLifecycles: []execution.EventLifecycleListener{panicking, healthy},
	}

	e.runEventLifecycles(context.Background(), func(ctx context.Context, l execution.EventLifecycleListener) {
		l.OnNoFunctionMatch(ctx, nil)
	})

	waitForLifecycleCall(t, panicking.called, "panicking listener")
	waitForLifecycleCall(t, healthy.called, "healthy listener")

	// A panic in one listener must not affect subsequent lifecycle runs.
	e.runEventLifecycles(context.Background(), func(ctx context.Context, l execution.EventLifecycleListener) {
		l.OnNoFunctionMatch(ctx, nil)
	})

	waitForLifecycleCall(t, panicking.called, "panicking listener second run")
	waitForLifecycleCall(t, healthy.called, "healthy listener second run")
}

func waitForLifecycleCall(t *testing.T, ch chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

type panickingEventLifecycle struct {
	execution.NoopEventLifecycleListener

	called chan struct{}
}

func (l *panickingEventLifecycle) OnNoFunctionMatch(context.Context, event.TrackedEvent) {
	l.called <- struct{}{}
	panic("event lifecycle listener panic")
}

type recordingEventLifecycle struct {
	execution.NoopEventLifecycleListener

	called chan struct{}
}

func (l *recordingEventLifecycle) OnNoFunctionMatch(context.Context, event.TrackedEvent) {
	l.called <- struct{}{}
}
