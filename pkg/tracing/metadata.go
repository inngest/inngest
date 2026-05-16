package tracing

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
)

type MetadataSpanAttrOpts func(cfg *MetadataSpanConfig)

type MetadataSpanConfig struct {
	Attrs *meta.SerializableAttrs
}

func CreateMetadataSpan(ctx context.Context, tracerProvider TracerProvider, parent *meta.SpanReference, location, pkgName string, stateMetadata *statev2.Metadata, spanMetadata metadata.Structured, scope metadata.Scope, opts ...MetadataSpanAttrOpts) (*meta.SpanReference, error) {
	values, err := spanMetadata.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize metadata: %w", err)
	}

	return CreateMetadataSpanFromValues(ctx, tracerProvider, parent, location, pkgName, stateMetadata, spanMetadata.Kind(), spanMetadata.Op(), values, scope, opts...)
}

// CreateMetadataSpanFromValues creates a metadata span from pre-serialized values,
// avoiding redundant serialization when the caller has already called Serialize.
func CreateMetadataSpanFromValues(ctx context.Context, tracerProvider TracerProvider, parent *meta.SpanReference, location, pkgName string, stateMetadata *statev2.Metadata, kind metadata.Kind, op metadata.Opcode, values metadata.Values, scope metadata.Scope, opts ...MetadataSpanAttrOpts) (*meta.SpanReference, error) {
	spanSize := values.Size()

	// Per-span size limit
	if spanSize > consts.MaxMetadataSpanSize {
		return nil, metadata.ErrMetadataSpanTooLarge
	}

	// Per-run cumulative size limit. Skip when stateMetadata is nil.
	//
	// TryAddMetadataSize atomically checks the limit and increments the
	// counter under a mutex, which is safe for concurrent access from
	// parallel step handlers (handleGeneratorGroup).
	if stateMetadata != nil {
		if !stateMetadata.Metrics.TryAddMetadataSize(spanSize, consts.MaxRunMetadataSize) {
			return nil, metadata.ErrRunMetadataSizeExceeded
		}
	}

	attrs := RawMetadataAttrs(kind, values, op)
	meta.AddAttr(attrs, meta.Attrs.MetadataScope, &scope)

	cfg := MetadataSpanConfig{
		Attrs: attrs,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	kindTag := kind.String()
	if kind.IsUser() {
		kindTag = fmt.Sprintf("%s*", metadata.KindPrefixUserland)
	}

	metrics.IncrMetadataSpansTotal(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"kind": kindTag,
		},
	})
	ref, err := tracerProvider.CreateSpan(
		ctx,
		meta.SpanNameMetadata,
		&CreateSpanOptions{
			Debug:      &SpanDebugData{Location: location},
			Parent:     parent,
			Metadata:   stateMetadata,
			Attributes: cfg.Attrs,

			// Set the dynamic_span_id from (parent, kind) so every
			// metadata emission of this kind under this parent aggregates together.
			DynamicSpanIDOverride: DeterministicSpanID(MetadataSpanIDSeed(parent.DynamicSpanID, kind)).String(),
		},
	)
	if err != nil {
		// Roll back the optimistic size increment on span creation failure.
		if stateMetadata != nil {
			stateMetadata.Metrics.RollbackMetadataSize(spanSize)
		}
		return nil, err
	}

	return ref, nil
}

func RawMetadataAttrs(kind metadata.Kind, values metadata.Values, op metadata.Opcode) *meta.SerializableAttrs {
	rawAttrs := meta.NewAttrSet()

	meta.AddAttr(rawAttrs, meta.Attrs.MetadataKind, &kind)
	meta.AddAttr(rawAttrs, meta.Attrs.Metadata, &values)
	meta.AddAttr(rawAttrs, meta.Attrs.MetadataOp, &op)

	return rawAttrs
}

func MetadataAttrs(metadata metadata.Structured) (*meta.SerializableAttrs, error) {
	rawMetadata, err := metadata.Serialize()
	if err != nil {
		return nil, err
	}

	return RawMetadataAttrs(metadata.Kind(), rawMetadata, metadata.Op()), nil
}

func MetadataSpanIDSeed(parentID string, kind metadata.Kind) []byte {
	return fmt.Appendf(nil, "%s-metadata-%s", parentID, kind)
}
