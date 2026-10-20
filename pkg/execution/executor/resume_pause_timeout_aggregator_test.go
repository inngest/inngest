package executor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/expressions/expragg"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// timeoutAggLoader is an EvaluableLoader backed by an in-memory pause slice,
// standing in for the pause manager as the aggregator's source of pauses.
type timeoutAggLoader struct {
	mu     sync.Mutex
	pauses []*state.Pause
}

func (l *timeoutAggLoader) LoadEvaluablesSince(ctx context.Context, workspaceID uuid.UUID, eventName string, since time.Time, do func(context.Context, expr.Evaluable) error) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, p := range l.pauses {
		if p.WorkspaceID != workspaceID || p.GetEvent() == nil || *p.GetEvent() != eventName {
			continue
		}
		if err := do(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func (l *timeoutAggLoader) remove(id uuid.UUID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.pauses[:0]
	for _, p := range l.pauses {
		if p.ID != id {
			kept = append(kept, p)
		}
	}
	l.pauses = kept
}

// timeoutAggPauseMgr deletes pauses from the loader, mimicking the real pause
// manager: once deleted, the pause is no longer returned by future loads.
type timeoutAggPauseMgr struct {
	pauses.Manager
	loader *timeoutAggLoader
}

func (m *timeoutAggPauseMgr) Delete(_ context.Context, _ pauses.Index, pause state.Pause, _ ...state.DeletePauseOpt) error {
	m.loader.remove(pause.ID)
	return nil
}

// A waitForEvent pause that ends by timeout must be removed from the
// expression aggregator, not only from the pause store.  Otherwise the
// aggregator keeps evaluating the dead pause against every matching event
// for the lifetime of the process.
func TestResumePauseTimeoutRemovesPauseFromAggregator(t *testing.T) {
	ctx := context.Background()

	runID := ulid.MustNew(ulid.Now(), nil)
	wsID, fnID, aID := uuid.New(), uuid.New(), uuid.New()

	md := sv2.Metadata{
		ID: sv2.ID{
			RunID:      runID,
			FunctionID: fnID,
			Tenant:     sv2.Tenant{EnvID: wsID, AccountID: aID},
		},
		Config: *sv2.InitConfig(&sv2.Config{}),
	}

	eventName := "test/event"
	expression := "async.data.x == 1"
	opcode := enums.OpcodeWaitForEvent.String()

	pause := state.Pause{
		ID:          uuid.New(),
		WorkspaceID: wsID,
		DataKey:     "wait-step",
		Identifier: state.PauseIdentifier{
			RunID:      runID,
			FunctionID: fnID,
			AccountID:  aID,
		},
		Event:      &eventName,
		Expression: &expression,
		Opcode:     &opcode,
	}

	loader := &timeoutAggLoader{pauses: []*state.Pause{&pause}}
	agg := expragg.NewAggregator(ctx, 100, 100, loader, expressions.ExprEvaluator, nil, nil)

	e := &executor{
		smv2:           &stubRunServiceMD{md: md},
		pm:             &timeoutAggPauseMgr{loader: loader},
		queue:          &stubQueue{},
		log:            logger.From(ctx),
		tracerProvider: tracing.NewNoopTracerProvider(),
		exprAggregator: agg,
	}

	tracked := event.InternalEvent{
		ID:          ulid.MustNew(ulid.Now(), nil),
		AccountID:   aID,
		WorkspaceID: wsID,
		Event: event.Event{
			Name: eventName,
			Data: map[string]any{"x": 1},
		},
	}

	// Sanity: the aggregator matches the pause for a matching event.
	found, _, err := agg.EvaluateAsyncEvent(ctx, tracked)
	require.NoError(t, err)
	require.Len(t, found, 1)

	require.NoError(t, e.ResumePauseTimeout(ctx, pause, execution.ResumeRequest{}))

	// After the pause ends by timeout, the aggregator must no longer match it.
	found, _, err = agg.EvaluateAsyncEvent(ctx, tracked)
	require.NoError(t, err)
	require.Empty(t, found, "timed-out pause must be removed from the expression aggregator")
}
