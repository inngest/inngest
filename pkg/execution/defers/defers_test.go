package defers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// capturingTracer records the spans CreateSpan is asked to emit so tests can
// assert on the attributes that downstream linkage reconstruction reads back.
type capturingTracer struct {
	tracing.TracerProvider
	spans []capturedSpan
}

type capturedSpan struct {
	name  string
	attrs *meta.ExtractedValues
}

func (c *capturingTracer) CreateSpan(_ context.Context, name string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error) {
	// Round-trip through the same serialize -> extract path production uses, so
	// the test observes exactly what GetRunDefers would read off the span.
	raw := map[string]any{}
	if opts.Attributes != nil {
		for _, kv := range opts.Attributes.Serialize() {
			raw[string(kv.Key)] = kv.Value.AsInterface()
		}
	}
	extracted, err := meta.ExtractTypedValues(context.Background(), raw)
	if err != nil {
		return nil, err
	}
	c.spans = append(c.spans, capturedSpan{name: name, attrs: extracted})
	return &meta.SpanReference{}, nil
}

type fakeRunService struct {
	statev2.RunService

	saveDeferErr          error
	savedDefer            *statev2.Defer
	savedDeferCalls       int
	savedRejectedFnSlug   string
	savedRejectedHashedID string
	savedRejectedCalls    int
	setDeferStatusHashed  string
	setDeferStatusValue   enums.DeferStatus
}

func (f *fakeRunService) SaveDefer(_ context.Context, _ statev2.ID, d statev2.Defer) error {
	f.savedDeferCalls++
	f.savedDefer = &d
	return f.saveDeferErr
}

func (f *fakeRunService) SaveRejectedDefer(_ context.Context, _ statev2.ID, fnSlug string, hashedID string) error {
	f.savedRejectedCalls++
	f.savedRejectedFnSlug = fnSlug
	f.savedRejectedHashedID = hashedID
	return nil
}

func (f *fakeRunService) SetDeferStatus(_ context.Context, _ statev2.ID, hashedID string, status enums.DeferStatus) error {
	f.setDeferStatusHashed = hashedID
	f.setDeferStatusValue = status
	return nil
}

func runID() statev2.ID {
	return statev2.ID{
		RunID:      ulid.MustNew(ulid.Now(), nil),
		FunctionID: uuid.New(),
	}
}

func deferAddOp(t *testing.T, hashedID, userlandID string, opts state.DeferAddOpts) state.GeneratorOpcode {
	t.Helper()
	raw, err := json.Marshal(opts)
	require.NoError(t, err)
	op := state.GeneratorOpcode{
		Op:   enums.OpcodeDeferAdd,
		ID:   hashedID,
		Opts: json.RawMessage(raw),
	}
	if userlandID != "" {
		op.Userland = &struct {
			ID    string `json:"id"`
			Index int    `json:"index,omitempty"`
		}{ID: userlandID}
	}
	return op
}

func TestSaveFromOp_Accepted(t *testing.T) {
	t.Run("propagates UserlandID", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-1", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 1, fake.savedDeferCalls)
		require.Equal(t, 0, fake.savedRejectedCalls)
		require.Equal(t, "user-defer-id", fake.savedDefer.UserlandID)
		require.Equal(t, "hash-1", fake.savedDefer.HashedID)
		require.Equal(t, enums.DeferStatusAfterRun, fake.savedDefer.ScheduleStatus)
	})

	t.Run("Userland nil → empty UserlandID, still accepted", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-1", "", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 1, fake.savedDeferCalls)
		require.Empty(t, fake.savedDefer.UserlandID)
	})
}

