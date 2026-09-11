package cqrs

import (
	"context"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

func TestApplyExtractedSpanAttributesPromotesStatusAndTiming(t *testing.T) {
	span := &OtelSpan{
		RawOtelSpan: RawOtelSpan{
			Attributes: map[string]any{
				"_inngest.dynamic.status": "Completed",
			},
		},
	}
	require.NoError(t, ApplyExtractedSpanAttributes(context.Background(), span))
	require.Equal(t, enums.StepStatusCompleted, span.Status)
}

func TestApplyExtractedSpanAttributesMarksDroppedSpans(t *testing.T) {
	span := &OtelSpan{
		RawOtelSpan: RawOtelSpan{
			Attributes: map[string]any{"_inngest.executor.drop": true},
		},
	}
	require.NoError(t, ApplyExtractedSpanAttributes(context.Background(), span))
	require.True(t, span.MarkedAsDropped)
}

func TestUnwrapSpanOutputEnvelopeSplitsErrorFromData(t *testing.T) {
	so := &SpanOutput{}
	done := UnwrapSpanOutputEnvelope(so, []byte(`{"data":"ok"}`), nil)
	require.False(t, done)
	require.False(t, so.IsError)
	require.Equal(t, `"ok"`, string(so.Data))

	so = &SpanOutput{}
	UnwrapSpanOutputEnvelope(so, []byte(`{"error":"boom"}`), nil)
	require.True(t, so.IsError)
	require.Equal(t, `"boom"`, string(so.Data))
}

func TestUnwrapSpanOutputEnvelopeStopsAtWaitForEventShape(t *testing.T) {
	so := &SpanOutput{}
	done := UnwrapSpanOutputEnvelope(so, []byte(`{"name":"n","data":{},"ts":1}`), nil)
	require.True(t, done)
}

func TestUnwrapSpanOutputEnvelopeSetsInput(t *testing.T) {
	so := &SpanOutput{}
	UnwrapSpanOutputEnvelope(so, nil, []byte(`{"a":1}`))
	require.Equal(t, `{"a":1}`, string(so.Input))
}
