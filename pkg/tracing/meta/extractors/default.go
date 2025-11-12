package extractors

import "github.com/inngest/inngest/pkg/tracing/meta"

var Default = meta.MetadataExtractor{
	ExtendedTrace: meta.SpanMetadataExtractors{NewAITokenExtractor()},
}
