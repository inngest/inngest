package tracing

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestAddQueueTimestampAttrs(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	later := now.Add(5 * time.Second)
	earlier := now.Add(-5 * time.Second)

	t.Run("both zero: no attributes set", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{}
		AddQueueTimestampAttrs(attrs, item)

		_, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		_, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		assert.False(t, hasQueuedAt)
		assert.False(t, hasScheduledAt)
	})

	t.Run("only EnqueuedAt set: QueuedAt set, ScheduledAt absent", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{EnqueuedAt: now}
		AddQueueTimestampAttrs(attrs, item)

		queuedAt, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		_, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		require.True(t, hasQueuedAt)
		assert.Equal(t, now, *queuedAt)
		assert.False(t, hasScheduledAt)
	})

	t.Run("only At set: ScheduledAt equals At, QueuedAt absent", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{At: now}
		AddQueueTimestampAttrs(attrs, item)

		_, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		scheduledAt, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		assert.False(t, hasQueuedAt)
		require.True(t, hasScheduledAt)
		assert.Equal(t, now, *scheduledAt)
	})

	t.Run("At after EnqueuedAt: ScheduledAt equals At", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{EnqueuedAt: now, At: later}
		AddQueueTimestampAttrs(attrs, item)

		queuedAt, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		scheduledAt, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		require.True(t, hasQueuedAt)
		require.True(t, hasScheduledAt)
		assert.Equal(t, now, *queuedAt)
		assert.Equal(t, later, *scheduledAt)
	})

	t.Run("At before EnqueuedAt: EnqueuedAt fudged to ScheduledAt", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{EnqueuedAt: now, At: earlier}
		AddQueueTimestampAttrs(attrs, item)

		queuedAt, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		scheduledAt, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		require.True(t, hasQueuedAt)
		require.True(t, hasScheduledAt)
		assert.Equal(t, now, *queuedAt)
		// ScheduledAt must never be before QueuedAt
		assert.Equal(t, now, *scheduledAt)
		assert.False(t, scheduledAt.Before(*queuedAt))
	})

	t.Run("At equals EnqueuedAt: ScheduledAt equals both", func(t *testing.T) {
		attrs := meta.NewAttrSet()
		item := queue.Item{EnqueuedAt: now, At: now}
		AddQueueTimestampAttrs(attrs, item)

		queuedAt, hasQueuedAt := meta.GetAttr(attrs, meta.Attrs.QueuedAt)
		scheduledAt, hasScheduledAt := meta.GetAttr(attrs, meta.Attrs.ScheduledAt)
		require.True(t, hasQueuedAt)
		require.True(t, hasScheduledAt)
		assert.Equal(t, now, *queuedAt)
		assert.Equal(t, now, *scheduledAt)
	})
}

func TestExecutionProcessorStepStartWhenQueued(t *testing.T) {
	queuedAt := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	sdkStartedAt := queuedAt.Add(30 * time.Second)
	sdkEndedAt := sdkStartedAt.Add(time.Second)

	// emitStep creates an executor.step span the way the executor does, while
	// executing a queue item of the given kind, and returns the exported
	// span's typed attributes.
	emitStep := func(t *testing.T, queueKind string, op enums.Opcode, opts CreateSpanOptions) (*meta.ExtractedValues, time.Time) {
		t.Helper()

		exp := tracetest.NewInMemoryExporter()

		tp := NewOtelTracerProvider(exp, time.Millisecond)

		ctx := WithExecutionContext(t.Context(), ExecutionContext{QueueKind: queueKind})

		stepID := "step-id"

		attrs := meta.NewAttrSet(
			meta.Attr(meta.Attrs.StepID, &stepID),
			meta.Attr(meta.Attrs.StepOp, &op),
		)

		opts.Attributes = attrs.Merge(opts.Attributes)

		opts.Parent = RunSpanRefFromMetadata(&statev2.Metadata{ID: statev2.ID{RunID: ulid.Make()}})

		_, err := tp.CreateSpan(ctx, meta.SpanNameStep, &opts)
		require.NoError(t, err)

		spans := exp.GetSpans()
		require.Len(t, spans, 1)

		raw := map[string]any{}
		for _, kv := range spans[0].Attributes {
			raw[string(kv.Key)] = kv.Value.AsInterface()
		}

		ev, err := meta.ExtractTypedValues(t.Context(), raw)
		require.NoError(t, err)

		return ev, spans[0].StartTime
	}

	t.Run("failed step run while resuming a sleep keeps its SDK start", func(t *testing.T) {
		timing := meta.NewAttrSet()
		AddTimingAttrs(timing, queuedAt, queuedAt, sdkStartedAt, sdkEndedAt)

		ev, _ := emitStep(t, queue.KindSleep, enums.OpcodeStepFailed, CreateSpanOptions{
			Attributes: timing,
			StartTime:  queuedAt,
			EndTime:    sdkEndedAt,
		})

		require.NotNil(t, ev.StartedAt)
		assert.Equal(t, sdkStartedAt.UnixMilli(), ev.StartedAt.UnixMilli())
	})

	t.Run("step run while resuming a sleep keeps its SDK start", func(t *testing.T) {
		timing := meta.NewAttrSet()
		AddTimingAttrs(timing, queuedAt, queuedAt, sdkStartedAt, sdkEndedAt)

		ev, _ := emitStep(t, queue.KindSleep, enums.OpcodeStepRun, CreateSpanOptions{
			Attributes: timing,
			StartTime:  queuedAt,
			EndTime:    sdkEndedAt,
		})

		require.NotNil(t, ev.StartedAt)
		assert.Equal(t, sdkStartedAt.UnixMilli(), ev.StartedAt.UnixMilli())
	})

	t.Run("sleep's own span starts when queued", func(t *testing.T) {
		ev, spanStart := emitStep(t, queue.KindSleep, enums.OpcodeSleep, CreateSpanOptions{})

		require.NotNil(t, ev.StartedAt)
		assert.Equal(t, spanStart.UnixMilli(), ev.StartedAt.UnixMilli())
	})

	t.Run("waitForEvent starts when queued", func(t *testing.T) {
		ev, spanStart := emitStep(t, queue.KindEdge, enums.OpcodeWaitForEvent, CreateSpanOptions{})

		require.NotNil(t, ev.StartedAt)
		assert.Equal(t, spanStart.UnixMilli(), ev.StartedAt.UnixMilli())
	})

	t.Run("step run outside a sleep keeps its SDK start", func(t *testing.T) {
		timing := meta.NewAttrSet()
		AddTimingAttrs(timing, queuedAt, queuedAt, sdkStartedAt, sdkEndedAt)

		ev, _ := emitStep(t, queue.KindEdge, enums.OpcodeStepRun, CreateSpanOptions{
			Attributes: timing,
			StartTime:  queuedAt,
			EndTime:    sdkEndedAt,
		})

		require.NotNil(t, ev.StartedAt)
		assert.Equal(t, sdkStartedAt.UnixMilli(), ev.StartedAt.UnixMilli())
	})
}

