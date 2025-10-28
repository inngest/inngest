package tracing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"lukechampine.com/frand"
)

var (
	defaultPropagator = propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	idGen = idGenerator{}
)

// TracerProvider defines the interface for tracing providers.
type TracerProvider interface {
	CreateSpan(ctx context.Context, name string, opts *CreateSpanOptions) (*meta.SpanReference, error)
	CreateDroppableSpan(ctx context.Context, name string, opts *CreateSpanOptions) (*DroppableSpan, error)
	UpdateSpan(ctx context.Context, opts *UpdateSpanOptions) error
}

type SpanDebugData struct {
	Location string
}

type DroppableSpan struct {
	span trace.Span
	Ref  *meta.SpanReference
}

type CreateSpanOptions struct {
	Attributes         *meta.SerializableAttrs
	Carriers           []map[string]any
	Debug              *SpanDebugData
	FollowsFrom        *meta.SpanReference
	Metadata           *statev2.Metadata
	Parent             *meta.SpanReference
	QueueItem          *queue.Item
	RawOtelSpanOptions []trace.SpanStartOption
	StartTime          time.Time
	EndTime            time.Time

	Seed []byte
}

type UpdateSpanOptions struct {
	Attributes         *meta.SerializableAttrs
	Debug              *SpanDebugData
	EndTime            time.Time
	Metadata           *statev2.Metadata
	QueueItem          *queue.Item
	RawOtelSpanOptions []trace.SpanStartOption
	Status             enums.StepStatus
	TargetSpan         *meta.SpanReference
}

// otelTracerProvider implements TracerProvider.
type otelTracerProvider struct {
	exp sdktrace.SpanExporter
	bt  time.Duration
}

func NewOtelTracerProvider(exp sdktrace.SpanExporter, batchTimeout time.Duration) TracerProvider {
	return &otelTracerProvider{
		exp: exp,
		bt:  batchTimeout,
	}
}

func (tp *otelTracerProvider) getTracer(md *statev2.Metadata) trace.Tracer {
	base := sdktrace.NewSimpleSpanProcessor(tp.exp)

	otelTP := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(newExecutionProcessor(md, base)),
		sdktrace.WithIDGenerator(idGen),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	return otelTP.Tracer(
		"inngest",
		trace.WithInstrumentationVersion(version.Print()),
	)
}

func (d *DroppableSpan) Drop() {
	d.span.SetAttributes(attribute.Bool(meta.Attrs.DropSpan.Key(), true))
	// Send span but we don't care if it makes it or not, as we're dropping
	// anyway
	d.span.End()
}

func (d *DroppableSpan) Send() error {
	d.span.End()
	return nil
}

