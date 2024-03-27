package telemetry

import (
	"context"
	"time"

	"github.com/inngest/inngest/pkg/inngest/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type SpanOpt func(s *span)

// NewSpan creates a new span from the provided context, and overrides the internals with
// additional options provided.
func NewSpan(ctx context.Context, opts ...SpanOpt) (context.Context, *span) {
	s := &span{
		StartedAt:  time.Now(),
		Attrs:      map[string]string{},
		SpanEvents: []tracesdk.Event{},
		SpanLinks:  []tracesdk.Link{},
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
	tracesdk.ReadOnlySpan

	TraceID      string         `json:"traceID"`
	SpanID       string         `json:"spanID"`
	TraceState   string         `json:"traceState"`
	ParentSpanID *string        `json:"parentSpanID,omitempty"`
	Flags        [4]byte        `json:"flags"`
	SpanName     string         `json:"name"`
	Kind         trace.SpanKind `json:"kind"`
	StartedAt    time.Time      `json:"startts"`
	EndedAt      time.Time      `json:"endts"`

	ServiceName  string `json:"serviceName"`
	ScopeName    string `json:"scopeName"`
	ScopeVersion string `json:"scopeVersion"`

	Attrs map[string]string `json:"attrs"`

	SpanEvents []tracesdk.Event `json:"events"`
	SpanLinks  []tracesdk.Link  `json:"links"`
}

// Implement the functions to fulfill trace.ReadOnlySpan
func (s *span) Name() string {
	return s.SpanName
}

func (s *span) SpanContext() trace.SpanContext {
	return trace.SpanContext{}
}

func (s *span) Parent() trace.SpanContext {
	return trace.SpanContext{}
}

func (s *span) SpanKind() trace.SpanKind {
	return s.Kind
}

func (s *span) StartTime() time.Time {
	return s.StartedAt
}

func (s *span) EndTime() time.Time {
	return s.EndedAt
}

func (s *span) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{}
}

func (s *span) Links() []tracesdk.Link {
	return s.SpanLinks
}

func (s *span) Events() []tracesdk.Event {
	return s.SpanEvents
}

func (s *span) Status() tracesdk.Status {
	return tracesdk.Status{
		Code: codes.Unset,
	}
}

func (s *span) InstrumentationScope() instrumentation.Scope {
	return instrumentation.Scope{}
}

func (s *span) InstrumentationLibrary() instrumentation.Library {
	return instrumentation.Library{}
}

func (s *span) Resource() *resource.Resource {
	return nil
}

func (s *span) DroppedAttributes() int {
	return 0
}

func (s *span) DroppedLinks() int {
	return 0
}

func (s *span) DroppedEvents() int {
	return 0
}

func (s *span) ChildSpanCount() int {
	return 0
}

// End utilizes the internal tracer's processors to send spans
func (s *span) End() {
	if err := UserTracer().Export(s); err != nil {
		ctx := context.Background()
		log.From(ctx).Error().Err(err).Msg("error ending span")
	}
}