func TestExecutionProcessorConnectInstanceID(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := NewOtelTracerProvider(exp, time.Millisecond)

	requestID := "request-id"
	base := WithExecutionContext(t.Context(), ExecutionContext{
		QueueKind: queue.KindEdge,
		RequestID: &requestID,
	})
	wrapped := WithConnectInstanceID(base, "worker-1")

	parent := RunSpanRefFromMetadata(&statev2.Metadata{ID: statev2.ID{RunID: ulid.Make()}})

	// extract returns the typed attributes of the most recently exported span.
	extract := func(t *testing.T) *meta.ExtractedValues {
		t.Helper()

		spans := exp.GetSpans()
		require.NotEmpty(t, spans)

		raw := map[string]any{}
		for _, kv := range spans[len(spans)-1].Attributes {
			raw[string(kv.Key)] = kv.Value.AsInterface()
		}

		ev, err := meta.ExtractTypedValues(t.Context(), raw)
		require.NoError(t, err)
		return ev
	}

	t.Run("step span without the instance ID has no attribute", func(t *testing.T) {
		_, err := tp.CreateSpan(base, meta.SpanNameStep, &CreateSpanOptions{
			Parent: parent,
		})
		require.NoError(t, err)

		ev := extract(t)
		assert.Nil(t, ev.ConnectInstanceID)
		require.NotNil(t, ev.RequestID)
		assert.Equal(t, requestID, *ev.RequestID)
	})

	t.Run("step span with the instance ID has the attribute", func(t *testing.T) {
		_, err := tp.CreateSpan(wrapped, meta.SpanNameStep, &CreateSpanOptions{
			Parent: parent,
		})
		require.NoError(t, err)

		spans := exp.GetSpans()
		found := false
		for _, kv := range spans[len(spans)-1].Attributes {
			if string(kv.Key) == "_inngest.connect.instance_id" {
				found = true
				assert.Equal(t, "worker-1", kv.Value.AsString())
			}
		}
		assert.True(t, found)

		ev := extract(t)
		require.NotNil(t, ev.ConnectInstanceID)
		assert.Equal(t, "worker-1", *ev.ConnectInstanceID)
		require.NotNil(t, ev.RequestID)
		assert.Equal(t, requestID, *ev.RequestID)
	})

	t.Run("extension span from UpdateSpan has the attribute", func(t *testing.T) {
		ref, err := tp.CreateSpan(base, meta.SpanNameExecution, &CreateSpanOptions{
			Parent: parent,
		})
		require.NoError(t, err)

		err = tp.UpdateSpan(wrapped, &UpdateSpanOptions{
			TargetSpan: ref,
			Status:     enums.StepStatusCompleted,
		})
		require.NoError(t, err)

		ev := extract(t)
		require.NotNil(t, ev.ConnectInstanceID)
		assert.Equal(t, "worker-1", *ev.ConnectInstanceID)
	})

	t.Run("the original execution context is not changed", func(t *testing.T) {
		ec := getExecutionContext(base)
		require.NotNil(t, ec)
		assert.Empty(t, ec.InstanceID)

		wrappedEC := getExecutionContext(wrapped)
		require.NotNil(t, wrappedEC)
		assert.NotSame(t, ec, wrappedEC)
		assert.Equal(t, "worker-1", wrappedEC.InstanceID)
		assert.Equal(t, ec.RequestID, wrappedEC.RequestID)
	})

	t.Run("empty ID or no execution context returns ctx unchanged", func(t *testing.T) {
		assert.Equal(t, base, WithConnectInstanceID(base, ""))

		plain := t.Context()
		assert.Equal(t, plain, WithConnectInstanceID(plain, "worker-1"))
		assert.Nil(t, getExecutionContext(WithConnectInstanceID(plain, "worker-1")))
	})
}
