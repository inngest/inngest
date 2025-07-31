package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

type SpanReference struct {
	TraceParent string `json:"tp"`
	TraceState  string `json:"ts"`

	// If this is a dynamic span, store enough information to be able to safely
	// extend the span with only this context.
	DynamicSpanTraceParent string `json:"dstp,omitempty"`
	DynamicSpanTraceState  string `json:"dsts,omitempty"`
	DynamicSpanID          string `json:"dsid,omitempty"`
}

type ctxKey struct{}

func (sr *SpanReference) Validate() error {
	if sr.TraceParent == "" {
		return fmt.Errorf("span reference missing traceparent; every span must have a traceparent")
	}

	return nil
}

func (sr *SpanReference) TraceID() (string, error) {
	// Return the trace ID from the traceparent
	if sr.TraceParent == "" {
		return "", fmt.Errorf("span reference missing traceparent; cannot get trace ID")
	}

	parts := strings.Split(sr.TraceParent, "-")
	if len(parts) != 4 || len(parts[1]) != 32 {
		return "", fmt.Errorf("invalid traceparent format")
	}

	return parts[1], nil
}

func (sr *SpanReference) SpanID() (string, error) {
	// Return the span ID from the traceparent
	if sr.TraceParent == "" {
		return "", fmt.Errorf("span reference missing traceparent; cannot get span ID")
	}

	parts := strings.Split(sr.TraceParent, "-")
	if len(parts) != 4 || len(parts[2]) != 12 {
		return "", fmt.Errorf("invalid traceparent format")
	}

	return parts[2], nil
}

// SetParentSpanID alters the TraceParent to include the parent span ID.
func (sr *SpanReference) SetParentSpanID(parentSpanID trace.SpanID) (*SpanReference, error) {
	err := sr.Validate()
	if err != nil {
		return nil, fmt.Errorf("span reference was not valid, so cannot set parent span ID: %w", err)
	}

	parts := strings.Split(sr.TraceParent, "-")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid traceparent format; expected 4 parts, got %d", len(parts))
	}

	parts[2] = parentSpanID.String()

	// If we're updating this, it's for userland spans, so we intentionally
	// remove all dynamic span information.
	return &SpanReference{
		TraceParent: strings.Join(parts, "-"),
		TraceState:  sr.TraceState,
	}, nil

}

func (sr *SpanReference) SetToCtx(ctx context.Context) context.Context {
	if sr == nil {
		return ctx
	}

	return context.WithValue(ctx, ctxKey{}, sr)
}

func (sr *SpanReference) QueryString() (string, error) {
	if sr == nil {
		return "", fmt.Errorf("span reference is nil")
	}

	byt, err := json.Marshal(sr)
	if err != nil {
		return "", fmt.Errorf("failed to marshal span reference: %w", err)
	}

	escaped := url.QueryEscape(string(byt))

	return escaped, nil
}

func GetSpanReferenceFromCtx(ctx context.Context) (*SpanReference, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}

	val := ctx.Value(ctxKey{})
	if val == nil {
		return nil, fmt.Errorf("no span reference found in context")
	}

	sr, ok := val.(*SpanReference)
	if !ok {
		return nil, fmt.Errorf("span reference in context is not of type *SpanReference")
	}

	if sr == nil {
		return nil, fmt.Errorf("span reference in context is nil")
	}

	return sr, sr.Validate()
}
