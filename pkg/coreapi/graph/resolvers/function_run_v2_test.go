package resolvers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// fakeInvokedFromManager is a minimal cqrs.Manager stub for the invokedFrom
// resolvers. Methods that are not implemented fall through to the embedded
// nil interface and panic if called.
type fakeInvokedFromManager struct {
	cqrs.Manager

	events    map[ulid.ULID]*cqrs.Event
	eventsErr error
	eventCall int

	functions map[uuid.UUID]*cqrs.Function
	runs      map[ulid.ULID]*cqrs.TraceRun
	runErr    error
}

func (f *fakeInvokedFromManager) GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*cqrs.Event, error) {
	f.eventCall++
	if f.eventsErr != nil {
		return nil, f.eventsErr
	}
	out := []*cqrs.Event{}
	for _, id := range ids {
		if evt, ok := f.events[id]; ok {
			out = append(out, evt)
		}
	}
	return out, nil
}

func (f *fakeInvokedFromManager) GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*cqrs.Function, error) {
	if fn, ok := f.functions[fnID]; ok {
		return fn, nil
	}
	return nil, sql.ErrNoRows
}

func (f *fakeInvokedFromManager) GetTraceRun(ctx context.Context, id cqrs.TraceRunIdentifier) (*cqrs.TraceRun, error) {
	if f.runErr != nil {
		return nil, f.runErr
	}
	if run, ok := f.runs[id.RunID]; ok {
		return run, nil
	}
	return nil, sql.ErrNoRows
}

// storedEvent converts an event to the shape it has when read back from the
// events table: its data goes through JSON, so typed metadata becomes a map.
func storedEvent(t *testing.T, internalID ulid.ULID, evt event.Event) *cqrs.Event {
	t.Helper()
	byt, err := json.Marshal(evt)
	require.NoError(t, err)
	var decoded event.Event
	require.NoError(t, json.Unmarshal(byt, &decoded))
	stored := cqrs.ConvertFromEvent(internalID, decoded)
	return &stored
}

func invocationEvent(t *testing.T, correlationID *string, sourceFnID string) (ulid.ULID, *cqrs.Event) {
	t.Helper()
	tracked := event.NewInvocationEvent(event.NewInvocationEventOpts{
		Event:         event.Event{Data: map[string]any{"foo": "bar"}},
		FnID:          "app-child",
		CorrelationID: correlationID,
		SourceAppID:   uuid.NewString(),
		SourceFnID:    sourceFnID,
	})
	return tracked.ID, storedEvent(t, tracked.ID, tracked.Event)
}

