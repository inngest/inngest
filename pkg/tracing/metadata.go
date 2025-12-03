package tracing

import (
	"context"
	"fmt"

	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
)

type MetadataSpanAttrOpts func(attr *meta.SerializableAttrs)

func CreateMetadataSpan(ctx context.Context, tracerProvider TracerProvider, parent *meta.SpanReference, location, pkgName string, stateMetadata *statev2.Metadata, spanMetadata metadata.Structured, scope metadata.Scope, opts ...MetadataSpanAttrOpts) (*meta.SpanReference, error) {
	attrs, err := MetadataAttrs(spanMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	meta.AddAttr(attrs, meta.Attrs.MetadataScope, &scope)

	for _, opt := range opts {
		opt(attrs)
	}

	kindTag := spanMetadata.Kind().String()
	if spanMetadata.Kind().IsUser() {
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
			Attributes: attrs,

			DynamicSeed: MetadataSpanIDSeed(parent.DynamicSpanID, spanMetadata.Kind()),
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
