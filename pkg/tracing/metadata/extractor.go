package metadata

import (
	"context"

	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

type SpanExtractor interface {
	ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]Structured, error)
}

type SpanExtractorFunc func(context.Context, *tracev1.Span) ([]Structured, error)

func (f SpanExtractorFunc) ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]Structured, error) {
	return f(ctx, span)
}

type SpanExtractors []SpanExtractor

func (me SpanExtractors) ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]Structured, error) {
	var metadata []Structured
	for _, extractor := range me {
		subMetadata, err := extractor.ExtractSpanMetadata(ctx, span)
		if err != nil {
			return nil, err
		}

		metadata = append(metadata, subMetadata...)
	}

	return metadata, nil
}
