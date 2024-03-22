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
var gen tracesdk.IDGenerator = newSpanIDGenerator()

type SpanOpt func(s *spanOpt)

func WithName(n string) SpanOpt {
	return func(s *spanOpt) {
		s.name = n
	}
}

func WithSpanAttributes(attr ...attribute.KeyValue) SpanOpt {
	return func(s *spanOpt) {
		s.attr = attr
	}
}

func WithScope(scope string) SpanOpt {
	return func(s *spanOpt) {
		s.scope = scope
	}
}

func WithServiceName(s string) SpanOpt {
	return func(s *spanOpt) {
		// TODO: implement
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

func WithParentSpanID(psid trace.SpanID) SpanOpt {
	return func(s *spanOpt) {
		s.psid = &psid
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
	scope      string
	name       string
	root       bool
	links      []tracesdk.Link
	attr       []attribute.KeyValue
	kind       trace.SpanKind
	stacktrace bool
	ts         time.Time
	// Parent SpanID
	psid *trace.SpanID
}

func (so *spanOpt) Attributes() []attribute.KeyValue {
	return so.attr
}

func (so *spanOpt) Links() []tracesdk.Link {
	return so.links
}

func (so *spanOpt) NewRoot() bool {
	return so.root
}

func (so *spanOpt) SpanName() string {
	return so.name
}

func (so *spanOpt) SpanScope() string {
	return so.scope
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

func (so *spanOpt) PreserveSpan() bool {
	return so.psid != nil
}

func (so *spanOpt) ParentSpanID() *trace.SpanID {
	return so.psid
}

// NewSpan creates a new span from the provided context, and overrides the internals with
// additional options provided.
func NewSpan(ctx context.Context, opts ...SpanOpt) (context.Context, *Span) {
	so := newSpanOpt(opts...)

	var psc trace.SpanContext
	if so.NewRoot() {
		ctx = trace.ContextWithSpanContext(ctx, psc)
	} else {
		psc = trace.SpanContextFromContext(ctx)
	}

	// prepare the IDs
	tid := psc.TraceID()
	var sid trace.SpanID
	if !psc.TraceID().IsValid() {
		tid, sid = gen.NewIDs(ctx)
	} else {
		sid = gen.NewSpanID(ctx, tid)
	}
	// TODO: how to get grantparent span to override psc's spanID?
	if so.PreserveSpan() {
		sid = psc.SpanID()
		pid := so.ParentSpanID()
		psc = psc.WithSpanID(*pid)
	}

	sconf := trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceState: psc.TraceState(),
		TraceFlags: psc.TraceFlags(),
	}

	s := &Span{
		parent: psc,
		name:   so.SpanName(),
		start:  so.Timestamp(),
		attrs:  so.Attributes(),
		events: []tracesdk.Event{},
		links:  so.Links(),
		status: tracesdk.Status{Code: codes.Unset},
		conf:   sconf,
		kind:   so.SpanKind(),
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
	sync.Mutex

	start time.Time
	end   time.Time

	name   string
	attrs  []attribute.KeyValue
	status tracesdk.Status
	events []tracesdk.Event
	links  []tracesdk.Link
	kind   trace.SpanKind

	parent trace.SpanContext
	conf   trace.SpanContextConfig

	childSpanCount    int
	droppedAttributes int
}

// Send is just an alias for End
func (s *Span) Send() {
	s.End()
}

//
// trace.ReadOnlySpan interface functions
//

func (s *Span) Name() string {
	return s.name
}

func (s *Span) SpanContext() trace.SpanContext {
	return trace.NewSpanContext(s.conf)
}

func (s *Span) Parent() trace.SpanContext {
	return s.parent
}

func (s *Span) SpanKind() trace.SpanKind {
	return s.kind
}

func (s *Span) StartTime() time.Time {
	return s.start
}

func (s *Span) EndTime() time.Time {
	if s.end.IsZero() {
		return time.Now()
	}
	return s.end
}

func (s *Span) Attributes() []attribute.KeyValue {
	return s.attrs
}

func (s *Span) Links() []tracesdk.Link {
	return s.links
}

func (s *Span) Events() []tracesdk.Event {
	return s.events
}

func (s *Span) Status() tracesdk.Status {
	return s.status
}

func (s *Span) InstrumentationScope() instrumentation.Scope {
	// TODO: implement
	return instrumentation.Scope{}
}

func (s *Span) InstrumentationLibrary() instrumentation.Library {
	// TODO: implement
	return instrumentation.Library{}
}

func (s *Span) Resource() *resource.Resource {
	// TODO: implement
	return nil
}

func (s *Span) DroppedAttributes() int {
	return s.droppedAttributes
}

func (s *Span) DroppedLinks() int {
	// TODO: verify
	return 0
}

func (s *Span) DroppedEvents() int {
	// TODO: verify
	return 0
}

func (s *Span) ChildSpanCount() int {
	return s.childSpanCount
}

// Span interface functions

// End utilizes the internal tracer's processors to send spans
func (s *Span) End(opts ...trace.SpanEndOption) {
	s.end = time.Now()

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

	s.events = append(s.events, evt)
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
	s.status = tracesdk.Status{
		Code:        code,
		Description: desc,
	}
}

func (s *Span) SetName(name string) {
	s.Lock()
	defer s.Unlock()
	s.name = name
}

// SetAttributes mimics the official SetAttributes method, but with
// reduced checks. We're not doing crazy stuff with it so there's
// less of a need to do so.
func (s *Span) SetAttributes(attrs ...attribute.KeyValue) {
	s.Lock()
	defer s.Unlock()

	if s.attrs == nil {
		s.attrs = []attribute.KeyValue{}
	}

	// dedup if the sum of existing and new attr could exceed limit
	if len(s.attrs)+len(attrs) > attrCountLimit {
		// dedup the existing list of attributes and take the latest one
		exists := make(map[attribute.Key]int)
		dedup := []attribute.KeyValue{}
		for _, a := range s.attrs {
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
				s.attrs[idx] = a
				continue
			}

			// don't bother appending if it's at limits
			if len(s.attrs) >= attrCountLimit {
				s.droppedAttributes++
				continue
			}

			s.attrs = append(s.attrs, a)
			exists[a.Key] = len(s.attrs) - 1
		}
	}

	// otherwise, just append
	for _, a := range attrs {
		if !a.Valid() {
			s.droppedAttributes++
			continue
		}
		s.attrs = append(s.attrs, a)
	}
}

func (s *Span) TracerProvider() trace.TracerProvider {
	return UserTracer().Provider()
}

func newSpanIDGenerator() *spanIDGenerator {
	var seed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &seed)

	return &spanIDGenerator{
		randSrc: rand.New(rand.NewSource(seed)),
	}
}

// spanIDGenerator provides a way to generate TraceID and SpanID
type spanIDGenerator struct {
	sync.Mutex
	randSrc *rand.Rand
}

func (sc *spanIDGenerator) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	sc.Lock()
	defer sc.Unlock()

	sid := trace.SpanID{}
	_, _ = sc.randSrc.Read(sid[:])
	return sid
}

func (sc *spanIDGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
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
