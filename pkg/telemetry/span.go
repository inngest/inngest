package telemetry

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
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

// type assertion
var _ tracesdk.IDGenerator = &spanCtx{}

type SpanOpt func(s *spanOpt)

func WithSpanAttributes(attr ...attribute.KeyValue) SpanOpt {
	return func(s *spanOpt) {
		s.attr = attr
	}
}

func WithNewRoot() SpanOpt {
	return func(s *spanOpt) {
		s.root = true
	}
}

func WithSpanKind(k trace.SpanKind) SpanOpt {
	return func(s *spanOpt) {
		s.kind = k
	}
}

func WithLinks(l []tracesdk.Link) SpanOpt {
	return func(s *spanOpt) {
		s.links = l
	}
}

func WithTimestamp(ts time.Time) SpanOpt {
	return func(s *spanOpt) {
		s.ts = ts
	}
}

func WithSpanID(sid *trace.SpanID) SpanOpt {
	return func(s *spanOpt) {
		s.spanID = sid
	}
}

func newSpanOpt(opts ...SpanOpt) *spanOpt {
	s := &spanOpt{
		kind:  trace.SpanKindUnspecified,
		ts:    time.Now(),
		links: []tracesdk.Link{},
		attr:  []attribute.KeyValue{},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type spanOpt struct {
	root       bool
	links      []tracesdk.Link
	attr       []attribute.KeyValue
	kind       trace.SpanKind
	stacktrace bool
	ts         time.Time
	spanID     *trace.SpanID
}

func (so *spanOpt) Attributes() []attribute.KeyValue {
	return so.attr
}

func (so *spanOpt) NewRoot() bool {
	return so.root
}

func (so *spanOpt) SpanID() *trace.SpanID {
	return so.spanID
}

func (so *spanOpt) SpanKind() trace.SpanKind {
	return so.kind
}

func (so *spanOpt) StackTrace() bool {
	return so.stacktrace
}

func (so *spanOpt) Timestamp() time.Time {
	return so.ts
}

// NewSpan creates a new span from the provided context, and overrides the internals with
// additional options provided.
func NewSpan(ctx context.Context, opts ...SpanOpt) (context.Context, *Span) {
	// conf := newSpanOpt(opts...)

	// var psc trace.SpanContext

	// Steps
	// - [ ] extract span context from passed in context
	// - [ ] check if it's valid
	// - [ ] if so
	// 	 + [ ] check if this should be a root span
	// 	 + [ ] if so
	// 	 	 > [ ] do not store current spanID as parent
	// 	 + [ ] extract traceID and set it in config
	// 	 + [ ] extract spanID and set it as parent
	// 	 + [ ] generate new spanID and set it to config
	// - [ ] if not
	//   + [ ] create a new span context config
	//   + [ ] generate a new traceID and spanID

	// TODO: construct a trace correctly from passed in context
	// sctx := trace.SpanContextFromContext(ctx)

	s := &Span{
		StartedAt:  time.Now(),
		Attrs:      []attribute.KeyValue{},
		SpanEvents: []tracesdk.Event{},
		SpanLinks:  []tracesdk.Link{},
		SpanStatus: tracesdk.Status{Code: codes.Unset},
		Kind:       trace.SpanKindUnspecified,
	}

	// for _, opt := range opts {
	// 	opt(s)
	// }

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
	sync.Mutex

	SpanName     string          `json:"name"`
	ParentSpanID *trace.SpanID   `json:"parentSpanID,omitempty"`
	Kind         trace.SpanKind  `json:"kind"`
	StartedAt    time.Time       `json:"startts"`
	EndedAt      time.Time       `json:"endts"`
	SpanStatus   tracesdk.Status `json:"status"`

	ServiceName  string `json:"serviceName"`
	ScopeName    string `json:"scopeName"`
	ScopeVersion string `json:"scopeVersion"`

	Attrs []attribute.KeyValue `json:"attrs"`

	SpanConf   trace.SpanContextConfig `json:"conf"`
	SpanEvents []tracesdk.Event        `json:"events"`
	SpanLinks  []tracesdk.Link         `json:"links"`

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
	return trace.NewSpanContext(s.SpanConf)
}

func (s *Span) Parent() trace.SpanContext {
	conf := trace.SpanContextConfig{
		TraceID:    s.SpanConf.TraceID,
		SpanID:     trace.SpanID{},
		TraceFlags: s.SpanConf.TraceFlags,
		TraceState: s.SpanConf.TraceState,
		Remote:     s.SpanConf.Remote,
	}
	if s.ParentSpanID != nil {
		conf.SpanID = *s.ParentSpanID
	}

	return trace.NewSpanContext(conf)
}

func (s *Span) SpanKind() trace.SpanKind {
	return s.Kind
}

func (s *Span) StartTime() time.Time {
	return s.StartedAt
}

func (s *Span) EndTime() time.Time {
	if s.EndedAt.IsZero() {
		return time.Now()
	}
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
	return s.SpanStatus
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
	s.EndedAt = time.Now()

	if err := UserTracer().Export(s); err != nil {
		ctx := context.Background()
		log.From(ctx).Error().Err(err).Msg("error ending span")
	}
}

func (s *Span) AddEvent(name string, opts ...trace.EventOption) {
	s.Lock()
	defer s.Unlock()

	config := trace.NewEventConfig(opts...)

	evt := tracesdk.Event{
		Name:       name,
		Time:       time.Now(),
		Attributes: config.Attributes(),
	}
	if !config.Timestamp().IsZero() {
		evt.Time = config.Timestamp()
	}

	s.SpanEvents = append(s.SpanEvents, evt)
}

func (s *Span) IsRecording() bool {
	return true
}

// official one doesn't actually set the status, but we'll just do it here
// for convinence's sake.
func (s *Span) RecordError(err error, opts ...trace.EventOption) {
	s.Lock()
	defer s.Unlock()

	s.AddEvent(err.Error(), opts...)
	s.SetStatus(codes.Error, err.Error())
}

func (s *Span) SetStatus(code codes.Code, desc string) {
	s.Lock()
	defer s.Unlock()
	s.SpanStatus = tracesdk.Status{
		Code:        code,
		Description: desc,
	}
}

func (s *Span) SetName(name string) {
	s.Lock()
	defer s.Unlock()
	s.SpanName = name
}

// SetAttributes mimics the official SetAttributes method, but with
// reduced checks. We're not doing crazy stuff with it so there's
// less of a need to do so.
func (s *Span) SetAttributes(attrs ...attribute.KeyValue) {
	s.Lock()
	defer s.Unlock()

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

func NewSpanCtx() *spanCtx {
	var seed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &seed)

	return &spanCtx{
		randSrc: rand.New(rand.NewSource(seed)),
	}
}

// spanCtx provides a way to generate TraceID and SpanID,
// and also a new SpanContext from those values
type spanCtx struct {
	tID *trace.TraceID
	sID *trace.SpanID

	sync.Mutex
	randSrc *rand.Rand
}

func (sc *spanCtx) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	sc.Lock()
	defer sc.Unlock()

	sid := trace.SpanID{}
	_, _ = sc.randSrc.Read(sid[:])
	return sid
}

func (sc *spanCtx) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	sc.Lock()
	defer sc.Unlock()
	var (
		tid trace.TraceID
		sid trace.SpanID
	)

	_, _ = sc.randSrc.Read(tid[:])
	_, _ = sc.randSrc.Read(sid[:])
	return tid, sid
}

func (sc *spanCtx) Context() trace.SpanContext {
	return trace.SpanContext{}
}