func (tp *otelTracerProvider) CreateSpan(
	ctx context.Context,
	name string,
	opts *CreateSpanOptions,
) (*meta.SpanReference, error) {
	ds, err := tp.CreateDroppableSpan(ctx, name, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to C{reateSpan: %w", err)
	}

	err = ds.Send()
	if err != nil {
		return nil, fmt.Errorf("failed to send span during creation: %w", err)
	}

	return ds.Ref, nil
}

// CreateDroppableSpan creates a span that can be dropped and relies on us
// calling `.End()`.
func (tp *otelTracerProvider) CreateDroppableSpan(
	ctx context.Context,
	name string,
	opts *CreateSpanOptions,
) (*DroppableSpan, error) {
	attrs := opts.Attributes
	if attrs == nil {
		attrs = meta.NewAttrSet()
	}

	st := opts.StartTime
	if st.IsZero() {
		st = time.Now()
	} else {
		meta.AddAttr(attrs, meta.Attrs.StartedAt, &st)
	}

	if opts.Parent != nil {
		carrier := propagation.MapCarrier{
			"traceparent": opts.Parent.TraceParent,
			"tracestate":  opts.Parent.TraceState,
		}
		ctx = mixinExecutonContext(
			ctx,
			// extract the propagator from a blank contexct, and mixin the execution
			// context from the parent.  this creates a blank ctx with just the executor context
			// and propagator, which is necessary to tie parents <> children.
			defaultPropagator.Extract(context.Background(), carrier),
		)
	} else {
		// Use a fresh context for parent traces so that there's no pollution from any
		// other tracing.
		ctx = context.Background()
	}

	if opts.Debug != nil {
		if opts.Debug.Location != "" {
			meta.AddAttr(attrs, meta.Attrs.InternalLocation, &opts.Debug.Location)
		}
	}
	if !opts.StartTime.IsZero() {
		meta.AddAttr(attrs, meta.Attrs.StartedAt, &opts.StartTime)
	}
	if !opts.EndTime.IsZero() {
		meta.AddAttr(attrs, meta.Attrs.EndedAt, &opts.EndTime)
	}

	spanOptions := append(
		[]trace.SpanStartOption{
			trace.WithAttributes(attrs.Serialize()...),
			trace.WithTimestamp(st),
		},
		opts.RawOtelSpanOptions...,
	)

	spanOptions = append(spanOptions, trace.WithSpanKind(trace.SpanKindServer))

	if opts.FollowsFrom != nil {
		spanOptions = append(
			spanOptions,
			trace.WithLinks(trace.Link{
				SpanContext: spanContextFromMetadata(opts.FollowsFrom),
				Attributes: []attribute.KeyValue{
					attribute.String(meta.LinkAttributeType, meta.LinkAttributeTypeFollowsFrom),
				},
			}),
		)
	}

	// IF THERE IS SEED, we're creating something with deterministic span and trace IDs.
	// YAY.  We love determinism.  This is important for eg. root spans.
	if len(opts.Seed) > 0 {
		ctx = setDeteterministicIDs(ctx, DeterministicSpanConfig(opts.Seed))
	}

	tracer := tp.getTracer(opts.Metadata)
	ctx, span := tracer.Start(ctx, name, spanOptions...)

	carrier := propagation.MapCarrier{}
	defaultPropagator.Inject(ctx, carrier)
	refTp := carrier["traceparent"]
	refTs := carrier["tracestate"]

	spanRef := &meta.SpanReference{
		TraceParent: refTp,
		TraceState:  refTs,
	}

	spanRef.DynamicSpanID = span.SpanContext().SpanID().String()

	if opts.Parent != nil {
		// If the span has a parent, set some attributes so we can extend it later
		// and pick the same trace and parent span IDs for the extension span.
		spanRef.DynamicSpanTraceParent = opts.Parent.TraceParent
		spanRef.DynamicSpanTraceState = opts.Parent.TraceState
	} else {
		// If we don't have a parent, this is a top-level span (e.g. the run
		// span), so we use this span as the dynamic reference instead.
		//
		// In this case, we forcibly set the span ID part of the traceparent
		// to the expected zero value, to be the same as the top-level span.
		// e.g. for "00-c0b6b7b1d103cd383d594e9ffa128965-930c339a6dbccb41-01",
		// produce "00-c0b6b7b1d103cd383d594e9ffa128965-0000000000000000-01"
		splitRefTp := strings.Split(refTp, "-")
		if len(splitRefTp) != 4 {
			return nil, fmt.Errorf("invalid traceparent format when setting dynamic span data: %q", refTp)
		}
		splitRefTp[2] = "0000000000000000"

		spanRef.DynamicSpanTraceParent = strings.Join(splitRefTp, "-")
		spanRef.DynamicSpanTraceState = refTs
	}

	span.SetAttributes(
		attribute.String(meta.Attrs.DynamicSpanID.Key(), spanRef.DynamicSpanID),
	)

	if len(opts.Carriers) > 0 {
		byt, err := json.Marshal(spanRef)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal span metadata when injecting to carriers: %w", err)
		}

		for _, carrier := range opts.Carriers {
			carrier[meta.PropagationKey] = string(byt)
		}
	}

	return &DroppableSpan{
		span: span,
		Ref:  spanRef,
	}, nil
}