func TestSaveFromOp_Rejected(t *testing.T) {
	t.Run("per_defer_size writes sentinel", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-too-big", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`"` + strings.Repeat("a", consts.MaxDeferInputSize+1) + `"`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 0, fake.savedDeferCalls)
		require.Equal(t, 1, fake.savedRejectedCalls)
		require.Equal(t, "child-fn", fake.savedRejectedFnSlug)
		require.Equal(t, "hash-too-big", fake.savedRejectedHashedID)
	})

	t.Run("invalid_opts with FnSlug present writes sentinel", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-noinput", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 0, fake.savedDeferCalls)
		require.Equal(t, 1, fake.savedRejectedCalls)
		require.Equal(t, "child-fn", fake.savedRejectedFnSlug)
		require.Equal(t, "hash-noinput", fake.savedRejectedHashedID)
	})

	t.Run("invalid_opts without FnSlug absorbs without sentinel", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-empty", "user-defer-id", state.DeferAddOpts{
			Input: json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 0, fake.savedDeferCalls)
		require.Equal(t, 0, fake.savedRejectedCalls)
	})

	t.Run("SaveDefer sentinel error soft-rejects", func(t *testing.T) {
		fake := &fakeRunService{saveDeferErr: statev2.ErrDeferLimitExceeded}
		op := deferAddOp(t, "hash-1", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 1, fake.savedDeferCalls)
		require.Equal(t, 0, fake.savedRejectedCalls)
	})

	t.Run("infra error surfaces", func(t *testing.T) {
		fake := &fakeRunService{saveDeferErr: errors.New("redis dead")}
		op := deferAddOp(t, "hash-1", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.Error(t, err)
	})
}

func TestAbortFromOp(t *testing.T) {
	t.Run("flips status to Aborted", func(t *testing.T) {
		fake := &fakeRunService{}
		raw, err := json.Marshal(state.DeferAbortOpts{TargetHashedID: "hash-abort"})
		require.NoError(t, err)
		op := state.GeneratorOpcode{
			Op:   enums.OpcodeDeferAbort,
			ID:   "step-id",
			Opts: json.RawMessage(raw),
		}

		err = AbortFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, "hash-abort", fake.setDeferStatusHashed)
		require.Equal(t, enums.DeferStatusAborted, fake.setDeferStatusValue)
	})

	// The UI reconstructs defer status purely from executor.defer spans, so an
	// abort that only mutates run state would keep displaying "Scheduled".
	// Assert the abort emits a defer span whose surfaced status is Aborted.
	t.Run("emits an executor.defer span carrying the Aborted status", func(t *testing.T) {
		fake := &fakeRunService{}
		tracer := &capturingTracer{}
		raw, err := json.Marshal(state.DeferAbortOpts{TargetHashedID: "hash-abort"})
		require.NoError(t, err)
		op := state.GeneratorOpcode{
			Op:   enums.OpcodeDeferAbort,
			ID:   "step-id",
			Opts: json.RawMessage(raw),
		}

		err = AbortFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracer, statev2.Metadata{}, time.Now())
		require.NoError(t, err)

		require.Len(t, tracer.spans, 1, "abort must emit exactly one executor.defer span")
		span := tracer.spans[0]
		require.Equal(t, meta.SpanNameDefer, span.name)
		require.NotNil(t, span.attrs.DeferHashedID)
		require.Equal(t, "hash-abort", *span.attrs.DeferHashedID,
			"abort span must reference the aborted defer's hashed ID so GetRunDefers collapses it onto the schedule span")
		require.NotNil(t, span.attrs.DeferStatus)
		require.Equal(t, enums.DeferStatusAborted, *span.attrs.DeferStatus,
			"the surfaced defer status must flip to Aborted")
	})

	t.Run("surfaces parse error", func(t *testing.T) {
		fake := &fakeRunService{}
		tracer := &capturingTracer{}
		op := state.GeneratorOpcode{
			Op:   enums.OpcodeDeferAbort,
			Opts: json.RawMessage(`{}`),
		}

		err := AbortFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracer, statev2.Metadata{}, time.Time{})

		require.Error(t, err)
		require.Empty(t, fake.setDeferStatusHashed)
		require.Empty(t, tracer.spans, "no span should be emitted when opts fail to parse")
	})
}