func TestFunctionRunV2InvokedFrom(t *testing.T) {
	ctx := context.Background()
	parentRunID := ulid.Make()
	parentFnID := uuid.New()
	correlationID := parentRunID.String() + "." + "step-hash"

	t.Run("returns the invoking run and function for step.invoke runs", func(t *testing.T) {
		evtID, evt := invocationEvent(t, &correlationID, parentFnID.String())
		mgr := &fakeInvokedFromManager{events: map[ulid.ULID]*cqrs.Event{evtID: evt}}
		r := &functionRunV2Resolver{&Resolver{Data: mgr}}

		got, err := r.InvokedFrom(ctx, &models.FunctionRunV2{TriggerIDs: []ulid.ULID{evtID}})
		require.NoError(t, err)
		require.Equal(t, &models.RunInvokedFrom{RunID: parentRunID, FunctionID: parentFnID}, got)
	})

	t.Run("keeps the run link when the function ID cannot be parsed", func(t *testing.T) {
		evtID, evt := invocationEvent(t, &correlationID, "not-a-uuid")
		mgr := &fakeInvokedFromManager{events: map[ulid.ULID]*cqrs.Event{evtID: evt}}
		r := &functionRunV2Resolver{&Resolver{Data: mgr}}

		got, err := r.InvokedFrom(ctx, &models.FunctionRunV2{TriggerIDs: []ulid.ULID{evtID}})
		require.NoError(t, err)
		require.Equal(t, &models.RunInvokedFrom{RunID: parentRunID, FunctionID: uuid.Nil}, got)
	})

	t.Run("returns null for runs invoked without a parent run", func(t *testing.T) {
		// Invocations made through the API carry no correlation ID.
		evtID, evt := invocationEvent(t, nil, "")
		mgr := &fakeInvokedFromManager{events: map[ulid.ULID]*cqrs.Event{evtID: evt}}
		r := &functionRunV2Resolver{&Resolver{Data: mgr}}

		got, err := r.InvokedFrom(ctx, &models.FunctionRunV2{TriggerIDs: []ulid.ULID{evtID}})
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("returns null for runs triggered by other events", func(t *testing.T) {
		evtID := ulid.Make()
		evt := storedEvent(t, evtID, event.Event{
			Name: "app/user.created",
			Data: map[string]any{
				// A user event can't impersonate an invocation by copying
				// its metadata.
				"_inngest": map[string]any{"correlation_id": correlationID},
			},
		})
		mgr := &fakeInvokedFromManager{events: map[ulid.ULID]*cqrs.Event{evtID: evt}}
		r := &functionRunV2Resolver{&Resolver{Data: mgr}}

		got, err := r.InvokedFrom(ctx, &models.FunctionRunV2{TriggerIDs: []ulid.ULID{evtID}})
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("returns null when the trigger event no longer exists", func(t *testing.T) {
		mgr := &fakeInvokedFromManager{}
		r := &functionRunV2Resolver{&Resolver{Data: mgr}}

		got, err := r.InvokedFrom(ctx, &models.FunctionRunV2{TriggerIDs: []ulid.ULID{ulid.Make()}})
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("skips the event lookup for runs without exactly one trigger", func(t *testing.T) {
		mgr := &fakeInvokedFromManager{}
		r := &functionRunV2Resolver{&Resolver{Data: mgr}}

		for _, triggerIDs := range [][]ulid.ULID{
			nil,
			{ulid.Make(), ulid.Make()},
		} {
			got, err := r.InvokedFrom(ctx, &models.FunctionRunV2{TriggerIDs: triggerIDs})
			require.NoError(t, err)
			require.Nil(t, got)
		}
		require.Zero(t, mgr.eventCall)
	})

	t.Run("returns the error when the trigger event can't be loaded", func(t *testing.T) {
		mgr := &fakeInvokedFromManager{eventsErr: errors.New("db unavailable")}
		r := &functionRunV2Resolver{&Resolver{Data: mgr}}

		_, err := r.InvokedFrom(ctx, &models.FunctionRunV2{TriggerIDs: []ulid.ULID{ulid.Make()}})
		require.ErrorContains(t, err, "db unavailable")
	})
}

func TestRunInvokedFromFunction(t *testing.T) {
	ctx := context.Background()
	fnID := uuid.New()
	mgr := &fakeInvokedFromManager{
		functions: map[uuid.UUID]*cqrs.Function{
			fnID: {
				ID:     fnID,
				Name:   "Parent",
				Slug:   "app-parent",
				Config: json.RawMessage(`{"name":"Parent","slug":"app-parent","steps":[{"id":"step","name":"step","uri":"http://localhost:3000/api/inngest"}]}`),
			},
		},
	}
	r := &runInvokedFromResolver{&Resolver{Data: mgr}}

	fn, err := r.Function(ctx, &models.RunInvokedFrom{FunctionID: fnID})
	require.NoError(t, err)
	require.NotNil(t, fn)
	require.Equal(t, "app-parent", fn.Slug)
	require.Equal(t, "Parent", fn.Name)

	fn, err = r.Function(ctx, &models.RunInvokedFrom{FunctionID: uuid.New()})
	require.NoError(t, err, "a deleted function must not fail the query")
	require.Nil(t, fn)

	fn, err = r.Function(ctx, &models.RunInvokedFrom{FunctionID: uuid.Nil})
	require.NoError(t, err)
	require.Nil(t, fn)
}

func TestRunInvokedFromRun(t *testing.T) {
	ctx := context.Background()
	parentRunID := ulid.Make()
	mgr := &fakeInvokedFromManager{
		runs: map[ulid.ULID]*cqrs.TraceRun{
			parentRunID: {
				RunID:    parentRunID.String(),
				QueuedAt: time.Now(),
				Status:   enums.RunStatusRunning,
			},
		},
	}
	r := &runInvokedFromResolver{&Resolver{Data: mgr}}

	run, err := r.Run(ctx, &models.RunInvokedFrom{RunID: parentRunID})
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, parentRunID, run.ID)
	require.Equal(t, models.FunctionRunStatusRunning, run.Status)

	run, err = r.Run(ctx, &models.RunInvokedFrom{RunID: ulid.Make()})
	require.NoError(t, err, "a missing invoking run must not fail the query")
	require.Nil(t, run)

	mgr.runErr = errors.New("db unavailable")
	_, err = r.Run(ctx, &models.RunInvokedFrom{RunID: parentRunID})
	require.Error(t, err)
}
