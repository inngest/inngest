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

	saveDeferErr error
	saved        []statev2.Defer
}

func (f *fakeRunService) SaveDefer(_ context.Context, _ statev2.ID, d statev2.Defer) error {
	f.saved = append(f.saved, d)
	return f.saveDeferErr
}

func (f *fakeRunService) countByStatus(s enums.DeferStatus) int {
	n := 0
	for _, d := range f.saved {
		if d.ScheduleStatus == s {
			n++
		}
	}
	return n
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
		require.Len(t, fake.saved, 1)
		require.Equal(t, "user-defer-id", fake.saved[0].UserlandID)
		require.Equal(t, "hash-1", fake.saved[0].HashedID)
		require.Equal(t, enums.DeferStatusAfterRun, fake.saved[0].ScheduleStatus)
	})

	t.Run("Userland nil → empty UserlandID, still accepted", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-1", "", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Len(t, fake.saved, 1)
		require.Empty(t, fake.saved[0].UserlandID)
	})
}

func TestSaveFromOp_Rejected(t *testing.T) {
	t.Run("per_defer_size persists Rejected", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-too-big", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`"` + strings.Repeat("a", consts.MaxDeferInputSize+1) + `"`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Len(t, fake.saved, 1)
		require.Equal(t, enums.DeferStatusRejected, fake.saved[0].ScheduleStatus)
		require.Equal(t, "child-fn", fake.saved[0].FnSlug)
		require.Equal(t, "hash-too-big", fake.saved[0].HashedID)
	})

	t.Run("invalid_opts with FnSlug present persists Rejected", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-noinput", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Len(t, fake.saved, 1)
		require.Equal(t, enums.DeferStatusRejected, fake.saved[0].ScheduleStatus)
		require.Equal(t, "child-fn", fake.saved[0].FnSlug)
		require.Equal(t, "hash-noinput", fake.saved[0].HashedID)
	})

	t.Run("invalid_opts without FnSlug absorbs without persisting", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-empty", "user-defer-id", state.DeferAddOpts{
			Input: json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Empty(t, fake.saved)
	})

	t.Run("SaveDefer sentinel error soft-rejects", func(t *testing.T) {
		fake := &fakeRunService{saveDeferErr: statev2.ErrDeferLimitExceeded}
		op := deferAddOp(t, "hash-1", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 1, fake.countByStatus(enums.DeferStatusAfterRun))
		require.Equal(t, 0, fake.countByStatus(enums.DeferStatusRejected))
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

