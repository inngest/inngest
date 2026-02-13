package extractors

import "github.com/inngest/inngest/pkg/tracing/metadata"

var DefaultSpanExtractors = metadata.SpanExtractors{
	NewAIMetadataExtractor(),
	NewHTTPMetadataExtractor(),
	NewResponseHeaderMetadataExtractor(),
}
