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

func TestSplitSpanOutputEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name         string
		output       string
		data, errVal *string
		waitForEvent bool
	}{
		{name: "empty"},
		{name: "data", output: `{"data":{"b":1,"a":2}}`, data: new(`{"b":1,"a":2}`)},
		{name: "error", output: `{"error":{"name":"Error","message":"boom"}}`, errVal: new(`{"name":"Error","message":"boom"}`)},
		{name: "error and data", output: `{"data":1,"error":"boom"}`, data: new(`1`), errVal: new(`"boom"`)},
		{name: "null error", output: `{"error":null}`, errVal: new(`null`)},
		{name: "null data", output: `{"data":null}`, data: new(`null`)},
		{name: "wait for event", output: `{"name":"n","data":{"x":1},"ts":1}`, data: new(`{"name":"n","data":{"x":1},"ts":1}`), waitForEvent: true},
		{name: "unrecognized object", output: `{"ok":true}`, data: new(`{"ok":true}`)},
		{name: "non-object", output: `[1,2]`, data: new(`[1,2]`)},
		{name: "string", output: `"hi"`, data: new(`"hi"`)},
		{name: "null", output: `null`, data: new(`null`)},
		{name: "invalid JSON", output: `not json`, data: new(`not json`)},
		{name: "number text", output: `{"data":{"z":1.10,"big":12345678901234567890,"a":1e3}}`, data: new(`{"z":1.10,"big":12345678901234567890,"a":1e3}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, errVal, waitForEvent := SplitSpanOutputEnvelope([]byte(tc.output))
			require.Equal(t, tc.waitForEvent, waitForEvent)
			if tc.data == nil {
				require.Nil(t, data)
			} else {
				require.Equal(t, *tc.data, string(data))
			}
			if tc.errVal == nil {
				require.Nil(t, errVal)
			} else {
				require.Equal(t, *tc.errVal, string(errVal))
			}

			// Reading the split columns agrees w/ unwrapping the envelope.
			unwrapped := &SpanOutput{}
			UnwrapSpanOutputEnvelope(unwrapped, []byte(tc.output), nil)
			applied := &SpanOutput{}
			ApplySpanOutputColumns(applied, data, errVal, nil)
			require.Equal(t, unwrapped, applied)
		})
	}
}

func TestSpanOutputEnvelope(t *testing.T) {
	require.Nil(t, SpanOutputEnvelope(nil, nil))
	require.Equal(t, `{"data":{"b":1.10,"a":2}}`, string(SpanOutputEnvelope([]byte(`{"b":1.10,"a":2}`), nil)))
	require.Equal(t, `{"error":"boom"}`, string(SpanOutputEnvelope([]byte(`1`), []byte(`"boom"`))))
	require.Equal(t, `{"data":null}`, string(SpanOutputEnvelope([]byte(`null`), nil)))
}
