package executor

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/require"
)

func TestRunFunctionMatchLifecycleSnapshotsRequestContextForAsyncListeners(t *testing.T) {
	listener := &requestContextRaceLifecycle{
		ready:   make(chan struct{}),
		start:   make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan any, 1),
	}
	e := &executor{
		evtLifecycles: []execution.EventLifecycleListener{listener},
	}

	req := execution.ScheduleRequest{
		Context: map[string]any{
			"stable": "before",
		},
	}

	e.RunFunctionMatchLifecycle(context.Background(), req)

	select {
	case <-listener.ready:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for lifecycle listener")
	}

	req.Context["stable"] = "after"
	close(listener.start)

	for i := 0; i < 1_000; i++ {
		req.Context[fmt.Sprintf("key-%d", i)] = i
		runtime.Gosched()
	}
	close(listener.release)

	select {
	case observed := <-listener.done:
		require.Equal(t, "before", observed)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for lifecycle listener to finish")
	}
}

func TestHandleFunctionSkippedSnapshotsMetadataContextForAsyncListeners(t *testing.T) {
	listener := &metadataContextRaceLifecycle{
		ready:   make(chan struct{}),
		start:   make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan any, 1),
	}
	e := &executor{
		evtLifecycles: []execution.EventLifecycleListener{listener},
	}

	metadata := sv2.Metadata{
		Config: *sv2.InitConfig(&sv2.Config{
			Context: map[string]any{
				"stable": "before",
			},
		}),
	}

	_, _, err := e.handleFunctionSkipped(context.Background(), execution.ScheduleRequest{}, metadata, nil, enums.SkipReasonFunctionPaused)
	require.ErrorIs(t, err, ErrFunctionSkipped)

	select {
	case <-listener.ready:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for lifecycle listener")
	}

	metadata.Config.Context["stable"] = "after"
	close(listener.start)

	for i := 0; i < 1_000; i++ {
		metadata.Config.Context[fmt.Sprintf("key-%d", i)] = i
		runtime.Gosched()
	}
	close(listener.release)

	select {
	case observed := <-listener.done:
		require.Equal(t, "before", observed)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for lifecycle listener to finish")
	}
}

type requestContextRaceLifecycle struct {
	execution.NoopEventLifecycleListener

	ready   chan struct{}
	start   chan struct{}
	release chan struct{}
	done    chan any
}

func (r *requestContextRaceLifecycle) OnFunctionMatch(_ context.Context, req execution.ScheduleRequest) {
	close(r.ready)
	<-r.start

	observed := req.Context["stable"]
	for {
		select {
		case <-r.release:
			r.done <- observed
			return
		default:
			_ = req.Context["stable"]
			runtime.Gosched()
		}
	}
}

type metadataContextRaceLifecycle struct {
	execution.NoopEventLifecycleListener

	ready   chan struct{}
	start   chan struct{}
	release chan struct{}
	done    chan any
}

func (r *metadataContextRaceLifecycle) OnFunctionSkipped(_ context.Context, _ execution.ScheduleRequest, metadata sv2.Metadata, _ enums.SkipReason) {
	close(r.ready)
	<-r.start

	observed := metadata.Config.Context["stable"]
	for {
		select {
		case <-r.release:
			r.done <- observed
			return
		default:
			_ = metadata.Config.Context["stable"]
			runtime.Gosched()
		}
	}
}
