package telemetry

import (
	"context"
	"time"
)

type SpanOpt func(s *span)

// NewSpan creates a new span from the provided context, and overrides the internals with
// additional options provided.
func NewSpan(ctx context.Context, opts ...SpanOpt) (context.Context, *span) {
	s := &span{
		StartTime: time.Now(),
		Attrs:     map[string]string{},
		Events:    []spanEvent{},
		Links:     []spanLink{},
	}

	for _, opt := range opts {
		opt(s)
	}

	return ctx, s
}

// span is an attempt to mimic the otel span data structure following the protobuf spec at
// https://github.com/open-telemetry/opentelemetry-proto/blob/v1.1.0/opentelemetry/proto/trace/v1/trace.proto
//
// Due to the limitations of the otel lib's API interface, we can't reconstruct spans over boundaries,
// and in order to make sure the execution data looks like how it looks from the SDK side,
// we'll need to work around the otel library and have slightly different way of working with the data
//
// This file is an attempt to make it as close as possible to official libs so we can minimize deviations.
//
// NOTE: to make sure it doesn't conflict the the ReadOnlySpan interface functions,
// certain fields are named in a little weird way.
type span struct {
	TraceID      string    `json:"traceID"`
	SpanID       string    `json:"spanID"`
	TraceState   string    `json:"traceState"`
	ParentSpanID *string   `json:"parentSpanID,omitempty"`
	Flags        [4]byte   `json:"flags"`
	SpanName     string    `json:"name"`
	Kind         string    `json:"kind"`
	StartTime    time.Time `json:"startts"`
	EndTime      time.Time `json:"endts"`

	ServiceName  string `json:"serviceName"`
	ScopeName    string `json:"scopeName"`
	ScopeVersion string `json:"scopeVersion"`

	Attrs map[string]string `json:"attrs"`

	Events []spanEvent `json:"events"`
	Links  []spanLink  `json:"links"`
}

type spanEvent struct {
	Timestamp time.Time         `json:"ts"`
	Name      string            `json:"name"`
	Attr      map[string]string `json:"attr"`
}

type spanLink struct {
	TraceID    string            `json:"traceID"`
	SpanID     string            `json:"spanID"`
	TraceState string            `json:"traceState"`
	Attr       map[string]string `json:"attr"`
	Flags      [4]byte           `json:"flags"`
}

// Implement the functions to fulfill trace.ReadOnlySpan
func (s *span) Name() string {
	return s.SpanName
}

func (s *span) End() {
	UserTracer().Export(s)
}