// Returns nothing, as the span is only extended and no further context is given
func (tp *otelTracerProvider) UpdateSpan(
	ctx context.Context,
	opts *UpdateSpanOptions,
) error {
	ts := opts.EndTime
	if ts.IsZero() {
		ts = time.Now()
	}

	if opts.TargetSpan == nil {
		return fmt.Errorf("no target span")
	}

	if opts.TargetSpan.DynamicSpanID == "" {
		return fmt.Errorf("target span is not dynamic; has no DynamicSpanID")
	}

	attrs := meta.NewAttrSet(
		meta.Attr(meta.Attrs.DynamicSpanID, &opts.TargetSpan.DynamicSpanID),
		meta.Attr(meta.Attrs.DynamicStatus, &opts.Status),
	)

	if opts.TargetSpan.DynamicSpanTraceParent != "" {
		splitTp := strings.Split(opts.TargetSpan.DynamicSpanTraceParent, "-")
		if len(splitTp) != 4 {
			attrs.AddErr(fmt.Errorf("invalid traceparent format when setting dynamic span data: %q", opts.TargetSpan.DynamicSpanTraceParent))
		} else {
			meta.AddAttr(attrs, meta.Attrs.DynamicTraceID, &splitTp[1])
		}
	}

	carrier := propagation.MapCarrier{
		"traceparent": opts.TargetSpan.DynamicSpanTraceParent,
		"tracestate":  opts.TargetSpan.DynamicSpanTraceState,
	}
	ctx = mixinExecutonContext(
		ctx,
		// extract the propagator from a blank contexct, and mixin the execution
		// context from the parent.  this creates a blank ctx with just the executor context
		// and propagator, which is necessary to tie parents <> children.
		defaultPropagator.Extract(context.Background(), carrier),
	)

	if opts.Status.IsEnded() {
		meta.AddAttr(attrs, meta.Attrs.EndedAt, &ts)
	}

	if opts.Debug != nil {
		if opts.Debug.Location != "" {
			meta.AddAttr(attrs, meta.Attrs.InternalLocation, &opts.Debug.Location)
		}
	}

	// Be careful to make sure that whatever attrs we specify here are
	// overwritten by whatever is given in options; the caller knows best.
	if opts.Attributes != nil {
		attrs = attrs.Merge(opts.Attributes)
	}

	spanOpts := append(
		[]trace.SpanStartOption{
			trace.WithAttributes(attrs.Serialize()...),
			trace.WithTimestamp(ts),
		},
		opts.RawOtelSpanOptions...,
	)

	tracer := tp.getTracer(opts.Metadata)
	_, span := tracer.Start(ctx, meta.SpanNameDynamicExtension, spanOpts...)

	span.End()
	return nil
}

func DeterministicSpanConfig(seed []byte) DeterministicIDs {
	sum := sha256.Sum256(seed)
	// XXX: can we not allocate here?
	r := frand.NewCustom(sum[:], 16+8, 10)
	return DeterministicIDs{
		TraceID: [16]byte(r.Bytes(16)[:16]),
		SpanID:  [8]byte(r.Bytes(8)[:8]),
	}
}

type deterministicIDKeyT = struct{}

var deterministicIDKeyV deterministicIDKeyT

func getDeterministicIDs(ctx context.Context) (DeterministicIDs, bool) {
	did, ok := ctx.Value(deterministicIDKeyV).(DeterministicIDs)
	return did, ok
}

func setDeteterministicIDs(ctx context.Context, did DeterministicIDs) context.Context {
	return context.WithValue(ctx, deterministicIDKeyV, did)
}

type DeterministicIDs struct {
	TraceID trace.TraceID
	SpanID  trace.SpanID
}

// idGenerator returns stable trace and span IDs using the SpanContext data.
//
// This does a passthrough:  if you have NOT generated a deterministic span
// via DeterministicSpanConfig, this will generate random numbers.
type idGenerator struct{}

// NewIDs returns a new trace and span ID.
func (idGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	// Does span context exist?
	if did, ok := getDeterministicIDs(ctx); ok {
		return did.TraceID, did.SpanID
	}
	tID := frand.Entropy128()
	sID := frand.Entropy128()
	return trace.TraceID([16]byte(tID)), trace.SpanID([8]byte(sID[:8]))
}

// NewSpanID returns a ID for a new span in the trace with TraceID.
func (idGenerator) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	if did, ok := getDeterministicIDs(ctx); ok {
		return did.SpanID
	}

	sID := frand.Entropy128()
	return trace.SpanID([8]byte(sID[:8]))
}
