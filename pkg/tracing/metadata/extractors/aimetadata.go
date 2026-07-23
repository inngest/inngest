package extractors

import (
	"context"

	"github.com/inngest/inngest/pkg/tracing/metadata"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

type AIMetadataExtractor struct{}

func NewAIMetadataExtractor() *AIMetadataExtractor {
	return &AIMetadataExtractor{}
}

func (e *AIMetadataExtractor) ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]metadata.Structured, error) {
	aiMetadata, ok := e.extractAIMetadata(span)
	if !ok {
		return []metadata.Structured{}, nil
	}
	return []metadata.Structured{aiMetadata}, nil
}

// extractAIMetadata populates AIMetadata and returns ok=false when no AI
// attributes are recognized.
func (e *AIMetadataExtractor) extractAIMetadata(span *tracev1.Span) (md AIMetadata, ok bool) {
	foundAny := extractAIMetadataFromAttributes(span.Attributes, &md)
	if !foundAny {
		return md, false
	}

	var spanDurMs int64
	if span.EndTimeUnixNano > span.StartTimeUnixNano {
		spanDurMs = int64((span.EndTimeUnixNano - span.StartTimeUnixNano) / 1_000_000)
	}
	md.Enrich(AIEnrichOpts{FallbackLatencyMs: spanDurMs})

	return md, true
}
