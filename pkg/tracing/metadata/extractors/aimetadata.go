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

	// calculate latency from span duration
	if span.EndTimeUnixNano > span.StartTimeUnixNano {
		latencyMs := int64((span.EndTimeUnixNano - span.StartTimeUnixNano) / 1_000_000)
		md.LatencyMs = &latencyMs
	}

	// calculate total tokens if a provider value wasn't supplied
	if md.TotalTokens == nil && (md.InputTokens > 0 || md.OutputTokens > 0) {
		totalTokens := md.InputTokens + md.OutputTokens
		md.TotalTokens = &totalTokens
	}

	md.EstimatedCost = EstimateCost(md.RequestModel, md.InputTokens, md.OutputTokens)

	return md, true
}
