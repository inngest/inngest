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
	if values.Size() > consts.MaxMetadataSpanSize {
		return nil, metadata.ErrMetadataSpanTooLarge
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
	return tracerProvider.CreateSpan(
		ctx,
		meta.SpanNameMetadata,
		&CreateSpanOptions{
			Debug:      &SpanDebugData{Location: location},
			Parent:     parent,
			Metadata:   stateMetadata,
			Attributes: cfg.Attrs,

			DynamicSeed: MetadataSpanIDSeed(parent.DynamicSpanID, kind),
		},
	)
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
