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

type SpanOpt func(s *Span)

func WithSpanAttributes(attr ...attribute.KeyValue) SpanOpt {
	return func(s *Span) {
		s.SetAttributes(attr...)
	}
}

// NewSpan creates a new span from the provided context, and overrides the internals with
// additional options provided.
func NewSpan(ctx context.Context, opts ...SpanOpt) (context.Context, *Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	// TODO: construct a trace correctly from passed in context
	spanCtx := trace.SpanContextFromContext(ctx)

	s := &Span{
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
type Span struct {
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

func (s *Span) Name() string {
	return s.SpanName
}

func (s *Span) SpanContext() trace.SpanContext {
	return trace.SpanContext{}
}

func (s *Span) Parent() trace.SpanContext {
	return trace.SpanContext{}
}

func (s *Span) SpanKind() trace.SpanKind {
	return s.Kind
}

func (s *Span) StartTime() time.Time {
	return s.StartedAt
}

func (s *Span) EndTime() time.Time {
	return s.EndedAt
}

func (s *Span) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{}
}

func (s *Span) Links() []tracesdk.Link {
	return s.SpanLinks
}

func (s *Span) Events() []tracesdk.Event {
	return s.SpanEvents
}

func (s *Span) Status() tracesdk.Status {
	return tracesdk.Status{
		Code: codes.Unset,
	}
}

func (s *Span) InstrumentationScope() instrumentation.Scope {
	return instrumentation.Scope{}
}

func (s *Span) InstrumentationLibrary() instrumentation.Library {
	return instrumentation.Library{}
}

func (s *Span) Resource() *resource.Resource {
	return nil
}

func (s *Span) DroppedAttributes() int {
	return s.droppedAttributes
}

func (s *Span) DroppedLinks() int {
	return 0
}

func (s *Span) DroppedEvents() int {
	return 0
}

func (s *Span) ChildSpanCount() int {
	return s.childSpanCount
}

// Span interface functions

// End utilizes the internal tracer's processors to send spans
func (s *Span) End(opts ...trace.SpanEndOption) {
	if err := UserTracer().Export(s); err != nil {
		ctx := context.Background()
		log.From(ctx).Error().Err(err).Msg("error ending span")
	}
}

func (s *Span) AddEvent(name string, opts ...trace.EventOption) {}

func (s *Span) IsRecording() bool {
	return true
}

func (s *Span) RecordError(err error, opts ...trace.EventOption) {}

func (s *Span) SetStatus(code codes.Code, desc string) {}

func (s *Span) SetName(name string) {}

// SetAttributes mimics the official SetAttributes method, but with
// reduced checks. We're not doing crazy stuff with it so there's
// less of a need to do so.
func (s *Span) SetAttributes(attrs ...attribute.KeyValue) {
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

func (s *Span) TracerProvider() trace.TracerProvider {
	return UserTracer().Provider()
}
