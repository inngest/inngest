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

	IDGenerator = idGenerator{}
)

// SetDeterministicIDs stores ids in ctx so IDGenerator returns them for the
// next span started against that ctx. Exported so a second TracerProvider
// implementation living outside this package (see pkg/tracing/v3) can drive
// the same IDGenerator this package uses. See setDeteterministicIDs, the
// unexported version this package's own otelTracerProvider uses directly.
func SetDeterministicIDs(ctx context.Context, ids DeterministicIDs) context.Context {
	return setDeteterministicIDs(ctx, ids)
}

// TracerProvider defines the interface for tracing providers.
type TracerProvider interface {
	CreateSpan(ctx context.Context, name string, opts *CreateSpanOptions) (*meta.SpanReference, error)
	CreateDroppableSpan(ctx context.Context, name string, opts *CreateSpanOptions) (*DroppableSpan, error)
	UpdateSpan(ctx context.Context, opts *UpdateSpanOptions) error
	// Shutdown flushes the exporter and releases the underlying
	// TracerProvider. Call it during process shutdown so pending spans are
	// written and the SpanProcessor's internal goroutines exit cleanly.
	// Safe to call on a nil receiver or an already-shutdown provider; the
	// OTel SDK makes Shutdown idempotent.
	Shutdown(ctx context.Context) error
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

	Seed   []byte
	SpanID trace.SpanID

	// DynamicSpanIDOverride, if non-empty, sets the persisted DynamicSpanID
	// attribute to exactly this value, leaving the concrete OTel span_id
	// random.
	//
	// Use this when multiple physical span rows must share one
	// logical identity but have have unique span_ids.
	DynamicSpanIDOverride string
}

type UpdateSpanOptions struct {
	Attributes         *meta.SerializableAttrs
	Debug              *SpanDebugData
	EndTime            time.Time
	EndTimeOffset      time.Duration
	Metadata           *statev2.Metadata
	QueueItem          *queue.Item
	RawOtelSpanOptions []trace.SpanStartOption
	Status             enums.StepStatus
	TargetSpan         *meta.SpanReference
}

// otelTracerProvider implements TracerProvider.
//
// Historical note (issue #3892): earlier revisions built a brand-new
// sdktrace.NewTracerProvider + SimpleSpanProcessor + executionProcessor on
// every CreateSpan / UpdateSpan call. None of them were ever Shutdown(),
// so every tenant / run leaked one TracerProvider's worth of goroutines
// and internal state. Under sustained traffic the Go runtime retained
// gigabytes of these across days (RSS 500MB → 10GB over 48h in the
// reporter's cluster).
//
// We now build the OTel TracerProvider ONCE in NewOtelTracerProvider and
// reuse the same trace.Tracer forever. Per-span metadata is injected into
// context via WithExecutionMetadata and read back by executionProcessor
// in OnStart (standard OTel pattern for per-span attribute enrichment).
type otelTracerProvider struct {
	exp sdktrace.SpanExporter
	bt  time.Duration

	otelTP *sdktrace.TracerProvider
	tracer trace.Tracer
}

func NewOtelTracerProvider(exp sdktrace.SpanExporter, batchTimeout time.Duration) TracerProvider {
	tp := &otelTracerProvider{
		exp: exp,
		bt:  batchTimeout,
	}
	// BatchSpanProcessor instead of SimpleSpanProcessor so a slow exporter
	// write (e.g. a slow Postgres InsertSpan from dbExporter) does not block
	// unrelated runs trying to finish a span. SimpleSpanProcessor holds a
	// single mutex across its synchronous export call; before this fix each
	// span had its own processor, so writes were accidentally concurrent.
	// With a shared TracerProvider (the fix for #3892) we must preserve that
	// concurrency by moving to the batched/async processor. Timeout matches
	// the original batchTimeout argument so callers see the same latency
	// bound they configured. (Greptile P1 on PR #5004.)
	var base sdktrace.SpanProcessor
	if exp != nil {
		base = sdktrace.NewBatchSpanProcessor(exp,
			sdktrace.WithBatchTimeout(batchTimeout),
		)
	}
	tp.otelTP = sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(newExecutionProcessor(nil, base)),
		sdktrace.WithIDGenerator(IDGenerator),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	tp.tracer = tp.otelTP.Tracer(
		"inngest",
		trace.WithInstrumentationVersion(version.Print()),
	)
	return tp
}

