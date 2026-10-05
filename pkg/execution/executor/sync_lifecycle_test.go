package executor

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type recordingSyncLifecycle struct {
	execution.NoopSyncLifecycleListener
	finishedCalls   int
	scheduledCalls  int
	scheduledEvents event.SerializedEvents
}

func (r *recordingSyncLifecycle) OnFunctionFinished(context.Context, sv2.Metadata, queue.Item, []json.RawMessage, statev1.DriverResponse, time.Time) {
	r.finishedCalls++
}

func (r *recordingSyncLifecycle) OnFunctionScheduled(_ context.Context, _ sv2.Metadata, _ queue.Item, events event.SerializedEvents) {
	r.scheduledCalls++
	r.scheduledEvents = events
}

func TestRunFunctionScheduledSyncListenersReuseSerializedEvents(t *testing.T) {
	ctx := context.Background()
	serialized, err := event.NewSerializedEvents([]json.RawMessage{
		json.RawMessage(`{"name":"first"}`),
		json.RawMessage(`{"name":"second"}`),
	})
	require.NoError(t, err)
	e := &executor{log: logger.VoidLogger()}
	sync := &recordingSyncLifecycle{}
	e.syncLifecycles = []execution.SyncLifecycleListener{sync}
	e.notifyFunctionScheduledSyncListeners(ctx, sv2.Metadata{}, queue.Item{}, serialized)
	require.Equal(t, 1, sync.scheduledCalls)
	require.True(t, serialized.Equal(sync.scheduledEvents))
	require.Equal(t, serialized.Input(), sync.scheduledEvents.Input())
}

type recordingScheduleRunService struct {
	sv2.RunService
	createCalled   bool
	createEvents   event.SerializedEvents
	persistedEvent []json.RawMessage
}

func (r *recordingScheduleRunService) Create(_ context.Context, s sv2.CreateState) (sv2.State, error) {
	r.createCalled = true
	r.createEvents = s.SerializedEvents
	r.persistedEvent = s.SerializedEvents.RawMessages()
	return sv2.State{Metadata: s.Metadata, Events: r.persistedEvent}, nil
}

type mutatingScheduledSyncLifecycle struct {
	execution.NoopSyncLifecycleListener
	runService        *recordingScheduleRunService
	called            bool
	createCalledFirst bool
	receivedEvents    event.SerializedEvents
}

type recordingSkippedLifecycle struct {
	execution.NoopLifecyceListener
	events chan event.SerializedEvents
}

func (l *recordingSkippedLifecycle) OnFunctionSkipped(_ context.Context, _ sv2.Metadata, state execution.SkipState) {
	l.events <- state.Events
}

func TestHandleFunctionSkippedPassesSerializedEvents(t *testing.T) {
	listener := &recordingSkippedLifecycle{events: make(chan event.SerializedEvents, 1)}
	e := &executor{lifecycles: []execution.LifecycleListener{listener}}
	req := execution.ScheduleRequest{Events: testSerializedEvents(t,
		event.NewBaseTrackedEvent(event.Event{Name: "test/skipped", Data: map[string]any{}}, nil),
	)}

	_, _, err := e.handleFunctionSkipped(context.Background(), req, sv2.Metadata{}, enums.SkipReasonFunctionPaused)
	require.ErrorIs(t, err, ErrFunctionSkipped)
	select {
	case events := <-listener.events:
		require.True(t, req.Events.Equal(events))
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for skipped lifecycle")
	}
}

func (l *mutatingScheduledSyncLifecycle) OnFunctionScheduled(_ context.Context, _ sv2.Metadata, _ queue.Item, events event.SerializedEvents) {
	l.called = true
	l.createCalledFirst = l.runService.createCalled
	l.receivedEvents = events
	raw := events.RawMessages()
	raw[0][0] = 'x'
}

