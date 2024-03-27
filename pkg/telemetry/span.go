package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/inngest/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ref: https://opentelemetry.io/docs/specs/otel/common/#configurable-parameters
	attrCountLimit = 128
)

type SpanOpt func(s *span)

func WithSpanAttributes(attr ...attribute.KeyValue) SpanOpt {
	return func(s *span) {
		s.SetAttributes(attr...)
	}
}

// NewSpan creates a new span from the provided context, and overrides the internals with
// additional options provided.
func NewSpan(ctx context.Context, opts ...SpanOpt) (context.Context, *span) {
	if ctx == nil {
		ctx = context.Background()
	}

	// TODO: construct a trace correctly from passed in context
	spanCtx := trace.SpanContextFromContext(ctx)

	s := &span{
		TraceID:    spanCtx.TraceID(),
		StartedAt:  time.Now(),
		Attrs:      []attribute.KeyValue{},
		SpanEvents: []tracesdk.Event{},
		SpanLinks:  []tracesdk.Link{},
		mu:         sync.Mutex{},
	}

	for _, opt := range opts {
		opt(s)
	}

	return trace.ContextWithSpan(ctx, s), s
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
	tracesdk.ReadWriteSpan // embeds both span interfaces

	TraceID      trace.TraceID  `json:"traceID"`
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

	Attrs []attribute.KeyValue `json:"attrs"`

	SpanEvents []tracesdk.Event `json:"events"`
	SpanLinks  []tracesdk.Link  `json:"links"`

	mu                sync.Mutex
	childSpanCount    int
	droppedAttributes int
}

//
// trace.ReadOnlySpan interface functions
//

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
	return s.droppedAttributes
}

func (s *span) DroppedLinks() int {
	return 0
}

func (s *span) DroppedEvents() int {
	return 0
}

func (s *span) ChildSpanCount() int {
	return s.childSpanCount
}

// Span interface functions

// End utilizes the internal tracer's processors to send spans
func (s *span) End(opts ...trace.SpanEndOption) {
	if err := UserTracer().Export(s); err != nil {
		ctx := context.Background()
		log.From(ctx).Error().Err(err).Msg("error ending span")
	}
}

func (s *span) AddEvent(name string, opts ...trace.EventOption) {}

func (s *span) IsRecording() bool {
	return true
}

func (s *span) RecordError(err error, opts ...trace.EventOption) {}

func (s *span) SetStatus(code codes.Code, desc string) {}

func (s *span) SetName(name string) {}

// SetAttributes mimics the official SetAttributes method, but with
// reduced checks. We're not doing crazy stuff with it so there's
// less of a need to do so.
func (s *span) SetAttributes(attrs ...attribute.KeyValue) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Attrs == nil {
		s.Attrs = []attribute.KeyValue{}
	}

	// dedup if the sum of existing and new attr could exceed limit
	if len(s.Attrs)+len(attrs) > attrCountLimit {
		// dedup the existing list of attributes and take the latest one
		exists := make(map[attribute.Key]int)
		dedup := []attribute.KeyValue{}
		for _, a := range s.Attrs {
			if idx, ok := exists[a.Key]; ok {
				dedup[idx] = a
			} else {
				dedup = append(dedup, a)
				exists[a.Key] = len(dedup) - 1
			}
		}

		for _, a := range attrs {
			if !a.Valid() {
				// Drop invalid attributes
				s.droppedAttributes++
				continue
			}

			// if a key is already there, take the latest one
			if idx, ok := exists[a.Key]; ok {
				s.Attrs[idx] = a
				continue
			}

			// don't bother appending if it's at limits
			if len(s.Attrs) >= attrCountLimit {
				s.droppedAttributes++
				continue
			}

			s.Attrs = append(s.Attrs, a)
			exists[a.Key] = len(s.Attrs) - 1
		}
	}

	// otherwise, just append
	for _, a := range attrs {
		if !a.Valid() {
			s.droppedAttributes++
			continue
		}
		s.Attrs = append(s.Attrs, a)
	}
}

func (s *span) TracerProvider() trace.TracerProvider {
	return UserTracer().Provider()
}
