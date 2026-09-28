package trace

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestPropagatorRoundTrip(t *testing.T) {
	traceID, err := oteltrace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := oteltrace.SpanIDFromHex("0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	traceState, err := oteltrace.ParseTraceState("vendor=value")
	if err != nil {
		t.Fatal(err)
	}

	source := oteltrace.ContextWithRemoteSpanContext(context.Background(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		TraceState: traceState,
		Remote:     true,
	}))
	member, err := baggage.NewMember("customer", "acme")
	if err != nil {
		t.Fatal(err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatal(err)
	}
	source = baggage.ContextWithBaggage(source, bag)

	tests := map[string]func() propagation.TextMapCarrier{
		"HTTP headers": func() propagation.TextMapCarrier {
			return propagation.HeaderCarrier(http.Header{})
		},
		"event, pubsub, and queue metadata": func() propagation.TextMapCarrier {
			return propagation.MapCarrier{}
		},
	}

	for name, newCarrier := range tests {
		t.Run(name, func(t *testing.T) {
			carrier := newCarrier()
			Propagator().Inject(source, carrier)

			if got := carrier.Get("traceparent"); got != "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01" {
				t.Fatalf("unexpected traceparent: %q", got)
			}
			if got := carrier.Get("tracestate"); got != "vendor=value" {
				t.Fatalf("unexpected tracestate: %q", got)
			}

			extracted := Propagator().Extract(context.Background(), carrier)
			if got := oteltrace.SpanContextFromContext(extracted); !got.Equal(oteltrace.SpanContextFromContext(source)) {
				t.Fatalf("span context did not round trip: got %v", got)
			}
			if got := baggage.FromContext(extracted).Member("customer").Value(); got != "acme" {
				t.Fatalf("baggage did not round trip: got %q", got)
			}
		})
	}
}

func TestHeadersFromTraceStatePreservesSDKCorrelationAndBaggage(t *testing.T) {
	traceID, err := oteltrace.TraceIDFromHex("fedcba9876543210fedcba9876543210")
	if err != nil {
		t.Fatal(err)
	}
	parentSpanID, err := oteltrace.SpanIDFromHex("1111111111111111")
	if err != nil {
		t.Fatal(err)
	}
	sdkSpanID := "2222222222222222"

	ctx := oteltrace.ContextWithSpanContext(context.Background(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     parentSpanID,
		TraceFlags: oteltrace.FlagsSampled,
	}))
	member, err := baggage.NewMember("session", "session-123")
	if err != nil {
		t.Fatal(err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatal(err)
	}
	ctx = baggage.ContextWithBaggage(ctx, bag)

	headers, err := HeadersFromTraceState(ctx, sdkSpanID, "app-123", "fn-456")
	if err != nil {
		t.Fatal(err)
	}
	if got := headers["traceparent"]; got != "00-fedcba9876543210fedcba9876543210-2222222222222222-01" {
		t.Fatalf("unexpected SDK traceparent: %q", got)
	}

	extracted := Propagator().Extract(context.Background(), propagation.MapCarrier(headers))
	spanContext := oteltrace.SpanContextFromContext(extracted)
	if got := spanContext.TraceState().Get("inngest@app"); got != "app-123" {
		t.Fatalf("unexpected app tracestate: %q", got)
	}
	if got := spanContext.TraceState().Get("inngest@fn"); got != "fn-456" {
		t.Fatalf("unexpected function tracestate: %q", got)
	}
	if got := baggage.FromContext(extracted).Member("session").Value(); got != "session-123" {
		t.Fatalf("SDK baggage did not round trip: %q", got)
	}
}