func TestScheduleCreatesStateBeforeInvokingSyncListenerWithSerializedEvents(t *testing.T) {
	runService := &recordingScheduleRunService{}
	listener := &mutatingScheduledSyncLifecycle{runService: runService}
	e := &executor{
		log:               logger.VoidLogger(),
		smv2:              runService,
		syncLifecycles:    []execution.SyncLifecycleListener{listener},
		tracerProvider:    tracing.NewNoopTracerProvider(),
		conditionalTracer: telemetrytrace.NoopConditionalTracer(),
	}
	firstEventID := ulid.Make()
	secondEventID := ulid.Make()
	req := execution.ScheduleRequest{
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       uuid.New(),
		Function: inngest.Function{
			ID:              uuid.New(),
			FunctionVersion: 1,
			Name:            "sync-listener-ordering",
		},
		Events: testSerializedEvents(t,
			event.InternalEvent{
				ID: firstEventID,
				Event: event.Event{
					ID:        firstEventID.String(),
					Name:      "test/sync-listener-ordering.first",
					Timestamp: time.Now().UnixMilli(),
					Data:      map[string]any{},
				},
			},
			event.InternalEvent{
				ID: secondEventID,
				Event: event.Event{
					ID:        secondEventID.String(),
					Name:      "test/sync-listener-ordering.second",
					Timestamp: time.Now().UnixMilli(),
					Data:      map[string]any{},
				},
			},
		),
		RunMode: enums.RunModeSync,
	}

	_, _, err := e.schedule(context.Background(), req, ulid.Make(), "test-key", false, nil)
	require.NoError(t, err)
	require.True(t, listener.called)
	require.True(t, listener.createCalledFirst, "state creation must complete before invoking the listener")
	require.True(t, req.Events.Equal(listener.receivedEvents))
	require.True(t, req.Events.Equal(runService.createEvents))
	require.Equal(t, req.Events.RawMessages(), runService.persistedEvent)
	require.Equal(t, byte('{'), req.Events.Event(0)[0], "listener materialization must not mutate the immutable snapshot")
}

func TestRunFunctionFinishedLifecycleCallsSyncListenerInline(t *testing.T) {
	sync := &recordingSyncLifecycle{}
	e := &executor{
		syncLifecycles: []execution.SyncLifecycleListener{sync},
	}

	e.RunFunctionFinishedLifecycle(context.Background(), sv2.Metadata{}, queue.Item{}, nil, statev1.DriverResponse{})

	// No channel/timeout wait needed: if this passes without any
	// synchronization, the call happened inline before
	// RunFunctionFinishedLifecycle returned — proving synchronous dispatch.
	require.Equal(t, 1, sync.finishedCalls)
}

type asyncOnlyLifecycle struct {
	execution.NoopLifecyceListener
	done chan struct{}
}

func (a *asyncOnlyLifecycle) OnFunctionFinished(context.Context, sv2.Metadata, queue.Item, []json.RawMessage, statev1.DriverResponse) {
	close(a.done)
}

func TestRunFunctionFinishedLifecycleSyncListenerDoesNotAffectAsyncListeners(t *testing.T) {
	async := &asyncOnlyLifecycle{done: make(chan struct{})}
	sync := &recordingSyncLifecycle{}
	e := &executor{
		lifecycles:     []execution.LifecycleListener{async},
		syncLifecycles: []execution.SyncLifecycleListener{sync},
	}

	e.RunFunctionFinishedLifecycle(context.Background(), sv2.Metadata{}, queue.Item{}, nil, statev1.DriverResponse{})

	// The sync listener already ran inline by the time we get here.
	require.Equal(t, 1, sync.finishedCalls)

	// The async listener runs on its own goroutine — it must eventually run,
	// but is not required to have run yet.
	select {
	case <-async.done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async listener")
	}
}

// TestSyncLifecycleRegistrationIsStaticOptionOnly locks the registration
// contract documented on execution.SyncLifecycleListener: sync listeners are
// supplied once at construction, via WithSyncLifecycleListeners, and there is
// deliberately no post-construction mutator. A dynamic AddSyncLifecycleListener
// (the counterpart AddLifecycleListener provides for async listeners) would let
// a consumer register with some dispatchers but not others, leaving a listener
// observing only part of a run's lifecycle -- so its absence is part of the
// contract, not an oversight.
func TestSyncLifecycleRegistrationIsStaticOptionOnly(t *testing.T) {
	// The option is a real registration path: a listener supplied through it
	// lands in the same slice the dispatchers fan out over.
	sync := &recordingSyncLifecycle{}
	e := &executor{log: logger.VoidLogger()}
	require.NoError(t, WithSyncLifecycleListeners(sync)(e))

	e.RunFunctionFinishedLifecycle(context.Background(), sv2.Metadata{}, queue.Item{}, nil, statev1.DriverResponse{})
	require.Equal(t, 1, sync.finishedCalls)

	// ...and it is the *only* registration path. Neither the concrete
	// *executor nor the execution.Executor interface may expose a mutator for
	// sync listeners.
	for _, typ := range []reflect.Type{
		reflect.TypeOf(&executor{}),
		reflect.TypeOf((*execution.Executor)(nil)).Elem(),
	} {
		for i := 0; i < typ.NumMethod(); i++ {
			name := typ.Method(i).Name
			if !strings.Contains(name, "Sync") {
				continue
			}
			require.False(t,
				strings.HasPrefix(name, "Add") || strings.HasPrefix(name, "Set") || strings.HasPrefix(name, "Register"),
				"%s.%s: sync lifecycle listeners are registered statically via WithSyncLifecycleListeners; adding a runtime mutator breaks that contract",
				typ, name,
			)
		}
	}
}