// Shutdown implements TracerProvider.Shutdown — flushes the exporter and
// releases the underlying sdktrace.TracerProvider. Wired from the devserver's
// defer block (devserver.go) so pending spans are written on process exit.
// The underlying OTel SDK makes Shutdown idempotent, so repeated calls are
// safe. Returns nil if the provider was never fully constructed.
func (tp *otelTracerProvider) Shutdown(ctx context.Context) error {
	if tp == nil || tp.otelTP == nil {
		return nil
	}
	return tp.otelTP.Shutdown(ctx)
}

// getTracer returns the single shared Tracer. The md argument is ignored
// at this layer — it reaches executionProcessor.OnStart via the context
// chain (see WithExecutionMetadata / tracer.go call sites). Kept for
// signature stability with callers that still pass opts.Metadata.
func (tp *otelTracerProvider) getTracer(_ *statev2.Metadata) trace.Tracer {
	return tp.tracer
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
		return nil, fmt.Errorf("failed to CreateSpan: %w", err)
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
		meta.AddAttrIfUnset(attrs, meta.Attrs.StartedAt, &st)
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
		meta.AddAttrIfUnset(attrs, meta.Attrs.StartedAt, &opts.StartTime)
	}
	if !opts.EndTime.IsZero() {
		meta.AddAttrIfUnset(attrs, meta.Attrs.EndedAt, &opts.EndTime)
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
				SpanContext: SpanContextFromMetadata(opts.FollowsFrom),
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
	} else if opts.SpanID.IsValid() && opts.Parent != nil {
		ctx = setDeteterministicIDs(ctx, DeterministicIDs{
			TraceID: trace.SpanContextFromContext(ctx).TraceID(),
			SpanID:  opts.SpanID,
		})
	}

	// Inject per-span metadata into context so the shared executionProcessor
	// can enrich this specific span with tenant attributes (issue #3892).
	ctx = WithExecutionMetadata(ctx, opts.Metadata)
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
	if opts.DynamicSpanIDOverride != "" {
		spanRef.DynamicSpanID = opts.DynamicSpanIDOverride
	}

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
	)
	if opts.Status != enums.StepStatusUnknown {
		meta.AddAttr(attrs, meta.Attrs.DynamicStatus, &opts.Status)
	}

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

	tsWithOffset := ts.Add(opts.EndTimeOffset)

	spanOpts := append(
		[]trace.SpanStartOption{
			trace.WithAttributes(attrs.Serialize()...),
			trace.WithTimestamp(tsWithOffset),
		},
		opts.RawOtelSpanOptions...,
	)

	// Inject per-span metadata into context so the shared executionProcessor
	// can enrich this specific span with tenant attributes (issue #3892).
	ctx = WithExecutionMetadata(ctx, opts.Metadata)
	tracer := tp.getTracer(opts.Metadata)
	_, span := tracer.Start(ctx, meta.SpanNameDynamicExtension, spanOpts...)

	span.End(trace.WithTimestamp(tsWithOffset))
	return nil
}

// DeterministicSpanID creates a new span ID based off of a deterministic seed.
// WARN: this is different from DeterministicSpanConfig.SpanID because this is the first 8 bytes while the
// DeterministicSpanConfig.SpanID is the 16-24th bytes. This was/is a mistake but we're stuck with it now
// for metadata. Don't use this for anything else new.
func DeterministicSpanID(seed []byte) trace.SpanID {
	sum := sha256.Sum256(seed)
	r := frand.NewCustom(sum[:], 8, 10)
	return trace.SpanID(r.Bytes(8))
}

// DeterministicSpanConfig creates a new span config based off of a deterministic seed.
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
