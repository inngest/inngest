package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// recordingTracerProvider captures UpdateSpan calls so tests can inspect them.
type recordingTracerProvider struct {
	tracing.TracerProvider
	updateCalls []*tracing.UpdateSpanOptions
	updateErr   error
}

func (r *recordingTracerProvider) UpdateSpan(_ context.Context, opts *tracing.UpdateSpanOptions) error {
	r.updateCalls = append(r.updateCalls, opts)
	return r.updateErr
}

func newRecordingTracerProvider() *recordingTracerProvider {
	return &recordingTracerProvider{TracerProvider: tracing.NewNoopTracerProvider()}
}

func newSpanRef() *meta.SpanReference {
	return &meta.SpanReference{
		TraceParent: "00-00112233445566778899aabbccddeeff-0011223344556677-01",
	}
}

// invocationEvent builds an InternalEvent shaped like a function.invoked event
// with the supplied InngestMetadata. The InvokeCorrelationId is set so that
// (*InngestMetadata).RunID() returns sourceRunID.
func invocationEvent(t *testing.T, accountID, envID uuid.UUID, sourceRunID ulid.ULID, md event.InngestMetadata) event.InternalEvent {
	t.Helper()
	md.InvokeCorrelationId = sourceRunID.String() + ".step-1"
	if md.SourceFnID == "" {
		md.SourceFnID = uuid.NewString()
	}
	if md.SourceAppID == "" {
		md.SourceAppID = uuid.NewString()
	}
	id := ulid.MustNew(ulid.Now(), nil)
	return event.InternalEvent{
		ID:          id,
		AccountID:   accountID,
		WorkspaceID: envID,
		Event: event.Event{
			ID:        id.String(),
			Name:      consts.FnInvokeName,
			Timestamp: time.Now().UnixMilli(),
			Data: map[string]any{
				consts.InngestEventDataPrefix: md,
			},
		},
	}
}

func TestUpdateInvokeSpanWithInvokedRunID_WritesAttributesToInvokeSpan(t *testing.T) {
	rec := newRecordingTracerProvider()
	e := &executor{
		log:            logger.From(context.Background()),
		tracerProvider: rec,
	}

	accountID := uuid.New()
	envID := uuid.New()
	sourceRunID := ulid.MustNew(ulid.Now(), nil)
	invokedRunID := ulid.MustNew(ulid.Now(), nil)
	spanRef := newSpanRef()

	evt := invocationEvent(t, accountID, envID, sourceRunID, event.InngestMetadata{
		InvokeFnID:    "app-fn",
		InvokeSpanRef: spanRef,
	})

	e.updateInvokeSpanWithInvokedRunID(context.Background(), e.log, []event.TrackedEvent{evt}, invokedRunID)

	require.Len(t, rec.updateCalls, 1)
	call := rec.updateCalls[0]
	require.NotNil(t, call.TargetSpan)
	require.Equal(t, spanRef.TraceParent, call.TargetSpan.TraceParent,
		"target span must be the caller's invoke span (round-tripped through the invocation event)")
	require.NotNil(t, call.Debug)
	require.Equal(t, "executor.Schedule.invokeRunID", call.Debug.Location)

	gotInvokedRunID, ok := call.Attributes.Get(meta.Attrs.StepInvokeRunID.Key()).(*ulid.ULID)
	require.True(t, ok, "step.invoke.run.id must be set as *ulid.ULID")
	require.Equal(t, invokedRunID, *gotInvokedRunID)

	gotSourceRunID, ok := call.Attributes.Get(meta.Attrs.RunID.Key()).(*ulid.ULID)
	require.True(t, ok, "caller run id must be set to scope the target span")
	require.Equal(t, sourceRunID, *gotSourceRunID)

	gotAccountID, ok := call.Attributes.Get(meta.Attrs.AccountID.Key()).(*uuid.UUID)
	require.True(t, ok)
	require.Equal(t, accountID, *gotAccountID)

	gotEnvID, ok := call.Attributes.Get(meta.Attrs.EnvID.Key()).(*uuid.UUID)
	require.True(t, ok)
	require.Equal(t, envID, *gotEnvID)
}

func TestUpdateInvokeSpanWithInvokedRunID_NoCallsWhenNotInvoke(t *testing.T) {
	rec := newRecordingTracerProvider()
	e := &executor{
		log:            logger.From(context.Background()),
		tracerProvider: rec,
	}
	invokedRunID := ulid.MustNew(ulid.Now(), nil)

	// Same metadata shape, but the event name is not FnInvokeName — this run
	// was not triggered by an invoke step, so nothing should be written.
	evt := event.InternalEvent{
		ID: ulid.MustNew(ulid.Now(), nil),
		Event: event.Event{
			Name: "user/cron.triggered",
			Data: map[string]any{
				consts.InngestEventDataPrefix: event.InngestMetadata{
					InvokeSpanRef: newSpanRef(),
				},
			},
		},
	}

	e.updateInvokeSpanWithInvokedRunID(context.Background(), e.log, []event.TrackedEvent{evt}, invokedRunID)
	require.Empty(t, rec.updateCalls, "non-invoke events must not produce UpdateSpan calls")
}

func TestUpdateInvokeSpanWithInvokedRunID_SkipsEventsMissingSpanRef(t *testing.T) {
	rec := newRecordingTracerProvider()
	e := &executor{
		log:            logger.From(context.Background()),
		tracerProvider: rec,
	}

	accountID := uuid.New()
	envID := uuid.New()
	sourceRunID := ulid.MustNew(ulid.Now(), nil)
	invokedRunID := ulid.MustNew(ulid.Now(), nil)

	// First event has no span ref (older invocations that predate the field);
	// second event has the span ref — only the second should produce an update.
	noRef := invocationEvent(t, accountID, envID, sourceRunID, event.InngestMetadata{
		InvokeFnID: "app-fn",
	})
	withRef := invocationEvent(t, accountID, envID, sourceRunID, event.InngestMetadata{
		InvokeFnID:    "app-fn",
		InvokeSpanRef: newSpanRef(),
	})

	e.updateInvokeSpanWithInvokedRunID(context.Background(), e.log, []event.TrackedEvent{noRef, withRef}, invokedRunID)
	require.Len(t, rec.updateCalls, 1, "only the event carrying an invoke span ref produces an UpdateSpan call")
}

func TestUpdateInvokeSpanWithInvokedRunID_SwallowsTracerError(t *testing.T) {
	rec := newRecordingTracerProvider()
	rec.updateErr = errors.New("tracer offline")
	e := &executor{
		log:            logger.From(context.Background()),
		tracerProvider: rec,
	}

	evt := invocationEvent(t, uuid.New(), uuid.New(), ulid.MustNew(ulid.Now(), nil), event.InngestMetadata{
		InvokeFnID:    "app-fn",
		InvokeSpanRef: newSpanRef(),
	})

	// The Schedule path must not abort if the tracer fails — the invoke span
	// update is best-effort (the invoking function's invoke step still
	// completes when the invoked run finishes), so a tracer error is logged
	// and swallowed.
	require.NotPanics(t, func() {
		e.updateInvokeSpanWithInvokedRunID(context.Background(), e.log, []event.TrackedEvent{evt}, ulid.MustNew(ulid.Now(), nil))
	})
	require.Len(t, rec.updateCalls, 1)
}
