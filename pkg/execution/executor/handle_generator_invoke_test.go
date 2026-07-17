package executor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// stubPauseManager satisfies pauses.Manager for tests that only need Write to
// succeed (the invoke path writes one pause before publishing the event).
type stubPauseManager struct{ pauses.Manager }

func (s *stubPauseManager) Write(_ context.Context, _ pauses.Index, _ ...*state.Pause) (int, error) {
	return 1, nil
}

// TestHandleGeneratorInvokeFunction_ResolvesSessionsBeforePublish proves the
// executor's step.invoke path actually calls EventMeta.ResolveSessions() on the
// invocation event before publishing it — the internal invocation event never
// passes through the API ingest path (ReceiveEvent) that resolves regular
// events, so this is the only fast guard for that wiring.
//
// It captures the published event via the injected handleInvokeEvent seam and
// asserts the merge happened (manual wins per key, disjoint propagated fills,
// propagated layer cleared). The merge *semantics* are covered exhaustively in
// pkg/event/sessions_test.go; here we only assert the executor invokes them.
func TestHandleGeneratorInvokeFunction_ResolvesSessionsBeforePublish(t *testing.T) {
	var captured event.TrackedEvent
	e := &executor{
		log:            logger.From(context.Background()),
		tracerProvider: tracing.NewNoopTracerProvider(),
		pm:             &stubPauseManager{},
		queue:          &stubQueue{},
		handleInvokeEvent: func(_ context.Context, evt event.TrackedEvent) error {
			captured = evt
			return nil
		},
	}

	rc := &mockRunContext{
		md: sv2.Metadata{
			ID:     sv2.ID{RunID: ulid.MustNew(ulid.Now(), nil), FunctionID: uuid.New()},
			Config: *sv2.InitConfig(&sv2.Config{}),
		},
	}

	// Raw-bytes Opts round-trips through EventMeta.UnmarshalJSON, exercising the
	// same two-layer wire (and null-tombstone capture) the SDK stamps.
	gen := state.GeneratorOpcode{
		Op: enums.OpcodeInvokeFunction,
		ID: "step-invoke",
		Opts: []byte(`{"function_id":"app-fn","payload":{
			"name":"some/event","data":{"foo":"bar"},
			"meta":{
				"sessions":{"conv_id":"manual","cut_me":null},
				"propagatedSessions":{"conv_id":"parent","org_id":"42","cut_me":"inherited"}
			}}}`),
	}
	edge := queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}

	err := e.handleGeneratorInvokeFunction(context.Background(), rc, gen, edge, OpcodeGroup{})
	require.NoError(t, err)
	require.NotNil(t, captured, "handleInvokeEvent must be called with the invocation event")

	got := captured.GetEvent()
	require.Equal(t, event.InvokeFnName, got.Name)
	// Manual wins conv_id; org_id fills from propagated; the null tombstone cuts
	// the inherited cut_me; the propagated layer is consumed.
	require.Equal(t, event.Sessions{"conv_id": "manual", "org_id": "42"}, got.Meta.Sessions)
	require.Nil(t, got.Meta.PropagatedSessions, "propagated layer must be cleared before publish")
}

// TestHandleGeneratorInvokeFunction_ValidatesSessionsAfterResolve proves the
// executor validates sessions *after* the merge, closing a gap the pre-merge
// InvokeFunctionOpts.Validate() (manual layer only) leaves open
//
// The error must be non-retryable (a bad payload never self-heals).
func TestHandleGeneratorInvokeFunction_ValidatesSessionsAfterResolve(t *testing.T) {
	var called bool
	e := &executor{
		log:            logger.From(context.Background()),
		tracerProvider: tracing.NewNoopTracerProvider(),
		pm:             &stubPauseManager{},
		queue:          &stubQueue{},
		handleInvokeEvent: func(_ context.Context, _ event.TrackedEvent) error {
			called = true
			return nil
		},
	}

	rc := &mockRunContext{
		md: sv2.Metadata{
			ID:     sv2.ID{RunID: ulid.MustNew(ulid.Now(), nil), FunctionID: uuid.New()},
			Config: *sv2.InitConfig(&sv2.Config{}),
		},
	}

	// Legal manual layer (empty), but the propagated layer has an empty id. The
	// pre-merge manual-only check passes; only the post-merge validate catches it.
	gen := state.GeneratorOpcode{
		Op: enums.OpcodeInvokeFunction,
		ID: "step-invoke",
		Opts: []byte(`{"function_id":"app-fn","payload":{
			"name":"some/event","data":{"foo":"bar"},
			"meta":{"propagatedSessions":{"org_id":""}}
		}}`),
	}
	edge := queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}

	err := e.handleGeneratorInvokeFunction(context.Background(), rc, gen, edge, OpcodeGroup{})
	require.Error(t, err)
	require.False(t, called, "an invalid invocation event must never be published")

	var execErr execError
	require.ErrorAs(t, err, &execErr)
	require.False(t, execErr.Retryable(), "session validation failure is a final user error")
}
