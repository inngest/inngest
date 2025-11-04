package meta

import (
	"context"
	"encoding/json"

	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

var DefaultMetadataExtractor = MetadataExtractor{

	ExtendedTrace: SpanMetadataExtractors{
		SpanMetadataExtractorFunc(func(ctx context.Context, span *tracev1.Span) ([]StructuredMetadata, error) {
			return []StructuredMetadata{
				AnyStructuredMetadata(
					"example",
					map[string]any{
						"key": "value",
					},
					MetadataOpMerge,
				),
			}, nil
		}),
	},
}

type RawMetadata map[string]json.RawMessage

type StructuredMetadata interface {
	Kind() MetadataKind
	Serialize() (RawMetadata, error)

	Op() MetadataOp
}

func AnyStructuredMetadata(kind MetadataKind, data any, op MetadataOp) StructuredMetadata {
	return anyStructuredMetadata{
		kind: kind,
		data: data,
		op:   op,
	}
}

type anyStructuredMetadata struct {
	kind MetadataKind
	data any
	op   MetadataOp
}

func (m anyStructuredMetadata) Kind() MetadataKind {
	return m.kind
}

func (m anyStructuredMetadata) Serialize() (RawMetadata, error) {
	b, err := json.Marshal(m.data)
	if err != nil {
		return nil, err
	}

	var raw RawMetadata
	err = json.Unmarshal(b, &raw)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func (m anyStructuredMetadata) Op() MetadataOp {
	return m.op
}

type MetadataKind string

type MetadataOp string // TODO real enum

const (
	MetadataOpMerge  MetadataOp = "merge"
	MetadataOpSet    MetadataOp = "set"
	MetadataOpDelete MetadataOp = "delete"
	MetadataOpAdd    MetadataOp = "add"
)

type SpanMetadataExtractor interface {
	ExtractMetadata(ctx context.Context, span *tracev1.Span) ([]StructuredMetadata, error)
}

type SpanMetadataExtractorFunc func(context.Context, *tracev1.Span) ([]StructuredMetadata, error)

func (f SpanMetadataExtractorFunc) ExtractMetadata(ctx context.Context, span *tracev1.Span) ([]StructuredMetadata, error) {
	return f(ctx, span)
}

type SpanMetadataExtractors []SpanMetadataExtractor

func (me SpanMetadataExtractors) ExtractMetadata(ctx context.Context, span *tracev1.Span) ([]StructuredMetadata, error) {
	var metadata []StructuredMetadata
	for _, extractor := range me {
		subMetadata, err := extractor.ExtractMetadata(ctx, span)
		if err != nil {
			return nil, err
		}

		metadata = append(metadata, subMetadata...)
	}

	return metadata, nil
}

type MetadataExtractor struct {
	ExtendedTrace SpanMetadataExtractor
}
