package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// fixedSpanRefTracerProvider returns a known ref for droppable (planned) spans
// so tests can assert what metadata is parented on.
type fixedSpanRefTracerProvider struct {
	tracing.TracerProvider
	ref *meta.SpanReference
}

func (p *fixedSpanRefTracerProvider) CreateDroppableSpan(ctx context.Context, name string, opts *tracing.CreateSpanOptions) (*tracing.DroppableSpan, error) {
	span, err := p.TracerProvider.CreateDroppableSpan(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	span.Ref = p.ref

	return span, nil
}

// TestHandleGeneratorInvokeFunction_KeepsStepMetadata proves a planned invoke
// op carrying step metadata produces the same metadata records as a run step
// does, differing only in the parent: the invoke span has a random ID, so the
// records hang off the planned span's own ref rather than the deterministic
// finalized step span.
func TestHandleGeneratorInvokeFunction_KeepsStepMetadata(t *testing.T) {
	ctx := context.Background()

	invokeSpanRef := &meta.SpanReference{DynamicSpanID: "invoke-span-id"}
	recorder := &metadataEntryRecorder{}
	e := &executor{
		log: logger.From(ctx),
		tracerProvider: &fixedSpanRefTracerProvider{
			TracerProvider: tracing.NewNoopTracerProvider(),
			ref:            invokeSpanRef,
		},
		pm:                &stubPauseManager{},
		queue:             &stubQueue{},
		syncLifecycles:    []execution.SyncLifecycleListener{recorder},
		handleInvokeEvent: func(context.Context, event.TrackedEvent) error { return nil },
	}

	rc := &mockRunContext{
		md: sv2.Metadata{
			ID:     sv2.ID{RunID: ulid.MustNew(ulid.Now(), nil), FunctionID: uuid.New()},
			Config: *sv2.InitConfig(&sv2.Config{}),
		},
	}

	updates := []metadata.ScopedUpdate{{
		Scope: enums.MetadataScopeStep,
		Update: metadata.Update{RawUpdate: metadata.RawUpdate{
			Kind:   "userland.test",
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{"foo": json.RawMessage(`"bar"`)},
		}},
	}}

	// The same metadata on a run step is the baseline.
	runGen := state.GeneratorOpcode{Op: enums.OpcodeStepRun, ID: "step-1", Metadata: updates}
	e.handleGeneratorMetadata(ctx, rc, &runGen)
	require.Len(t, recorder.entries, 1)
	runEntry := recorder.entries[0]
	recorder.entries = nil

	invokeGen := state.GeneratorOpcode{
		Op:       enums.OpcodeInvokeFunction,
		ID:       "step-1",
		Opts:     []byte(`{"function_id":"app-fn","payload":{"name":"some/event","data":{}}}`),
		Metadata: updates,
	}
	edge := queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}
	require.NoError(t, e.handleGeneratorInvokeFunction(ctx, rc, invokeGen, edge, OpcodeGroup{}))

	require.Len(t, recorder.entries, 1, "invoke metadata must be recorded at planning time")
	invokeEntry := recorder.entries[0]

	require.Equal(t, runEntry.Kind, invokeEntry.Kind)
	require.Equal(t, runEntry.Scope, invokeEntry.Scope)
	require.Equal(t, runEntry.Values, invokeEntry.Values)
	require.Equal(t, runEntry.RunID, invokeEntry.RunID)
	require.Equal(t, runEntry.StepID, invokeEntry.StepID)
	require.Equal(t, runEntry.StepHashedID, invokeEntry.StepHashedID)
	require.Equal(t, runEntry.StepAttempt, invokeEntry.StepAttempt)

	require.Equal(t, tracing.FinalizedStepSpanRefFromMetadataAndStepID(&rc.md, "step-1"), runEntry.Parent)
	require.Equal(t, invokeSpanRef, invokeEntry.Parent)
}
