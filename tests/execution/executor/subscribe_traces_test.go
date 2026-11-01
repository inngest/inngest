package executor

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// traceHarness is a subscribeHarness whose spans are kept, so a test can see
// what the trace of the run is made of.
type traceHarness struct {
	*subscribeHarness
	spans *tracetest.InMemoryExporter
}

func newTraceHarness(t *testing.T, ops ...*state.GeneratorOpcode) *traceHarness {
	t.Helper()

	spans := tracetest.NewInMemoryExporter()
	h := newSubscribeHarnessWith(t,
		[]executor.ExecutorOpt{
			executor.WithTracerProvider(tracing.NewOtelTracerProvider(spans, time.Millisecond)),
		},
		ops...,
	)

	return &traceHarness{subscribeHarness: h, spans: spans}
}

type seenSpan struct {
	name   string
	parent string
	attrs  map[string]attribute.Value
}

// seen is every span the executor has emitted whose name is name.
func (h *traceHarness) seen(name string) []seenSpan {
	h.t.Helper()

	out := []seenSpan{}
	for _, span := range h.spans.GetSpans() {
		if span.Name != name {
			continue
		}

		attrs := map[string]attribute.Value{}
		for _, kv := range span.Attributes {
			attrs[string(kv.Key)] = kv.Value
		}

		out = append(out, seenSpan{
			name:   span.Name,
			parent: span.Parent.SpanID().String(),
			attrs:  attrs,
		})
	}

	return out
}

// rowID is the id of the span that draws the step with this id.
func rowID(step string) string {
	return tracing.DeterministicSpanConfig(tracing.FinalizedStepDynamicSeed(step)).SpanID.String()
}

func ticksOf(spans []seenSpan, id string) []seenSpan {
	out := []seenSpan{}
	for _, span := range spans {
		if span.attrs[meta.Attrs.TickID.Key()].AsString() == id {
			out = append(out, span)
		}
	}

	return out
}

// An arrival is a tick under the row of the op that owns the source it came
// through, one per message, and a redelivery adds no second row.
func TestSubscribeArrivalIsATickOnTheOpenRow(t *testing.T) {
	h := newTraceHarness(t, subscribeOp("inbox", "test/msg"))
	pause := h.sourcePause("inbox", "test/msg")
	require.NotNil(t, pause)

	id := newEventID()
	evt := event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}
	require.NoError(t, h.deliverID(id, evt))

	msg := messageStepID("inbox", id)
	ticks := ticksOf(h.seen(meta.SpanNameTick), msg)
	require.Len(t, ticks, 1)
	require.Equal(t, rowID("inbox"), ticks[0].parent, "the tick is a child of the open row")
	require.Equal(t, []string{pause.ID.String()}, ticks[0].attrs[meta.Attrs.TickSources.Key()].AsStringSlice())
	require.Contains(t, ticks[0].attrs, meta.Attrs.TickAt.Key())

	// A redelivery draws the same tick: the same dynamic span, not a new row.
	require.NoError(t, h.deliverID(id, evt))
	for _, tick := range ticksOf(h.seen(meta.SpanNameTick), msg) {
		require.Equal(t, ticks[0].attrs[meta.Attrs.DynamicSpanID.Key()], tick.attrs[meta.Attrs.DynamicSpanID.Key()])
	}
}

// A read that waits has a row from the moment it waits, and the row ends when
// a message wakes it.
func TestSubscribeWaitHasARowThatEndsOnWake(t *testing.T) {
	h := newTraceHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	var row *seenSpan
	for _, span := range h.seen(meta.SpanNameStep) {
		if span.attrs[meta.Attrs.StepID.Key()].AsString() == "next-1" {
			row = &span
		}
	}
	require.NotNil(t, row, "the waiting read has a row")
	require.Equal(t, enums.StepStatusWaiting.String(), row.attrs[meta.Attrs.DynamicStatus.Key()].AsString())

	require.NoError(t, h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))

	ended := false
	for _, span := range h.seen(meta.SpanNameDynamicExtension) {
		if span.attrs[meta.Attrs.DynamicSpanID.Key()].AsString() == row.attrs[meta.Attrs.DynamicSpanID.Key()].AsString() &&
			span.attrs[meta.Attrs.DynamicStatus.Key()].AsString() == enums.StepStatusCompleted.String() {
			ended = true
		}
	}
	require.True(t, ended, "the message ended the read's row")
}

// A read answered from state is reported as a lazy op: the executor draws its
// row and puts a tick for each message it took under it, stamped with the time
// it was read. Reporting it again draws the same ticks.
func TestSubscribeReadOpDrawsRowAndReadTicks(t *testing.T) {
	h := newTraceHarness(t, subscribeOp("inbox", "test/msg"))

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	msg := messageStepID("inbox", id)

	read := &state.GeneratorOpcode{
		Op:   enums.OpcodeSubscribeRead,
		ID:   "next-1",
		Name: "next-1",
		Opts: map[string]any{"inbox": "inbox", "messages": []string{msg}, "draw": true},
	}
	h.report(read)

	var drawn bool
	for _, span := range h.seen(meta.SpanNameStep) {
		if span.attrs[meta.Attrs.StepName.Key()].AsString() == "next-1" {
			drawn = true
			require.Equal(t, enums.StepStatusCompleted.String(), span.attrs[meta.Attrs.DynamicStatus.Key()].AsString())
		}
	}
	require.True(t, drawn, "the read has a row")

	var readTicks []seenSpan
	for _, tick := range ticksOf(h.seen(meta.SpanNameTick), msg) {
		if _, ok := tick.attrs[meta.Attrs.TickReadAt.Key()]; ok {
			readTicks = append(readTicks, tick)
		}
	}
	require.Len(t, readTicks, 1)
	require.Equal(t, rowID("next-1"), readTicks[0].parent, "the read tick is under the read's row")

	h.report(read)
	for _, tick := range ticksOf(h.seen(meta.SpanNameTick), msg) {
		if _, ok := tick.attrs[meta.Attrs.TickReadAt.Key()]; ok {
			require.Equal(t, readTicks[0].attrs[meta.Attrs.DynamicSpanID.Key()], tick.attrs[meta.Attrs.DynamicSpanID.Key()])
		}
	}
}
