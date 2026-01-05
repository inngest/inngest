package extractors

import (
	"context"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

const KindInngestSkip metadata.Kind = "inngest.skip"

type SkipMetadata struct {
	Reason string `json:"reason"`
}

func (m SkipMetadata) Kind() metadata.Kind { return KindInngestSkip }
func (m SkipMetadata) Op() metadata.Opcode { return enums.MetadataOpcodeSet }
func (m SkipMetadata) Serialize() (metadata.Values, error) {
	var v metadata.Values
	return v, v.FromStruct(m)
}

type SkipMetadataExtractor struct{}

func NewSkipMetadataExtractor() *SkipMetadataExtractor {
	return &SkipMetadataExtractor{}
}

func (e *SkipMetadataExtractor) ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]metadata.Structured, error) {
	for _, attr := range span.Attributes {
		if attr.Key == consts.OtelSysFunctionSkipReason {
			return []metadata.Structured{
				SkipMetadata{Reason: attr.Value.GetStringValue()},
			}, nil
		}
	}
	return nil, nil
}